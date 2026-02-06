package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

type DomainRepository struct {
	db *database.DB
}

func NewDomainRepository(db *database.DB) *DomainRepository {
	return &DomainRepository{db: db}
}

func (r *DomainRepository) GetByID(id int64) (*models.Domain, error) {
	domain := &models.Domain{}
	err := r.db.QueryRow(`
		SELECT id, site_id, hostname, is_primary, ssl_enabled, ssl_expires_at, created_at
		FROM domains WHERE id = ?
	`, id).Scan(&domain.ID, &domain.SiteID, &domain.Hostname, &domain.IsPrimary, &domain.SSLEnabled, &domain.SSLExpiresAt, &domain.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return domain, err
}

func (r *DomainRepository) GetByHostname(hostname string) (*models.Domain, error) {
	domain := &models.Domain{}
	err := r.db.QueryRow(`
		SELECT id, site_id, hostname, is_primary, ssl_enabled, ssl_expires_at, created_at
		FROM domains WHERE hostname = ?
	`, hostname).Scan(&domain.ID, &domain.SiteID, &domain.Hostname, &domain.IsPrimary, &domain.SSLEnabled, &domain.SSLExpiresAt, &domain.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return domain, err
}

func (r *DomainRepository) Create(domain *models.Domain) error {
	domain.CreatedAt = time.Now()
	result, err := r.db.Exec(`
		INSERT INTO domains (site_id, hostname, is_primary, ssl_enabled, ssl_expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, domain.SiteID, domain.Hostname, domain.IsPrimary, domain.SSLEnabled, domain.SSLExpiresAt, domain.CreatedAt)
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

func (r *DomainRepository) Update(domain *models.Domain) error {
	_, err := r.db.Exec(`
		UPDATE domains SET hostname = ?, is_primary = ?, ssl_enabled = ?, ssl_expires_at = ?
		WHERE id = ?
	`, domain.Hostname, domain.IsPrimary, domain.SSLEnabled, domain.SSLExpiresAt, domain.ID)
	return err
}

func (r *DomainRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM domains WHERE id = ?`, id)
	return err
}

func (r *DomainRepository) ListBySite(siteID int64) ([]*models.Domain, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, hostname, is_primary, ssl_enabled, ssl_expires_at, created_at
		FROM domains WHERE site_id = ? ORDER BY is_primary DESC, hostname ASC
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*models.Domain
	for rows.Next() {
		domain := &models.Domain{}
		if err := rows.Scan(&domain.ID, &domain.SiteID, &domain.Hostname, &domain.IsPrimary, &domain.SSLEnabled, &domain.SSLExpiresAt, &domain.CreatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}

func (r *DomainRepository) SetPrimary(siteID, domainID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Reset all domains for site
	if _, err := tx.Exec(`UPDATE domains SET is_primary = 0 WHERE site_id = ?`, siteID); err != nil {
		return err
	}

	// Set new primary
	if _, err := tx.Exec(`UPDATE domains SET is_primary = 1 WHERE id = ?`, domainID); err != nil {
		return err
	}

	return tx.Commit()
}
