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
