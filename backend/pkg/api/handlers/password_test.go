package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/email"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

// CustomValidator is a custom validator for Echo
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates a struct
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// newTestEchoWithValidator creates an Echo instance with validator configured
func newTestEchoWithValidator() *echo.Echo {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	return e
}

// setupPasswordTestHandler creates a test AuthHandler with Redis cache for password reset tests
func setupPasswordTestHandler(t *testing.T) (*AuthHandler, *ent.Client, *cache.Client, func()) {
	t.Helper()

	// Create in-memory SQLite database
	dbClient := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// Create Redis client for password reset token storage
	redisClient, err := cache.NewClient("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	// Create email service
	emailService := email.NewService("noreply@test.com", "IndustryDB Test", "http://localhost:5678")

	// Create audit logger
	auditLogger := audit.NewService(dbClient)

	// Create test config
	testConfig := &config.Config{
		JWTSecret:          "test-secret-key",
		JWTExpirationHours: 24,
	}

	// Create handler with test dependencies
	handler := &AuthHandler{
		db:           dbClient,
		emailService: emailService,
		config:       testConfig,
		cache:        redisClient,
		auditLogger:  auditLogger,
		validator:    validator.New(),
	}

	// Cleanup function
	cleanup := func() {
		dbClient.Close()
	}

	return handler, dbClient, redisClient, cleanup
}

// createPasswordTestUser creates a user for password reset testing
func createPasswordTestUser(ctx context.Context, client *ent.Client, email, name string) (*ent.User, error) {
	passwordHash, err := auth.HashPassword("oldpassword123")
	if err != nil {
		return nil, err
	}

	return client.User.Create().
		SetEmail(email).
		SetName(name).
		SetPasswordHash(passwordHash).
		SetEmailVerified(true).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		Save(ctx)
}

// TestForgotPassword_Success tests successful password reset request for existing user
func TestForgotPassword_Success(t *testing.T) {
	handler, client, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	testUser, err := createPasswordTestUser(ctx, client, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := newTestEchoWithValidator()
	requestBody := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ForgotPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expectedMessage := "If an account exists with this email, you will receive a password reset link"
	if response["message"] != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, response["message"])
	}

	// Note: Cannot verify token was created in Redis without exposing the token
	// The email service logs the token, but we're testing behavior, not implementation details
	_ = testUser // Avoid unused variable warning
}

