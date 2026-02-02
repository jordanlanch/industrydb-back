package models

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required,min=2"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token string     `json:"token"`
	User  *UserInfo  `json:"user"`
}

// UserInfo represents user information in responses
type UserInfo struct {
	ID                  int    `json:"id"`
	Email               string `json:"email"`
	Name                string `json:"name"`
	SubscriptionTier    string `json:"subscription_tier"`
	UsageCount          int    `json:"usage_count"`
	UsageLimit          int    `json:"usage_limit"`
	EmailVerified       bool   `json:"email_verified"`
	OnboardingCompleted bool   `json:"onboarding_completed"`
	OnboardingStep      int    `json:"onboarding_step"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
