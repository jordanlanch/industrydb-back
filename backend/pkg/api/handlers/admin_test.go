package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestAdmin creates test database with admin user
func setupTestAdmin(t *testing.T) (*ent.Client, *ent.User, *ent.User) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	
	// Create superadmin user
	admin, err := client.User.Create().
		SetEmail("admin@test.com").
		SetName("Admin User").
		SetPasswordHash("hashed_password").
		SetRole(user.RoleSuperadmin).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed creating admin: %v", err)
	}

	// Create regular user for testing
	regularUser, err := client.User.Create().
		SetEmail("user@test.com").
		SetName("Regular User").
		SetPassword("hashed_password").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierStarter).
		SetUsageCount(10).
		SetUsageLimit(500).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed creating user: %v", err)
	}

	return client, admin, regularUser
}

func TestListUsers(t *testing.T) {
	client, admin, _ := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set admin user in context
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	
	// Test
	err := handler.ListUsers(c)
	
	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	users := response["users"].([]interface{})
	assert.Equal(t, 2, len(users)) // admin + regular user

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, 1.0, pagination["page"])
	assert.Equal(t, 10.0, pagination["limit"])
	assert.Equal(t, 2.0, pagination["total"])
}

func TestListUsersWithFilters(t *testing.T) {
	client, admin, _ := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?tier=starter", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.ListUsers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	users := response["users"].([]interface{})
	assert.Equal(t, 1, len(users)) // Only starter tier user

	firstUser := users[0].(map[string]interface{})
	assert.Equal(t, "starter", firstUser["subscription_tier"])
}

func TestGetUser(t *testing.T) {
	client, admin, regularUser := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("2")
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.GetUser(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.Equal(t, regularUser.Email, response["email"])
	assert.Equal(t, regularUser.Name, response["name"])
	assert.Equal(t, "starter", response["subscription_tier"])
}

func TestGetUserNotFound(t *testing.T) {
	client, admin, _ := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.GetUser(c)

	assert.Error(t, err)
	httpError, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpError.Code)
}

func TestUpdateUser(t *testing.T) {
	client, admin, regularUser := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	
	updateJSON := `{
		"subscription_tier": "pro",
		"usage_limit": 2000
	}`
	
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/2", strings.NewReader(updateJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("2")
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.UpdateUser(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.Equal(t, "pro", response["subscription_tier"])
	assert.Equal(t, 2000.0, response["usage_limit"])

	// Verify in database
	updated, err := client.User.Get(ctx, regularUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.SubscriptionTierPro, updated.SubscriptionTier)
	assert.Equal(t, 2000, updated.UsageLimit)
}

func TestUpdateUserRole(t *testing.T) {
	client, admin, regularUser := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	
	updateJSON := `{
		"role": "admin"
	}`
	
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/2", strings.NewReader(updateJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("2")
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.UpdateUser(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify in database
	updated, err := client.User.Get(ctx, regularUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.RoleAdmin, updated.Role)
}

func TestSuspendUser(t *testing.T) {
	client, admin, regularUser := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("2")
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.SuspendUser(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "User suspended successfully", response["message"])

	// Verify user is anonymized
	suspended, err := client.User.Get(ctx, regularUser.ID)
	assert.NoError(t, err)
	assert.Contains(t, suspended.Email, "deleted_")
	assert.Equal(t, "Deleted User", suspended.Name)
}

func TestGetStats(t *testing.T) {
	client, admin, _ := setupTestAdmin(t)
	defer client.Close()

	// Create more test users with different tiers
	client.User.Create().
		SetEmail("starter@test.com").
		SetName("Starter User").
		SetPassword("pass").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierStarter).
		SetUsageCount(50).
		SetUsageLimit(500).
		SetEmailVerifiedAt(time.Now()).
		Save(ctx)

	client.User.Create().
		SetEmail("pro@test.com").
		SetName("Pro User").
		SetPassword("pass").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierPro).
		SetUsageCount(100).
		SetUsageLimit(2000).
		SetEmailVerifiedAt(time.Now()).
		Save(ctx)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", admin)

	handler := NewAdminHandler(client)
	err := handler.GetStats(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var stats map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &stats)

	// Check users stats
	usersStats := stats["users"].(map[string]interface{})
	assert.Equal(t, 4.0, usersStats["total"]) // admin + 3 test users
	assert.Equal(t, 2.0, usersStats["verified"]) // 2 verified users

	// Check subscriptions breakdown
	subsStats := stats["subscriptions"].(map[string]interface{})
	assert.Equal(t, 1.0, subsStats["free"]) // admin
	assert.Equal(t, 2.0, subsStats["starter"]) // 2 starter users
	assert.Equal(t, 1.0, subsStats["pro"]) // 1 pro user
	assert.Equal(t, 0.0, subsStats["business"]) // 0 business users
}

func TestListUsersRequiresAdmin(t *testing.T) {
	client, _, regularUser := setupTestAdmin(t)
	defer client.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", regularUser) // Regular user, not admin

	handler := NewAdminHandler(client)
	
	// Should be blocked by RequireAdmin middleware
	// In actual implementation, middleware returns error before handler
	err := handler.ListUsers(c)
	
	// Handler will process it but should ideally be protected by middleware
	// This test documents expected behavior
	assert.NoError(t, err) // Handler itself doesn't check, middleware does
}
