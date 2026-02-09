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

// DetectLowDataHandler godoc
// @Summary Detect industries with low data
// @Description Detects industry-country combinations with fewer leads than the specified threshold. Requires admin role.
// @Tags Admin Jobs
// @Produce json
// @Security BearerAuth
// @Param threshold query integer false "Minimum lead count threshold" default(100)
// @Success 200 {object} map[string]interface{} "Low data industry-country pairs with threshold and count"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin role required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/jobs/detect-low-data [post]
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

// DetectMissingHandler godoc
// @Summary Detect missing industry-country combinations
// @Description Detects industry-country combinations that have no data at all. Requires admin role.
// @Tags Admin Jobs
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Missing industry-country pairs with count"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin role required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/jobs/detect-missing [post]
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

// TriggerFetchHandler godoc
// @Summary Trigger manual data fetch
// @Description Triggers a manual data acquisition fetch for a specific industry and country from OpenStreetMap. Requires admin role.
// @Tags Admin Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Fetch configuration" SchemaExample({"industry": "tattoo", "country": "US", "limit": 1000})
// @Success 202 {object} map[string]interface{} "Data fetch triggered"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin role required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/jobs/trigger-fetch [post]
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

// TriggerBatchFetchHandler godoc
// @Summary Trigger batch data fetch
// @Description Triggers data acquisition for multiple industry-country pairs concurrently. Requires admin role.
// @Tags Admin Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Batch fetch configuration with pairs, limit, and concurrency"
// @Success 202 {object} map[string]interface{} "Batch fetch triggered with count and configuration"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin role required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/jobs/trigger-batch-fetch [post]
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

// GetPopulationStatsHandler godoc
// @Summary Get data population statistics
// @Description Returns statistics about data population across industries and countries. Requires admin role.
// @Tags Admin Jobs
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Population statistics"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin role required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/jobs/stats [get]
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

// AutoPopulateHandler godoc
// @Summary Auto-populate low data industries
// @Description Detects industries with low data and automatically triggers data fetches to populate them. Optionally includes missing combinations. Requires admin role.
// @Tags Admin Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Auto-populate configuration" SchemaExample({"threshold": 100, "count": 10, "max_concurrent": 3, "limit": 1000, "include_missing": false})
// @Success 202 {object} map[string]interface{} "Auto-population triggered with pairs and configuration"
// @Success 200 {object} map[string]interface{} "No industries need population"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden - admin role required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/jobs/auto-populate [post]
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
