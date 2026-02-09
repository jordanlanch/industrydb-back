package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/jobs"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupJobsHandler(t *testing.T) (*JobsHandler, func()) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	monitor := jobs.NewDataMonitor(client, nil, nil) // nil cache and nil logger (uses default)
	handler := NewJobsHandler(monitor)
	return handler, func() { client.Close() }
}

// --- DetectLowDataHandler ---

func TestDetectLowDataHandler_Success_DefaultThreshold(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-low-data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.DetectLowDataHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, float64(100), resp["threshold"])
	assert.Contains(t, resp, "count")
	assert.Contains(t, resp, "pairs")
}

func TestDetectLowDataHandler_Success_CustomThreshold(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-low-data?threshold=50", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.DetectLowDataHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, float64(50), resp["threshold"])
}

func TestDetectLowDataHandler_InvalidThresholdUsesDefault(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	tests := []struct {
		name      string
		threshold string
		expected  float64
	}{
		{"non_numeric", "abc", 100},
		{"zero", "0", 100},
		{"negative", "-5", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-low-data?threshold="+tt.threshold, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.DetectLowDataHandler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.Equal(t, tt.expected, resp["threshold"])
		})
	}
}

// --- DetectMissingHandler ---

func TestDetectMissingHandler_Success(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-missing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.DetectMissingHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "count")
	assert.Contains(t, resp, "pairs")

	// With empty DB, all combinations should be missing
	count := resp["count"].(float64)
	assert.Greater(t, count, float64(0))
}

// --- TriggerFetchHandler ---

func TestTriggerFetchHandler_InvalidBody(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	body := `not valid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/trigger-fetch", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.TriggerFetchHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid request body", resp["error"])
}

// TriggerFetchHandler and TriggerBatchFetchHandler call cache.Client methods
// which require a real Redis connection. These handlers are tested via:
// - Validation tests (invalid body returns 400)
// - Admin middleware integration tests (block before reaching service call)
// Full integration tests would require Redis.

func TestTriggerBatchFetchHandler_InvalidBody(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	body := `not valid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/trigger-batch-fetch", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.TriggerBatchFetchHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetPopulationStatsHandler ---

func TestGetPopulationStatsHandler_Success(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetPopulationStatsHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "total_leads")
	assert.Equal(t, float64(0), resp["total_leads"]) // Empty DB
}

// --- AutoPopulateHandler ---

func TestAutoPopulateHandler_InvalidBody(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	body := `not valid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/auto-populate", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.AutoPopulateHandler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAutoPopulateHandler_EmptyBody_UsesDefaults(t *testing.T) {
	handler, cleanup := setupJobsHandler(t)
	defer cleanup()

	e := echo.New()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/auto-populate", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.AutoPopulateHandler(c)
	require.NoError(t, err)
	// With empty DB, DetectLowDataIndustries returns empty (no leads to be "low")
	// But IncludeMissing is false by default, so with 0 low data pairs, returns 200 "No industries need population"
	// Actually, DetectLowDataIndustries only returns pairs that exist but have low count
	// So with empty DB, it returns empty, and without include_missing, result is 0 pairs
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "No industries need population", resp["message"])
	assert.Equal(t, float64(0), resp["count"])
}

// TestAutoPopulateHandler_IncludeMissing is skipped because include_missing=true
// triggers TriggerDataFetchBatch which requires a Redis cache connection.
// The no-missing path (empty DB, default settings) is tested above.

// --- Admin Middleware Integration ---

func TestJobsHandler_AdminMiddleware_BlocksRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	monitor := jobs.NewDataMonitor(client, nil, nil)
	handler := NewJobsHandler(monitor)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-low-data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", regularUser.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.DetectLowDataHandler)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "insufficient_permissions", resp["error"])
}

func TestJobsHandler_AdminMiddleware_AllowsAdmin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	admin, _, _ := createAdminAndRegularUser(t, client)

	monitor := jobs.NewDataMonitor(client, nil, nil)
	handler := NewJobsHandler(monitor)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-low-data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", admin.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.DetectLowDataHandler)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJobsHandler_AdminMiddleware_BlocksUnauthenticated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	monitor := jobs.NewDataMonitor(client, nil, nil)
	handler := NewJobsHandler(monitor)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/detect-low-data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	mw := requireAdminMiddleware(client)
	h := mw(handler.DetectLowDataHandler)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJobsHandler_AdminMiddleware_AllMethods_BlockRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	monitor := jobs.NewDataMonitor(client, nil, nil)
	handler := NewJobsHandler(monitor)

	methods := []struct {
		name    string
		handler echo.HandlerFunc
		method  string
	}{
		{"DetectLowData", handler.DetectLowDataHandler, http.MethodPost},
		{"DetectMissing", handler.DetectMissingHandler, http.MethodPost},
		{"GetPopulationStats", handler.GetPopulationStatsHandler, http.MethodGet},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(m.method, "/api/v1/admin/jobs/endpoint", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_id", regularUser.ID)

			mw := requireAdminMiddleware(client)
			h := mw(m.handler)
			err := h(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}
