package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"micropanel/internal/config"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

var (
	ErrInvalidPathPrefix = errors.New("path prefix must start with /")
	ErrInvalidRealm      = errors.New("realm is required")
	ErrInvalidUsername   = errors.New("username is required")
	ErrInvalidPassword   = errors.New("password is required")
	ErrAuthZoneNotFound  = errors.New("auth zone not found")
)

type AuthZoneService struct {
	config       *config.Config
	authZoneRepo *repository.AuthZoneRepository
	nginxService *NginxService
}

func NewAuthZoneService(cfg *config.Config, authZoneRepo *repository.AuthZoneRepository, nginxService *NginxService) *AuthZoneService {
	return &AuthZoneService{
		config:       cfg,
		authZoneRepo: authZoneRepo,
		nginxService: nginxService,
	}
}

func (s *AuthZoneService) Create(siteID int64, pathPrefix, realm string) (*models.AuthZone, error) {
	if err := s.validateZone(pathPrefix, realm); err != nil {
		return nil, err
	}

	zone := &models.AuthZone{
		SiteID:     siteID,
		PathPrefix: pathPrefix,
		Realm:      realm,
		IsEnabled:  true,
	}

	if err := s.authZoneRepo.Create(zone); err != nil {
		return nil, err
	}

	return zone, nil
}

func (s *AuthZoneService) GetByID(id int64) (*models.AuthZone, error) {
	return s.authZoneRepo.GetByID(id)
}

func (s *AuthZoneService) GetByIDWithUsers(id int64) (*models.AuthZone, error) {
	return s.authZoneRepo.GetByIDWithUsers(id)
}

func (s *AuthZoneService) ListBySite(siteID int64) ([]*models.AuthZone, error) {
	return s.authZoneRepo.ListBySite(siteID)
}

func (s *AuthZoneService) ListBySiteWithUsers(siteID int64) ([]*models.AuthZone, error) {
	return s.authZoneRepo.ListBySiteWithUsers(siteID)
}

func (s *AuthZoneService) Update(zone *models.AuthZone) error {
	if err := s.validateZone(zone.PathPrefix, zone.Realm); err != nil {
		return err
	}

	if err := s.authZoneRepo.Update(zone); err != nil {
		return err
	}

	// Regenerate htpasswd and nginx config
	return s.regenerateConfig(zone.SiteID)
}

func (s *AuthZoneService) Delete(id int64) error {
	zone, err := s.authZoneRepo.GetByID(id)
	if err != nil {
		return err
	}

	siteID := zone.SiteID

	if err := s.authZoneRepo.Delete(id); err != nil {
		return err
	}

	// Remove htpasswd file
	htpasswdPath := s.getHtpasswdPath(siteID, id)
	os.Remove(htpasswdPath)

	// Regenerate nginx config
	return s.nginxService.ApplyConfig(siteID)
}

func (s *AuthZoneService) Toggle(id int64) error {
	zone, err := s.authZoneRepo.GetByID(id)
	if err != nil {
		return err
	}

	zone.IsEnabled = !zone.IsEnabled
	if err := s.authZoneRepo.Update(zone); err != nil {
		return err
	}

	return s.regenerateConfig(zone.SiteID)
}

// User management

func (s *AuthZoneService) CreateUser(zoneID int64, username, password string) (*models.AuthZoneUser, error) {
	if username == "" {
		return nil, ErrInvalidUsername
	}
	if password == "" {
		return nil, ErrInvalidPassword
	}

	zone, err := s.authZoneRepo.GetByID(zoneID)
	if err != nil {
		return nil, ErrAuthZoneNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.AuthZoneUser{
		AuthZoneID:   zoneID,
		Username:     username,
		PasswordHash: string(hash),
	}

	if err := s.authZoneRepo.CreateUser(user); err != nil {
		return nil, err
	}

	// Regenerate htpasswd
	if err := s.regenerateHtpasswd(zone.SiteID, zoneID); err != nil {
		return user, err
	}

	return user, nil
}

func (s *AuthZoneService) UpdateUserPassword(userID int64, password string) error {
	if password == "" {
		return ErrInvalidPassword
	}

	user, err := s.authZoneRepo.GetUserByID(userID)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	if err := s.authZoneRepo.UpdateUser(user); err != nil {
		return err
	}

	zone, err := s.authZoneRepo.GetByID(user.AuthZoneID)
	if err != nil {
		return err
	}

	return s.regenerateHtpasswd(zone.SiteID, zone.ID)
}

func (s *AuthZoneService) DeleteUser(userID int64) error {
	user, err := s.authZoneRepo.GetUserByID(userID)
	if err != nil {
		return err
	}

	zone, err := s.authZoneRepo.GetByID(user.AuthZoneID)
	if err != nil {
		return err
	}

	if err := s.authZoneRepo.DeleteUser(userID); err != nil {
		return err
	}

	return s.regenerateHtpasswd(zone.SiteID, zone.ID)
}

// Helper functions

func (s *AuthZoneService) validateZone(pathPrefix, realm string) error {
	if !strings.HasPrefix(pathPrefix, "/") {
		return ErrInvalidPathPrefix
	}
	if realm == "" {
		return ErrInvalidRealm
	}
	return nil
}

func (s *AuthZoneService) getHtpasswdPath(siteID, zoneID int64) string {
	return filepath.Join(s.config.Sites.Path, fmt.Sprintf("%d", siteID), "auth", fmt.Sprintf("zone_%d.htpasswd", zoneID))
}

func (s *AuthZoneService) regenerateHtpasswd(siteID, zoneID int64) error {
	users, err := s.authZoneRepo.ListUsers(zoneID)
	if err != nil {
		return err
	}

	htpasswdPath := s.getHtpasswdPath(siteID, zoneID)

	// Create auth directory if not exists
	authDir := filepath.Dir(htpasswdPath)
	if err := os.MkdirAll(authDir, 0755); err != nil {
		return fmt.Errorf("create auth dir: %w", err)
	}

	// Generate htpasswd content
	var lines []string
	for _, user := range users {
		lines = append(lines, fmt.Sprintf("%s:%s", user.Username, user.PasswordHash))
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	if err := os.WriteFile(htpasswdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write htpasswd: %w", err)
	}

	return nil
}

func (s *AuthZoneService) regenerateConfig(siteID int64) error {
	zones, err := s.authZoneRepo.ListBySiteWithUsers(siteID)
	if err != nil {
		return err
	}

	// Regenerate all htpasswd files for this site
	for _, zone := range zones {
		if err := s.regenerateHtpasswd(siteID, zone.ID); err != nil {
			return err
		}
	}

	// Regenerate nginx config
	return s.nginxService.ApplyConfig(siteID)
}

func (s *AuthZoneService) GetHtpasswdPath(siteID, zoneID int64) string {
	return s.getHtpasswdPath(siteID, zoneID)
}
