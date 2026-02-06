package models

type Redirect struct {
	ID            int64  `json:"id"`
	SiteID        int64  `json:"site_id"`
	SourcePath    string `json:"source_path"`
	TargetURL     string `json:"target_url"`
	Code          int    `json:"code"`
	PreservePath  bool   `json:"preserve_path"`
	PreserveQuery bool   `json:"preserve_query"`
	Priority      int    `json:"priority"`
	IsEnabled     bool   `json:"is_enabled"`
}
