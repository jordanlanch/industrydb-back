package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/auditlog"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/leadassignment"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// LeadAssignmentHandler handles lead assignment operations.
type LeadAssignmentHandler struct {
	service     *leadassignment.Service
	auditLogger *audit.Service
}

// NewLeadAssignmentHandler creates a new lead assignment handler.
func NewLeadAssignmentHandler(db *ent.Client, auditLogger *audit.Service) *LeadAssignmentHandler {
	return &LeadAssignmentHandler{
		service:     leadassignment.NewService(db),
		auditLogger: auditLogger,
	}
}

// AssignLead godoc
// @Summary Manually assign lead to user
// @Description Assign a lead to a specific user with a reason
// @Tags Lead Assignment
// @Accept json
// @Produce json
// @Param id path int true "Lead ID"
// @Param request body leadassignment.AssignLeadRequest true "Assignment details"
// @Success 200 {object} leadassignment.AssignmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/assign [post]
func (h *LeadAssignmentHandler) AssignLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Get("user_id").(int)

	// Parse request body
	var req leadassignment.AssignLeadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Override lead ID from path
	req.LeadID = leadID

	// Assign lead
	result, err := h.service.AssignLead(ctx, req, userID)
	if err != nil {
		if err.Error() == "lead not found" || err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	// Audit log (non-blocking)
	resourceType := "lead_assignment"
	resourceID := strconv.Itoa(result.ID)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	description := "Manually assigned lead to user"
	go h.auditLogger.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       auditlog.ActionDataExport, // Reuse existing action
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Description:  &description,
		Severity:     auditlog.SeverityInfo,
	})

	return c.JSON(http.StatusOK, result)
}

// AutoAssignLead godoc
// @Summary Auto-assign lead using round-robin
// @Description Automatically assign lead to user with fewest active leads
// @Tags Lead Assignment
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} leadassignment.AssignmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/auto-assign [post]
func (h *LeadAssignmentHandler) AutoAssignLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	// Get user ID from context (for audit logging)
	userID := c.Get("user_id").(int)

	// Auto-assign lead
	result, err := h.service.AutoAssignLead(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		if err.Error() == "no available users for assignment" {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "no_users",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	// Audit log (non-blocking)
	resourceType := "lead_assignment"
	resourceID := strconv.Itoa(result.ID)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	description := "Auto-assigned lead using round-robin"
	go h.auditLogger.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       auditlog.ActionDataExport, // Reuse existing action
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Description:  &description,
		Severity:     auditlog.SeverityInfo,
	})

	return c.JSON(http.StatusOK, result)
}

// GetUserLeads godoc
// @Summary Get user's assigned leads
// @Description Get all active leads assigned to the current user
// @Tags Lead Assignment
// @Produce json
// @Param limit query int false "Limit (default 50, max 100)" default(50)
// @Success 200 {array} leadassignment.AssignmentResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user/assigned-leads [get]
func (h *LeadAssignmentHandler) GetUserLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get user ID from context
	userID := c.Get("user_id").(int)

	// Parse limit
	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get user leads
	results, err := h.service.GetUserLeads(ctx, userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, results)
}

// GetLeadAssignmentHistory godoc
// @Summary Get lead assignment history
// @Description Get complete assignment history for a lead
// @Tags Lead Assignment
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {array} leadassignment.AssignmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/assignment-history [get]
func (h *LeadAssignmentHandler) GetLeadAssignmentHistory(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	// Get assignment history
	results, err := h.service.GetLeadAssignmentHistory(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, results)
}

// GetCurrentAssignment godoc
// @Summary Get current lead assignment
// @Description Get the current active assignment for a lead
// @Tags Lead Assignment
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} leadassignment.AssignmentResponse
// @Success 204 "No assignment"
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/current-assignment [get]
func (h *LeadAssignmentHandler) GetCurrentAssignment(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	// Get current assignment
	result, err := h.service.GetCurrentAssignment(ctx, leadID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	// No assignment
	if result == nil {
		return c.NoContent(http.StatusNoContent)
	}

	return c.JSON(http.StatusOK, result)
}
