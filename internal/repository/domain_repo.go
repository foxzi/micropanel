package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

// DomainRepository manages alias domains for sites
type DomainRepository struct {
	db *database.DB
}

func NewDomainRepository(db *database.DB) *DomainRepository {
	return &DomainRepository{db: db}
}

func (r *DomainRepository) GetByID(id int64) (*models.Domain, error) {
	domain := &models.Domain{}
	err := r.db.QueryRow(`
		SELECT id, site_id, hostname, created_at
		FROM domains WHERE id = ?
	`, id).Scan(&domain.ID, &domain.SiteID, &domain.Hostname, &domain.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return domain, err
}

func (r *DomainRepository) GetByHostname(hostname string) (*models.Domain, error) {
	domain := &models.Domain{}
	err := r.db.QueryRow(`
		SELECT id, site_id, hostname, created_at
		FROM domains WHERE hostname = ?
	`, hostname).Scan(&domain.ID, &domain.SiteID, &domain.Hostname, &domain.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return domain, err
}

func (r *DomainRepository) Create(domain *models.Domain) error {
	domain.CreatedAt = time.Now()
	result, err := r.db.Exec(`
		INSERT INTO domains (site_id, hostname, created_at)
		VALUES (?, ?, ?)
	`, domain.SiteID, domain.Hostname, domain.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	domain.ID = id
	return nil
}

func (r *DomainRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM domains WHERE id = ?`, id)
	return err
}

func (r *DomainRepository) ListBySite(siteID int64) ([]*models.Domain, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, hostname, created_at
		FROM domains WHERE site_id = ? ORDER BY hostname ASC
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*models.Domain
	for rows.Next() {
		domain := &models.Domain{}
		if err := rows.Scan(&domain.ID, &domain.SiteID, &domain.Hostname, &domain.CreatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}

func (r *DomainRepository) DeleteBySite(siteID int64) error {
	_, err := r.db.Exec(`DELETE FROM domains WHERE site_id = ?`, siteID)
	return err
}
