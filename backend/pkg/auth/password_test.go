package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	// Test hashing
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Hash should not be empty
	if hashed == "" {
		t.Error("Hashed password should not be empty")
	}

	// Hash should be different from original
	if hashed == password {
		t.Error("Hashed password should be different from original")
	}

	// Hash should be consistent
	hashed2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password second time: %v", err)
	}

	if hashed == hashed2 {
		t.Error("Different hashes should be generated for same password (bcrypt salt)")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	// Hash the password
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	if !CheckPassword(hashed, password) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Test wrong password
	if CheckPassword(hashed, wrongPassword) {
		t.Error("CheckPassword should return false for wrong password")
	}

	// Test empty password
	if CheckPassword(hashed, "") {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestHashPasswordEmptyString(t *testing.T) {
	// Bcrypt can hash empty strings, so we just check it doesn't panic
	hashed, err := HashPassword("")
	if err != nil {
		t.Errorf("HashPassword failed for empty string: %v", err)
	}
	if hashed == "" {
		t.Error("Hash should not be empty even for empty password")
	}
}

func TestCheckPasswordEmptyHash(t *testing.T) {
	if CheckPassword("", "password") {
		t.Error("CheckPassword should return false for empty hash")
	}
}
