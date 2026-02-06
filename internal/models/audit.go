package models

import "time"

type AuditLog struct {
	ID         int64     `json:"id"`
	UserID     *int64    `json:"user_id,omitempty"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   *int64    `json:"entity_id,omitempty"`
	Details    string    `json:"details,omitempty"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}
