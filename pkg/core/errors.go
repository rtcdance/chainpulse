package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
)

// Error variables for validation
var (
	ErrInvalidBlockNumber     = errors.New("invalid block number")
	ErrInvalidTransactionHash = errors.New("invalid transaction hash")
	ErrInvalidContractAddress = errors.New("invalid contract address")
	ErrInvalidEventName       = errors.New("invalid event name")
	ErrInvalidAddress         = errors.New("invalid address")
	ErrInvalidBlockHash       = errors.New("invalid block hash")
	ErrInvalidLogIndex        = errors.New("invalid log index")
	ErrInvalidEventData       = errors.New("invalid event data")
	ErrInvalidTimestamp       = errors.New("invalid timestamp")
)

// Typed sentinel errors for error classification via errors.Is().
// Each wraps a SystemError with the appropriate ErrorType so that
// DefaultErrorClassifier can use type assertions instead of string matching.

// Transient errors — retryable, typically network or timeout related.
var (
	ErrTimeout           = NewSystemError(ErrorTypeTransient, ErrorCodeTimeout, "timeout", nil)
	ErrConnectionRefused = NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "connection refused", nil)
	ErrConnectionReset   = NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "connection reset", nil)
	ErrUnavailable       = NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "service unavailable", nil)
	ErrDeadlineExceeded  = NewSystemError(ErrorTypeTransient, ErrorCodeTimeout, "deadline exceeded", nil)
	ErrTemporaryFailure  = NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "temporary failure", nil)
)

// Permanent errors — not retryable, indicate a client or logic error.
var (
	ErrUnauthorized = NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "unauthorized", nil)
	ErrForbidden    = NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "forbidden", nil)
	ErrNotFound     = NewSystemError(ErrorTypePermanent, ErrorCodeNotFound, "not found", nil)
	ErrBadRequest   = NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "bad request", nil)
	ErrInvalidState = NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "invalid state", nil)
	ErrAuthFailed   = NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "authentication failed", nil)
)

// Critical errors — require immediate attention, indicate data corruption or system failure.
var (
	ErrDataCorruption  = NewSystemError(ErrorTypeCritical, ErrorCodeInternalError, "data corruption", nil)
	ErrCriticalFailure = NewSystemError(ErrorTypeCritical, ErrorCodeInternalError, "critical failure", nil)
	ErrFatalError      = NewSystemError(ErrorTypeCritical, ErrorCodeInternalError, "fatal error", nil)
)

// Web3-specific sentinel errors
var (
	ErrBlockNotFound     = NewSystemError(ErrorTypePermanent, ErrorCodeBlockNotFound, "block not found", nil)
	ErrEventNotFound     = NewSystemError(ErrorTypePermanent, ErrorCodeEventNotFound, "event not found", nil)
	ErrChainNotFound     = NewSystemError(ErrorTypePermanent, ErrorCodeChainNotFound, "chain not found", nil)
	ErrChainNotSupported = NewSystemError(ErrorTypePermanent, ErrorCodeChainNotSupported, "chain not supported", nil)
	ErrTxNotFound        = NewSystemError(ErrorTypePermanent, ErrorCodeTxNotFound, "transaction not found", nil)
	ErrContractNotFound  = NewSystemError(ErrorTypePermanent, ErrorCodeContractNotFound, "contract not found", nil)
	ErrRPCError          = NewSystemError(ErrorTypeTransient, ErrorCodeRPCError, "RPC call failed", nil)
	ErrRPCRateLimited    = NewSystemError(ErrorTypeTransient, ErrorCodeRPCRateLimited, "RPC rate limit exceeded", nil)
	ErrEventDecodeFailed = NewSystemError(ErrorTypePermanent, ErrorCodeEventDecodeFailed, "event decode failed", nil)
	ErrABINotFound       = NewSystemError(ErrorTypePermanent, ErrorCodeABINotFound, "ABI not found for contract", nil)
	ErrFinalityNotReady  = NewSystemError(ErrorTypeTransient, ErrorCodeFinalityNotReady, "block not yet finalized", nil)
	ErrRPCUnreachable    = NewSystemError(ErrorTypeTransient, ErrorCodeRPCUnreachable, "RPC endpoint unreachable", nil)
	ErrStaleBlock        = NewSystemError(ErrorTypePermanent, ErrorCodeStaleBlock, "block is too old, past safe reorg depth", nil)
	ErrReorgDetected     = NewSystemError(ErrorTypeTransient, ErrorCodeReorgDetected, "blockchain reorganization detected", nil)
)

// Constants for configuration
const (
	DefaultWorkerPoolSize = 10
	DefaultBatchSize      = 100
	DefaultMaxRetries     = 3
	DefaultRetryBackoff   = 100  // milliseconds
	DefaultCacheTTL       = 3600 // seconds (1 hour)
	DefaultAPIPort        = 8080
)

