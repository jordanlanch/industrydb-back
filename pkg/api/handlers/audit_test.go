package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupAuditHandler(t *testing.T) (*AuditHandler, *ent.Client, func()) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	auditService := audit.NewService(client)
	handler := NewAuditHandler(auditService)
	return handler, client, func() { client.Close() }
}

func createAuditTestUser(t *testing.T, client *ent.Client) *ent.User {
	t.Helper()
	ctx := context.Background()
	u, err := client.User.Create().
		SetEmail("audit-test@example.com").
		SetName("Audit Test User").
		SetPasswordHash("hashed_password").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		SetEmailVerifiedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return u
}

func createAuditTestLogs(t *testing.T, client *ent.Client, userID int, count int) {
	t.Helper()
	ctx := context.Background()
	auditService := audit.NewService(client)

	for i := 0; i < count; i++ {
		err := auditService.LogUserLogin(ctx, userID, "127.0.0.1", "test-agent")
		require.NoError(t, err)
	}
}

// --- GetUserLogs ---

func TestGetUserLogs_Success(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)
	createAuditTestLogs(t, client, testUser.ID, 5)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", testUser.ID)

	err := handler.GetUserLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "logs")
	assert.Contains(t, resp, "count")
	assert.Equal(t, float64(5), resp["count"])
}

func TestGetUserLogs_WithLimit(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)
	createAuditTestLogs(t, client, testUser.ID, 10)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs?limit=3", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", testUser.ID)

	err := handler.GetUserLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, float64(3), resp["count"])
}

func TestGetUserLogs_DefaultLimit(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", testUser.ID)

	err := handler.GetUserLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUserLogs_InvalidLimitUsesDefault(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)

	tests := []struct {
		name  string
		limit string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
		{"over_max", "101"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs?limit="+tt.limit, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_id", testUser.ID)

			err := handler.GetUserLogs(c)
			require.NoError(t, err)
			// Invalid limits silently fall back to default (50)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestGetUserLogs_Unauthorized(t *testing.T) {
	handler, _, cleanup := setupAuditHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	err := handler.GetUserLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "unauthorized", resp["error"])
}

func TestGetUserLogs_NoLogs(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", testUser.ID)

	err := handler.GetUserLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["count"])
}

// --- GetRecentLogs (admin only) ---

func TestGetRecentLogs_Success(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)
	createAuditTestLogs(t, client, testUser.ID, 5)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/recent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetRecentLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "logs")
	assert.Contains(t, resp, "count")
	assert.Equal(t, float64(5), resp["count"])
}

func TestGetRecentLogs_WithLimit(t *testing.T) {
	handler, client, cleanup := setupAuditHandler(t)
	defer cleanup()

	testUser := createAuditTestUser(t, client)
	createAuditTestLogs(t, client, testUser.ID, 10)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/recent?limit=2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetRecentLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, float64(2), resp["count"])
}

func TestGetRecentLogs_InvalidLimitUsesDefault(t *testing.T) {
	handler, _, cleanup := setupAuditHandler(t)
	defer cleanup()

	tests := []struct {
		name  string
		limit string
	}{
		{"non_numeric", "xyz"},
		{"zero", "0"},
		{"over_max", "501"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/recent?limit="+tt.limit, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetRecentLogs(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

// --- GetCriticalLogs (admin only) ---

func TestGetCriticalLogs_Success(t *testing.T) {
	handler, _, cleanup := setupAuditHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/critical", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCriticalLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "logs")
	assert.Contains(t, resp, "count")
}

func TestGetCriticalLogs_WithLimit(t *testing.T) {
	handler, _, cleanup := setupAuditHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/critical?limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCriticalLogs(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCriticalLogs_InvalidLimitUsesDefault(t *testing.T) {
	handler, _, cleanup := setupAuditHandler(t)
	defer cleanup()

	tests := []struct {
		name  string
		limit string
	}{
		{"non_numeric", "abc"},
		{"over_max", "201"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/critical?limit="+tt.limit, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCriticalLogs(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

// --- Admin Middleware Integration ---

func TestAuditHandler_AdminMiddleware_RecentLogs_BlocksRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	auditService := audit.NewService(client)
	handler := NewAuditHandler(auditService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/recent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", regularUser.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetRecentLogs)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuditHandler_AdminMiddleware_RecentLogs_AllowsAdmin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	admin, _, _ := createAdminAndRegularUser(t, client)

	auditService := audit.NewService(client)
	handler := NewAuditHandler(auditService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/recent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", admin.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetRecentLogs)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuditHandler_AdminMiddleware_CriticalLogs_BlocksUnauthenticated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	auditService := audit.NewService(client)
	handler := NewAuditHandler(auditService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/critical", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetCriticalLogs)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuditHandler_AdminMiddleware_CriticalLogs_BlocksRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	auditService := audit.NewService(client)
	handler := NewAuditHandler(auditService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/critical", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", regularUser.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetCriticalLogs)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
