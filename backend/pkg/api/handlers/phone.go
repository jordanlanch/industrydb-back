package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/jordanlanch/industrydb/pkg/phone"
	"github.com/labstack/echo/v4"
)

// PhoneHandler handles phone validation endpoints.
type PhoneHandler struct{}

// NewPhoneHandler creates a new phone handler.
func NewPhoneHandler() *PhoneHandler {
	return &PhoneHandler{}
}

// ValidatePhoneRequest represents a phone validation request.
type ValidatePhoneRequest struct {
	Phone       string `json:"phone" validate:"required"`
	CountryCode string `json:"country_code,omitempty"` // Optional, defaults to US
}

// ValidatePhone godoc
// @Summary Validate a phone number
// @Description Validate and normalize a phone number with international format support
// @Tags Phone
// @Accept json
// @Produce json
// @Param request body ValidatePhoneRequest true "Phone validation request"
// @Success 200 {object} phone.ValidationResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/phone/validate [post]
func (h *PhoneHandler) ValidatePhone(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	_ = ctx // Unused but available for future use

	var req ValidatePhoneRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate required fields
	if req.Phone == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Phone number is required",
		})
	}

	// Default country code to US if not provided
	if req.CountryCode == "" {
		req.CountryCode = "US"
	}

	// Validate the phone number
	result, err := phone.ValidatePhone(req.Phone, req.CountryCode)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// NormalizePhoneRequest represents a phone normalization request.
type NormalizePhoneRequest struct {
	Phone       string `json:"phone" validate:"required"`
	CountryCode string `json:"country_code,omitempty"`
}

// NormalizePhoneResponse represents a phone normalization response.
type NormalizePhoneResponse struct {
	Original   string `json:"original"`
	Normalized string `json:"normalized"`
	IsValid    bool   `json:"is_valid"`
}

// NormalizePhone godoc
// @Summary Normalize a phone number to E.164 format
// @Description Convert a phone number to E.164 international format (+15551234567)
// @Tags Phone
// @Accept json
// @Produce json
// @Param request body NormalizePhoneRequest true "Phone normalization request"
// @Success 200 {object} NormalizePhoneResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/phone/normalize [post]
func (h *PhoneHandler) NormalizePhone(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	_ = ctx

	var req NormalizePhoneRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if req.Phone == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Phone number is required",
		})
	}

	if req.CountryCode == "" {
		req.CountryCode = "US"
	}

	// Normalize the phone number
	normalized, err := phone.NormalizePhone(req.Phone, req.CountryCode)
	isValid := err == nil

	// If normalization failed, return the original
	if !isValid {
		normalized = req.Phone
	}

	return c.JSON(http.StatusOK, NormalizePhoneResponse{
		Original:   req.Phone,
		Normalized: normalized,
		IsValid:    isValid,
	})
}

// BatchValidateRequest represents a batch phone validation request.
type BatchValidateRequest struct {
	Phones      []string `json:"phones" validate:"required,min=1,max=100"`
	CountryCode string   `json:"country_code,omitempty"`
}

// BatchValidateResponse represents a batch validation response.
type BatchValidateResponse struct {
	Results []phone.ValidationResult `json:"results"`
	Valid   int                      `json:"valid"`
	Invalid int                      `json:"invalid"`
}

// BatchValidatePhones godoc
// @Summary Validate multiple phone numbers
// @Description Validate up to 100 phone numbers in a single request
// @Tags Phone
// @Accept json
// @Produce json
// @Param request body BatchValidateRequest true "Batch validation request (max 100 phones)"
// @Success 200 {object} BatchValidateResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/phone/batch-validate [post]
func (h *PhoneHandler) BatchValidatePhones(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	_ = ctx

	var req BatchValidateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if len(req.Phones) == 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "At least one phone number is required",
		})
	}

	if len(req.Phones) > 100 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Maximum 100 phone numbers allowed per request",
		})
	}

	if req.CountryCode == "" {
		req.CountryCode = "US"
	}

	results := make([]phone.ValidationResult, 0, len(req.Phones))
	validCount := 0
	invalidCount := 0

	for _, phoneNum := range req.Phones {
		result, err := phone.ValidatePhone(phoneNum, req.CountryCode)
		if err != nil {
			// For invalid phones, create a result with is_valid = false
			results = append(results, phone.ValidationResult{
				IsValid:            false,
				E164Format:         phoneNum,
				InternationalFormat: phoneNum,
				NationalFormat:     phoneNum,
				CountryCode:        "",
				PhoneType:          phone.TypeUnknown,
			})
			invalidCount++
		} else {
			results = append(results, *result)
			if result.IsValid {
				validCount++
			} else {
				invalidCount++
			}
		}
	}

	return c.JSON(http.StatusOK, BatchValidateResponse{
		Results: results,
		Valid:   validCount,
		Invalid: invalidCount,
	})
}
