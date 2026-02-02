package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/config"
	"github.com/jordanlanch/industrydb/pkg/email"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

// setupPasswordTestHandler creates a test handler for password reset tests
func setupPasswordTestHandler(t *testing.T) (*AuthHandler, *ent.Client, *cache.MockClient, func()) {
	// Create in-memory SQLite database for testing
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// Create mock cache
	mockCache := cache.NewMockClient()

	// Create test services
	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}
	blacklist := auth.NewTokenBlacklist(mockCache)
	emailService := email.NewService("test@industrydb.io", "IndustryDB Test", "")
	auditLogger := audit.NewService(client)

	// Create handler
	handler := NewAuthHandler(client, cfg, blacklist, mockCache, auditLogger, emailService)

	// Return cleanup function
	cleanup := func() {
		client.Close()
	}

	return handler, client, mockCache, cleanup
}

// createPasswordTestUser creates a test user for password reset tests
func createPasswordTestUser(t *testing.T, client *ent.Client) *ent.User {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetEmail("password-test@example.com").
		SetPasswordHash(string(passwordHash)).
		SetName("Password Test User").
		SetSubscriptionTier("free").
		SetRole("user").
		SetUsageCount(0).
		SetUsageLimit(50).
		SetEmailVerified(true).
		SetAcceptedTermsAt(time.Now()).
		Save(context.Background())

	require.NoError(t, err)
	return user
}

// TestForgotPassword_Success tests successful forgot password request
func TestForgotPassword_Success(t *testing.T) {
	handler, client, mockCache, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Create test user
	user := createPasswordTestUser(t, client)

	// Setup Echo context
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"email": user.Email,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err := handler.ForgotPassword(c)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "If an account exists")

	// Verify token was saved in cache (we can't check exact token but verify cache was called)
	// In real implementation, mockCache would track Set() calls
}

// TestForgotPassword_UserNotFound tests forgot password with non-existent email
func TestForgotPassword_UserNotFound(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo context with non-existent email
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"email": "nonexistent@example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err := handler.ForgotPassword(c)
	require.NoError(t, err)

	// Assertions - should still return 200 to prevent email enumeration
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "If an account exists")
}

// TestForgotPassword_InvalidEmail tests forgot password with invalid email format
func TestForgotPassword_InvalidEmail(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo context with invalid email
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"email": "invalid-email",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err := handler.ForgotPassword(c)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response models.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
}

// TestResetPassword_Success tests successful password reset
func TestResetPassword_Success(t *testing.T) {
	handler, client, mockCache, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Create test user
	user := createPasswordTestUser(t, client)

	// Generate reset token and store in cache
	resetToken := "test-reset-token-12345678"
	tokenHash := sha256.Sum256([]byte(resetToken))
	tokenKey := fmt.Sprintf("password_reset:%s", hex.EncodeToString(tokenHash[:]))

	// Store user ID in cache
	err := mockCache.Set(context.Background(), tokenKey, fmt.Sprintf("%d", user.ID), time.Hour)
	require.NoError(t, err)

	// Setup Echo context
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"token":        resetToken,
		"new_password": "newpassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err = handler.ResetPassword(c)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Password reset successfully", response["message"])

	// Verify password was updated
	updatedUser, err := client.User.Get(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify new password works
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte("newpassword123"))
	assert.NoError(t, err)

	// Verify token was deleted from cache
	cachedValue, _ := mockCache.Get(context.Background(), tokenKey)
	assert.Empty(t, cachedValue)
}

// TestResetPassword_InvalidToken tests password reset with invalid token
func TestResetPassword_InvalidToken(t *testing.T) {
	handler, _, _, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Setup Echo context with invalid token (not in cache)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"token":        "invalid-token-12345",
		"new_password": "newpassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err := handler.ResetPassword(c)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response models.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_token", response.Error)
}

// TestResetPassword_ExpiredToken tests password reset with expired token
func TestResetPassword_ExpiredToken(t *testing.T) {
	handler, client, mockCache, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Create test user
	user := createPasswordTestUser(t, client)

	// Generate reset token and store in cache with immediate expiration
	resetToken := "test-expired-token-12345"
	tokenHash := sha256.Sum256([]byte(resetToken))
	tokenKey := fmt.Sprintf("password_reset:%s", hex.EncodeToString(tokenHash[:]))

	// Store user ID in cache with very short TTL
	err := mockCache.Set(context.Background(), tokenKey, fmt.Sprintf("%d", user.ID), 1*time.Nanosecond)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Setup Echo context
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"token":        resetToken,
		"new_password": "newpassword123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err = handler.ResetPassword(c)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response models.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_token", response.Error) // Expired token returns same error as invalid (security)
}

// TestResetPassword_WeakPassword tests password reset with weak password
func TestResetPassword_WeakPassword(t *testing.T) {
	handler, client, mockCache, cleanup := setupPasswordTestHandler(t)
	defer cleanup()

	// Create test user
	user := createPasswordTestUser(t, client)

	// Generate reset token and store in cache
	resetToken := "test-reset-token-12345678"
	tokenHash := sha256.Sum256([]byte(resetToken))
	tokenKey := fmt.Sprintf("password_reset:%s", hex.EncodeToString(tokenHash[:]))

	// Store user ID in cache
	err := mockCache.Set(context.Background(), tokenKey, fmt.Sprintf("%d", user.ID), time.Hour)
	require.NoError(t, err)

	// Setup Echo context with weak password (< 8 chars)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	reqBody := map[string]string{
		"token":        resetToken,
		"new_password": "weak",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err = handler.ResetPassword(c)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response models.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
}
