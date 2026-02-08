package services

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_HashPassword(t *testing.T) {
	svc := &AuthService{}

	password := "testpassword123"
	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() returned empty string")
	}

	if hash == password {
		t.Error("HashPassword() returned plain password")
	}

	// Verify hash is valid bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Errorf("Hash verification failed: %v", err)
	}

	// Verify wrong password fails
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrongpassword"))
	if err == nil {
		t.Error("Hash verification should fail for wrong password")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID() error = %v", err)
	}

	if len(id1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Session ID length = %d, want 64", len(id1))
	}

	// Generate another and verify they're different
	id2, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID() error = %v", err)
	}

	if id1 == id2 {
		t.Error("Two generated session IDs should be different")
	}
}

func TestSessionConstants(t *testing.T) {
	if SessionCookieKey != "session_id" {
		t.Errorf("SessionCookieKey = %q, want %q", SessionCookieKey, "session_id")
	}

	// Session duration should be 24 hours
	expectedHours := 24
	actualHours := int(SessionDuration.Hours())
	if actualHours != expectedHours {
		t.Errorf("SessionDuration = %d hours, want %d hours", actualHours, expectedHours)
	}
}

func TestAuthErrors(t *testing.T) {
	if ErrInvalidCredentials.Error() != "invalid credentials" {
		t.Errorf("ErrInvalidCredentials = %q, want %q", ErrInvalidCredentials.Error(), "invalid credentials")
	}
	if ErrUserNotActive.Error() != "user is not active" {
		t.Errorf("ErrUserNotActive = %q, want %q", ErrUserNotActive.Error(), "user is not active")
	}
	if ErrSessionExpired.Error() != "session expired" {
		t.Errorf("ErrSessionExpired = %q, want %q", ErrSessionExpired.Error(), "session expired")
	}
}
