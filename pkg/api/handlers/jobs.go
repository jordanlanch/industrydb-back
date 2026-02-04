package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/pkg/jobs"
	"github.com/labstack/echo/v4"
)

// JobsHandler handles data acquisition job endpoints
type JobsHandler struct {
	monitor *jobs.DataMonitor
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(monitor *jobs.DataMonitor) *JobsHandler {
	return &JobsHandler{
		monitor: monitor,
	}
}

// DetectLowDataHandler triggers detection of industries with low data
// POST /api/v1/admin/jobs/detect-low-data
func (h *JobsHandler) DetectLowDataHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Get threshold from query params (default: 100)
	thresholdStr := c.QueryParam("threshold")
	threshold := 100
	if thresholdStr != "" {
		if t, err := strconv.Atoi(thresholdStr); err == nil && t > 0 {
			threshold = t
		}
	}

	// Detect low data industries
	pairs, err := h.monitor.DetectLowDataIndustries(ctx, threshold)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to detect low data industries",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"threshold": threshold,
		"count":     len(pairs),
		"pairs":     pairs,
	})
}

// DetectMissingHandler triggers detection of missing combinations
// POST /api/v1/admin/jobs/detect-missing
func (h *JobsHandler) DetectMissingHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Detect missing combinations
	pairs, err := h.monitor.DetectMissingCombinations(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to detect missing combinations",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"count": len(pairs),
		"pairs": pairs,
	})
}

// TriggerFetchHandler triggers a manual data fetch
// POST /api/v1/admin/jobs/trigger-fetch
func (h *JobsHandler) TriggerFetchHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse request body
	var req struct {
		Industry string `json:"industry" validate:"required"`
		Country  string `json:"country" validate:"required"`
		Limit    int    `json:"limit"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Default limit
	if req.Limit == 0 {
		req.Limit = 1000
	}

	// Trigger fetch
	if err := h.monitor.TriggerDataFetch(ctx, req.Industry, req.Country, req.Limit); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to trigger data fetch",
		})
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":  "Data fetch triggered",
		"industry": req.Industry,
		"country":  req.Country,
		"limit":    req.Limit,
	})
}

// TriggerBatchFetchHandler triggers batch data fetch
// POST /api/v1/admin/jobs/trigger-batch-fetch
func (h *JobsHandler) TriggerBatchFetchHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse request body
	var req struct {
		Pairs         []jobs.IndustryCountryPair `json:"pairs" validate:"required"`
		Limit         int                         `json:"limit"`
		MaxConcurrent int                         `json:"max_concurrent"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Defaults
	if req.Limit == 0 {
		req.Limit = 1000
	}
	if req.MaxConcurrent == 0 {
		req.MaxConcurrent = 3
	}

	// Trigger batch fetch
	if err := h.monitor.TriggerDataFetchBatch(ctx, req.Pairs, req.Limit, req.MaxConcurrent); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to trigger batch fetch",
		})
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":        "Batch fetch triggered",
		"count":          len(req.Pairs),
		"limit":          req.Limit,
		"max_concurrent": req.MaxConcurrent,
	})
}

// GetPopulationStatsHandler returns data population statistics
// GET /api/v1/admin/jobs/stats
func (h *JobsHandler) GetPopulationStatsHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Get stats
	stats, err := h.monitor.GetPopulationStats(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get population stats",
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// AutoPopulateHandler triggers automatic population workflow
// POST /api/v1/admin/jobs/auto-populate
func (h *JobsHandler) AutoPopulateHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
	defer cancel()

	// Parse request body
	var req struct {
		Threshold     int  `json:"threshold"`      // Default: 100
		Count         int  `json:"count"`          // Number of pairs to populate (default: 10)
		MaxConcurrent int  `json:"max_concurrent"` // Max concurrent fetches (default: 3)
		Limit         int  `json:"limit"`          // Leads per fetch (default: 1000)
		IncludeMissing bool `json:"include_missing"` // Include missing combinations (default: false)
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Defaults
	if req.Threshold == 0 {
		req.Threshold = 100
	}
	if req.Count == 0 {
		req.Count = 10
	}
	if req.MaxConcurrent == 0 {
		req.MaxConcurrent = 3
	}
	if req.Limit == 0 {
		req.Limit = 1000
	}

	// Detect low data industries
	pairs, err := h.monitor.DetectLowDataIndustries(ctx, req.Threshold)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to detect low data industries",
		})
	}

	// Include missing combinations if requested
	if req.IncludeMissing {
		missing, err := h.monitor.DetectMissingCombinations(ctx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to detect missing combinations",
			})
		}
		pairs = append(pairs, missing...)
	}

	// Limit to requested count
	if len(pairs) > req.Count {
		pairs = pairs[:req.Count]
	}

	if len(pairs) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "No industries need population",
			"count":   0,
		})
	}

	// Trigger batch fetch
	if err := h.monitor.TriggerDataFetchBatch(ctx, pairs, req.Limit, req.MaxConcurrent); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to trigger batch fetch",
		})
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":        "Auto-population triggered",
		"count":          len(pairs),
		"threshold":      req.Threshold,
		"limit":          req.Limit,
		"max_concurrent": req.MaxConcurrent,
		"pairs":          pairs,
	})
}
