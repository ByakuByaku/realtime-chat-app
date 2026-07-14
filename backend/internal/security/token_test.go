package security

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	token, err := GenerateJWT(userID, "secret", time.Hour)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	claims, err := ValidateJWT(token, "secret")
	if err != nil {
		t.Fatalf("validate jwt: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, claims.UserID)
	}
}

func TestGenerateRefreshTokenAndHash(t *testing.T) {
	token, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	if token == "" {
		t.Fatal("expected refresh token to be non-empty")
	}

	hash1 := HashRefreshToken(token)
	hash2 := HashRefreshToken(token)
	if hash1 == "" || hash1 != hash2 {
		t.Fatal("expected stable non-empty refresh token hash")
	}
}