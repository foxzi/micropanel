package models

import "time"

type Domain struct {
	ID           int64      `json:"id"`
	SiteID       int64      `json:"site_id"`
	Hostname     string     `json:"hostname"`
	IsPrimary    bool       `json:"is_primary"`
	SSLEnabled   bool       `json:"ssl_enabled"`
	SSLExpiresAt *time.Time `json:"ssl_expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
