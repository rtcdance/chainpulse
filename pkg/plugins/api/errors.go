package api

import "fmt"

// APIError represents an API-related error
type APIError struct {
	Code    string
	Message string
	Status  int
}

// Error implements the error interface
func (ae *APIError) Error() string {
	return fmt.Sprintf("[%s] %s", ae.Code, ae.Message)
}

// Error constructors

// ErrInvalidRequest creates an invalid request error
func ErrInvalidRequest(message string) error {
	return &APIError{
		Code:    "INVALID_REQUEST",
		Message: message,
		Status:  400,
	}
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) error {
	return &APIError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  401,
	}
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) error {
	return &APIError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  403,
	}
}

// ErrNotFound creates a not found error
func ErrNotFound(resource string) error {
	return &APIError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found", resource),
		Status:  404,
	}
}

// ErrInternalServer creates an internal server error
func ErrInternalServer(message string) error {
	return &APIError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: message,
		Status:  500,
	}
}

// ErrServiceUnavailable creates a service unavailable error
func ErrServiceUnavailable(message string) error {
	return &APIError{
		Code:    "SERVICE_UNAVAILABLE",
		Message: message,
		Status:  503,
	}
}

// ErrRateLimited creates a rate limit error
func ErrRateLimited() error {
	return &APIError{
		Code:    "RATE_LIMITED",
		Message: "too many requests",
		Status:  429,
	}
}

// ErrValidationFailed creates a validation failed error
func ErrValidationFailed(field, reason string) error {
	return &APIError{
		Code:    "VALIDATION_FAILED",
		Message: fmt.Sprintf("validation failed for field %s: %s", field, reason),
		Status:  400,
	}
}
