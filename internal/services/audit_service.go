package services

import (
	"encoding/json"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

// Action constants
const (
	ActionLogin         = "login"
	ActionLogout        = "logout"
	ActionLoginFailed   = "login_failed"
	ActionSiteCreate    = "site_create"
	ActionSiteUpdate    = "site_update"
	ActionSiteDelete    = "site_delete"
	ActionSiteEnable    = "site_enable"
	ActionSiteDisable   = "site_disable"
	ActionDomainAdd     = "domain_add"
	ActionDomainDelete  = "domain_delete"
	ActionDomainPrimary = "domain_primary"
	ActionSSLIssue      = "ssl_issue"
	ActionSSLRenew      = "ssl_renew"
	ActionDeploy        = "deploy"
	ActionRollback      = "rollback"
	ActionRedirectAdd   = "redirect_add"
	ActionRedirectEdit  = "redirect_update"
	ActionRedirectDel   = "redirect_delete"
	ActionAuthZoneAdd   = "auth_zone_add"
	ActionAuthZoneEdit  = "auth_zone_update"
	ActionAuthZoneDel   = "auth_zone_delete"
	ActionAuthUserAdd   = "auth_user_add"
	ActionAuthUserDel   = "auth_user_delete"
	ActionFileCreate    = "file_create"
	ActionFileEdit      = "file_edit"
	ActionFileDelete    = "file_delete"
	ActionFileRename    = "file_rename"
	ActionFileUpload    = "file_upload"
	ActionUserCreate    = "user_create"
	ActionUserUpdate    = "user_update"
	ActionUserDelete    = "user_delete"
	ActionUserBlock     = "user_block"
	ActionUserUnblock   = "user_unblock"
)

// Entity types
const (
	EntityUser     = "user"
	EntitySite     = "site"
	EntityDomain   = "domain"
	EntityDeploy   = "deploy"
	EntityRedirect = "redirect"
	EntityAuthZone = "auth_zone"
	EntityAuthUser = "auth_user"
	EntityFile     = "file"
)

type AuditService struct {
	repo *repository.AuditRepository
}

func NewAuditService(repo *repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(userID *int64, action, entityType string, entityID *int64, details interface{}, ip string) error {
	var detailsStr string
	if details != nil {
		if str, ok := details.(string); ok {
			detailsStr = str
		} else {
			data, _ := json.Marshal(details)
			detailsStr = string(data)
		}
	}

	log := &models.AuditLog{
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    detailsStr,
		IP:         ip,
	}
	return s.repo.Create(log)
}

func (s *AuditService) LogUser(userID int64, action, entityType string, entityID *int64, details interface{}, ip string) error {
	return s.Log(&userID, action, entityType, entityID, details, ip)
}

func (s *AuditService) LogAnonymous(action, entityType string, details interface{}, ip string) error {
	return s.Log(nil, action, entityType, nil, details, ip)
}

func (s *AuditService) List(page, perPage int) ([]*models.AuditLog, int, error) {
	if perPage <= 0 {
		perPage = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	logs, err := s.repo.List(perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count()
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *AuditService) ListByUser(userID int64, page, perPage int) ([]*models.AuditLog, int, error) {
	if perPage <= 0 {
		perPage = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	logs, err := s.repo.ListByUser(userID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *AuditService) ListByEntity(entityType string, entityID int64, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListByEntity(entityType, entityID, limit, 0)
}

func (s *AuditService) Cleanup(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	return s.repo.DeleteOlderThan(retentionDays)
}
