package models

// CheckoutRequest represents a request to create a checkout session
type CheckoutRequest struct {
	Tier           string `json:"tier" validate:"required,oneof=starter pro business"`
	OrganizationID *int   `json:"organization_id,omitempty"` // Optional: If set, subscription applies to organization
}

// CheckoutResponse represents a checkout session response
type CheckoutResponse struct {
	SessionID  string `json:"session_id"`
	URL        string `json:"url"`
	ExpiresAt  int64  `json:"expires_at"`
}

// CustomerPortalResponse represents a customer portal session response
type CustomerPortalResponse struct {
	URL string `json:"url"`
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// SubscriptionInfo represents subscription information
type SubscriptionInfo struct {
	ID                 int    `json:"id"`
	Tier               string `json:"tier"`
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   string `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
	StripeCustomerID   string `json:"stripe_customer_id,omitempty"`
}

// PricingTier represents a pricing tier with details
type PricingTier struct {
	Name        string `json:"name"`
	Price       int    `json:"price"`
	LeadsLimit  int    `json:"leads_limit"`
	Description string `json:"description"`
	Features    []string `json:"features"`
}

// PricingResponse represents pricing information
type PricingResponse struct {
	Tiers []PricingTier `json:"tiers"`
}
