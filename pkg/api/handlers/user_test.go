package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestHandler creates a test handler with in-memory database
func setupTestHandler(t *testing.T) (*UserHandler, *ent.Client, func()) {
	// Create in-memory SQLite database for testing
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// Create test services
	leadService := leads.NewService(client, nil) // nil cache for testing
	auditLogger := audit.NewService(client)

	// Create handler (nil billingService for tests)
	handler := NewUserHandler(client, leadService, auditLogger, nil)

	// Return cleanup function
	cleanup := func() {
		client.Close()
	}

	return handler, client, cleanup
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, client *ent.Client) *ent.User {
	user, err := client.User.Create().
		SetEmail("test@example.com").
		SetPasswordHash("$2a$10$test_hash").
		SetName("Test User").
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(10).
		SetUsageLimit(50).
		SetEmailVerified(true).
		SetAcceptedTermsAt(time.Now()).
		Save(context.Background())

	require.NoError(t, err)
	return user
}

func TestExportPersonalData_Success(t *testing.T) {
	handler, client, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test user
	user := createTestUser(t, client)

	// Create test subscription
	sub, err := client.Subscription.Create().
		SetUserID(user.ID).
		SetTier("pro").
		SetStatus("active").
		SetStripeSubscriptionID("sub_test123").
		SetCurrentPeriodStart(time.Now()).
		SetCurrentPeriodEnd(time.Now().Add(30 * 24 * time.Hour)).
		Save(context.Background())
	require.NoError(t, err)

	// Create test export
	exp, err := client.Export.Create().
		SetUserID(user.ID).
		SetFormat("csv").
		SetFiltersApplied(map[string]interface{}{"industry": "tattoo"}).
		SetLeadCount(100).
		SetStatus("ready").
		SetFileURL("https://example.com/export.csv").
		SetExpiresAt(time.Now().Add(7 * 24 * time.Hour)).
		Save(context.Background())
	require.NoError(t, err)

	// Setup Echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/data-export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	err = handler.ExportPersonalData(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify user data
	userData, ok := response["user"].(map[string]interface{})
	require.True(t, ok, "user field should exist")
	assert.Equal(t, float64(user.ID), userData["id"])
	assert.Equal(t, user.Email, userData["email"])
	assert.Equal(t, user.Name, userData["name"])
	assert.Equal(t, string(user.SubscriptionTier), userData["subscription_tier"])

	// Verify usage data
	usageData, ok := response["usage"].(map[string]interface{})
	require.True(t, ok, "usage field should exist")
	assert.Equal(t, float64(user.UsageCount), usageData["usage_count"])
	assert.Equal(t, float64(user.UsageLimit), usageData["usage_limit"])

	// Verify subscription history
	subscriptions, ok := response["subscription_history"].([]interface{})
	require.True(t, ok, "subscription_history field should exist")
	assert.Len(t, subscriptions, 1)
	subData := subscriptions[0].(map[string]interface{})
	assert.Equal(t, float64(sub.ID), subData["id"])
	assert.Equal(t, string(sub.Tier), subData["tier"])
	assert.Equal(t, string(sub.Status), subData["status"])

	// Verify export history
	exports, ok := response["export_history"].([]interface{})
	require.True(t, ok, "export_history field should exist")
	assert.Len(t, exports, 1)
	expData := exports[0].(map[string]interface{})
	assert.Equal(t, float64(exp.ID), expData["id"])
	assert.Equal(t, string(exp.Format), expData["format"])
	assert.Equal(t, float64(exp.LeadCount), expData["lead_count"])

	// Verify metadata
	metadata, ok := response["export_metadata"].(map[string]interface{})
	require.True(t, ok, "export_metadata field should exist")
	assert.Equal(t, "JSON", metadata["format"])
	assert.Equal(t, "1.0", metadata["version"])
	assert.NotEmpty(t, metadata["exported_at"])

	// Verify headers
	assert.Equal(t, "attachment; filename=industrydb-personal-data.json", rec.Header().Get("Content-Disposition"))
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestExportPersonalData_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Setup Echo context without user_id
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/data-export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	// Execute handler
	err := handler.ExportPersonalData(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response["error"])
}

func TestExportPersonalData_UserNotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Setup Echo context with non-existent user
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/data-export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 99999) // Non-existent user

	// Execute handler
	err := handler.ExportPersonalData(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportPersonalData_EmptyHistory(t *testing.T) {
	handler, client, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test user without subscriptions or exports
	user := createTestUser(t, client)

	// Setup Echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/data-export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	err := handler.ExportPersonalData(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify empty arrays
	subscriptions, ok := response["subscription_history"].([]interface{})
	require.True(t, ok)
	assert.Len(t, subscriptions, 0)

	exports, ok := response["export_history"].([]interface{})
	require.True(t, ok)
	assert.Len(t, exports, 0)
}

func TestDeleteAccount_Success(t *testing.T) {
	handler, client, cleanup := setupTestHandler(t)
	defer cleanup()

	// Generate proper password hash
	passwordHash, err := auth.HashPassword("password123")
	require.NoError(t, err)

	// Create test user with known password
	user, err := client.User.Create().
		SetEmail("delete@example.com").
		SetPasswordHash(passwordHash).
		SetName("Delete User").
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		SetEmailVerified(true).
		Save(context.Background())
	require.NoError(t, err)

	// Create test export
	_, err = client.Export.Create().
		SetUserID(user.ID).
		SetFormat("csv").
		SetFiltersApplied(map[string]interface{}{}).
		SetLeadCount(50).
		SetStatus("ready").
		Save(context.Background())
	require.NoError(t, err)

	// Setup Echo context
	e := echo.New()
	reqBody := `{"password":"password123"}`
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/account", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	err = handler.DeleteAccount(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify user was anonymized
	deletedUser, err := client.User.Get(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Contains(t, deletedUser.Email, "deleted_")
	assert.Contains(t, deletedUser.Email, "@deleted.local")
	assert.Equal(t, "Deleted User", deletedUser.Name)
	assert.Equal(t, "deleted", deletedUser.PasswordHash)
	assert.False(t, deletedUser.EmailVerified)
	assert.Nil(t, deletedUser.StripeCustomerID)

	// Verify exports were marked as expired
	exports, err := client.Export.Query().Where().All(context.Background())
	require.NoError(t, err)
	for _, exp := range exports {
		if exp.UserID == user.ID {
			assert.Equal(t, "expired", string(exp.Status))
		}
	}
}

func TestDeleteAccount_InvalidPassword(t *testing.T) {
	handler, client, cleanup := setupTestHandler(t)
	defer cleanup()

	// Generate proper password hash
	passwordHash, err := auth.HashPassword("password123")
	require.NoError(t, err)

	// Create test user
	user, err := client.User.Create().
		SetEmail("delete@example.com").
		SetPasswordHash(passwordHash).
		SetName("Delete User").
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		SetEmailVerified(true).
		Save(context.Background())
	require.NoError(t, err)

	// Setup Echo context with wrong password
	e := echo.New()
	reqBody := `{"password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/account", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	err = handler.DeleteAccount(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_password", response["error"])
}

func TestDeleteAccount_MissingPassword(t *testing.T) {
	handler, client, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create test user
	user := createTestUser(t, client)

	// Setup Echo context without password
	e := echo.New()
	reqBody := `{}`
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/account", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	err := handler.DeleteAccount(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteAccount_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Setup Echo context without user_id
	e := echo.New()
	reqBody := `{"password":"password123"}`
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/account", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	// Execute handler
	err := handler.DeleteAccount(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
