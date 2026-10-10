package services

import (
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
