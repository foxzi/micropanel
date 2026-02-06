package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

type DeployRepository struct {
	db *database.DB
}

func NewDeployRepository(db *database.DB) *DeployRepository {
	return &DeployRepository{db: db}
}

func (r *DeployRepository) GetByID(id int64) (*models.Deploy, error) {
	deploy := &models.Deploy{}
	err := r.db.QueryRow(`
		SELECT id, site_id, user_id, filename, status, error_message, created_at
		FROM deploys WHERE id = ?
	`, id).Scan(&deploy.ID, &deploy.SiteID, &deploy.UserID, &deploy.Filename, &deploy.Status, &deploy.ErrorMessage, &deploy.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return deploy, err
}

func (r *DeployRepository) Create(deploy *models.Deploy) error {
	deploy.CreatedAt = time.Now()
	result, err := r.db.Exec(`
		INSERT INTO deploys (site_id, user_id, filename, status, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, deploy.SiteID, deploy.UserID, deploy.Filename, deploy.Status, deploy.ErrorMessage, deploy.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	deploy.ID = id
	return nil
}

func (r *DeployRepository) UpdateStatus(id int64, status models.DeployStatus, errorMsg string) error {
	_, err := r.db.Exec(`
		UPDATE deploys SET status = ?, error_message = ? WHERE id = ?
	`, status, errorMsg, id)
	return err
}

func (r *DeployRepository) ListBySite(siteID int64, limit int) ([]*models.Deploy, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, user_id, filename, status, error_message, created_at
		FROM deploys WHERE site_id = ? ORDER BY created_at DESC LIMIT ?
	`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deploys []*models.Deploy
	for rows.Next() {
		deploy := &models.Deploy{}
		if err := rows.Scan(&deploy.ID, &deploy.SiteID, &deploy.UserID, &deploy.Filename, &deploy.Status, &deploy.ErrorMessage, &deploy.CreatedAt); err != nil {
			return nil, err
		}
		deploys = append(deploys, deploy)
	}
	return deploys, rows.Err()
}

func (r *DeployRepository) GetLastSuccessful(siteID int64) (*models.Deploy, error) {
	deploy := &models.Deploy{}
	err := r.db.QueryRow(`
		SELECT id, site_id, user_id, filename, status, error_message, created_at
		FROM deploys WHERE site_id = ? AND status = ? ORDER BY created_at DESC LIMIT 1
	`, siteID, models.DeployStatusSuccess).Scan(&deploy.ID, &deploy.SiteID, &deploy.UserID, &deploy.Filename, &deploy.Status, &deploy.ErrorMessage, &deploy.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return deploy, err
}

func (r *DeployRepository) CountBySite(siteID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM deploys WHERE site_id = ?`, siteID).Scan(&count)
	return count, err
}
