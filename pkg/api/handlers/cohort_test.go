package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupCohortHandler(t *testing.T) (*CohortHandler, func()) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	handler := NewCohortHandler(client)
	return handler, func() { client.Close() }
}

// --- GetCohorts ---

func TestGetCohorts_Success_DefaultParams(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohorts(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCohorts_Success_CustomParams(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts?period=month&count=6", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohorts(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCohorts_Success_AllPeriods(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	periods := []string{"day", "week", "month"}
	for _, period := range periods {
		t.Run("period_"+period, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts?period="+period, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohorts(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestGetCohorts_InvalidPeriod(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts?period=quarter", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohorts(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_period", errResp.Error)
}

func TestGetCohorts_InvalidCount(t *testing.T) {
	tests := []struct {
		name  string
		count string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
		{"too_large", "53"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupCohortHandler(t)
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts?count="+tt.count, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohorts(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_count", errResp.Error)
		})
	}
}

func TestGetCohorts_CountBoundary(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	tests := []struct {
		name   string
		count  string
		expect int
	}{
		{"min_valid", "1", http.StatusOK},
		{"max_valid", "52", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts?count="+tt.count, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohorts(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, rec.Code)
		})
	}
}

// --- GetCohortRetention ---

func TestGetCohortRetention_ServiceError_EmptyDB(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	cohortStart := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/retention?cohort_start="+cohortStart, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortRetention(c)
	require.NoError(t, err)
	// Service returns error "no users in cohort" on empty DB
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "server_error", errResp.Error)
}

func TestGetCohortRetention_MissingCohortStart(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/retention", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortRetention(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "missing_cohort_start", errResp.Error)
}

func TestGetCohortRetention_InvalidCohortStart(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/retention?cohort_start=not-a-date", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortRetention(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_cohort_start", errResp.Error)
}

func TestGetCohortRetention_InvalidPeriod(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	cohortStart := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/retention?cohort_start="+cohortStart+"&period=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortRetention(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_period", errResp.Error)
}

func TestGetCohortRetention_InvalidPeriods(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	cohortStart := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

	tests := []struct {
		name    string
		periods string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"too_large", "53"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/retention?cohort_start="+cohortStart+"&periods="+tt.periods, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohortRetention(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_periods", errResp.Error)
		})
	}
}

// --- GetCohortComparison ---

func TestGetCohortComparison_Success_DefaultParams(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/comparison", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortComparison(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCohortComparison_InvalidPeriod(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/comparison?period=yearly", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortComparison(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetCohortComparison_InvalidCohortCount(t *testing.T) {
	tests := []struct {
		name  string
		count string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"too_large", "13"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupCohortHandler(t)
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/comparison?cohort_count="+tt.count, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohortComparison(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_cohort_count", errResp.Error)
		})
	}
}

func TestGetCohortComparison_InvalidRetentionPeriods(t *testing.T) {
	tests := []struct {
		name    string
		periods string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"too_large", "53"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, cleanup := setupCohortHandler(t)
			defer cleanup()

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/comparison?retention_periods="+tt.periods, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohortComparison(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_retention_periods", errResp.Error)
		})
	}
}

func TestGetCohortComparison_CohortCountBoundary(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	tests := []struct {
		name   string
		count  string
		expect int
	}{
		{"min_valid", "1", http.StatusOK},
		{"max_valid", "12", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/comparison?cohort_count="+tt.count, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohortComparison(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, rec.Code)
		})
	}
}

// --- GetCohortActivityMetrics ---

func TestGetCohortActivityMetrics_ServiceError_EmptyDB(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	cohortStart := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/activity?cohort_start="+cohortStart, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortActivityMetrics(c)
	require.NoError(t, err)
	// Service returns error "no users in cohort" on empty DB
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "server_error", errResp.Error)
}

func TestGetCohortActivityMetrics_MissingCohortStart(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/activity", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortActivityMetrics(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "missing_cohort_start", errResp.Error)
}

func TestGetCohortActivityMetrics_InvalidCohortStart(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/activity?cohort_start=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetCohortActivityMetrics(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp models.ErrorResponse
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_cohort_start", errResp.Error)
}

func TestGetCohortActivityMetrics_InvalidWeeks(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	cohortStart := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

	tests := []struct {
		name  string
		weeks string
	}{
		{"non_numeric", "abc"},
		{"zero", "0"},
		{"too_large", "53"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/activity?cohort_start="+cohortStart+"&weeks="+tt.weeks, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohortActivityMetrics(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp models.ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.Equal(t, "invalid_weeks", errResp.Error)
		})
	}
}

func TestGetCohortActivityMetrics_WeeksBoundary(t *testing.T) {
	handler, cleanup := setupCohortHandler(t)
	defer cleanup()

	cohortStart := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

	tests := []struct {
		name  string
		weeks string
	}{
		{"min_valid", "1"},
		{"max_valid", "52"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts/activity?cohort_start="+cohortStart+"&weeks="+tt.weeks, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.GetCohortActivityMetrics(c)
			require.NoError(t, err)
			// Valid weeks params pass validation; service may return 500 on empty DB
			// but the important thing is no 400 error from parameter validation
			assert.NotEqual(t, http.StatusBadRequest, rec.Code)
		})
	}
}

// --- Admin Middleware Integration ---

func TestCohortHandler_AdminMiddleware_BlocksRegularUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, regularUser, _ := createAdminAndRegularUser(t, client)

	handler := NewCohortHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", regularUser.ID)

	// Simulate middleware wrapping
	mw := requireAdminMiddleware(client)
	h := mw(handler.GetCohorts)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "insufficient_permissions", resp["error"])
}

func TestCohortHandler_AdminMiddleware_AllowsAdmin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	admin, _, _ := createAdminAndRegularUser(t, client)

	handler := NewCohortHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", admin.ID)

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetCohorts)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCohortHandler_AdminMiddleware_BlocksUnauthenticated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	handler := NewCohortHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cohorts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set - unauthenticated

	mw := requireAdminMiddleware(client)
	h := mw(handler.GetCohorts)
	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
