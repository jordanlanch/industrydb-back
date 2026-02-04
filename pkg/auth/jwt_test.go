package auth

import (
	"testing"
	"time"
)

func TestGenerateJWT(t *testing.T) {
	userID := 1
	email := "test@example.com"
	tier := "free"
	secret := "test-secret-key-minimum-32-characters-long"
	expirationHours := 24

	token, err := GenerateJWT(userID, email, tier, secret, expirationHours)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	// Token should contain 3 parts separated by dots (header.payload.signature)
	if len(token) < 10 {
		t.Error("Token seems too short")
	}
}

func TestValidateJWT(t *testing.T) {
	userID := 123
	email := "test@example.com"
	tier := "pro"
	secret := "test-secret-key-minimum-32-characters-long"
	expirationHours := 24

	// Generate a valid token
	token, err := GenerateJWT(userID, email, tier, secret, expirationHours)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Validate the token
	claims, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	// Check claims
	if claims.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}

	if claims.Tier != tier {
		t.Errorf("Expected Tier %s, got %s", tier, claims.Tier)
	}
}

func TestValidateJWTInvalidToken(t *testing.T) {
	secret := "test-secret-key-minimum-32-characters-long"

	// Test with invalid token
	_, err := ValidateJWT("invalid.token.here", secret)
	if err == nil {
		t.Error("ValidateJWT should return error for invalid token")
	}

	// Test with empty token
	_, err = ValidateJWT("", secret)
	if err == nil {
		t.Error("ValidateJWT should return error for empty token")
	}
}

func TestValidateJWTWrongSecret(t *testing.T) {
	userID := 1
	email := "test@example.com"
	tier := "free"
	secret := "test-secret-key-minimum-32-characters-long"
	wrongSecret := "wrong-secret-key-minimum-32-characters-long"
	expirationHours := 24

	// Generate token with one secret
	token, err := GenerateJWT(userID, email, tier, secret, expirationHours)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Try to validate with different secret
	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Error("ValidateJWT should return error when using wrong secret")
	}
}

func TestJWTExpiration(t *testing.T) {
	userID := 1
	email := "test@example.com"
	tier := "free"
	secret := "test-secret-key-minimum-32-characters-long"

	token, err := GenerateJWT(userID, email, tier, secret, 24)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Validate immediately should work
	claims, err := ValidateJWT(token, secret)
	if err != nil {
		t.Errorf("Token should be valid immediately: %v", err)
	}

	// Check expiration is in the future
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		t.Error("Token expiration should be in the future")
	}
}

func TestGenerateJWTDifferentTiers(t *testing.T) {
	secret := "test-secret-key-minimum-32-characters-long"
	tiers := []string{"free", "starter", "pro", "business"}

	for _, tier := range tiers {
		token, err := GenerateJWT(1, "test@example.com", tier, secret, 24)
		if err != nil {
			t.Errorf("Failed to generate JWT for tier %s: %v", tier, err)
			continue
		}

		claims, err := ValidateJWT(token, secret)
		if err != nil {
			t.Errorf("Failed to validate JWT for tier %s: %v", tier, err)
			continue
		}

		if claims.Tier != tier {
			t.Errorf("Expected tier %s, got %s", tier, claims.Tier)
		}
	}
}
