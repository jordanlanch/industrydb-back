package handlers

import (
	"net/http"
	"strconv"

	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/labstack/echo/v4"
)

// AuditHandler handles audit log endpoints
type AuditHandler struct {
	auditService *audit.Service
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService *audit.Service) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// GetUserLogs returns audit logs for the current user
func (h *AuditHandler) GetUserLogs(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	// Get limit from query param (default 50, max 100)
	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get logs
	logs, err := h.auditService.GetUserLogs(c.Request().Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed_to_fetch_logs",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetRecentLogs returns recent audit logs (admin only)
func (h *AuditHandler) GetRecentLogs(c echo.Context) error {
	// Get limit from query param (default 100, max 500)
	limitStr := c.QueryParam("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	// Get logs
	logs, err := h.auditService.GetRecentLogs(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed_to_fetch_logs",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetCriticalLogs returns critical severity logs (admin only)
func (h *AuditHandler) GetCriticalLogs(c echo.Context) error {
	// Get limit from query param (default 50, max 200)
	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	// Get logs
	logs, err := h.auditService.GetCriticalLogs(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed_to_fetch_logs",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}
