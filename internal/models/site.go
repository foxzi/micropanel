package models

import "time"

type Site struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	OwnerID   int64     `json:"owner_id"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations (loaded separately)
	Owner   *User    `json:"owner,omitempty"`
	Domains []Domain `json:"domains,omitempty"`
}
