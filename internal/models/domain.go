package models

import "time"

// Domain represents an alias domain for a site
type Domain struct {
	ID        int64     `json:"id"`
	SiteID    int64     `json:"site_id"`
	Hostname  string    `json:"hostname"`
	CreatedAt time.Time `json:"created_at"`
}
