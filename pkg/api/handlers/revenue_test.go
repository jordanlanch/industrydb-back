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

func setupRevenueHandler(t *testing.T) (*RevenueHandler, func()) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	handler := NewRevenueHandler(client)
	return handler, func() { client.Close() }
}

// --- GetMonthlyRevenueForecast ---

func TestGetMonthlyRevenueForecast_Success_DefaultMonths(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetMonthlyRevenueForecast(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetMonthlyRevenueForecast_Success_CustomMonths(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast?months=6", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetMonthlyRevenueForecast(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetMonthlyRevenueForecast_InvalidMonths(t *testing.T) {
	tests := []struct {
		name   string
		months string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
		{"too_large", "25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupRevenueHandler(t)
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast?months="+tt.months, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetMonthlyRevenueForecast(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_months", errResp.Error)
		})
	}
}

func TestGetMonthlyRevenueForecast_MonthsBoundary(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	tests := []struct {
		name   string
		months string
		expect int
	}{
		{"min_valid", "1", http.StatusOK},
		{"max_valid", "24", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast?months="+tt.months, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetMonthlyRevenueForecast(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, rec.Code)
		})
	}
}

// --- GetAnnualRevenueForecast ---

func TestGetAnnualRevenueForecast_Success(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/annual-forecast", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetAnnualRevenueForecast(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- GetRevenueByTier ---

func TestGetRevenueByTier_Success(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/by-tier", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetRevenueByTier(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- GetGrowthRate ---

func TestGetGrowthRate_Success_DefaultMonths(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/growth-rate", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetGrowthRate(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "growth_rate")
	assert.Contains(t, resp, "months")
	assert.Equal(t, float64(3), resp["months"])
}

func TestGetGrowthRate_Success_CustomMonths(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/growth-rate?months=6", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetGrowthRate(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, float64(6), resp["months"])
}

func TestGetGrowthRate_InvalidMonths(t *testing.T) {
	tests := []struct {
		name   string
		months string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
		{"too_large", "13"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupRevenueHandler(t)
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/growth-rate?months="+tt.months, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetGrowthRate(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_months", errResp.Error)
		})
	}
}

func TestGetGrowthRate_MonthsBoundary(t *testing.T) {
	handler, cleanup := setupRevenueHandler(t)
	defer cleanup()

	tests := []struct {
		name   string
		months string
		expect int
	}{
		{"min_valid", "1", http.StatusOK},
		{"max_valid", "12", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/growth-rate?months="+tt.months, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetGrowthRate(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, rec.Code)
		})
	}
}

// --- Admin Middleware Integration ---

func TestRevenueHandler_AdminMiddleware_BlocksRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	handler := NewRevenueHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", regularUser.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetMonthlyRevenueForecast)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRevenueHandler_AdminMiddleware_AllowsAdmin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	admin, _, _ := createAdminAndRegularUser(t, client)

	handler := NewRevenueHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", admin.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetMonthlyRevenueForecast)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRevenueHandler_AdminMiddleware_BlocksUnauthenticated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	handler := NewRevenueHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/revenue/monthly-forecast", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetMonthlyRevenueForecast)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
