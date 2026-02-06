package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"micropanel/internal/models"
	"micropanel/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user is not active")
	ErrSessionExpired     = errors.New("session expired")
)

const (
	SessionDuration  = 24 * time.Hour
	SessionCookieKey = "session_id"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
}

func NewAuthService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *AuthService) Login(email, password string) (*models.Session, error) {
	user, err := s.userRepo.GetByEmail(email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *AuthService) Logout(sessionID string) error {
	return s.sessionRepo.Delete(sessionID)
}

func (s *AuthService) ValidateSession(sessionID string) (*models.User, error) {
	session, err := s.sessionRepo.GetByID(sessionID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		s.sessionRepo.Delete(sessionID)
		return nil, ErrSessionExpired
	}

	user, err := s.userRepo.GetByID(session.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	// Extend session if more than half the time has passed
	if time.Until(session.ExpiresAt) < SessionDuration/2 {
		s.sessionRepo.Extend(sessionID, time.Now().Add(SessionDuration))
	}

	return user, nil
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *AuthService) CleanupExpiredSessions() error {
	return s.sessionRepo.DeleteExpired()
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
