package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/email"
	custommiddleware "github.com/jordanlanch/industrydb/pkg/middleware"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

// setupAuthTestHandler creates a test AuthHandler with in-memory SQLite database
func setupAuthTestHandler(t *testing.T) (*AuthHandler, *ent.Client, func()) {
	t.Helper()

	// Create in-memory SQLite database for testing
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// Create email service (test mode - no SendGrid key)
	emailService := email.NewService("noreply@test.com", "IndustryDB Test", "http://localhost:5678", "")

	// Create test config
	testConfig := &config.Config{
		JWTSecret:          "test-secret-key",
		JWTExpirationHours: 24,
	}

	// Create handler with test dependencies (minimal setup for email verification tests)
	handler := &AuthHandler{
		db:           client,
		emailService: emailService,
		config:       testConfig,
	}

	// Cleanup function
	cleanup := func() {
		client.Close()
	}

	return handler, client, cleanup
}

// createVerificationTestUser creates a user for email verification testing
func createVerificationTestUser(ctx context.Context, client *ent.Client, email, name, verificationToken string, verified bool, tokenExpired bool) (*ent.User, error) {
	passwordHash, err := auth.HashPassword("password123")
	if err != nil {
		return nil, err
	}

	builder := client.User.Create().
		SetEmail(email).
		SetName(name).
		SetPasswordHash(passwordHash).
		SetEmailVerified(verified).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50)

	if verificationToken != "" {
		builder = builder.SetEmailVerificationToken(verificationToken)

		if tokenExpired {
			// Token expired 25 hours ago
			builder = builder.SetEmailVerificationTokenExpiresAt(time.Now().Add(-25 * time.Hour))
		} else {
			// Token expires in 23 hours
			builder = builder.SetEmailVerificationTokenExpiresAt(time.Now().Add(23 * time.Hour))
		}
	}

	return builder.Save(ctx)
}

