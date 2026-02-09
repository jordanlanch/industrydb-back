package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupFunnelHandler(t *testing.T) (*FunnelHandler, func()) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	handler := NewFunnelHandler(client)
	return handler, func() { client.Close() }
}

// --- GetFunnelMetrics ---

func TestGetFunnelMetrics_Success_DefaultDays(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetFunnelMetrics(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFunnelMetrics_Success_CustomDays(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics?days=90", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetFunnelMetrics(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFunnelMetrics_InvalidDays(t *testing.T) {
	tests := []struct {
		name string
		days string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
		{"too_large", "366"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupFunnelHandler(t)
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics?days="+tt.days, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetFunnelMetrics(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_days", errResp.Error)
		})
	}
}

func TestGetFunnelMetrics_DaysBoundary(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	tests := []struct {
		name   string
		days   string
		expect int
	}{
		{"min_valid", "1", http.StatusOK},
		{"max_valid", "365", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics?days="+tt.days, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetFunnelMetrics(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, rec.Code)
		})
	}
}

// --- GetFunnelDetails ---

func TestGetFunnelDetails_Success(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/details", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetFunnelDetails(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFunnelDetails_InvalidDays(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/details?days=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetFunnelDetails(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_days", errResp.Error)
}

// --- GetDropoffAnalysis ---

func TestGetDropoffAnalysis_Success(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/dropoff", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetDropoffAnalysis(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetDropoffAnalysis_InvalidDays(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/dropoff?days=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetDropoffAnalysis(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetTimeToConversion ---

func TestGetTimeToConversion_Success(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/time-to-conversion", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetTimeToConversion(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetTimeToConversion_InvalidDays(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/time-to-conversion?days=-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetTimeToConversion(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetTimeToConversion_CustomDays(t *testing.T) {
	handler, cleanup := setupFunnelHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/time-to-conversion?days=180", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetTimeToConversion(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Admin Middleware Integration ---

func TestFunnelHandler_AdminMiddleware_BlocksRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	handler := NewFunnelHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", regularUser.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetFunnelMetrics)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFunnelHandler_AdminMiddleware_AllowsAdmin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	admin, _, _ := createAdminAndRegularUser(t, client)

	handler := NewFunnelHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", admin.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetFunnelMetrics)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestFunnelHandler_AdminMiddleware_BlocksUnauthenticated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	handler := NewFunnelHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/funnel/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetFunnelMetrics)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
