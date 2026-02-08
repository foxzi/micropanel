package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

type APITokenRepository struct {
	db *database.DB
}

func NewAPITokenRepository(db *database.DB) *APITokenRepository {
	return &APITokenRepository{db: db}
}

func (r *APITokenRepository) Create(token *models.APIToken) error {
	token.CreatedAt = time.Now()
	result, err := r.db.Exec(`
		INSERT INTO api_tokens (user_id, name, token, created_at)
		VALUES (?, ?, ?, ?)
	`, token.UserID, token.Name, token.Token, token.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	token.ID = id
	return nil
}

func (r *APITokenRepository) GetByToken(tokenString string) (*models.APIToken, error) {
	token := &models.APIToken{}
	err := r.db.QueryRow(`
		SELECT id, user_id, name, token, created_at, last_used_at
		FROM api_tokens WHERE token = ?
	`, tokenString).Scan(&token.ID, &token.UserID, &token.Name, &token.Token, &token.CreatedAt, &token.LastUsedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return token, err
}

func (r *APITokenRepository) GetByID(id int64) (*models.APIToken, error) {
	token := &models.APIToken{}
	err := r.db.QueryRow(`
		SELECT id, user_id, name, token, created_at, last_used_at
		FROM api_tokens WHERE id = ?
	`, id).Scan(&token.ID, &token.UserID, &token.Name, &token.Token, &token.CreatedAt, &token.LastUsedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return token, err
}

func (r *APITokenRepository) GetByUserID(userID int64) ([]*models.APIToken, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, name, token, created_at, last_used_at
		FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.APIToken
	for rows.Next() {
		token := &models.APIToken{}
		if err := rows.Scan(&token.ID, &token.UserID, &token.Name, &token.Token, &token.CreatedAt, &token.LastUsedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (r *APITokenRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM api_tokens WHERE id = ?`, id)
	return err
}

func (r *APITokenRepository) UpdateLastUsed(id int64) error {
	_, err := r.db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, time.Now(), id)
	return err
}
