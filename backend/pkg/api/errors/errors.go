package errors

import (
	"log"
	"net/http"

	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// ValidationError returns a generic validation error without exposing internal details
func ValidationError(c echo.Context, err error) error {
	// Log the actual error for debugging
	log.Printf("[VALIDATION ERROR] Path: %s, Error: %v", c.Request().URL.Path, err)

	return c.JSON(http.StatusBadRequest, models.ErrorResponse{
		Error:   "validation_error",
		Message: "Invalid request data. Please check your input and try again.",
	})
}

// DatabaseError returns a generic database error without exposing internal details
func DatabaseError(c echo.Context, err error) error {
	// Log the actual error for debugging
	log.Printf("[DATABASE ERROR] Path: %s, Error: %v", c.Request().URL.Path, err)

	return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
		Error:   "database_error",
		Message: "A database error occurred. Please try again later.",
	})
}

// InternalError returns a generic internal server error
func InternalError(c echo.Context, err error) error {
	// Log the actual error for debugging
	log.Printf("[INTERNAL ERROR] Path: %s, Error: %v", c.Request().URL.Path, err)

	return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
		Error:   "internal_error",
		Message: "An internal error occurred. Please try again later.",
	})
}

// UnauthorizedError returns a generic unauthorized error
func UnauthorizedError(c echo.Context, reason string) error {
	return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
		Error:   "unauthorized",
		Message: "You are not authorized to access this resource.",
	})
}

// ForbiddenError returns a generic forbidden error
func ForbiddenError(c echo.Context, reason string) error {
	return c.JSON(http.StatusForbidden, models.ErrorResponse{
		Error:   "forbidden",
		Message: "You do not have permission to access this resource.",
	})
}

// NotFoundError returns a generic not found error
func NotFoundError(c echo.Context, resource string) error {
	return c.JSON(http.StatusNotFound, models.ErrorResponse{
		Error:   "not_found",
		Message: "The requested resource was not found.",
	})
}

// ConflictError returns a generic conflict error
func ConflictError(c echo.Context, message string) error {
	return c.JSON(http.StatusConflict, models.ErrorResponse{
		Error:   "conflict",
		Message: message, // Message is safe to expose (e.g., "User already exists")
	})
}
