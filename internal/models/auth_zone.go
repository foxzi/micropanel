package models

type AuthZone struct {
	ID         int64  `json:"id"`
	SiteID     int64  `json:"site_id"`
	PathPrefix string `json:"path_prefix"`
	Realm      string `json:"realm"`
	IsEnabled  bool   `json:"is_enabled"`

	// Relations
	Users []AuthZoneUser `json:"users,omitempty"`
}

type AuthZoneUser struct {
	ID           int64  `json:"id"`
	AuthZoneID   int64  `json:"auth_zone_id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}
