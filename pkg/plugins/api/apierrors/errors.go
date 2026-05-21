package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// APIError represents a structured API error with consistent JSON output.
// All API handlers should use this type for error responses.
type APIError struct {
	Code    string         `json:"error"`
	Message string         `json:"message"`
	Status  int            `json:"statusCode"`
	Details map[string]any `json:"details,omitempty"` // optional structured context
	Err     error          `json:"-"`                 // underlying error, not exposed in JSON
}

// Error implements the error interface
func (ae *APIError) Error() string {
	if ae.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", ae.Code, ae.Message, ae.Err)
	}
	return fmt.Sprintf("[%s] %s", ae.Code, ae.Message)
}

// Unwrap returns the underlying error, enabling errors.Is() and errors.As()
// to traverse the error chain through APIError.
func (ae *APIError) Unwrap() error { return ae.Err }

// ToJSON returns the JSON representation of the error.
func (ae *APIError) ToJSON() []byte {
	data, _ := json.Marshal(ae)
	return data
}

// WriteHTTP writes the error as an HTTP JSON response.
func (ae *APIError) WriteHTTP(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ae.Status)
	_, _ = w.Write(ae.ToJSON())
}

// IsAPIError checks if an error is an APIError.
func IsAPIError(err error) (*APIError, bool) {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// MapErrorToAPIError converts any error to an APIError.
// If the error is already an APIError it is returned directly.
// If the error is a core.SystemError, it is mapped by code and type.
// Otherwise a generic 500 is returned with no internal details.
func MapErrorToAPIError(err error) *APIError {
	if err == nil {
		return &APIError{Code: "OK", Message: "", Status: 200}
	}

	// Already an APIError — return as-is
	var ae *APIError
	if errors.As(err, &ae) {
		return ae
	}

	// core.SystemError — map by code and type
	var sysErr *core.SystemError
	if errors.As(err, &sysErr) {
		return mapSystemError(sysErr)
	}

	// Unknown error — do NOT expose internal details
	return &APIError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "an internal error occurred",
		Status:  500,
	}
}

// mapSystemError maps a core.SystemError to an APIError using
// error code and type to determine HTTP status.
func mapSystemError(se *core.SystemError) *APIError {
	status := 500
	code := "INTERNAL_SERVER_ERROR"
	message := "an internal error occurred"

	switch se.Code {
	case core.ErrorCodeValidation:
		status = 400
		code = "VALIDATION_FAILED"
		message = se.Message
	case core.ErrorCodeNotFound:
		status = 404
		code = "NOT_FOUND"
		message = se.Message
	case core.ErrorCodeDuplicate:
		status = 409
		code = "CONFLICT"
		message = se.Message
	case core.ErrorCodeDatabaseError:
		code = "SERVICE_UNAVAILABLE"
		message = "service temporarily unavailable"
		if se.IsTransient() {
			status = 503
		}
	case core.ErrorCodeCacheError:
		status = 503
		code = "SERVICE_UNAVAILABLE"
		message = "cache temporarily unavailable"
	case core.ErrorCodeMQError:
		status = 503
		code = "SERVICE_UNAVAILABLE"
		message = "message queue temporarily unavailable"
	case core.ErrorCodeNetworkError:
		status = 502
		code = "BAD_GATEWAY"
		message = "upstream service unavailable"
	case core.ErrorCodeTimeout:
		status = 504
		code = "GATEWAY_TIMEOUT"
		message = "request timed out"
	case core.ErrorCodeConfigError:
		status = 500
		code = "INTERNAL_SERVER_ERROR"
		message = "configuration error"

	// Web3-specific error codes
	case core.ErrorCodeBlockNotFound:
		status = 404
		code = "BLOCK_NOT_FOUND"
		message = se.Message
	case core.ErrorCodeEventNotFound:
		status = 404
		code = "EVENT_NOT_FOUND"
		message = se.Message
	case core.ErrorCodeChainNotFound:
		status = 404
		code = "CHAIN_NOT_FOUND"
		message = se.Message
	case core.ErrorCodeChainNotSupported:
		status = 400
		code = "CHAIN_NOT_SUPPORTED"
		message = se.Message
	case core.ErrorCodeTxNotFound:
		status = 404
		code = "TRANSACTION_NOT_FOUND"
		message = se.Message
	case core.ErrorCodeContractNotFound:
		status = 404
		code = "CONTRACT_NOT_FOUND"
		message = se.Message
	case core.ErrorCodeRPCError:
		status = 502
		code = "RPC_ERROR"
		message = se.Message
	case core.ErrorCodeRPCRateLimited:
		status = 429
		code = "RPC_RATE_LIMITED"
		message = se.Message
	case core.ErrorCodeEventDecodeFailed:
		status = 422
		code = "EVENT_DECODE_FAILED"
		message = se.Message
	case core.ErrorCodeABINotFound:
		status = 404
		code = "ABI_NOT_FOUND"
		message = se.Message
	case core.ErrorCodeFinalityNotReady:
		status = 409
		code = "FINALITY_NOT_READY"
		message = se.Message
	case core.ErrorCodeReorgDetected:
		status = 409
		code = "REORG_DETECTED"
		message = se.Message
	case core.ErrorCodeInvalidEventData:
		status = 422
		code = "INVALID_EVENT_DATA"
		message = se.Message

	default:
		// Fallback by type
		if se.IsTransient() {
			status = 503
			code = "SERVICE_UNAVAILABLE"
			message = "service temporarily unavailable"
		}
	}

	return &APIError{Code: code, Message: message, Status: status}
}

// Error constructors

// ErrInvalidRequest creates an invalid request error
func ErrInvalidRequest(message string) error {
	return &APIError{Code: "INVALID_REQUEST", Message: message, Status: 400}
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) error {
	return &APIError{Code: "UNAUTHORIZED", Message: message, Status: 401}
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) error {
	return &APIError{Code: "FORBIDDEN", Message: message, Status: 403}
}

// ErrNotFound creates a not found error
func ErrNotFound(resource string) error {
	return &APIError{Code: "NOT_FOUND", Message: fmt.Sprintf("%s not found", resource), Status: 404}
}

// ErrInternalServer creates an internal server error
func ErrInternalServer(message string) error {
	return &APIError{Code: "INTERNAL_SERVER_ERROR", Message: message, Status: 500}
}

// ErrServiceUnavailable creates a service unavailable error
func ErrServiceUnavailable(message string) error {
	return &APIError{Code: "SERVICE_UNAVAILABLE", Message: message, Status: 503}
}

// ErrRateLimited creates a rate limit error
func ErrRateLimited() error {
	return &APIError{Code: "RATE_LIMIT_EXCEEDED", Message: "too many requests", Status: 429}
}

// ErrInvalidParameter creates an invalid parameter error
func ErrInvalidParameter(param string, reason string) error {
	return &APIError{Code: "INVALID_PARAMETER", Message: fmt.Sprintf("invalid parameter '%s': %s", param, reason), Status: 400}
}

// ErrMissingParameter creates a missing parameter error
func ErrMissingParameter(param string) error {
	return &APIError{Code: "MISSING_PARAMETER", Message: fmt.Sprintf("required parameter '%s' is missing", param), Status: 400}
}

// ErrValidationFailed creates a validation failed error
func ErrValidationFailed(field, reason string) error {
	return &APIError{Code: "VALIDATION_FAILED", Message: fmt.Sprintf("validation failed for field %s: %s", field, reason), Status: 400}
}
