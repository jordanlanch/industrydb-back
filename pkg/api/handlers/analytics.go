package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/labstack/echo/v4"
)

// AnalyticsHandler handles analytics endpoints
type AnalyticsHandler struct {
	analyticsService *analytics.Service
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetDailyUsage returns daily usage statistics
func (h *AnalyticsHandler) GetDailyUsage(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	// Get days parameter (default: 30)
	daysStr := c.QueryParam("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get daily usage
	usage, err := h.analyticsService.GetDailyUsage(ctx, userID, days)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"daily_usage": usage,
		"days":        days,
	})
}

// GetUsageSummary returns aggregated usage statistics
func (h *AnalyticsHandler) GetUsageSummary(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	// Get days parameter (default: 30)
	daysStr := c.QueryParam("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get summary
	summary, err := h.analyticsService.GetUsageSummary(ctx, userID, days)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	return c.JSON(http.StatusOK, summary)
}

// GetActionBreakdown returns usage breakdown by action type
func (h *AnalyticsHandler) GetActionBreakdown(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	// Get days parameter (default: 30)
	daysStr := c.QueryParam("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get breakdown
	breakdown, err := h.analyticsService.GetActionBreakdown(ctx, userID, days)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"breakdown": breakdown,
		"days":      days,
	})
}
