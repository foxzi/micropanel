package config

import (
	"os"
	"testing"
)

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{App: AppConfig{Env: tt.env}}
			if got := cfg.IsDevelopment(); got != tt.expected {
				t.Errorf("IsDevelopment() = %v, want %v for env %q", got, tt.expected, tt.env)
			}
		})
	}
}

func TestConfig_ValidateAPIToken(t *testing.T) {
	cfg := &Config{
		API: APIConfig{
			Enabled: true,
			Tokens: []APIToken{
				{Name: "bot1", Token: "secret-token-1"},
				{Name: "bot2", Token: "secret-token-2"},
			},
		},
	}

	tests := []struct {
		token    string
		expected string // expected token name, empty if nil
	}{
		{"secret-token-1", "bot1"},
		{"secret-token-2", "bot2"},
		{"invalid-token", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := cfg.ValidateAPIToken(tt.token)
			if tt.expected == "" {
				if result != nil {
					t.Errorf("ValidateAPIToken(%q) = %v, want nil", tt.token, result)
				}
			} else {
				if result == nil {
					t.Errorf("ValidateAPIToken(%q) = nil, want %q", tt.token, tt.expected)
				} else if result.Name != tt.expected {
					t.Errorf("ValidateAPIToken(%q).Name = %q, want %q", tt.token, result.Name, tt.expected)
				}
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	// Clear any environment variables that might affect the test
	envVars := []string{"APP_ENV", "APP_PORT", "APP_SECRET", "DB_PATH", "SITES_PATH"}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults
	if cfg.App.Host != "127.0.0.1" {
		t.Errorf("Default Host = %q, want %q", cfg.App.Host, "127.0.0.1")
	}
	if cfg.App.Port != 8080 {
		t.Errorf("Default Port = %d, want %d", cfg.App.Port, 8080)
	}
	if cfg.Sites.User != "micropanel" {
		t.Errorf("Default Sites.User = %q, want %q", cfg.Sites.User, "micropanel")
	}
	if cfg.Sites.Group != "micropanel" {
		t.Errorf("Default Sites.Group = %q, want %q", cfg.Sites.Group, "micropanel")
	}
	if cfg.Limits.MaxZipSize != 100*1024*1024 {
		t.Errorf("Default MaxZipSize = %d, want %d", cfg.Limits.MaxZipSize, 100*1024*1024)
	}
	if cfg.API.Enabled != false {
		t.Errorf("Default API.Enabled = %v, want %v", cfg.API.Enabled, false)
	}
}

func TestConfig_EnvOverride(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	os.Setenv("APP_PORT", "9000")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("APP_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Env != "production" {
		t.Errorf("App.Env = %q, want %q", cfg.App.Env, "production")
	}
	if cfg.App.Port != 9000 {
		t.Errorf("App.Port = %d, want %d", cfg.App.Port, 9000)
	}
}
