package services

import (
	"micropanel/internal/models"
	"micropanel/internal/repository"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"micropanel/internal/config"
)

// recordCommands replaces the commands the nginx service runs with `true`
// and records them.
func recordCommands(s *NginxService) *[]string {
	var calls []string
	s.command = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return exec.Command("true")
	}
	return &calls
}

func TestRemoveAndReloadStopsServingTheSite(t *testing.T) {
	cfg := &config.Config{}
	cfg.Nginx.ConfigPath = "/etc/nginx/sites-enabled"
	s := NewNginxService(cfg, nil, nil)
	calls := recordCommands(s)

	if err := s.RemoveAndReload(7); err != nil {
		t.Fatalf("RemoveAndReload: %v", err)
	}
	// Removing the file alone leaves the site served from nginx's memory:
	// nginx must be restarted after the removal.
	want := []string{
		"sudo rm -f /etc/nginx/sites-enabled/panel-7.conf",
		"sudo nginx -t",
		"sudo systemctl restart nginx",
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("commands = %q, want %q", *calls, want)
	}
}

// rootRedirectConfig generates the config of a site holding one redirect.
func rootRedirectConfig(t *testing.T, sslEnabled, enabled bool) string {
	t.Helper()
	db := openTestDB(t)
	res, err := db.Exec(`INSERT INTO users (email, password_hash, role, is_active) VALUES ('owner@example.test', 'x', 'user', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := res.LastInsertId()
	res, err = db.Exec(`INSERT INTO sites (name, owner_id, ssl_enabled) VALUES ('second.example.test', ?, ?)`, userID, sslEnabled)
	if err != nil {
		t.Fatal(err)
	}
	siteID, _ := res.LastInsertId()

	cfg := &config.Config{}
	cfg.Sites.Path = t.TempDir()
	redirectRepo := repository.NewRedirectRepository(db)
	s := NewNginxService(cfg, repository.NewSiteRepository(db), repository.NewDomainRepository(db))
	s.SetRedirectRepo(redirectRepo)
	if err := redirectRepo.Create(&models.Redirect{SiteID: siteID, SourcePath: "/", TargetURL: "https://main.example.test", Code: 301, PreservePath: true, PreserveQuery: true, IsEnabled: enabled}); err != nil {
		t.Fatal(err)
	}
	out, err := s.GenerateConfig(siteID)
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}
	return out
}

func TestRootRedirectReplacesTheDefaultLocation(t *testing.T) {
	for _, ssl := range []bool{false, true} {
		out := rootRedirectConfig(t, ssl, true)
		// nginx rejects the same location twice, so the static fallback goes;
		// the port 80 server of an SSL site has its own `location /`.
		want := 1
		if ssl {
			want = 2
		}
		if n := strings.Count(out, "location / {"); n != want {
			t.Fatalf("ssl=%v: %d `location / {` blocks, want %d:\n%s", ssl, n, want, out)
		}
		if !strings.Contains(out, "return 301 https://main.example.test$uri$is_args$args;") {
			t.Fatalf("ssl=%v: no 301 to the primary domain:\n%s", ssl, out)
		}
		if strings.Contains(out, "try_files $uri $uri/ =404;") {
			t.Fatalf("ssl=%v: static fallback kept:\n%s", ssl, out)
		}
		if !strings.Contains(out, "location ^~ /.well-known/acme-challenge/") {
			t.Fatalf("ssl=%v: ACME challenge location lost:\n%s", ssl, out)
		}
	}
}

func TestDisabledRootRedirectKeepsTheDefaultLocation(t *testing.T) {
	out := rootRedirectConfig(t, false, false)
	if !strings.Contains(out, "try_files $uri $uri/ =404;") {
		t.Fatalf("static fallback missing:\n%s", out)
	}
	if strings.Contains(out, "return 301 https://main.example.test") {
		t.Fatalf("disabled redirect rendered:\n%s", out)
	}
}
