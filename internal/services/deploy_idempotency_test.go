package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"micropanel/internal/config"
	"micropanel/internal/database"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

// openTestDB returns a migrated SQLite database in a temporary directory.
func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "micropanel.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	migrations, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// deployFixture is a deploy service over a real database with one user and
// one site.
type deployFixture struct {
	svc        *DeployService
	deployRepo *repository.DeployRepository
	sitesPath  string
	siteID     int64
	userID     int64
}

func newDeployFixture(t *testing.T) *deployFixture {
	t.Helper()
	db := openTestDB(t)
	res, err := db.Exec(`INSERT INTO users (email, password_hash, role, is_active) VALUES ('owner@example.test', 'x', 'user', 1)`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, _ := res.LastInsertId()
	res, err = db.Exec(`INSERT INTO sites (name, owner_id) VALUES ('site.example.test', ?)`, userID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
	siteID, _ := res.LastInsertId()

	cfg := &config.Config{}
	cfg.Sites.Path = t.TempDir()
	deployRepo := repository.NewDeployRepository(db)
	return &deployFixture{
		svc:        NewDeployService(cfg, deployRepo, repository.NewSiteRepository(db)),
		deployRepo: deployRepo,
		sitesPath:  cfg.Sites.Path,
		siteID:     siteID,
		userID:     userID,
	}
}

// tarGz builds a .tar.gz archive with a single index.html.
func tarGz(t *testing.T, index string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "index.html", Mode: 0644, Size: int64(len(index)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(index)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (f *deployFixture) deploy(t *testing.T, key, index string) (*models.Deploy, bool, error) {
	t.Helper()
	archive := tarGz(t, index)
	return f.svc.DeployWithKey(f.siteID, f.userID, key, "site.tar.gz", bytes.NewReader(archive), int64(len(archive)))
}

func (f *deployFixture) read(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.sitesPath, fmt.Sprint(f.siteID), dir, "index.html"))
	if err != nil {
		return ""
	}
	return string(b)
}

func TestDeployWithKeyReplaysASuccessfulDeploy(t *testing.T) {
	f := newDeployFixture(t)
	if _, _, err := f.deploy(t, "key-a", "A"); err != nil {
		t.Fatalf("deploy A: %v", err)
	}
	first, replayed, err := f.deploy(t, "key-b", "B")
	if err != nil || replayed {
		t.Fatalf("deploy B: replayed=%v err=%v", replayed, err)
	}

	// The response to B was lost; the client retries with the same key.
	again, replayed, err := f.deploy(t, "key-b", "C")
	if err != nil {
		t.Fatalf("replay B: %v", err)
	}
	if !replayed || again.ID != first.ID || again.Status != models.DeployStatusSuccess {
		t.Fatalf("replay = %+v replayed=%v, want deploy %d success replayed", again, replayed, first.ID)
	}
	if got := f.read(t, "public"); got != "B" {
		t.Errorf("public = %q, want B: a replay must not unpack the archive", got)
	}
	if got := f.read(t, "public_prev"); got != "A" {
		t.Errorf("public_prev = %q, want A: a replay must not overwrite the previous version", got)
	}
	if n, _ := f.deployRepo.CountBySite(f.siteID); n != 2 {
		t.Errorf("deploys = %d, want 2", n)
	}
}

func TestDeployWithKeyInProgressConflicts(t *testing.T) {
	f := newDeployFixture(t)
	running := &models.Deploy{SiteID: f.siteID, UserID: f.userID, Filename: "site.tar.gz", Status: models.DeployStatusPending, IdempotencyKey: "key-a"}
	if err := f.deployRepo.Create(running); err != nil {
		t.Fatal(err)
	}

	got, _, err := f.deploy(t, "key-a", "A")
	if !errors.Is(err, ErrDeployInProgress) {
		t.Fatalf("err = %v, want ErrDeployInProgress", err)
	}
	if got == nil || got.ID != running.ID {
		t.Fatalf("deploy = %+v, want the running deploy %d", got, running.ID)
	}
	if f.read(t, "public") != "" {
		t.Error("an in-progress key must not unpack the archive")
	}
}

func TestDeployWithKeyRetriesAFailedDeploy(t *testing.T) {
	f := newDeployFixture(t)
	broken := []byte("not a gzip stream")
	failed, _, err := f.svc.DeployWithKey(f.siteID, f.userID, "key-a", "site.tar.gz", bytes.NewReader(broken), int64(len(broken)))
	if err == nil || failed.Status != models.DeployStatusFailed {
		t.Fatalf("broken archive: deploy=%+v err=%v, want failed", failed, err)
	}

	retried, replayed, err := f.deploy(t, "key-a", "A")
	if err != nil || replayed {
		t.Fatalf("retry: replayed=%v err=%v", replayed, err)
	}
	if retried.ID != failed.ID || retried.Status != models.DeployStatusSuccess {
		t.Fatalf("retry = %+v, want deploy %d success", retried, failed.ID)
	}
	if got := f.read(t, "public"); got != "A" {
		t.Errorf("public = %q, want A", got)
	}
	if n, _ := f.deployRepo.CountBySite(f.siteID); n != 1 {
		t.Errorf("deploys = %d, want 1: a retried key reuses its record", n)
	}
}

func TestDeployWithoutKeyKeepsTheOldBehaviour(t *testing.T) {
	f := newDeployFixture(t)
	for _, index := range []string{"A", "A"} {
		if _, replayed, err := f.deploy(t, "", index); err != nil || replayed {
			t.Fatalf("deploy: replayed=%v err=%v", replayed, err)
		}
	}
	if n, _ := f.deployRepo.CountBySite(f.siteID); n != 2 {
		t.Errorf("deploys = %d, want 2: without a key every request is a new deploy", n)
	}
}

func TestRecoverInterruptedDeploysFreesTheirKeys(t *testing.T) {
	f := newDeployFixture(t)
	stuck := &models.Deploy{SiteID: f.siteID, UserID: f.userID, Filename: "site.tar.gz", Status: models.DeployStatusPending, IdempotencyKey: "key-a"}
	if err := f.deployRepo.Create(stuck); err != nil {
		t.Fatal(err)
	}

	n, err := f.svc.RecoverInterrupted()
	if err != nil || n != 1 {
		t.Fatalf("recover = %d, %v; want 1", n, err)
	}
	after, err := f.deployRepo.GetByID(stuck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != models.DeployStatusFailed || after.ErrorMessage == "" {
		t.Fatalf("after recover = %+v, want failed with a reason", after)
	}
	if _, _, err := f.deploy(t, "key-a", "A"); err != nil {
		t.Fatalf("retry after recover: %v", err)
	}
}

func TestDeploysOfOneSiteAreSerialised(t *testing.T) {
	f := newDeployFixture(t)
	const n = 16
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, errs[i] = f.deploy(t, "", "same")
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("deploy %d: %v", i, err)
		}
	}
	if got := f.read(t, "public"); got != "same" {
		t.Errorf("public = %q, want same", got)
	}
}

func TestListDeploysByKey(t *testing.T) {
	f := newDeployFixture(t)
	for _, key := range []string{"key-a", "key-b"} {
		if _, _, err := f.deploy(t, key, key); err != nil {
			t.Fatal(err)
		}
	}
	all, err := f.svc.ListDeploysFiltered(f.siteID, "", 10)
	if err != nil || len(all) != 2 {
		t.Fatalf("all = %d, %v; want 2", len(all), err)
	}
	if all[0].IdempotencyKey != "key-b" {
		t.Errorf("newest first: got %q, want key-b", all[0].IdempotencyKey)
	}
	one, err := f.svc.ListDeploysFiltered(f.siteID, "key-a", 10)
	if err != nil || len(one) != 1 || one[0].IdempotencyKey != "key-a" {
		t.Fatalf("by key = %+v, %v; want only key-a", one, err)
	}
}
