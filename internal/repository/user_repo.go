package repository

import (
	"database/sql"
	"errors"
	"time"

	"micropanel/internal/database"
	"micropanel/internal/models"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(id int64) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(`
		SELECT id, email, password_hash, role, is_active, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return user, err
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(`
		SELECT id, email, password_hash, role, is_active, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return user, err
}

func (r *UserRepository) Create(user *models.User) error {
	result, err := r.db.Exec(`
		INSERT INTO users (email, password_hash, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.Email, user.PasswordHash, user.Role, user.IsActive, time.Now(), time.Now())
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (r *UserRepository) Update(user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE users SET email = ?, password_hash = ?, role = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, user.Email, user.PasswordHash, user.Role, user.IsActive, user.UpdatedAt, user.ID)
	return err
}

func (r *UserRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *UserRepository) List() ([]*models.User, error) {
	rows, err := r.db.Query(`
		SELECT id, email, password_hash, role, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
