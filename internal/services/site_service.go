package services

import (
	"fmt"
	"os"
	"path/filepath"

	"micropanel/internal/config"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

type SiteService struct {
	siteRepo   *repository.SiteRepository
	domainRepo *repository.DomainRepository
	config     *config.Config
}

func NewSiteService(siteRepo *repository.SiteRepository, domainRepo *repository.DomainRepository, cfg *config.Config) *SiteService {
	return &SiteService{
		siteRepo:   siteRepo,
		domainRepo: domainRepo,
		config:     cfg,
	}
}

func (s *SiteService) Create(name string, ownerID int64) (*models.Site, error) {
	site := &models.Site{
		Name:      name,
		OwnerID:   ownerID,
		IsEnabled: true,
		WWWAlias:  true, // www alias enabled by default
	}

	if err := s.siteRepo.Create(site); err != nil {
		return nil, err
	}

	// Create site directory structure
	if err := s.createSiteDirectories(site.ID); err != nil {
		s.siteRepo.Delete(site.ID)
		return nil, fmt.Errorf("create site directories: %w", err)
	}

	return site, nil
}

func (s *SiteService) GetByID(id int64) (*models.Site, error) {
	site, err := s.siteRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Load aliases
	aliases, err := s.domainRepo.ListBySite(id)
	if err != nil {
		return nil, err
	}
	site.Aliases = make([]models.Domain, len(aliases))
	for i, d := range aliases {
		site.Aliases[i] = *d
	}

	return site, nil
}

func (s *SiteService) GetByName(name string) (*models.Site, error) {
	return s.siteRepo.GetByName(name)
}

func (s *SiteService) List(user *models.User) ([]*models.Site, error) {
	if user.IsAdmin() {
		return s.siteRepo.ListAll()
	}
	return s.siteRepo.ListByOwner(user.ID)
}

func (s *SiteService) ListAll() ([]*models.Site, error) {
	return s.siteRepo.ListAll()
}

func (s *SiteService) ListPaginated(user *models.User, search string, page, limit int) ([]*models.Site, error) {
	if user.IsAdmin() {
		return s.siteRepo.ListAllPaginated(search, page, limit)
	}
	return s.siteRepo.ListByOwnerPaginated(user.ID, search, page, limit)
}

func (s *SiteService) Count(user *models.User, search string) (int, error) {
	if user.IsAdmin() {
		return s.siteRepo.CountAll(search)
	}
	return s.siteRepo.CountByOwner(user.ID, search)
}

func (s *SiteService) Update(site *models.Site) error {
	return s.siteRepo.Update(site)
}

func (s *SiteService) Delete(id int64) error {
	// Delete site directory
	sitePath := s.GetSitePath(id)
	if err := os.RemoveAll(sitePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove site directory: %w", err)
	}

	return s.siteRepo.Delete(id)
}

func (s *SiteService) CanAccess(site *models.Site, user *models.User) bool {
	return user.IsAdmin() || site.OwnerID == user.ID
}

func (s *SiteService) GetSitePath(siteID int64) string {
	return filepath.Join(s.config.Sites.Path, fmt.Sprintf("%d", siteID))
}

func (s *SiteService) GetPublicPath(siteID int64) string {
	return filepath.Join(s.GetSitePath(siteID), "public")
}

func (s *SiteService) createSiteDirectories(siteID int64) error {
	dirs := []string{
		s.GetPublicPath(siteID),
		filepath.Join(s.GetSitePath(siteID), "deploys"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create default index.html
	indexPath := filepath.Join(s.GetPublicPath(siteID), "index.html")
	defaultContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Site %d</title></head>
<body><h1>Welcome to Site %d</h1><p>Deploy your files to see your content.</p></body>
</html>`, siteID, siteID)

	return os.WriteFile(indexPath, []byte(defaultContent), 0644)
}
