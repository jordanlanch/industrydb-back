package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimiter holds the rate limiters for different IPs
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex
	r        rate.Limit // requests per second
	b        int        // burst
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	// Convert requests per minute to requests per second
	rps := float64(requestsPerMinute) / 60.0

	rl := &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		r:        rate.Limit(rps),
		b:        burst,
	}

	// Clean up old visitors every 3 minutes
	go rl.cleanupVisitors()

	return rl
}

// GetLimiter returns the rate limiter for the given IP
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.visitors[ip] = limiter
	}

	return limiter
}

// cleanupVisitors removes inactive visitors every 3 minutes
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(3 * time.Minute)

		rl.mu.Lock()
		// Remove visitors that haven't made requests recently
		for ip, limiter := range rl.visitors {
			// If limiter has full tokens (hasn't been used), remove it
			if limiter.Tokens() >= float64(rl.b) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates an Echo middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get IP address
			ip := c.RealIP()
			if ip == "" {
				ip = c.Request().RemoteAddr
			}

			// Get limiter for this IP
			limiter := rl.GetLimiter(ip)

			// Check if request is allowed
			if !limiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
				})
			}

			return next(c)
		}
	}
}

// NewPerEndpointRateLimiter creates a rate limiter with custom limits per endpoint
func NewPerEndpointRateLimiter(requestsPerMinute, burst int) *PerEndpointRateLimiter {
	rps := float64(requestsPerMinute) / 60.0

	return &PerEndpointRateLimiter{
		limiters: make(map[string]*RateLimiter),
		defaultR: rate.Limit(rps),
		defaultB: burst,
	}
}

// PerEndpointRateLimiter allows different rate limits per endpoint
type PerEndpointRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	defaultR rate.Limit
	defaultB int
}

// SetEndpointLimit sets a custom rate limit for a specific endpoint
func (perl *PerEndpointRateLimiter) SetEndpointLimit(endpoint string, requestsPerMinute, burst int) {
	perl.mu.Lock()
	defer perl.mu.Unlock()

	perl.limiters[endpoint] = NewRateLimiter(requestsPerMinute, burst)
}

// RateLimitMiddleware creates middleware with endpoint-specific limits
func (perl *PerEndpointRateLimiter) RateLimitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			endpoint := c.Request().Method + " " + c.Path()

			perl.mu.RLock()
			limiter, exists := perl.limiters[endpoint]
			perl.mu.RUnlock()

			// If no specific limiter, use default
			if !exists {
				limiter = NewRateLimiter(int(perl.defaultR*60), perl.defaultB)
				perl.mu.Lock()
				perl.limiters[endpoint] = limiter
				perl.mu.Unlock()
			}

			// Apply rate limiting
			return limiter.RateLimitMiddleware()(next)(c)
		}
	}
}
