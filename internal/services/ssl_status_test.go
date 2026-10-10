package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"micropanel/internal/config"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

// selfSignedPEM is a certificate that GetCertificateExpiry can parse.
func selfSignedPEM(t *testing.T, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "site.example.test"}, NotBefore: time.Now(), NotAfter: notAfter}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

type fakeNginx struct{ applied atomic.Int32 }

func (n *fakeNginx) ApplyConfig(int64) error { n.applied.Add(1); return nil }
func (n *fakeNginx) Reload() error           { return nil }

// sslFixture is an SSL service over a real database whose certbot is a
// function the test controls.
type sslFixture struct {
	svc      *SSLService
	siteRepo *repository.SiteRepository
	siteID   int64
	nginx    *fakeNginx
	calls    atomic.Int32
	certbot  func(args ...string) ([]byte, error)
}

func newSSLFixture(t *testing.T) *sslFixture {
	t.Helper()
	db := openTestDB(t)
	res, err := db.Exec(`INSERT INTO users (email, password_hash, role, is_active) VALUES ('owner@example.test', 'x', 'user', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := res.LastInsertId()
	res, err = db.Exec(`INSERT INTO sites (name, owner_id) VALUES ('site.example.test', ?)`, userID)
	if err != nil {
		t.Fatal(err)
	}
	siteID, _ := res.LastInsertId()

	f := &sslFixture{siteRepo: repository.NewSiteRepository(db), siteID: siteID, nginx: &fakeNginx{}}
	f.certbot = func(...string) ([]byte, error) { return []byte("ok"), nil }
	cert := selfSignedPEM(t, time.Now().Add(90*24*time.Hour))
	f.svc = NewSSLService(&config.Config{}, f.siteRepo, repository.NewDomainRepository(db), nil)
	f.svc.SetCommands(
		func(args ...string) ([]byte, error) { f.calls.Add(1); return f.certbot(args...) },
		func(string) ([]byte, error) { return cert, nil },
		f.nginx,
	)
	return f
}

func (f *sslFixture) site(t *testing.T) *models.Site {
	t.Helper()
	site, err := f.siteRepo.GetByID(f.siteID)
	if err != nil {
		t.Fatal(err)
	}
	return site
}

// waitStatus polls the site until its ssl_status leaves "issuing".
func (f *sslFixture) waitStatus(t *testing.T) *models.Site {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if site := f.site(t); site.SSLStatus != models.SSLStatusIssuing {
			return site
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("issuance did not finish")
	return nil
}

func TestNewSiteHasNoCertificateStatus(t *testing.T) {
	f := newSSLFixture(t)
	if got := f.site(t).SSLStatus; got != models.SSLStatusNone {
		t.Fatalf("ssl_status = %q, want none", got)
	}
}

func TestAsyncIssuanceRunsInTheBackgroundOnce(t *testing.T) {
	f := newSSLFixture(t)
	release := make(chan struct{})
	f.certbot = func(...string) ([]byte, error) { <-release; return []byte("ok"), nil }

	started, err := f.svc.IssueCertificateForDomainsAsync(f.siteID, []string{"site.example.test"})
	if err != nil || !started {
		t.Fatalf("start = %v, %v; want started", started, err)
	}
	if got := f.site(t).SSLStatus; got != models.SSLStatusIssuing {
		t.Fatalf("while running: ssl_status = %q, want issuing", got)
	}
	again, err := f.svc.IssueCertificateForDomainsAsync(f.siteID, []string{"site.example.test"})
	if err != nil || again {
		t.Fatalf("second start = %v, %v; want it to join the running issuance", again, err)
	}

	close(release)
	site := f.waitStatus(t)
	if site.SSLStatus != models.SSLStatusIssued || site.SSLError != "" || !site.SSLEnabled || site.SSLExpiresAt == nil {
		t.Fatalf("after issuance: %+v, want issued with an expiry", site)
	}
	if n := f.calls.Load(); n != 1 {
		t.Errorf("certbot ran %d times, want 1", n)
	}
	if f.nginx.applied.Load() != 1 {
		t.Errorf("nginx config applied %d times, want 1", f.nginx.applied.Load())
	}
}

func TestAsyncIssuanceFailureKeepsTheReason(t *testing.T) {
	f := newSSLFixture(t)
	f.certbot = func(...string) ([]byte, error) {
		return nil, errors.New(strings.Repeat("x", 3000) + " Challenge failed for domain site.example.test")
	}

	if _, err := f.svc.IssueCertificateForDomainsAsync(f.siteID, []string{"site.example.test"}); err != nil {
		t.Fatal(err)
	}
	site := f.waitStatus(t)
	if site.SSLStatus != models.SSLStatusFailed || site.SSLEnabled {
		t.Fatalf("after failure: %+v, want failed without SSL", site)
	}
	if !strings.HasSuffix(site.SSLError, "Challenge failed for domain site.example.test") || len(site.SSLError) > maxSSLErrorLength+3 {
		t.Errorf("ssl_error = %d bytes ending %q, want the tail of the reason", len(site.SSLError), site.SSLError[max(0, len(site.SSLError)-50):])
	}

	// A new request after a failure starts a new issuance.
	f.certbot = func(...string) ([]byte, error) { return []byte("ok"), nil }
	if started, err := f.svc.IssueCertificateForDomainsAsync(f.siteID, []string{"site.example.test"}); err != nil || !started {
		t.Fatalf("retry = %v, %v; want started", started, err)
	}
	if site := f.waitStatus(t); site.SSLStatus != models.SSLStatusIssued {
		t.Fatalf("retry: ssl_status = %q, want issued", site.SSLStatus)
	}
}

func TestSyncIssuanceRecordsItsOutcome(t *testing.T) {
	f := newSSLFixture(t)
	if err := f.svc.IssueCertificateForDomains(f.siteID, []string{"site.example.test"}); err != nil {
		t.Fatal(err)
	}
	if got := f.site(t).SSLStatus; got != models.SSLStatusIssued {
		t.Fatalf("ssl_status = %q, want issued", got)
	}
}

func TestRecoverInterruptedIssuance(t *testing.T) {
	f := newSSLFixture(t)
	if ok, err := f.siteRepo.BeginSSLIssuance(f.siteID); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if n, err := f.svc.RecoverInterrupted(); err != nil || n != 1 {
		t.Fatalf("recover = %d, %v; want 1", n, err)
	}
	site := f.site(t)
	if site.SSLStatus != models.SSLStatusFailed || site.SSLError != interruptedSSLReason {
		t.Fatalf("after recover: %+v", site)
	}
}

func TestSiteUpdateKeepsTheCertificateStatus(t *testing.T) {
	f := newSSLFixture(t)
	stale := f.site(t) // loaded before the issuance finished
	if err := f.siteRepo.SetSSLStatus(f.siteID, models.SSLStatusIssued, ""); err != nil {
		t.Fatal(err)
	}
	stale.FixMimeTypes = true
	if err := f.siteRepo.Update(stale); err != nil {
		t.Fatal(err)
	}
	if got := f.site(t).SSLStatus; got != models.SSLStatusIssued {
		t.Fatalf("ssl_status = %q after a stale save, want issued", got)
	}
}
