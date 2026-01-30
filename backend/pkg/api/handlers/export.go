package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// ExportHandler handles export endpoints
type ExportHandler struct {
	exportService    *export.Service
	analyticsService *analytics.Service
	validator        *validator.Validate
}

// NewExportHandler creates a new export handler
func NewExportHandler(exportService *export.Service, analyticsService *analytics.Service) *ExportHandler {
	return &ExportHandler{
		exportService:    exportService,
		analyticsService: analyticsService,
		validator:        validator.New(),
	}
}

// Create handles creating a new export
func (h *ExportHandler) Create(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse request
	var req models.ExportRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create export
	exportResp, err := h.exportService.CreateExport(c.Request().Context(), userID, req)
	if err != nil {
		return errors.InternalError(c, err)
	}

	// Analytics will be logged after export completes (in export service)
	return c.JSON(http.StatusCreated, exportResp)
}

// Get handles retrieving a single export
func (h *ExportHandler) Get(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse export ID
	exportIDStr := c.Param("id")
	exportID, err := strconv.Atoi(exportIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Export ID must be a number",
		})
	}

	// Get export
	exportResp, err := h.exportService.GetExport(c.Request().Context(), userID, exportID)
	if err != nil {
		if err.Error() == "export not found" {
			return errors.NotFoundError(c, "export")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, exportResp)
}

// List handles listing all exports for the current user
func (h *ExportHandler) List(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// List exports
	exports, err := h.exportService.ListExports(c.Request().Context(), userID, page, limit)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, exports)
}

// Download handles downloading an export file
func (h *ExportHandler) Download(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse export ID
	exportIDStr := c.Param("id")
	exportID, err := strconv.Atoi(exportIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Export ID must be a number",
		})
	}

	// Get file path
	filePath, err := h.exportService.GetFilePath(c.Request().Context(), userID, exportID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	// Get filename
	filename := filepath.Base(filePath)

	// Set headers for download
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.Response().Header().Set("Content-Type", "application/octet-stream")

	// Send file
	return c.File(filePath)
}