// Constants for error codes
const (
	ErrorCodeValidation    = "VALIDATION_ERROR"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeDuplicate     = "DUPLICATE"
	ErrorCodeDatabaseError = "DATABASE_ERROR"
	ErrorCodeCacheError    = "CACHE_ERROR"
	ErrorCodeMQError       = "MQ_ERROR"
	ErrorCodeNetworkError  = "NETWORK_ERROR"
	ErrorCodeTimeout       = "TIMEOUT"
	ErrorCodeInternalError = "INTERNAL_ERROR"
	ErrorCodeConfigError   = "CONFIG_ERROR"

	// Web3-specific error codes
	ErrorCodeBlockNotFound     = "BLOCK_NOT_FOUND"
	ErrorCodeEventNotFound     = "EVENT_NOT_FOUND"
	ErrorCodeChainNotFound     = "CHAIN_NOT_FOUND"
	ErrorCodeChainNotSupported = "CHAIN_NOT_SUPPORTED"
	ErrorCodeTxNotFound        = "TRANSACTION_NOT_FOUND"
	ErrorCodeContractNotFound  = "CONTRACT_NOT_FOUND"
	ErrorCodeRPCError          = "RPC_ERROR"
	ErrorCodeRPCRateLimited    = "RPC_RATE_LIMITED"
	ErrorCodeEventDecodeFailed = "EVENT_DECODE_FAILED"
	ErrorCodeABINotFound       = "ABI_NOT_FOUND"
	ErrorCodeFinalityNotReady  = "FINALITY_NOT_READY"
	ErrorCodeReorgDetected     = "REORG_DETECTED"
	ErrorCodeRPCUnreachable    = "RPC_UNREACHABLE"
	ErrorCodeStaleBlock        = "STALE_BLOCK"
	ErrorCodeInvalidEventData  = "INVALID_EVENT_DATA"
)

// NewSystemError creates a new system error
func NewSystemError(errorType ErrorType, code, message string, err error) *SystemError {
	return &SystemError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Err:     err,
		Details: make(map[string]any),
	}
}

// Error implements the error interface
func (e *SystemError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (%v)", e.Type, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// Unwrap returns the underlying error, enabling errors.Is() and errors.As()
// to traverse the error chain through SystemError.
func (e *SystemError) Unwrap() error { return e.Err }

// WithDetail adds a detail to the error
func (e *SystemError) WithDetail(key string, value any) *SystemError {
	e.Details[key] = value
	return e
}

// IsTransient checks if error is transient
func (e *SystemError) IsTransient() bool {
	return e.Type == ErrorTypeTransient
}

// IsPermanent checks if error is permanent
func (e *SystemError) IsPermanent() bool {
	return e.Type == ErrorTypePermanent
}

// IsCritical checks if error is critical
func (e *SystemError) IsCritical() bool {
	return e.Type == ErrorTypeCritical
}

// ClassifyError classifies an error as transient, permanent, or critical
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypePermanent
	}

	// 1. Check if it's a SystemError (covers all typed sentinel errors)
	var sysErr *SystemError
	if errors.As(err, &sysErr) {
		return sysErr.Type
	}

	// 2. Check typed sentinels via errors.Is (handles wrapped errors)
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrConnectionRefused) ||
		errors.Is(err, ErrConnectionReset) || errors.Is(err, ErrUnavailable) ||
		errors.Is(err, ErrDeadlineExceeded) || errors.Is(err, ErrTemporaryFailure) {
		return ErrorTypeTransient
	}
	if errors.Is(err, ErrDataCorruption) || errors.Is(err, ErrCriticalFailure) ||
		errors.Is(err, ErrFatalError) {
		return ErrorTypeCritical
	}

	// 3. Check for net.Error interface (transient network errors)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return ErrorTypeTransient
	}

	// 4. Check for syscall errors (transient)
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return ErrorTypeTransient
	}

	// 5. Context deadline exceeded (transient)
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTypeTransient
	}

	// Default to permanent
	return ErrorTypePermanent
}

// IsContextError checks if error is a context error
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// ClassifyErrorCode returns a stable error code string suitable for metrics tagging.
// Falls back to "UNKNOWN" if the error is not a SystemError or a known sentinel.
func ClassifyErrorCode(err error) string {
	if err == nil {
		return "OK"
	}

	var sysErr *SystemError
	if errors.As(err, &sysErr) {
		return sysErr.Code
	}

	switch {
	case errors.Is(err, ErrTimeout):
		return ErrorCodeTimeout
	case errors.Is(err, ErrConnectionRefused):
		return ErrorCodeNetworkError
	case errors.Is(err, ErrNotFound) || errors.Is(err, ErrBlockNotFound) ||
		errors.Is(err, ErrEventNotFound) || errors.Is(err, ErrTxNotFound) ||
		errors.Is(err, ErrContractNotFound):
		return ErrorCodeNotFound
	case errors.Is(err, ErrUnauthorized):
		return ErrorCodeValidation
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorCodeTimeout
	case errors.Is(err, context.Canceled):
		return ErrorCodeTimeout
	}

	return "UNKNOWN"
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
