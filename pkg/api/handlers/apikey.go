package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/apikey"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// APIKeyHandler handles API key endpoints
type APIKeyHandler struct {
	apiKeyService *apikey.Service
	validator     *validator.Validate
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(apiKeyService *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
		validator:     validator.New(),
	}
}

// Create handles creating a new API key
func (h *APIKeyHandler) Create(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse request
	var req apikey.CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Create API key
	response, err := h.apiKeyService.CreateAPIKey(ctx, userID, req)
	if err != nil {
		if err.Error() == "API keys require Business tier subscription" {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "upgrade_required",
				Message: "API keys are only available on Business tier",
			})
		}
		return errors.InternalError(c, err)
	}

	// Return response with warning about key visibility
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"api_key": response,
		"warning": "This is the only time the API key will be displayed. Store it securely.",
	})
}

// List handles listing all API keys for the current user
func (h *APIKeyHandler) List(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// List API keys
	keys, err := h.apiKeyService.ListAPIKeys(ctx, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"api_keys": keys,
		"total":    len(keys),
	})
}

// Get handles retrieving a single API key
func (h *APIKeyHandler) Get(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get API key
	key, err := h.apiKeyService.GetAPIKey(ctx, userID, keyID)
	if err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, key)
}

// Revoke handles revoking an API key
func (h *APIKeyHandler) Revoke(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Revoke API key
	if err := h.apiKeyService.RevokeAPIKey(ctx, userID, keyID); err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}

// Delete handles deleting an API key
func (h *APIKeyHandler) Delete(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Delete API key
	if err := h.apiKeyService.DeleteAPIKey(ctx, userID, keyID); err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key deleted successfully",
	})
}

// UpdateName handles updating an API key's name
func (h *APIKeyHandler) UpdateName(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Parse request
	var req struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
	}
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Update API key name
	if err := h.apiKeyService.UpdateAPIKeyName(ctx, userID, keyID, req.Name); err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key name updated successfully",
	})
}

// GetStats handles retrieving API key statistics
func (h *APIKeyHandler) GetStats(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get stats
	stats, err := h.apiKeyService.GetAPIKeyStats(ctx, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, stats)
}
