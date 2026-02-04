package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/customfields"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// CustomFieldsHandler handles custom fields operations for leads.
type CustomFieldsHandler struct {
	service *customfields.Service
}

// NewCustomFieldsHandler creates a new custom fields handler.
func NewCustomFieldsHandler(client *ent.Client) *CustomFieldsHandler {
	return &CustomFieldsHandler{
		service: customfields.NewService(client),
	}
}

// GetCustomFields godoc
// @Summary Get all custom fields for a lead
// @Description Retrieve all user-defined custom fields for a specific lead
// @Tags Custom Fields
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} customfields.CustomFieldsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/custom-fields [get]
func (h *CustomFieldsHandler) GetCustomFields(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Get custom fields
	result, err := h.service.GetCustomFields(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch custom fields",
		})
	}

	return c.JSON(http.StatusOK, result)
}

// SetCustomField godoc
// @Summary Set a single custom field
// @Description Set or update a single custom field for a lead
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param id path int true "Lead ID"
// @Param request body customfields.SetCustomFieldRequest true "Custom field data"
// @Success 200 {object} customfields.CustomFieldsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/custom-fields/set [post]
func (h *CustomFieldsHandler) SetCustomField(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Parse request body
	var req customfields.SetCustomFieldRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate
	if req.Key == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Key is required",
		})
	}

	// Set custom field
	result, err := h.service.SetCustomField(ctx, leadID, req.Key, req.Value)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		if err.Error() == "key cannot be empty" || err.Error() == "key too long (max 50 characters)" {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to set custom field",
		})
	}

	return c.JSON(http.StatusOK, result)
}

// RemoveCustomField godoc
// @Summary Remove a custom field
// @Description Remove a specific custom field from a lead
// @Tags Custom Fields
// @Produce json
// @Param id path int true "Lead ID"
// @Param key path string true "Field key to remove"
// @Success 200 {object} customfields.CustomFieldsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/custom-fields/{key} [delete]
func (h *CustomFieldsHandler) RemoveCustomField(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Get key from path
	key := c.Param("key")
	if key == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Key is required",
		})
	}

	// Remove custom field
	result, err := h.service.RemoveCustomField(ctx, leadID, key)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to remove custom field",
		})
	}

	return c.JSON(http.StatusOK, result)
}

// UpdateCustomFields godoc
// @Summary Update all custom fields (bulk)
// @Description Replace all custom fields for a lead with new values
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param id path int true "Lead ID"
// @Param request body customfields.UpdateCustomFieldsRequest true "Custom fields data"
// @Success 200 {object} customfields.CustomFieldsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/custom-fields [put]
func (h *CustomFieldsHandler) UpdateCustomFields(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Parse request body
	var req customfields.UpdateCustomFieldsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Update custom fields
	result, err := h.service.UpdateCustomFields(ctx, leadID, req.CustomFields)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		if err.Error() == "custom field key cannot be empty" ||
		   err.Error() == "key too long (max 50 characters)" ||
		   (len(err.Error()) > 0 && err.Error()[0:3] == "cus") { // Starts with "custom field key"
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update custom fields",
		})
	}

	return c.JSON(http.StatusOK, result)
}

// ClearCustomFields godoc
// @Summary Clear all custom fields
// @Description Remove all custom fields from a lead
// @Tags Custom Fields
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} customfields.CustomFieldsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/custom-fields [delete]
func (h *CustomFieldsHandler) ClearCustomFields(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Clear custom fields
	result, err := h.service.ClearCustomFields(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to clear custom fields",
		})
	}

	return c.JSON(http.StatusOK, result)
}
