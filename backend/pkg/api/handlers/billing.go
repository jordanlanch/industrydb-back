package handlers

import (
	"io"
	"net/http"
	"net/url"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/billing"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// BillingHandler handles billing endpoints
type BillingHandler struct {
	billingService *billing.Service
	validator      *validator.Validate
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService *billing.Service) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
		validator:      validator.New(),
	}
}

// validateReturnURL validates and sanitizes return URL to prevent open redirect attacks
// Returns a safe URL from whitelist or default URL if validation fails
func validateReturnURL(returnURL string) string {
	const defaultURL = "https://industrydb.io/dashboard/settings/billing"

	// If empty, return default
	if returnURL == "" {
		return defaultURL
	}

	// Parse URL
	parsed, err := url.Parse(returnURL)
	if err != nil {
		return defaultURL
	}

	// Only allow http and https schemes (prevents javascript:, data:, ftp:, etc.)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return defaultURL
	}

	// Reject URLs with userinfo (prevents phishing: https://attacker@legitimate.com)
	if parsed.User != nil && parsed.User.String() != "" {
		return defaultURL
	}

	// Whitelist of allowed hosts (DDD: bounded context - billing domain)
	allowedHosts := []string{
		"localhost:5678",      // Development (root docker-compose)
		"localhost:5566",      // Development (modular setup)
		"industrydb.io",       // Production
		"www.industrydb.io",   // Production WWW
	}

	// Validate host against whitelist
	for _, allowedHost := range allowedHosts {
		if parsed.Host == allowedHost {
			return returnURL
		}
	}

	// Host not in whitelist, return default
	return defaultURL
}

// CreateCheckout handles creating a checkout session
func (h *BillingHandler) CreateCheckout(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse request
	var req models.CheckoutRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create checkout session
	session, err := h.billingService.CreateCheckoutSession(c.Request().Context(), userID, req.Tier)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, session)
}

// CreatePortalSession handles creating a customer portal session
func (h *BillingHandler) CreatePortalSession(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Get and validate return URL (prevents open redirect attacks)
	returnURL := validateReturnURL(c.QueryParam("return_url"))

	// Create portal session
	portal, err := h.billingService.CreateCustomerPortalSession(c.Request().Context(), userID, returnURL)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, portal)
}

// HandleWebhook handles Stripe webhook events
func (h *BillingHandler) HandleWebhook(c echo.Context) error {
	// Get raw body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_body",
			Message: "Failed to read request body",
		})
	}

	// Get Stripe signature
	signature := c.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "missing_signature",
		})
	}

	// Handle webhook
	err = h.billingService.HandleWebhook(c.Request().Context(), body, signature)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Webhook processed successfully",
	})
}

// GetPricing handles returning pricing information
func (h *BillingHandler) GetPricing(c echo.Context) error {
	pricing := h.billingService.GetPricing()
	return c.JSON(http.StatusOK, pricing)
}
