package domain

import "fmt"

// DomainError represents a domain-specific error with a code and message
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// Error codes
const (
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeUsageLimitExceeded = "USAGE_LIMIT_EXCEEDED"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInternal          = "INTERNAL_ERROR"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeBadRequest        = "BAD_REQUEST"
)

// Error constructors

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string) error {
	return &DomainError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// NewValidationError creates a new validation error
func NewValidationError(msg string) error {
	return &DomainError{
		Code:    ErrCodeValidation,
		Message: msg,
	}
}

// NewUsageLimitError creates a new usage limit exceeded error
func NewUsageLimitError(limit int) error {
	return &DomainError{
		Code:    ErrCodeUsageLimitExceeded,
		Message: fmt.Sprintf("Usage limit of %d exceeded. Please upgrade your plan.", limit),
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError() error {
	return &DomainError{
		Code:    ErrCodeUnauthorized,
		Message: "Authentication required",
	}
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(msg string) error {
	return &DomainError{
		Code:    ErrCodeForbidden,
		Message: msg,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(err error) error {
	return &DomainError{
		Code:    ErrCodeInternal,
		Message: "An internal error occurred",
		Err:     err,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(msg string) error {
	return &DomainError{
		Code:    ErrCodeConflict,
		Message: msg,
	}
}

// NewBadRequestError creates a new bad request error
func NewBadRequestError(msg string) error {
	return &DomainError{
		Code:    ErrCodeBadRequest,
		Message: msg,
	}
}

// Helper functions to check error types

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeNotFound
	}
	return false
}

// IsValidation checks if the error is a validation error
func IsValidation(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeValidation
	}
	return false
}

// IsUsageLimitExceeded checks if the error is a usage limit exceeded error
func IsUsageLimitExceeded(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeUsageLimitExceeded
	}
	return false
}

// IsUnauthorized checks if the error is an unauthorized error
func IsUnauthorized(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeUnauthorized
	}
	return false
}

// IsForbidden checks if the error is a forbidden error
func IsForbidden(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeForbidden
	}
	return false
}

// IsInternal checks if the error is an internal error
func IsInternal(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeInternal
	}
	return false
}

// IsConflict checks if the error is a conflict error
func IsConflict(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeConflict
	}
	return false
}

// IsBadRequest checks if the error is a bad request error
func IsBadRequest(err error) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == ErrCodeBadRequest
	}
	return false
}

// GetErrorCode extracts the error code from a domain error
func GetErrorCode(err error) string {
	if de, ok := err.(*DomainError); ok {
		return de.Code
	}
	return ErrCodeInternal
}
