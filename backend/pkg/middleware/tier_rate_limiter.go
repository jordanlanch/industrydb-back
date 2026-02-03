package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// TierLimits defines rate limits for each subscription tier
type TierLimits struct {
	RequestsPerMinute int
	Burst             int
}

// TierRateLimiter implements tier-based rate limiting
type TierRateLimiter struct {
	// Limiters for authenticated users (by user ID)
	userLimiters map[int]*rate.Limiter
	// Limiters for unauthenticated users (by IP)
	ipLimiters map[string]*rate.Limiter
	mu         sync.RWMutex

	// Rate limits by tier
	tierLimits map[string]TierLimits

	// Default limits for unauthenticated requests
	defaultLimits TierLimits
}

// NewTierRateLimiter creates a new tier-based rate limiter
func NewTierRateLimiter() *TierRateLimiter {
	trl := &TierRateLimiter{
		userLimiters: make(map[int]*rate.Limiter),
		ipLimiters:   make(map[string]*rate.Limiter),
		tierLimits: map[string]TierLimits{
			"free": {
				RequestsPerMinute: 60,  // 1 request per second
				Burst:             10,  // Allow burst of 10
			},
			"starter": {
				RequestsPerMinute: 120, // 2 requests per second
				Burst:             20,
			},
			"pro": {
				RequestsPerMinute: 300, // 5 requests per second
				Burst:             50,
			},
			"business": {
				RequestsPerMinute: 600,  // 10 requests per second
				Burst:             100,  // Allow larger bursts
			},
		},
		defaultLimits: TierLimits{
			RequestsPerMinute: 30, // Unauthenticated users: 30 req/min
			Burst:             5,
		},
	}

	// Cleanup goroutine
	go trl.cleanupLimiters()

	return trl
}

// getUserLimiter returns or creates a rate limiter for a user based on their tier
func (trl *TierRateLimiter) getUserLimiter(userID int, tier string) *rate.Limiter {
	trl.mu.Lock()
	defer trl.mu.Unlock()

	// Check if limiter exists
	if limiter, exists := trl.userLimiters[userID]; exists {
		return limiter
	}

	// Get limits for this tier
	limits, exists := trl.tierLimits[tier]
	if !exists {
		limits = trl.tierLimits["free"] // Default to free tier
	}

	// Create new limiter
	rps := float64(limits.RequestsPerMinute) / 60.0
	limiter := rate.NewLimiter(rate.Limit(rps), limits.Burst)
	trl.userLimiters[userID] = limiter

	return limiter
}

// getIPLimiter returns or creates a rate limiter for an IP address
func (trl *TierRateLimiter) getIPLimiter(ip string) *rate.Limiter {
	trl.mu.Lock()
	defer trl.mu.Unlock()

	if limiter, exists := trl.ipLimiters[ip]; exists {
		return limiter
	}

	rps := float64(trl.defaultLimits.RequestsPerMinute) / 60.0
	limiter := rate.NewLimiter(rate.Limit(rps), trl.defaultLimits.Burst)
	trl.ipLimiters[ip] = limiter

	return limiter
}

// cleanupLimiters removes inactive limiters every 5 minutes
func (trl *TierRateLimiter) cleanupLimiters() {
	for {
		time.Sleep(5 * time.Minute)

		trl.mu.Lock()

		// Cleanup user limiters
		for userID, limiter := range trl.userLimiters {
			// If limiter has full burst tokens, it hasn't been used recently
			if limiter.Tokens() >= float64(limiter.Burst()) {
				delete(trl.userLimiters, userID)
			}
		}

		// Cleanup IP limiters
		for ip, limiter := range trl.ipLimiters {
			if limiter.Tokens() >= float64(limiter.Burst()) {
				delete(trl.ipLimiters, ip)
			}
		}

		trl.mu.Unlock()
	}
}

// Middleware creates an Echo middleware for tier-based rate limiting
func (trl *TierRateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var limiter *rate.Limiter

			// Try to get user ID and tier from context (set by auth middleware)
			userID, hasUserID := c.Get("user_id").(int)
			tier, hasTier := c.Get("user_tier").(string)

			if hasUserID && hasTier {
				// Authenticated user - use tier-based limiting
				limiter = trl.getUserLimiter(userID, tier)
			} else {
				// Unauthenticated user - use IP-based limiting
				ip := c.RealIP()
				if ip == "" {
					ip = c.Request().RemoteAddr
				}
				limiter = trl.getIPLimiter(ip)
			}

			// Check if request is allowed
			if !limiter.Allow() {
				// Get user tier for error message
				tierInfo := "unauthenticated"
				if hasTier {
					tierInfo = tier
				}

				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"message": "Rate limit exceeded for " + tierInfo + " tier. Please upgrade for higher limits or try again later.",
					"tier":    tierInfo,
				})
			}

			return next(c)
		}
	}
}

// GetTierLimits returns the rate limits for a specific tier
func (trl *TierRateLimiter) GetTierLimits(tier string) (TierLimits, bool) {
	trl.mu.RLock()
	defer trl.mu.RUnlock()

	limits, exists := trl.tierLimits[tier]
	return limits, exists
}

// SetTierLimits allows customizing rate limits for a tier
func (trl *TierRateLimiter) SetTierLimits(tier string, requestsPerMinute, burst int) {
	trl.mu.Lock()
	defer trl.mu.Unlock()

	trl.tierLimits[tier] = TierLimits{
		RequestsPerMinute: requestsPerMinute,
		Burst:             burst,
	}
}
