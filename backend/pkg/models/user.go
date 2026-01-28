package models

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=2"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
}

// UserResponse represents a user in responses
type UserResponse struct {
	ID               int    `json:"id"`
	Email            string `json:"email"`
	Name             string `json:"name"`
	SubscriptionTier string `json:"subscription_tier"`
	Role             string `json:"role,omitempty"`
	UsageCount       int    `json:"usage_count"`
	UsageLimit       int    `json:"usage_limit"`
	EmailVerified    bool   `json:"email_verified"`
	CreatedAt        string `json:"created_at"`
}

// UsageResponse represents usage statistics
type UsageResponse struct {
	UsageCount int    `json:"usage_count"`
	UsageLimit int    `json:"usage_limit"`
	Remaining  int    `json:"remaining"`
	ResetAt    string `json:"reset_at"`
	Tier       string `json:"tier"`
}
