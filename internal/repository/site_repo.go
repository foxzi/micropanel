package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

type SiteRepository struct {
	db *database.DB
}

func NewSiteRepository(db *database.DB) *SiteRepository {
	return &SiteRepository{db: db}
}

func (r *SiteRepository) GetByID(id int64) (*models.Site, error) {
	site := &models.Site{}
	err := r.db.QueryRow(`
		SELECT id, name, owner_id, is_enabled, created_at, updated_at
		FROM sites WHERE id = ?
	`, id).Scan(&site.ID, &site.Name, &site.OwnerID, &site.IsEnabled, &site.CreatedAt, &site.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return site, err
}

func (r *SiteRepository) Create(site *models.Site) error {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO sites (name, owner_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, site.Name, site.OwnerID, site.IsEnabled, now, now)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	site.ID = id
	site.CreatedAt = now
	site.UpdatedAt = now
	return nil
}

func (r *SiteRepository) Update(site *models.Site) error {
	site.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE sites SET name = ?, is_enabled = ?, updated_at = ?
		WHERE id = ?
	`, site.Name, site.IsEnabled, site.UpdatedAt, site.ID)
	return err
}

func (r *SiteRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM sites WHERE id = ?`, id)
	return err
}

func (r *SiteRepository) ListByOwner(ownerID int64) ([]*models.Site, error) {
	rows, err := r.db.Query(`
		SELECT id, name, owner_id, is_enabled, created_at, updated_at
		FROM sites WHERE owner_id = ? ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSites(rows)
}

func (r *SiteRepository) ListAll() ([]*models.Site, error) {
	rows, err := r.db.Query(`
		SELECT id, name, owner_id, is_enabled, created_at, updated_at
		FROM sites ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSites(rows)
}

func (r *SiteRepository) scanSites(rows *sql.Rows) ([]*models.Site, error) {
	var sites []*models.Site
	for rows.Next() {
		site := &models.Site{}
		if err := rows.Scan(&site.ID, &site.Name, &site.OwnerID, &site.IsEnabled, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}
