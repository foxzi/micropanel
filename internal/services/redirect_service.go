package services

import (
	"errors"
	"strings"

	"micropanel/internal/models"
	"micropanel/internal/repository"
)

var (
	ErrInvalidRedirectCode = errors.New("redirect code must be 301 or 302")
	ErrInvalidSourcePath   = errors.New("source path must start with /")
	ErrInvalidTargetURL    = errors.New("target URL is required")
)

type RedirectService struct {
	redirectRepo *repository.RedirectRepository
	nginxService *NginxService
}

func NewRedirectService(redirectRepo *repository.RedirectRepository, nginxService *NginxService) *RedirectService {
	return &RedirectService{
		redirectRepo: redirectRepo,
		nginxService: nginxService,
	}
}

func (s *RedirectService) Create(siteID int64, sourcePath, targetURL string, code int, preservePath, preserveQuery bool, priority int) (*models.Redirect, error) {
	if err := s.validateRedirect(sourcePath, targetURL, code); err != nil {
		return nil, err
	}

	redirect := &models.Redirect{
		SiteID:        siteID,
		SourcePath:    sourcePath,
		TargetURL:     targetURL,
		Code:          code,
		PreservePath:  preservePath,
		PreserveQuery: preserveQuery,
		Priority:      priority,
		IsEnabled:     true,
	}

	if err := s.redirectRepo.Create(redirect); err != nil {
		return nil, err
	}

	// Regenerate nginx config
	if err := s.nginxService.ApplyConfig(siteID); err != nil {
		return redirect, err
	}

	return redirect, nil
}

func (s *RedirectService) GetByID(id int64) (*models.Redirect, error) {
	return s.redirectRepo.GetByID(id)
}

func (s *RedirectService) ListBySite(siteID int64) ([]*models.Redirect, error) {
	return s.redirectRepo.ListBySite(siteID)
}

func (s *RedirectService) Update(redirect *models.Redirect) error {
	if err := s.validateRedirect(redirect.SourcePath, redirect.TargetURL, redirect.Code); err != nil {
		return err
	}

	if err := s.redirectRepo.Update(redirect); err != nil {
		return err
	}

	// Regenerate nginx config
	return s.nginxService.ApplyConfig(redirect.SiteID)
}

func (s *RedirectService) Delete(id int64) error {
	redirect, err := s.redirectRepo.GetByID(id)
	if err != nil {
		return err
	}

	siteID := redirect.SiteID

	if err := s.redirectRepo.Delete(id); err != nil {
		return err
	}

	// Regenerate nginx config
	return s.nginxService.ApplyConfig(siteID)
}

func (s *RedirectService) Toggle(id int64) error {
	redirect, err := s.redirectRepo.GetByID(id)
	if err != nil {
		return err
	}

	redirect.IsEnabled = !redirect.IsEnabled
	if err := s.redirectRepo.Update(redirect); err != nil {
		return err
	}

	return s.nginxService.ApplyConfig(redirect.SiteID)
}

func (s *RedirectService) validateRedirect(sourcePath, targetURL string, code int) error {
	if code != 301 && code != 302 {
		return ErrInvalidRedirectCode
	}

	if !strings.HasPrefix(sourcePath, "/") {
		return ErrInvalidSourcePath
	}

	if targetURL == "" {
		return ErrInvalidTargetURL
	}

	return nil
}
