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
// @Summary Create Stripe checkout session
// @Description Create a new Stripe checkout session to upgrade/downgrade subscription tier
// @Tags Billing
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CheckoutRequest true "Checkout configuration with subscription tier"
// @Success 200 {object} map[string]string "Checkout session created with URL"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /billing/checkout [post]
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

	// Create checkout session (user or organization)
	session, err := h.billingService.CreateCheckoutSession(c.Request().Context(), userID, req.Tier, req.OrganizationID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, session)
}

// CreatePortalSession handles creating a customer portal session
// @Summary Create Stripe customer portal session
// @Description Create a session to access Stripe customer portal for managing subscriptions, payment methods, and billing history
// @Tags Billing
// @Produce json
// @Security BearerAuth
// @Param return_url query string false "URL to return to after portal session (validated against whitelist)"
// @Success 200 {object} map[string]string "Portal session created with URL"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /billing/portal [post]
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
// @Summary Handle Stripe webhook
// @Description Process Stripe webhook events for subscription updates, payment confirmations, and cancellations
// @Tags Billing
// @Accept json
// @Produce json
// @Param Stripe-Signature header string true "Stripe webhook signature for verification"
// @Param payload body object true "Stripe webhook event payload"
// @Success 200 {object} models.SuccessResponse "Webhook processed successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid request or missing signature"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /webhook/stripe [post]
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
// @Summary Get pricing tiers
// @Description Get all available subscription tiers with pricing, features, and limits
// @Tags Billing
// @Produce json
// @Success 200 {object} map[string]interface{} "Pricing information for all tiers"
// @Router /billing/pricing [get]
func (h *BillingHandler) GetPricing(c echo.Context) error {
	pricing := h.billingService.GetPricing()
	return c.JSON(http.StatusOK, pricing)
}
