package models

import "time"

type DeployStatus string

const (
	DeployStatusPending DeployStatus = "pending"
	DeployStatusSuccess DeployStatus = "success"
	DeployStatusFailed  DeployStatus = "failed"
)

type Deploy struct {
	ID           int64        `json:"id"`
	SiteID       int64        `json:"site_id"`
	UserID       int64        `json:"user_id"`
	Filename     string       `json:"filename"`
	Status       DeployStatus `json:"status"`
	ErrorMessage string       `json:"error_message,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}