// TestForgotPassword_EmailNotFound tests password reset for non-existent email (security: same response)
func TestForgotPassword_EmailNotFound(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo request with non-existent email
	e := newTestEchoWithValidator()
	requestBody := `{"email":"nonexistent@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ForgotPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response (should be identical to success case for security)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expectedMessage := "If an account exists with this email, you will receive a password reset link"
	if response["message"] != expectedMessage {
		t.Errorf("Expected generic message, got %q", response["message"])
	}
}

// TestForgotPassword_InvalidEmail tests password reset with invalid email format
func TestForgotPassword_InvalidEmail(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo request with invalid email
	e := newTestEchoWithValidator()
	requestBody := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ForgotPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "validation_error" {
		t.Errorf("Expected error 'validation_error', got %s", response.Error)
	}
}

// TestResetPassword_Success tests successful password reset with valid token
func TestResetPassword_Success(t *testing.T) {
	handler, client, redisClient, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	testUser, err := createPasswordTestUser(ctx, client, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Manually generate and store reset token (simulating forgot password flow)
	resetToken := "test-reset-token-12345678901234567890123456789012"
	tokenHash := auth.HashResetToken(resetToken)
	tokenKey := "password_reset:" + tokenHash

	err = redisClient.Set(ctx, tokenKey, testUser.ID, time.Hour)
	if err != nil {
		t.Fatalf("Failed to store reset token: %v", err)
	}

	// Setup Echo request
	e := newTestEchoWithValidator()
	requestBody := `{"token":"test-reset-token-12345678901234567890123456789012","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "Password reset successfully" {
		t.Errorf("Expected success message, got %s", response["message"])
	}

	// Verify password was updated in database
	updatedUser, err := client.User.Get(ctx, testUser.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	// Verify new password works
	if !auth.CheckPassword(updatedUser.PasswordHash, "newpassword123") {
		t.Error("New password should be set correctly")
	}

	// Verify old password no longer works
	if auth.CheckPassword(updatedUser.PasswordHash, "oldpassword123") {
		t.Error("Old password should no longer work")
	}

	// Verify token was deleted from Redis (one-time use)
	_, err = redisClient.Get(ctx, tokenKey)
	if err == nil {
		t.Error("Reset token should be deleted after successful use")
	}
}

// TestResetPassword_InvalidToken tests password reset with invalid token
func TestResetPassword_InvalidToken(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo request with invalid token
	e := newTestEchoWithValidator()
	requestBody := `{"token":"invalid-token","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "invalid_token" {
		t.Errorf("Expected error 'invalid_token', got %s", response.Error)
	}
}

// TestResetPassword_ExpiredToken tests password reset with expired token
func TestResetPassword_ExpiredToken(t *testing.T) {
	handler, client, redisClient, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	testUser, err := createPasswordTestUser(ctx, client, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Manually generate and store reset token with very short expiration
	resetToken := "expired-reset-token-123456789012345678901234567890"
	tokenHash := auth.HashResetToken(resetToken)
	tokenKey := "password_reset:" + tokenHash

	// Store with 1 millisecond expiration (will expire immediately)
	err = redisClient.Set(ctx, tokenKey, testUser.ID, time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to store reset token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Setup Echo request
	e := newTestEchoWithValidator()
	requestBody := `{"token":"expired-reset-token-123456789012345678901234567890","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "invalid_token" {
		t.Errorf("Expected error 'invalid_token' (for expired token), got %s", response.Error)
	}
}

// TestResetPassword_WeakPassword tests password reset with weak password (< 8 characters)
func TestResetPassword_WeakPassword(t *testing.T) {
	handler, client, redisClient, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	testUser, err := createPasswordTestUser(ctx, client, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Manually generate and store reset token
	resetToken := "valid-reset-token-12345678901234567890123456789012"
	tokenHash := auth.HashResetToken(resetToken)
	tokenKey := "password_reset:" + tokenHash

	err = redisClient.Set(ctx, tokenKey, testUser.ID, time.Hour)
	if err != nil {
		t.Fatalf("Failed to store reset token: %v", err)
	}

	// Setup Echo request with weak password
	e := newTestEchoWithValidator()
	requestBody := `{"token":"valid-reset-token-12345678901234567890123456789012","new_password":"weak"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "validation_error" {
		t.Errorf("Expected error 'validation_error', got %s", response.Error)
	}
}

// TestResetPassword_TokenReuse tests that reset token cannot be reused after successful reset
func TestResetPassword_TokenReuse(t *testing.T) {
	handler, client, redisClient, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	testUser, err := createPasswordTestUser(ctx, client, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Manually generate and store reset token
	resetToken := "reusable-reset-token-1234567890123456789012345678"
	tokenHash := auth.HashResetToken(resetToken)
	tokenKey := "password_reset:" + tokenHash

	err = redisClient.Set(ctx, tokenKey, testUser.ID, time.Hour)
	if err != nil {
		t.Fatalf("Failed to store reset token: %v", err)
	}

	// First reset (should succeed)
	e := newTestEchoWithValidator()
	requestBody := `{"token":"reusable-reset-token-1234567890123456789012345678","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("First reset failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("First reset: expected status 200, got %d", rec.Code)
	}

	// Second reset attempt with same token (should fail)
	req2 := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	if err := handler.ResetPassword(c2); err != nil {
		t.Fatalf("Second reset attempt failed to execute: %v", err)
	}

	// Verify second attempt was rejected
	if rec2.Code != http.StatusBadRequest {
		t.Errorf("Second reset: expected status 400, got %d", rec2.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "invalid_token" {
		t.Errorf("Expected error 'invalid_token' (token already used), got %s", response.Error)
	}
}

// TestResetPassword_MissingToken tests password reset without token
func TestResetPassword_MissingToken(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo request without token
	e := newTestEchoWithValidator()
	requestBody := `{"new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "validation_error" {
		t.Errorf("Expected error 'validation_error', got %s", response.Error)
	}
}

// TestGeneratePasswordResetToken tests token generation produces unique tokens
func TestGeneratePasswordResetToken(t *testing.T) {
	tokens := make(map[string]bool)

	// Generate 100 tokens
	for i := 0; i < 100; i++ {
		token, err := generatePasswordResetToken()
		if err != nil {
			t.Fatalf("Token generation failed: %v", err)
		}

		// Verify token is 64 characters (32 bytes hex encoded)
		if len(token) != 64 {
			t.Errorf("Expected token length 64, got %d", len(token))
		}

		// Verify token is unique
		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

// TestHashResetToken tests reset token hashing produces consistent results
func TestHashResetToken(t *testing.T) {
	token := "test-token-12345"

	// Hash same token multiple times
	hash1 := auth.HashResetToken(token)
	hash2 := auth.HashResetToken(token)
	hash3 := auth.HashResetToken(token)

	// Verify hashes are consistent
	if hash1 != hash2 || hash2 != hash3 {
		t.Error("Hash function should produce consistent results for same input")
	}

	// Verify different tokens produce different hashes
	differentToken := "different-token-67890"
	differentHash := auth.HashResetToken(differentToken)

	if hash1 == differentHash {
		t.Error("Different tokens should produce different hashes")
	}

	// Verify hash is 64 characters (SHA256 hex encoded)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}
