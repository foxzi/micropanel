package services

import (
	"errors"

	"micropanel/internal/models"
	"micropanel/internal/repository"
)

var (
	ErrTokenNameRequired = errors.New("token name is required")
	ErrTokenNotFound     = errors.New("token not found")
	ErrTokenNotOwned     = errors.New("token does not belong to user")
)

type APITokenService struct {
	tokenRepo *repository.APITokenRepository
}

func NewAPITokenService(tokenRepo *repository.APITokenRepository) *APITokenService {
	return &APITokenService{
		tokenRepo: tokenRepo,
	}
}

// CreateToken generates a new API token for the user
func (s *APITokenService) CreateToken(userID int64, name string) (*models.APIToken, error) {
	if name == "" {
		return nil, ErrTokenNameRequired
	}

	tokenString, err := models.GenerateToken()
	if err != nil {
		return nil, err
	}

	token := &models.APIToken{
		UserID: userID,
		Name:   name,
		Token:  tokenString,
	}

	if err := s.tokenRepo.Create(token); err != nil {
		return nil, err
	}

	return token, nil
}

// ListUserTokens returns all tokens for a user (tokens are masked)
func (s *APITokenService) ListUserTokens(userID int64) ([]*models.APIToken, error) {
	tokens, err := s.tokenRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Mask tokens - only show first 8 and last 4 characters
	for _, t := range tokens {
		if len(t.Token) > 12 {
			t.Token = t.Token[:8] + "..." + t.Token[len(t.Token)-4:]
		}
	}

	return tokens, nil
}

// DeleteToken deletes a token with ownership check
func (s *APITokenService) DeleteToken(id, userID int64) error {
	token, err := s.tokenRepo.GetByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrTokenNotFound
	}
	if err != nil {
		return err
	}

	if token.UserID != userID {
		return ErrTokenNotOwned
	}

	return s.tokenRepo.Delete(id)
}

// ValidateToken validates a token and returns the user ID
func (s *APITokenService) ValidateToken(tokenString string) (*models.APIToken, error) {
	token, err := s.tokenRepo.GetByToken(tokenString)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	// Update last used timestamp in background
	go s.tokenRepo.UpdateLastUsed(token.ID)

	return token, nil
}
