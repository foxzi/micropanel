package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/mattn/go-sqlite3"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

// ErrDuplicateKey is returned by Create when the site already has a deploy
// with the same idempotency key.
var ErrDuplicateKey = errors.New("duplicate idempotency key")

const deployColumns = `id, site_id, user_id, filename, status, error_message, idempotency_key, created_at`

type DeployRepository struct {
	db *database.DB
}

func NewDeployRepository(db *database.DB) *DeployRepository {
	return &DeployRepository{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDeploy(row rowScanner) (*models.Deploy, error) {
	deploy := &models.Deploy{}
	var errorMessage, key sql.NullString
	if err := row.Scan(&deploy.ID, &deploy.SiteID, &deploy.UserID, &deploy.Filename, &deploy.Status, &errorMessage, &key, &deploy.CreatedAt); err != nil {
		return nil, err
	}
	deploy.ErrorMessage = errorMessage.String
	deploy.IdempotencyKey = key.String
	return deploy, nil
}

func (r *DeployRepository) GetByID(id int64) (*models.Deploy, error) {
	deploy, err := scanDeploy(r.db.QueryRow(`SELECT `+deployColumns+` FROM deploys WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return deploy, err
}

// GetByKey returns the site's deploy with the given idempotency key.
func (r *DeployRepository) GetByKey(siteID int64, key string) (*models.Deploy, error) {
	deploy, err := scanDeploy(r.db.QueryRow(`SELECT `+deployColumns+` FROM deploys WHERE site_id = ? AND idempotency_key = ?`, siteID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return deploy, err
}

// Create inserts a deploy. A deploy with an idempotency key the site already
// used fails with ErrDuplicateKey.
func (r *DeployRepository) Create(deploy *models.Deploy) error {
	deploy.CreatedAt = time.Now()
	var key any
	if deploy.IdempotencyKey != "" {
		key = deploy.IdempotencyKey
	}
	result, err := r.db.Exec(`
		INSERT INTO deploys (site_id, user_id, filename, status, error_message, idempotency_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, deploy.SiteID, deploy.UserID, deploy.Filename, deploy.Status, deploy.ErrorMessage, key, deploy.CreatedAt)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return ErrDuplicateKey
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	deploy.ID = id
	return nil
}

// ClaimFailed moves a failed deploy back to pending for a retry with the same
// idempotency key. It reports false when the deploy is no longer failed, for
// example because a concurrent retry claimed it first.
func (r *DeployRepository) ClaimFailed(id int64, userID int64, filename string) (bool, error) {
	result, err := r.db.Exec(`
		UPDATE deploys SET status = ?, error_message = '', user_id = ?, filename = ?
		WHERE id = ? AND status = ?
	`, models.DeployStatusPending, userID, filename, id, models.DeployStatusFailed)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// FailPending marks every pending deploy as failed with the given reason and
// returns how many there were. Called at startup: a deploy still pending then
// was interrupted by a restart and will never finish.
func (r *DeployRepository) FailPending(reason string) (int64, error) {
	result, err := r.db.Exec(`UPDATE deploys SET status = ?, error_message = ? WHERE status = ?`,
		models.DeployStatusFailed, reason, models.DeployStatusPending)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *DeployRepository) UpdateStatus(id int64, status models.DeployStatus, errorMsg string) error {
	_, err := r.db.Exec(`
		UPDATE deploys SET status = ?, error_message = ? WHERE id = ?
	`, status, errorMsg, id)
	return err
}

func (r *DeployRepository) ListBySite(siteID int64, limit int) ([]*models.Deploy, error) {
	return r.list(`SELECT `+deployColumns+` FROM deploys WHERE site_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`, siteID, limit)
}

// ListBySiteAndKey returns the site's deploys with the given idempotency key
// (at most one, since the key is unique per site).
func (r *DeployRepository) ListBySiteAndKey(siteID int64, key string) ([]*models.Deploy, error) {
	return r.list(`SELECT `+deployColumns+` FROM deploys WHERE site_id = ? AND idempotency_key = ?`, siteID, key)
}

func (r *DeployRepository) list(query string, args ...any) ([]*models.Deploy, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deploys []*models.Deploy
	for rows.Next() {
		deploy, err := scanDeploy(rows)
		if err != nil {
			return nil, err
		}
		deploys = append(deploys, deploy)
	}
	return deploys, rows.Err()
}

func (r *DeployRepository) GetLastSuccessful(siteID int64) (*models.Deploy, error) {
	deploy, err := scanDeploy(r.db.QueryRow(`
		SELECT `+deployColumns+`
		FROM deploys WHERE site_id = ? AND status = ? ORDER BY created_at DESC, id DESC LIMIT 1
	`, siteID, models.DeployStatusSuccess))
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
