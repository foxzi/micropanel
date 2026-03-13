package models

import "time"

type Site struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"` // Primary hostname (domain)
	OwnerID      int64      `json:"owner_id"`
	IsEnabled    bool       `json:"is_enabled"`
	SSLEnabled   bool       `json:"ssl_enabled"`
	SSLExpiresAt *time.Time `json:"ssl_expires_at,omitempty"`
	SSLCertName  string     `json:"ssl_cert_name,omitempty"` // certbot --cert-name (may differ from Name)
	WWWAlias     bool       `json:"www_alias"`               // Add www. alias
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations (loaded separately)
	Owner   *User    `json:"owner,omitempty"`
	Aliases []Domain `json:"aliases,omitempty"` // Additional domains
}

// GetSSLCertName returns the certificate name for letsencrypt paths.
// Falls back to Site.Name if SSLCertName is not set.
func (s *Site) GetSSLCertName() string {
	if s.SSLCertName != "" {
		return s.SSLCertName
	}
	return s.Name
}

// GetAllHostnames returns all hostnames for nginx config (primary + www + aliases)
func (s *Site) GetAllHostnames() []string {
	hostnames := []string{s.Name}
	if s.WWWAlias {
		hostnames = append(hostnames, "www."+s.Name)
	}
	for _, alias := range s.Aliases {
		hostnames = append(hostnames, alias.Hostname)
	}
	return hostnames
}
