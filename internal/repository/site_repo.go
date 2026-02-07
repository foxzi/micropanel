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
		SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at
		FROM sites WHERE id = ?
	`, id).Scan(&site.ID, &site.Name, &site.OwnerID, &site.IsEnabled, &site.SSLEnabled, &site.SSLExpiresAt, &site.WWWAlias, &site.CreatedAt, &site.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return site, err
}

func (r *SiteRepository) GetByName(name string) (*models.Site, error) {
	site := &models.Site{}
	err := r.db.QueryRow(`
		SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at
		FROM sites WHERE name = ?
	`, name).Scan(&site.ID, &site.Name, &site.OwnerID, &site.IsEnabled, &site.SSLEnabled, &site.SSLExpiresAt, &site.WWWAlias, &site.CreatedAt, &site.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return site, err
}

func (r *SiteRepository) Create(site *models.Site) error {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO sites (name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, site.Name, site.OwnerID, site.IsEnabled, site.SSLEnabled, site.SSLExpiresAt, site.WWWAlias, now, now)
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
		UPDATE sites SET name = ?, is_enabled = ?, ssl_enabled = ?, ssl_expires_at = ?, www_alias = ?, updated_at = ?
		WHERE id = ?
	`, site.Name, site.IsEnabled, site.SSLEnabled, site.SSLExpiresAt, site.WWWAlias, site.UpdatedAt, site.ID)
	return err
}

func (r *SiteRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM sites WHERE id = ?`, id)
	return err
}

func (r *SiteRepository) ListByOwner(ownerID int64) ([]*models.Site, error) {
	rows, err := r.db.Query(`
		SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at
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
		SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at
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
		if err := rows.Scan(&site.ID, &site.Name, &site.OwnerID, &site.IsEnabled, &site.SSLEnabled, &site.SSLExpiresAt, &site.WWWAlias, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func (r *SiteRepository) ListByOwnerPaginated(ownerID int64, search string, page, limit int) ([]*models.Site, error) {
	offset := (page - 1) * limit
	query := `
		SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at
		FROM sites WHERE owner_id = ?`
	args := []interface{}{ownerID}

	if search != "" {
		query += ` AND name LIKE ?`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSites(rows)
}

func (r *SiteRepository) ListAllPaginated(search string, page, limit int) ([]*models.Site, error) {
	offset := (page - 1) * limit
	query := `
		SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at
		FROM sites`
	var args []interface{}

	if search != "" {
		query += ` WHERE name LIKE ?`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSites(rows)
}

func (r *SiteRepository) CountByOwner(ownerID int64, search string) (int, error) {
	query := `SELECT COUNT(*) FROM sites WHERE owner_id = ?`
	args := []interface{}{ownerID}

	if search != "" {
		query += ` AND name LIKE ?`
		args = append(args, "%"+search+"%")
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (r *SiteRepository) CountAll(search string) (int, error) {
	query := `SELECT COUNT(*) FROM sites`
	var args []interface{}

	if search != "" {
		query += ` WHERE name LIKE ?`
		args = append(args, "%"+search+"%")
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}
