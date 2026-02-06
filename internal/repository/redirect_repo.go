package repository

import (
	"micropanel/internal/database"
	"micropanel/internal/models"
)

type RedirectRepository struct {
	db *database.DB
}

func NewRedirectRepository(db *database.DB) *RedirectRepository {
	return &RedirectRepository{db: db}
}

func (r *RedirectRepository) Create(redirect *models.Redirect) error {
	result, err := r.db.Exec(
		`INSERT INTO redirects (site_id, source_path, target_url, code, preserve_path, preserve_query, priority, is_enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		redirect.SiteID, redirect.SourcePath, redirect.TargetURL, redirect.Code,
		redirect.PreservePath, redirect.PreserveQuery, redirect.Priority, redirect.IsEnabled,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	redirect.ID = id
	return nil
}

func (r *RedirectRepository) GetByID(id int64) (*models.Redirect, error) {
	redirect := &models.Redirect{}
	err := r.db.QueryRow(
		`SELECT id, site_id, source_path, target_url, code, preserve_path, preserve_query, priority, is_enabled
		 FROM redirects WHERE id = ?`,
		id,
	).Scan(
		&redirect.ID, &redirect.SiteID, &redirect.SourcePath, &redirect.TargetURL,
		&redirect.Code, &redirect.PreservePath, &redirect.PreserveQuery,
		&redirect.Priority, &redirect.IsEnabled,
	)
	if err != nil {
		return nil, err
	}
	return redirect, nil
}

func (r *RedirectRepository) ListBySite(siteID int64) ([]*models.Redirect, error) {
	rows, err := r.db.Query(
		`SELECT id, site_id, source_path, target_url, code, preserve_path, preserve_query, priority, is_enabled
		 FROM redirects WHERE site_id = ? ORDER BY priority DESC, id ASC`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirects []*models.Redirect
	for rows.Next() {
		redirect := &models.Redirect{}
		if err := rows.Scan(
			&redirect.ID, &redirect.SiteID, &redirect.SourcePath, &redirect.TargetURL,
			&redirect.Code, &redirect.PreservePath, &redirect.PreserveQuery,
			&redirect.Priority, &redirect.IsEnabled,
		); err != nil {
			return nil, err
		}
		redirects = append(redirects, redirect)
	}
	return redirects, nil
}

func (r *RedirectRepository) Update(redirect *models.Redirect) error {
	_, err := r.db.Exec(
		`UPDATE redirects SET source_path = ?, target_url = ?, code = ?, preserve_path = ?, preserve_query = ?, priority = ?, is_enabled = ?
		 WHERE id = ?`,
		redirect.SourcePath, redirect.TargetURL, redirect.Code,
		redirect.PreservePath, redirect.PreserveQuery, redirect.Priority, redirect.IsEnabled,
		redirect.ID,
	)
	return err
}

func (r *RedirectRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM redirects WHERE id = ?`, id)
	return err
}

func (r *RedirectRepository) DeleteBySite(siteID int64) error {
	_, err := r.db.Exec(`DELETE FROM redirects WHERE site_id = ?`, siteID)
	return err
}
