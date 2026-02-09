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
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// setupAnalyticsTest creates test database and analytics handler
func setupAnalyticsTest(t *testing.T) (*ent.Client, *AnalyticsHandler, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	analyticsService := analytics.NewService(client)
	handler := NewAnalyticsHandler(analyticsService)

	cleanup := func() {
		client.Close()
	}

	return client, handler, cleanup
}

// createTestUsageLogs creates test usage logs for a user
func createTestUsageLogs(t *testing.T, client *ent.Client, userID int, days int) {
	ctx := context.Background()
	now := time.Now()

	// Create logs for the last N days
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)

		// Create search action
		_, err := client.UsageLog.Create().
			SetUserID(userID).
			SetAction("search").
			SetCount(1).
			SetMetadata(map[string]interface{}{"industry": "tattoo"}).
			SetCreatedAt(date).
			Save(ctx)
		require.NoError(t, err)

		// Create export action every 3 days
		if i%3 == 0 {
			_, err := client.UsageLog.Create().
				SetUserID(userID).
				SetAction("export").
				SetCount(10).
				SetMetadata(map[string]interface{}{"format": "csv"}).
				SetCreatedAt(date).
				Save(ctx)
			require.NoError(t, err)
		}

		// Create API action every 5 days
		if i%5 == 0 {
			_, err := client.UsageLog.Create().
				SetUserID(userID).
				SetAction("api_call").
				SetCount(1).
				SetMetadata(map[string]interface{}{"endpoint": "/api/v1/leads"}).
				SetCreatedAt(date).
				Save(ctx)
			require.NoError(t, err)
		}
	}
}

// Helper function to create test user for analytics tests
func createAnalyticsTestUser(t *testing.T, client *ent.Client) *ent.User {
	ctx := context.Background()
	user, err := client.User.Create().
		SetEmail("analytics-test@example.com").
		SetPasswordHash("$2a$10$dummyhash").
		SetName("Analytics Test User").
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageLimit(50).
		SetUsageCount(0).
		SetLastResetAt(time.Now()).
		SetEmailVerified(true).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func TestGetDailyUsage_Success(t *testing.T) {
	client, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	// Create test user
	user := createAnalyticsTestUser(t, client)

	// Create usage logs for last 30 days
	createTestUsageLogs(t, client, user.ID, 30)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/daily?days=30", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	// Execute handler
	err := handler.GetDailyUsage(c)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(30), response["days"])
	assert.NotNil(t, response["daily_usage"])

	dailyUsage := response["daily_usage"].([]interface{})
	assert.Greater(t, len(dailyUsage), 0, "Should have daily usage data")
}

func TestGetDailyUsage_CustomDays(t *testing.T) {
	client, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	user := createAnalyticsTestUser(t, client)
	createTestUsageLogs(t, client, user.ID, 7)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/daily?days=7", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.GetDailyUsage(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(7), response["days"])
}

func TestGetDailyUsage_InvalidDays(t *testing.T) {
	_, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	testCases := []struct {
		name          string
		daysParam     string
		expectedDays  float64
		description   string
	}{
		{
			name:         "Negative days defaults to 30",
			daysParam:    "-5",
			expectedDays: 30,
			description:  "Invalid negative value should use default",
		},
		{
			name:         "Zero days defaults to 30",
			daysParam:    "0",
			expectedDays: 30,
			description:  "Zero should use default",
		},
		{
			name:         "Over 365 days defaults to 30",
			daysParam:    "400",
			expectedDays: 30,
			description:  "Over max should use default",
		},
		{
			name:         "Non-numeric defaults to 30",
			daysParam:    "abc",
			expectedDays: 30,
			description:  "Non-numeric should use default",
		},
		{
			name:         "Empty defaults to 30",
			daysParam:    "",
			expectedDays: 30,
			description:  "Empty should use default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			url := "/user/analytics/daily"
			if tc.daysParam != "" {
				url += "?days=" + tc.daysParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_id", 1)

			err := handler.GetDailyUsage(c)
			require.NoError(t, err)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDays, response["days"], tc.description)
		})
	}
}

func TestGetDailyUsage_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/daily", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Don't set user_id to simulate unauthorized access

	err := handler.GetDailyUsage(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUsageSummary_Success(t *testing.T) {
	client, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	user := createAnalyticsTestUser(t, client)
	createTestUsageLogs(t, client, user.ID, 30)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/summary?days=30", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.GetUsageSummary(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have summary fields
	assert.Contains(t, response, "total_actions")
	assert.Contains(t, response, "period_start")
	assert.Contains(t, response, "period_end")
}

func TestGetUsageSummary_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	err := handler.GetUsageSummary(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetActionBreakdown_Success(t *testing.T) {
	client, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	user := createAnalyticsTestUser(t, client)
	createTestUsageLogs(t, client, user.ID, 30)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/breakdown?days=30", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.GetActionBreakdown(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(30), response["days"])
	assert.NotNil(t, response["breakdown"])

	breakdown := response["breakdown"].([]interface{})
	assert.Greater(t, len(breakdown), 0, "Should have breakdown data")

	// Verify breakdown contains action types
	hasSearch := false
	hasExport := false
	hasAPI := false

	for _, item := range breakdown {
		entry := item.(map[string]interface{})
		action := entry["action"].(string)
		count := entry["count"].(float64)

		assert.Greater(t, count, float64(0), "Count should be greater than 0")

		switch action {
		case "search":
			hasSearch = true
		case "export":
			hasExport = true
		case "api_call":
			hasAPI = true
		}
	}

	assert.True(t, hasSearch, "Should have search actions")
	assert.True(t, hasExport, "Should have export actions")
	assert.True(t, hasAPI, "Should have API call actions")
}

func TestGetActionBreakdown_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/breakdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id

	err := handler.GetActionBreakdown(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetActionBreakdown_NoData(t *testing.T) {
	client, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	user := createAnalyticsTestUser(t, client)
	// Don't create any usage logs

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/user/analytics/breakdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.GetActionBreakdown(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	breakdown := response["breakdown"].([]interface{})
	assert.Equal(t, 0, len(breakdown), "Should have empty breakdown when no data")
}

func TestAnalyticsEndpoints_EdgeCases(t *testing.T) {
	client, handler, cleanup := setupAnalyticsTest(t)
	defer cleanup()

	user := createAnalyticsTestUser(t, client)

	// Test with exactly 1 day
	t.Run("One day period", func(t *testing.T) {
		createTestUsageLogs(t, client, user.ID, 1)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/user/analytics/daily?days=1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", user.ID)

		err := handler.GetDailyUsage(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	// Test with maximum days (365)
	t.Run("Maximum days period", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/user/analytics/daily?days=365", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", user.ID)

		err := handler.GetDailyUsage(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		assert.Equal(t, float64(365), response["days"])
	})
}
