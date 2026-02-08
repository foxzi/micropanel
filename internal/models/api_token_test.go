package models

import "testing"

func TestGenerateToken(t *testing.T) {
	token1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Token should be 64 hex characters (32 bytes * 2)
	if len(token1) != 64 {
		t.Errorf("GenerateToken() length = %d, expected 64", len(token1))
	}

	// Tokens should be unique
	token2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token1 == token2 {
		t.Error("GenerateToken() generated duplicate tokens")
	}

	// Token should be valid hex
	for _, c := range token1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("GenerateToken() contains invalid hex character: %c", c)
		}
	}
}
