package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

type SessionRepository struct {
	db *database.DB
}

func NewSessionRepository(db *database.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) GetByID(id string) (*models.Session, error) {
	session := &models.Session{}
	err := r.db.QueryRow(`
		SELECT id, user_id, expires_at, created_at
		FROM sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return session, err
}

func (r *SessionRepository) Create(session *models.Session) error {
	session.CreatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, session.ID, session.UserID, session.ExpiresAt, session.CreatedAt)
	return err
}

func (r *SessionRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (r *SessionRepository) DeleteByUserID(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (r *SessionRepository) DeleteExpired() error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}

func (r *SessionRepository) Extend(id string, expiresAt time.Time) error {
	_, err := r.db.Exec(`UPDATE sessions SET expires_at = ? WHERE id = ?`, expiresAt, id)
	return err
}
