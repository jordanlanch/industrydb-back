package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// CohortHandler handles cohort analytics operations
type CohortHandler struct {
	service *analytics.Service
}

// NewCohortHandler creates a new cohort analytics handler
func NewCohortHandler(db *ent.Client) *CohortHandler {
	return &CohortHandler{
		service: analytics.NewService(db),
	}
}

// GetCohorts godoc
// @Summary Get user cohorts
// @Description Get list of user cohorts grouped by time period
// @Tags Analytics
// @Produce json
// @Param period query string false "Time period (day, week, month)" default(week)
// @Param count query int false "Number of periods to retrieve" default(12)
// @Success 200 {array} analytics.Cohort
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/cohorts [get]
func (h *CohortHandler) GetCohorts(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse period parameter
	period := c.QueryParam("period")
	if period == "" {
		period = "week" // default
	}

	if period != "day" && period != "week" && period != "month" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_period",
			Message: "period must be 'day', 'week', or 'month'",
		})
	}

	// Parse count parameter
	countStr := c.QueryParam("count")
	count := 12 // default
	if countStr != "" {
		parsedCount, err := strconv.Atoi(countStr)
		if err != nil || parsedCount < 1 || parsedCount > 52 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_count",
				Message: "count must be between 1 and 52",
			})
		}
		count = parsedCount
	}

	cohorts, err := h.service.GetCohorts(ctx, period, count)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, cohorts)
}

// GetCohortRetention godoc
// @Summary Get cohort retention
// @Description Get retention data for a specific cohort over time
// @Tags Analytics
// @Produce json
// @Param cohort_start query string true "Cohort start date (RFC3339 format)"
// @Param period query string false "Time period (day, week, month)" default(week)
// @Param periods query int false "Number of periods to track" default(12)
// @Success 200 {object} analytics.CohortRetention
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/cohorts/retention [get]
func (h *CohortHandler) GetCohortRetention(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse cohort_start parameter
	cohortStartStr := c.QueryParam("cohort_start")
	if cohortStartStr == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_cohort_start",
			Message: "cohort_start is required (RFC3339 format)",
		})
	}

	cohortStart, err := time.Parse(time.RFC3339, cohortStartStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_cohort_start",
			Message: "cohort_start must be in RFC3339 format",
		})
	}

	// Parse period parameter
	period := c.QueryParam("period")
	if period == "" {
		period = "week" // default
	}

	if period != "day" && period != "week" && period != "month" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_period",
			Message: "period must be 'day', 'week', or 'month'",
		})
	}

	// Parse periods parameter
	periodsStr := c.QueryParam("periods")
	periods := 12 // default
	if periodsStr != "" {
		parsedPeriods, err := strconv.Atoi(periodsStr)
		if err != nil || parsedPeriods < 1 || parsedPeriods > 52 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_periods",
				Message: "periods must be between 1 and 52",
			})
		}
		periods = parsedPeriods
	}

	retention, err := h.service.GetCohortRetention(ctx, cohortStart, period, periods)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, retention)
}

// GetCohortComparison godoc
// @Summary Compare multiple cohorts
// @Description Compare retention across multiple cohorts
// @Tags Analytics
// @Produce json
// @Param period query string false "Time period (day, week, month)" default(week)
// @Param cohort_count query int false "Number of cohorts to compare" default(6)
// @Param retention_periods query int false "Number of retention periods" default(12)
// @Success 200 {object} analytics.CohortComparison
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/cohorts/comparison [get]
func (h *CohortHandler) GetCohortComparison(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 15*time.Second)
	defer cancel()

	// Parse period parameter
	period := c.QueryParam("period")
	if period == "" {
		period = "week" // default
	}

	if period != "day" && period != "week" && period != "month" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_period",
			Message: "period must be 'day', 'week', or 'month'",
		})
	}

	// Parse cohort_count parameter
	cohortCountStr := c.QueryParam("cohort_count")
	cohortCount := 6 // default
	if cohortCountStr != "" {
		parsedCount, err := strconv.Atoi(cohortCountStr)
		if err != nil || parsedCount < 1 || parsedCount > 12 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_cohort_count",
				Message: "cohort_count must be between 1 and 12",
			})
		}
		cohortCount = parsedCount
	}

	// Parse retention_periods parameter
	retentionPeriodsStr := c.QueryParam("retention_periods")
	retentionPeriods := 12 // default
	if retentionPeriodsStr != "" {
		parsedPeriods, err := strconv.Atoi(retentionPeriodsStr)
		if err != nil || parsedPeriods < 1 || parsedPeriods > 52 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_retention_periods",
				Message: "retention_periods must be between 1 and 52",
			})
		}
		retentionPeriods = parsedPeriods
	}

	comparison, err := h.service.GetCohortComparison(ctx, period, cohortCount, retentionPeriods)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, comparison)
}

// GetCohortActivityMetrics godoc
// @Summary Get cohort activity metrics
// @Description Get activity metrics for a specific cohort
// @Tags Analytics
// @Produce json
// @Param cohort_start query string true "Cohort start date (RFC3339 format)"
// @Param weeks query int false "Number of weeks to track" default(4)
// @Success 200 {object} analytics.CohortActivityMetrics
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/cohorts/activity [get]
func (h *CohortHandler) GetCohortActivityMetrics(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse cohort_start parameter
	cohortStartStr := c.QueryParam("cohort_start")
	if cohortStartStr == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_cohort_start",
			Message: "cohort_start is required (RFC3339 format)",
		})
	}

	cohortStart, err := time.Parse(time.RFC3339, cohortStartStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_cohort_start",
			Message: "cohort_start must be in RFC3339 format",
		})
	}

	// Parse weeks parameter
	weeksStr := c.QueryParam("weeks")
	weeks := 4 // default
	if weeksStr != "" {
		parsedWeeks, err := strconv.Atoi(weeksStr)
		if err != nil || parsedWeeks < 1 || parsedWeeks > 52 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_weeks",
				Message: "weeks must be between 1 and 52",
			})
		}
		weeks = parsedWeeks
	}

	metrics, err := h.service.GetCohortActivityMetrics(ctx, cohortStart, weeks)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, metrics)
}