// TestVerifyEmail_Success tests successful email verification with valid token
func TestVerifyEmail_Success(t *testing.T) {
	handler, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()
	token := "valid-token-12345"

	// Create unverified user
	user, err := createVerificationTestUser(ctx, client, "test@example.com", "Test User", token, false, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/auth/verify-email/%s", token), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues(token)

	// Execute handler
	if err := handler.VerifyEmail(c); err != nil {
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

	if response["message"] != "Email verified successfully" {
		t.Errorf("Expected success message, got %s", response["message"])
	}

	// Verify user is marked as verified in database
	updatedUser, err := client.User.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if !updatedUser.EmailVerified {
		t.Error("User should be marked as verified")
	}

	if updatedUser.EmailVerifiedAt == nil {
		t.Error("EmailVerifiedAt should be set")
	}

	if updatedUser.EmailVerificationToken != nil && *updatedUser.EmailVerificationToken != "" {
		t.Error("Verification token should be cleared")
	}

	// Note: Welcome email is sent asynchronously via goroutine, so we can't verify it in this test
}

// TestVerifyEmail_InvalidToken tests verification with invalid token
func TestVerifyEmail_InvalidToken(t *testing.T) {
	handler, _, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	// Setup Echo request with invalid token
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email/invalid-token", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues("invalid-token")

	// Execute handler
	if err := handler.VerifyEmail(c); err != nil {
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

// TestVerifyEmail_ExpiredToken tests verification with expired token
func TestVerifyEmail_ExpiredToken(t *testing.T) {
	handler, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()
	token := "expired-token-12345"

	// Create user with expired token (expired 25 hours ago)
	_, err := createVerificationTestUser(ctx, client, "test@example.com", "Test User", token, false, true)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/auth/verify-email/%s", token), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues(token)

	// Execute handler
	if err := handler.VerifyEmail(c); err != nil {
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

	if response.Error != "expired_token" {
		t.Errorf("Expected error 'expired_token', got %s", response.Error)
	}
}

// TestVerifyEmail_AlreadyVerified tests idempotent behavior for already verified user
func TestVerifyEmail_AlreadyVerified(t *testing.T) {
	handler, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()
	token := "valid-token-12345"

	// Create already verified user
	_, err := createVerificationTestUser(ctx, client, "test@example.com", "Test User", token, true, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/auth/verify-email/%s", token), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues(token)

	// Execute handler
	if err := handler.VerifyEmail(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response (should still return 200 OK, idempotent)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "Email already verified" {
		t.Errorf("Expected 'Email already verified' message, got %s", response["message"])
	}
}

// TestVerifyEmail_MissingToken tests verification without token parameter
func TestVerifyEmail_MissingToken(t *testing.T) {
	handler, _, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	// Setup Echo request without token
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues("")

	// Execute handler
	if err := handler.VerifyEmail(c); err != nil {
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

	if response.Error != "missing_token" {
		t.Errorf("Expected error 'missing_token', got %s", response.Error)
	}
}

// TestResendVerificationEmail_Success tests successful resend for unverified user
func TestResendVerificationEmail_Success(t *testing.T) {
	handler, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create unverified user
	user, err := createVerificationTestUser(ctx, client, "test@example.com", "Test User", "old-token", false, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID) // Simulate authenticated user

	// Execute handler
	if err := handler.ResendVerificationEmail(c); err != nil {
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

	if response["message"] != "Verification email sent" {
		t.Errorf("Expected success message, got %s", response["message"])
	}

	// Verify new token was generated
	updatedUser, err := client.User.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if updatedUser.EmailVerificationToken != nil && *updatedUser.EmailVerificationToken == "old-token" {
		t.Error("Verification token should have been regenerated")
	}

	if updatedUser.EmailVerificationToken == nil || *updatedUser.EmailVerificationToken == "" {
		t.Error("New verification token should be set")
	}

	// Note: Verification email is sent asynchronously via goroutine, so we can't verify it in this test
}

// TestResendVerificationEmail_AlreadyVerified tests resend for already verified user
func TestResendVerificationEmail_AlreadyVerified(t *testing.T) {
	handler, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create verified user
	user, err := createVerificationTestUser(ctx, client, "test@example.com", "Test User", "", true, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	if err := handler.ResendVerificationEmail(c); err != nil {
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

	if response.Error != "already_verified" {
		t.Errorf("Expected error 'already_verified', got %s", response.Error)
	}
}

// TestResendVerificationEmail_Unauthorized tests resend without authentication
func TestResendVerificationEmail_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	// Setup Echo request without user_id in context
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Don't set user_id in context

	// Execute handler
	if err := handler.ResendVerificationEmail(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "unauthorized" {
		t.Errorf("Expected error 'unauthorized', got %s", response.Error)
	}
}

// TestRequireEmailVerified_AllowsVerifiedUser tests middleware allows verified users
func TestRequireEmailVerified_AllowsVerifiedUser(t *testing.T) {
	_, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create verified user
	user, err := createVerificationTestUser(ctx, client, "verified@example.com", "Verified User", "", true, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Create middleware and next handler
	middleware := custommiddleware.RequireEmailVerified(client)
	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "Protected content")
	}

	// Execute middleware
	handler := middleware(next)
	if err := handler(c); err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	// Verify next handler was called
	if !nextCalled {
		t.Error("Next handler should have been called for verified user")
	}

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Protected content") {
		t.Error("Should return protected content for verified user")
	}
}

// TestRequireEmailVerified_BlocksUnverifiedUser tests middleware blocks unverified users
func TestRequireEmailVerified_BlocksUnverifiedUser(t *testing.T) {
	_, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create unverified user
	user, err := createVerificationTestUser(ctx, client, "unverified@example.com", "Unverified User", "token", false, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Setup Echo request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Create middleware and next handler
	middleware := custommiddleware.RequireEmailVerified(client)
	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "Protected content")
	}

	// Execute middleware
	handler := middleware(next)
	if err := handler(c); err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	// Verify next handler was NOT called
	if nextCalled {
		t.Error("Next handler should not have been called for unverified user")
	}

	// Verify response
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "email_not_verified" {
		t.Errorf("Expected error 'email_not_verified', got %v", response["error"])
	}

	if response["email"] != user.Email {
		t.Errorf("Expected email %s in response, got %v", user.Email, response["email"])
	}
}

// TestRequireEmailVerified_Unauthorized tests middleware without authentication
func TestRequireEmailVerified_Unauthorized(t *testing.T) {
	_, client, cleanup := setupAuthTestHandler(t)
	defer cleanup()

	// Setup Echo request without user_id in context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Don't set user_id in context

	// Create middleware and next handler
	middleware := custommiddleware.RequireEmailVerified(client)
	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "Protected content")
	}

	// Execute middleware
	handler := middleware(next)
	if err := handler(c); err != nil {
		t.Fatalf("Middleware returned error: %v", err)
	}

	// Verify next handler was NOT called
	if nextCalled {
		t.Error("Next handler should not have been called without authentication")
	}

	// Verify response
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "unauthorized" {
		t.Errorf("Expected error 'unauthorized', got %s", response["error"])
	}
}
