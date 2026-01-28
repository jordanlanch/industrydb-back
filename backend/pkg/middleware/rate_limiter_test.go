package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create rate limiter: 120 requests per minute (2 per second) with burst of 1
	rl := NewRateLimiter(120, 1)

	// Get limiter for IP
	limiter := rl.GetLimiter("192.168.1.1")

	// First request should be allowed
	assert.True(t, limiter.Allow(), "First request should be allowed")

	// Second request should be blocked (burst exhausted)
	assert.False(t, limiter.Allow(), "Second request should be blocked")

	// Wait for token refill (120 req/min = 2 req/sec = 0.5 seconds per token)
	time.Sleep(600 * time.Millisecond)

	// Third request should be allowed after waiting
	assert.True(t, limiter.Allow(), "Third request should be allowed after waiting")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(2, 1)

	// Get limiters for different IPs
	limiter1 := rl.GetLimiter("192.168.1.1")
	limiter2 := rl.GetLimiter("192.168.1.2")

	// Both should be allowed (different IPs have separate limiters)
	assert.True(t, limiter1.Allow(), "IP 1 first request should be allowed")
	assert.True(t, limiter2.Allow(), "IP 2 first request should be allowed")

	// Both should be blocked after burst
	assert.False(t, limiter1.Allow(), "IP 1 second request should be blocked")
	assert.False(t, limiter2.Allow(), "IP 2 second request should be blocked")
}

func TestRateLimitMiddleware(t *testing.T) {
	// Create Echo instance
	e := echo.New()
	rl := NewRateLimiter(2, 1)

	// Create test handler
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	// Wrap handler with rate limit middleware
	middleware := rl.RateLimitMiddleware()
	wrappedHandler := middleware(handler)

	// Test first request (should succeed)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)

	err := wrappedHandler(c1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Test second request from same IP (should be rate limited)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12346"
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err = wrappedHandler(c2)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "rate_limit_exceeded")
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	e := echo.New()
	rl := NewRateLimiter(2, 1)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	middleware := rl.RateLimitMiddleware()
	wrappedHandler := middleware(handler)

	// Request from IP 1
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)

	err := wrappedHandler(c1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Request from IP 2 (different IP, should still succeed)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err = wrappedHandler(c2)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestPerEndpointRateLimiter(t *testing.T) {
	perl := NewPerEndpointRateLimiter(60, 10)

	// Set custom limit for login endpoint: 5 per minute
	perl.SetEndpointLimit("POST /api/v1/auth/login", 5, 2)

	e := echo.New()
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	middleware := perl.RateLimitMiddleware()
	wrappedHandler := middleware(handler)

	// Create context with specific path
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/auth/login")

	// First 2 requests should succeed (burst = 2)
	err := wrappedHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/api/v1/auth/login")
	err = wrappedHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Third request should be rate limited
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/api/v1/auth/login")
	err = wrappedHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_BurstBehavior(t *testing.T) {
	// Rate limiter: 60 req/min (1 req/sec) with burst of 10
	rl := NewRateLimiter(60, 10)
	limiter := rl.GetLimiter("192.168.1.1")

	// Should allow burst of 10 requests immediately
	allowedCount := 0
	for i := 0; i < 15; i++ {
		if limiter.Allow() {
			allowedCount++
		}
	}

	// Should allow exactly burst size (10)
	assert.Equal(t, 10, allowedCount, "Should allow exactly burst size requests")

	// Wait for 1 second (1 token refill at 60 req/min)
	time.Sleep(1100 * time.Millisecond)

	// Should allow 1 more request after refill
	assert.True(t, limiter.Allow(), "Should allow 1 request after token refill")
}
