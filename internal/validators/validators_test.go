package validators

import (
	"strings"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		domain  string
		wantErr bool
	}{
		// Valid domains
		{"example.com", false},
		{"sub.example.com", false},
		{"my-site.example.org", false},
		{"a.co", false},
		{"test123.example.com", false},

		// Invalid domains
		{"", true},
		{"localhost", true},
		{"-example.com", true},
		{"example-.com", true},
		{"example.com;", true},
		{"example.com\n", true},
		{"example.com\r", true},
		{"example.com'", true},
		{"example.com\"", true},
		{"example.com`", true},
		{"example.com$var", true},
		{"example.com{}", true},
		{strings.Repeat("a", 254) + ".com", true}, // too long
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain(%q) error = %v, wantErr %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		// Valid paths
		{"/", false},
		{"/admin", false},
		{"/api/v1", false},
		{"/path/to/file.html", false},
		{"/my-path_123", false},

		// Invalid paths
		{"", true},
		{"admin", true},         // no leading /
		{"/path;injection", true},
		{"/path\ninjection", true},
		{"/path\rinjection", true},
		{"/path'injection", true},
		{"/path\"injection", true},
		{"/path/../etc/passwd", true},
		{"/path with spaces", true},
		{"/path$var", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRedirectURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		// Valid URLs
		{"https://example.com", false},
		{"https://example.com/path", false},
		{"http://example.com", false},
		{"/relative/path", false},
		{"https://example.com/path?query=1", false},

		// Invalid URLs
		{"", true},
		{"https://example.com;injection", true},
		{"https://example.com\ninjection", true},
		{"ftp://example.com", true},
		{"javascript:alert(1)", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := ValidateRedirectURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRedirectURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAuthRealm(t *testing.T) {
	tests := []struct {
		realm   string
		wantErr bool
	}{
		// Valid realms
		{"Admin Area", false},
		{"Private Zone", false},
		{"my-realm_123", false},
		{"Restricted", false},

		// Invalid realms
		{"", true},
		{"realm;injection", true},
		{"realm\ninjection", true},
		{"realm\"injection", true},
		{"realm'injection", true},
		{strings.Repeat("a", 129), true}, // too long
	}

	for _, tt := range tests {
		t.Run(tt.realm, func(t *testing.T) {
			err := ValidateAuthRealm(tt.realm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthRealm(%q) error = %v, wantErr %v", tt.realm, err, tt.wantErr)
			}
		})
	}
}

// TestNginxConfigInjection tests various nginx config injection attempts
func TestNginxConfigInjection(t *testing.T) {
	// These payloads attempt to break out of nginx config context
	injectionPayloads := []string{
		"example.com; location /evil { }",
		"example.com\nserver { listen 8080; }",
		"example.com\r\nlocation /pwned { }",
		"example.com' or '1'='1",
		`example.com"; return 200 "pwned`,
		"example.com`id`",
		"example.com${PATH}",
		"example.com{return 200;}",
		"example.com\\nserver{}",
	}

	t.Run("domain injection", func(t *testing.T) {
		for _, payload := range injectionPayloads {
			if err := ValidateDomain(payload); err == nil {
				t.Errorf("ValidateDomain should reject injection payload: %q", payload)
			}
		}
	})

	t.Run("path injection", func(t *testing.T) {
		pathPayloads := []string{
			"/admin; return 200 pwned;",
			"/admin\nlocation /evil { }",
			"/admin' or '1'='1",
			`/admin"; } server { listen 8080;`,
			"/admin`whoami`",
			"/admin$request_uri",
			"/admin{return 200;}",
		}
		for _, payload := range pathPayloads {
			if err := ValidatePath(payload); err == nil {
				t.Errorf("ValidatePath should reject injection payload: %q", payload)
			}
		}
	})

	t.Run("redirect URL injection", func(t *testing.T) {
		urlPayloads := []string{
			"https://evil.com; return 200 pwned",
			"https://evil.com\nX-Injected: true",
			"https://evil.com\r\nSet-Cookie: admin=true",
			"javascript:alert(document.cookie)",
			"data:text/html,<script>alert(1)</script>",
			"vbscript:msgbox(1)",
			"https://evil.com`id`",
			"https://evil.com{malicious}",
		}
		for _, payload := range urlPayloads {
			if err := ValidateRedirectURL(payload); err == nil {
				t.Errorf("ValidateRedirectURL should reject injection payload: %q", payload)
			}
		}
	})

	t.Run("auth realm injection", func(t *testing.T) {
		realmPayloads := []string{
			`Restricted"; auth_basic off; #`,
			"Admin\nauth_basic off;",
			"Admin; deny all;",
			"Admin`id`",
			"Admin${PATH}",
		}
		for _, payload := range realmPayloads {
			if err := ValidateAuthRealm(payload); err == nil {
				t.Errorf("ValidateAuthRealm should reject injection payload: %q", payload)
			}
		}
	})
}

func TestValidateHtpasswdUsername(t *testing.T) {
	tests := []struct {
		username string
		wantErr  bool
	}{
		// Valid usernames
		{"admin", false},
		{"user123", false},
		{"john_doe", false},
		{"test-user", false},
		{"user.name", false},

		// Invalid usernames
		{"", true},
		{"user:name", true},       // colon is htpasswd delimiter
		{"user\nname", true},      // newline breaks format
		{"user\rname", true},      // carriage return
		{"user name", true},       // space not allowed
		{"user;name", true},       // special chars
		{"user'name", true},       // quotes
		{strings.Repeat("a", 256), true}, // too long
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			err := ValidateHtpasswdUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHtpasswdUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

// TestXSSPayloads tests that XSS payloads are rejected
func TestXSSPayloads(t *testing.T) {
	xssPayloads := []string{
		"javascript:alert(1)",
		"javascript:alert(document.cookie)",
		"JAVASCRIPT:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"DATA:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
	}

	for _, payload := range xssPayloads {
		t.Run(payload, func(t *testing.T) {
			if err := ValidateRedirectURL(payload); err == nil {
				t.Errorf("ValidateRedirectURL should reject XSS payload: %q", payload)
			}
		})
	}
}
