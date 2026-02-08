package models

import (
	"testing"
	"time"
)

func TestSite_GetAllHostnames(t *testing.T) {
	tests := []struct {
		name     string
		site     Site
		expected []string
	}{
		{
			name: "primary only",
			site: Site{
				Name:     "example.com",
				WWWAlias: false,
				Aliases:  nil,
			},
			expected: []string{"example.com"},
		},
		{
			name: "with www alias",
			site: Site{
				Name:     "example.com",
				WWWAlias: true,
				Aliases:  nil,
			},
			expected: []string{"example.com", "www.example.com"},
		},
		{
			name: "with additional aliases",
			site: Site{
				Name:     "example.com",
				WWWAlias: false,
				Aliases: []Domain{
					{Hostname: "alias1.com"},
					{Hostname: "alias2.com"},
				},
			},
			expected: []string{"example.com", "alias1.com", "alias2.com"},
		},
		{
			name: "with www and aliases",
			site: Site{
				Name:     "example.com",
				WWWAlias: true,
				Aliases: []Domain{
					{Hostname: "alias.com"},
				},
			},
			expected: []string{"example.com", "www.example.com", "alias.com"},
		},
		{
			name: "empty aliases slice",
			site: Site{
				Name:     "test.org",
				WWWAlias: true,
				Aliases:  []Domain{},
			},
			expected: []string{"test.org", "www.test.org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.site.GetAllHostnames()
			if len(got) != len(tt.expected) {
				t.Errorf("GetAllHostnames() returned %d hostnames, expected %d", len(got), len(tt.expected))
				return
			}
			for i, hostname := range got {
				if hostname != tt.expected[i] {
					t.Errorf("GetAllHostnames()[%d] = %q, expected %q", i, hostname, tt.expected[i])
				}
			}
		})
	}
}

func TestSite_SSLExpiry(t *testing.T) {
	now := time.Now()
	future := now.Add(30 * 24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		isExpired bool
	}{
		{
			name:      "nil expiry",
			expiresAt: nil,
			isExpired: false,
		},
		{
			name:      "future expiry",
			expiresAt: &future,
			isExpired: false,
		},
		{
			name:      "past expiry",
			expiresAt: &past,
			isExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := Site{
				SSLExpiresAt: tt.expiresAt,
			}
			if tt.expiresAt != nil {
				isExpired := time.Now().After(*site.SSLExpiresAt)
				if isExpired != tt.isExpired {
					t.Errorf("SSL expired check = %v, expected %v", isExpired, tt.isExpired)
				}
			}
		})
	}
}
