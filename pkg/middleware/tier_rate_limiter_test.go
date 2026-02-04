package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestTierRateLimiter_FreeTier(t *testing.T) {
	trl := NewTierRateLimiter()
	e := echo.New()

	// Free tier: 60 requests/minute (1 per second), burst 10
	handler := trl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Set user context (free tier)
	for i := 0; i < 12; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", 1)
		c.Set("user_tier", "free")

		err := handler(c)

		if i < 10 {
			// First 10 requests should succeed (burst)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			// 11th and 12th requests should be rate limited
			assert.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		}
	}
}

func TestTierRateLimiter_BusinessTier(t *testing.T) {
	trl := NewTierRateLimiter()
	e := echo.New()

	// Business tier: 600 requests/minute (10 per second), burst 100
	handler := trl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	successCount := 0
	// Business tier should allow many more requests
	for i := 0; i < 105; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", 2)
		c.Set("user_tier", "business")

		err := handler(c)
		assert.NoError(t, err)

		if rec.Code == http.StatusOK {
			successCount++
		}
	}

	// Business tier should allow at least 100 requests (burst)
	assert.GreaterOrEqual(t, successCount, 100)
}

func TestTierRateLimiter_UnauthenticatedUser(t *testing.T) {
	trl := NewTierRateLimiter()
	e := echo.New()

	// Unauthenticated: 30 requests/minute, burst 5
	handler := trl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	for i := 0; i < 7; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)

		if i < 5 {
			// First 5 requests should succeed (burst)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			// 6th and 7th requests should be rate limited
			assert.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		}
	}
}

func TestTierRateLimiter_DifferentUsers(t *testing.T) {
	trl := NewTierRateLimiter()
	e := echo.New()

	handler := trl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// User 1 (free tier) makes requests
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", 1)
		c.Set("user_tier", "free")
		handler(c)
	}

	// User 2 (free tier) should have their own rate limit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 2)
	c.Set("user_tier", "free")

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code, "User 2 should not be rate limited by user 1's usage")
}

func TestTierRateLimiter_TierComparison(t *testing.T) {
	trl := NewTierRateLimiter()

	tiers := []string{"free", "starter", "pro", "business"}
	expectedLimits := []int{60, 120, 300, 600}

	for i, tier := range tiers {
		limits, exists := trl.GetTierLimits(tier)
		assert.True(t, exists, "Tier %s should exist", tier)
		assert.Equal(t, expectedLimits[i], limits.RequestsPerMinute, "Tier %s should have %d requests/minute", tier, expectedLimits[i])
	}
}

func TestTierRateLimiter_SetCustomLimits(t *testing.T) {
	trl := NewTierRateLimiter()

	// Set custom limits for enterprise tier
	trl.SetTierLimits("enterprise", 1200, 200)

	limits, exists := trl.GetTierLimits("enterprise")
	assert.True(t, exists)
	assert.Equal(t, 1200, limits.RequestsPerMinute)
	assert.Equal(t, 200, limits.Burst)
}

func TestTierRateLimiter_ErrorMessage(t *testing.T) {
	trl := NewTierRateLimiter()
	e := echo.New()

	handler := trl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Exceed free tier limit
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", 1)
		c.Set("user_tier", "free")
		handler(c)

		if i == 10 {
			// Check error message
			assert.Contains(t, rec.Body.String(), "free")
			assert.Contains(t, rec.Body.String(), "rate_limit_exceeded")
		}
	}
}

func TestTierRateLimiter_TokenRefill(t *testing.T) {
	trl := NewTierRateLimiter()
	e := echo.New()

	// Free tier: 60 req/min = 1 req/second
	handler := trl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Exhaust burst
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", 1)
		c.Set("user_tier", "free")
		handler(c)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)
	c.Set("user_tier", "free")
	handler(c)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Wait for token to refill (1 second for free tier)
	time.Sleep(1100 * time.Millisecond)

	// Should succeed now
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set("user_id", 1)
	c.Set("user_tier", "free")
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code, "Request should succeed after token refill")
}
