package handlers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"micropanel/internal/config"
	"micropanel/internal/database"
	"micropanel/internal/middleware"
	"micropanel/internal/models"
	"micropanel/internal/repository"
	"micropanel/internal/services"
)

const (
	ownerToken    = "owner-token"
	strangerToken = "stranger-token"
)

// apiFixture is the /api/v1 router over a migrated SQLite database with a
// site owned by the owner token's user and a second, unrelated user.
type apiFixture struct {
	router     *gin.Engine
	deployRepo *repository.DeployRepository
	siteID     int64
	ownerID    int64
	// certbot stands in for the certbot command; the default succeeds.
	certbot func(args ...string) ([]byte, error)
}

type nopNginx struct{}

func (nopNginx) ApplyConfig(int64) error { return nil }
func (nopNginx) Reload() error           { return nil }

func newAPIFixture(t *testing.T) *apiFixture {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "micropanel.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	migrations, _ := filepath.Abs("../../migrations")
	if err := db.Migrate(migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	insertUser := func(email string) int64 {
		res, err := db.Exec(`INSERT INTO users (email, password_hash, role, is_active) VALUES (?, 'x', 'user', 1)`, email)
		if err != nil {
			t.Fatalf("insert user: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	ownerID := insertUser("owner@example.test")
	strangerID := insertUser("stranger@example.test")
	res, err := db.Exec(`INSERT INTO sites (name, owner_id) VALUES ('site.example.test', ?)`, ownerID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
	siteID, _ := res.LastInsertId()

	cfg := &config.Config{}
	cfg.Sites.Path = t.TempDir()
	siteRepo := repository.NewSiteRepository(db)
	domainRepo := repository.NewDomainRepository(db)
	deployRepo := repository.NewDeployRepository(db)
	userRepo := repository.NewUserRepository(db)
	f := &apiFixture{deployRepo: deployRepo, siteID: siteID, ownerID: ownerID}
	f.certbot = func(...string) ([]byte, error) { return []byte("ok"), nil }
	sslService := services.NewSSLService(cfg, siteRepo, domainRepo, nil)
	sslService.SetCommands(
		func(args ...string) ([]byte, error) { return f.certbot(args...) },
		func(string) ([]byte, error) { return nil, errors.New("no certificate in tests") },
		nopNginx{},
	)
	h := NewAPIHandler(
		services.NewSiteService(siteRepo, domainRepo, cfg),
		services.NewDeployService(cfg, deployRepo, siteRepo),
		nil, sslService,
		services.NewAuditService(repository.NewAuditRepository(db)),
		domainRepo, userRepo,
	)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.APIToken([]config.APIToken{
		{Name: "owner", Token: ownerToken, UserID: ownerID},
		{Name: "stranger", Token: strangerToken, UserID: strangerID},
	}, nil))
	api.GET("/sites/:id", h.GetSite)
	api.POST("/sites/:id/deploy", h.Deploy)
	api.GET("/sites/:id/deploys", h.ListDeploys)
	api.POST("/sites/:id/ssl", h.IssueSSL)
	f.router = r
	return f
}

func (f *apiFixture) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func archiveWithIndex(t *testing.T, index string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "index.html", Mode: 0644, Size: int64(len(index)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte(index))
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// deploy posts an archive; key == nil sends no Idempotency-Key header.
func (f *apiFixture) deploy(t *testing.T, token string, key *string, index string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "site.tar.gz")
	part.Write(archiveWithIndex(t, index))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/sites/%d/deploy", f.siteID), &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	if key != nil {
		req.Header.Set("Idempotency-Key", *key)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *apiFixture) list(t *testing.T, token, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sites/%d/deploys%s", f.siteID, query), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return v
}

func ptr(s string) *string { return &s }

func TestAPIDeployReplaysAKeyedDeploy(t *testing.T) {
	f := newAPIFixture(t)

	first := f.deploy(t, ownerToken, ptr("publication-1"), "A")
	if first.Code != http.StatusOK {
		t.Fatalf("first = %d %s", first.Code, first.Body)
	}
	got := decode[map[string]any](t, first)
	if _, ok := got["replayed"]; ok {
		t.Errorf("first deploy reports replayed: %v", got)
	}

	again := f.deploy(t, ownerToken, ptr("publication-1"), "B")
	if again.Code != http.StatusOK {
		t.Fatalf("replay = %d %s", again.Code, again.Body)
	}
	replay := decode[deployResponse](t, again)
	if !replay.Replayed || replay.DeployID != int64(got["deploy_id"].(float64)) || replay.Status != "success" {
		t.Fatalf("replay = %+v, want the first deploy, replayed", replay)
	}
	if n, _ := f.deployRepo.CountBySite(f.siteID); n != 1 {
		t.Errorf("deploys = %d, want 1", n)
	}
}

func TestAPIDeployWithARunningKeyConflicts(t *testing.T) {
	f := newAPIFixture(t)
	running := &models.Deploy{SiteID: f.siteID, UserID: f.ownerID, Filename: "site.tar.gz", Status: models.DeployStatusPending, IdempotencyKey: "publication-1"}
	if err := f.deployRepo.Create(running); err != nil {
		t.Fatal(err)
	}

	w := f.deploy(t, ownerToken, ptr("publication-1"), "A")
	if w.Code != http.StatusConflict {
		t.Fatalf("code = %d %s, want 409", w.Code, w.Body)
	}
	if got := decode[map[string]any](t, w); int64(got["deploy_id"].(float64)) != running.ID {
		t.Errorf("409 body = %v, want deploy_id %d", got, running.ID)
	}
}

func TestAPIDeployRejectsABadKey(t *testing.T) {
	f := newAPIFixture(t)
	for _, key := range []string{"", "has space", string(bytes.Repeat([]byte("k"), 256)), "ключ"} {
		if w := f.deploy(t, ownerToken, ptr(key), "A"); w.Code != http.StatusBadRequest {
			t.Errorf("key %q: code = %d, want 400", key, w.Code)
		}
	}
	if n, _ := f.deployRepo.CountBySite(f.siteID); n != 0 {
		t.Errorf("deploys = %d, want 0", n)
	}
}

func TestAPIDeployWithoutAKeyIsUnchanged(t *testing.T) {
	f := newAPIFixture(t)
	for range 2 {
		if w := f.deploy(t, ownerToken, nil, "A"); w.Code != http.StatusOK {
			t.Fatalf("code = %d %s", w.Code, w.Body)
		}
	}
	if n, _ := f.deployRepo.CountBySite(f.siteID); n != 2 {
		t.Errorf("deploys = %d, want 2", n)
	}
}

func TestAPIListDeploys(t *testing.T) {
	f := newAPIFixture(t)
	for _, key := range []string{"k1", "k2", "k3"} {
		if w := f.deploy(t, ownerToken, ptr(key), key); w.Code != http.StatusOK {
			t.Fatalf("deploy %s = %d", key, w.Code)
		}
	}

	all := decode[[]deployListItem](t, f.list(t, ownerToken, ""))
	if len(all) != 3 || all[0].IdempotencyKey != "k3" || all[0].Status != "success" || all[0].CreatedAt == "" {
		t.Fatalf("list = %+v, want three, newest first", all)
	}
	if two := decode[[]deployListItem](t, f.list(t, ownerToken, "?limit=2")); len(two) != 2 {
		t.Errorf("limit=2 gave %d", len(two))
	}
	one := decode[[]deployListItem](t, f.list(t, ownerToken, "?idempotency_key=k1"))
	if len(one) != 1 || one[0].IdempotencyKey != "k1" {
		t.Errorf("by key = %+v, want k1", one)
	}
	if none := decode[[]deployListItem](t, f.list(t, ownerToken, "?idempotency_key=missing")); len(none) != 0 {
		t.Errorf("unknown key = %+v, want empty", none)
	}
	if w := f.list(t, ownerToken, "?limit=0"); w.Code != http.StatusBadRequest {
		t.Errorf("limit=0: code = %d, want 400", w.Code)
	}
	if w := f.list(t, strangerToken, ""); w.Code != http.StatusForbidden {
		t.Errorf("stranger: code = %d, want 403", w.Code)
	}
}

func TestAPIAsyncCertificateAnswers202AndReportsTheStatus(t *testing.T) {
	f := newAPIFixture(t)
	release := make(chan struct{})
	f.certbot = func(...string) ([]byte, error) { <-release; return []byte("ok"), nil }
	sitePath := fmt.Sprintf("/api/v1/sites/%d", f.siteID)

	w := f.do(t, http.MethodPost, sitePath+"/ssl", ownerToken, `{"mode":"primary","async":true}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("code = %d %s, want 202", w.Code, w.Body)
	}
	if got := decode[siteResponse](t, w); got.SSLStatus != "issuing" {
		t.Fatalf("202 body = %+v, want ssl_status issuing", got)
	}
	if again := f.do(t, http.MethodPost, sitePath+"/ssl", ownerToken, `{"mode":"primary","async":true}`); again.Code != http.StatusAccepted {
		t.Fatalf("repeat = %d, want 202", again.Code)
	}

	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := decode[siteResponse](t, f.do(t, http.MethodGet, sitePath, ownerToken, ""))
		if got.SSLStatus == "issued" {
			break
		}
		if got.SSLStatus != "issuing" || time.Now().After(deadline) {
			t.Fatalf("site = %+v, want issued", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAPIAsyncCertificateFailureIsVisible(t *testing.T) {
	f := newAPIFixture(t)
	f.certbot = func(...string) ([]byte, error) { return nil, errors.New("certbot command failed: Challenge failed") }
	sitePath := fmt.Sprintf("/api/v1/sites/%d", f.siteID)

	if w := f.do(t, http.MethodPost, sitePath+"/ssl", ownerToken, `{"async":true}`); w.Code != http.StatusAccepted {
		t.Fatalf("code = %d, want 202", w.Code)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := decode[siteResponse](t, f.do(t, http.MethodGet, sitePath, ownerToken, ""))
		if got.SSLStatus == "failed" {
			if got.SSLError != "certbot command failed: Challenge failed" {
				t.Errorf("ssl_error = %q", got.SSLError)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("site = %+v, want failed", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAPISyncCertificateIsUnchanged(t *testing.T) {
	f := newAPIFixture(t)
	w := f.do(t, http.MethodPost, fmt.Sprintf("/api/v1/sites/%d/ssl", f.siteID), ownerToken, `{"mode":"primary"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d %s, want 200", w.Code, w.Body)
	}
	if got := decode[siteResponse](t, w); got.SSLStatus != "issued" || !got.SSLEnabled {
		t.Fatalf("body = %+v, want issued", got)
	}

	f.certbot = func(...string) ([]byte, error) { return nil, errors.New("certbot command failed") }
	if w := f.do(t, http.MethodPost, fmt.Sprintf("/api/v1/sites/%d/ssl", f.siteID), ownerToken, `{}`); w.Code != http.StatusInternalServerError {
		t.Fatalf("failing certbot: code = %d, want 500", w.Code)
	}
}
