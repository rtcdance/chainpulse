package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "TEST", Message: "test message"}
	assert.Equal(t, "[TEST] test message", ae.Error())
}

func TestAPIError_ErrorWithWrapped(t *testing.T) {
	t.Parallel()
	inner := fmt.Errorf("inner error")
	ae := &APIError{Code: "TEST", Message: "test message", Err: inner}
	assert.Contains(t, ae.Error(), "inner error")
}

func TestAPIError_Unwrap(t *testing.T) {
	t.Parallel()
	inner := errors.New("inner")
	ae := &APIError{Code: "TEST", Message: "test", Err: inner}
	assert.True(t, errors.Is(ae, inner))
}

func TestAPIError_ToJSON(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "TEST", Message: "test message", Status: 400}
	data := ae.ToJSON()

	var decoded map[string]interface{}
	err := json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "TEST", decoded["error"])
	assert.Equal(t, "test message", decoded["message"])
	assert.Equal(t, float64(400), decoded["statusCode"])
}

func TestAPIError_ToJSONWithDetails(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "TEST", Message: "test", Status: 400, Details: map[string]any{"field": "name"}}
	data := ae.ToJSON()

	var decoded map[string]interface{}
	err := json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "name", decoded["details"].(map[string]interface{})["field"])
}

func TestAPIError_WriteHTTP(t *testing.T) {
	ae := &APIError{Code: "NOT_FOUND", Message: "resource not found", Status: 404}
	w := httptest.NewRecorder()
	ae.WriteHTTP(w)

	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var decoded map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &decoded)
	assert.Equal(t, "NOT_FOUND", decoded["error"])
}

func TestIsAPIError(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "TEST", Message: "test"}
	result, ok := IsAPIError(ae)
	assert.True(t, ok)
	assert.Equal(t, ae, result)
}

func TestIsAPIError_Wrapped(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "TEST", Message: "test"}
	wrapped := fmt.Errorf("wrapped: %w", ae)
	result, ok := IsAPIError(wrapped)
	assert.True(t, ok)
	assert.Equal(t, ae, result)
}

func TestIsAPIError_NotAPIError(t *testing.T) {
	t.Parallel()
	_, ok := IsAPIError(errors.New("plain error"))
	assert.False(t, ok)
}

func TestIsAPIError_Nil(t *testing.T) {
	t.Parallel()
	_, ok := IsAPIError(nil)
	assert.False(t, ok)
}

func TestMapErrorToAPIErrorNil(t *testing.T) {
	t.Parallel()
	result := MapErrorToAPIError(nil)
	assert.Equal(t, "OK", result.Code)
	assert.Equal(t, 200, result.Status)
}

func TestMapErrorToAPIErrorAlreadyAPIError(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "MY_CODE", Message: "my error", Status: 418}
	result := MapErrorToAPIError(ae)
	assert.Equal(t, ae, result)
}

func TestMapErrorToAPIErrorValidation(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeValidation, "bad input", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 400, result.Status)
	assert.Equal(t, "VALIDATION_FAILED", result.Code)
	assert.Equal(t, "bad input", result.Message)
}

func TestMapErrorToAPIErrorNotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeNotFound, "not found", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorDuplicate(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeDuplicate, "dup", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 409, result.Status)
	assert.Equal(t, "CONFLICT", result.Code)
}

func TestMapErrorToAPIErrorDatabaseTransient(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeDatabaseError, "db down", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 503, result.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", result.Code)
}

func TestMapErrorToAPIErrorDatabasePermanent(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeDatabaseError, "db issue", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 500, result.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", result.Code)
}

func TestMapErrorToAPIErrorCacheError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeCacheError, "cache miss", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 503, result.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", result.Code)
}

func TestMapErrorToAPIErrorMQError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeMQError, "mq issue", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 503, result.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", result.Code)
}

func TestMapErrorToAPIErrorNetworkError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeNetworkError, "timeout", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 502, result.Status)
	assert.Equal(t, "BAD_GATEWAY", result.Code)
}

func TestMapErrorToAPIErrorTimeout(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeTimeout, "deadline", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 504, result.Status)
	assert.Equal(t, "GATEWAY_TIMEOUT", result.Code)
}

func TestMapErrorToAPIErrorConfigError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeConfigError, "config bad", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 500, result.Status)
	assert.Equal(t, "INTERNAL_SERVER_ERROR", result.Code)
}

func TestMapErrorToAPIErrorBlockNotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeBlockNotFound, "blk not found", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "BLOCK_NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorEventNotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeEventNotFound, "evt not found", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "EVENT_NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorChainNotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeChainNotFound, "chain n/a", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "CHAIN_NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorChainNotSupported(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeChainNotSupported, "unsupported", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 400, result.Status)
	assert.Equal(t, "CHAIN_NOT_SUPPORTED", result.Code)
}

func TestMapErrorToAPIErrorTxNotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeTxNotFound, "tx n/a", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "TRANSACTION_NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorContractNotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeContractNotFound, "contract n/a", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "CONTRACT_NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorRPCError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeRPCError, "rpc fail", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 502, result.Status)
	assert.Equal(t, "RPC_ERROR", result.Code)
}

func TestMapErrorToAPIErrorRPCRateLimited(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeRPCRateLimited, "limited", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 429, result.Status)
	assert.Equal(t, "RPC_RATE_LIMITED", result.Code)
}

func TestMapErrorToAPIErrorEventDecodeFailed(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeEventDecodeFailed, "bad decode", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 422, result.Status)
	assert.Equal(t, "EVENT_DECODE_FAILED", result.Code)
}

func TestMapErrorToAPIErrorABINotFound(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeABINotFound, "no abi", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 404, result.Status)
	assert.Equal(t, "ABI_NOT_FOUND", result.Code)
}

func TestMapErrorToAPIErrorFinalityNotReady(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeFinalityNotReady, "wait", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 409, result.Status)
	assert.Equal(t, "FINALITY_NOT_READY", result.Code)
}

func TestMapErrorToAPIErrorReorgDetected(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, core.ErrorCodeReorgDetected, "reorg", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 409, result.Status)
	assert.Equal(t, "REORG_DETECTED", result.Code)
}

func TestMapErrorToAPIErrorInvalidEventData(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeInvalidEventData, "bad event", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 422, result.Status)
	assert.Equal(t, "INVALID_EVENT_DATA", result.Code)
}

func TestMapErrorToAPIErrorUnknownSystemError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypeTransient, "UNKNOWN_CODE", "some error", nil)
	result := MapErrorToAPIError(se)
	assert.Equal(t, 503, result.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", result.Code)
}

func TestMapErrorToAPIErrorUnknownPlainError(t *testing.T) {
	t.Parallel()
	result := MapErrorToAPIError(errors.New("something went wrong"))
	assert.Equal(t, 500, result.Status)
	assert.Equal(t, "INTERNAL_SERVER_ERROR", result.Code)
	assert.Equal(t, "an internal error occurred", result.Message)
}

func TestErrInvalidRequest(t *testing.T) {
	t.Parallel()
	err := ErrInvalidRequest("bad request")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 400, ae.Status)
	assert.Equal(t, "INVALID_REQUEST", ae.Code)
}

func TestErrUnauthorized(t *testing.T) {
	t.Parallel()
	err := ErrUnauthorized("no access")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 401, ae.Status)
	assert.Equal(t, "UNAUTHORIZED", ae.Code)
}

func TestErrForbidden(t *testing.T) {
	t.Parallel()
	err := ErrForbidden("denied")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 403, ae.Status)
	assert.Equal(t, "FORBIDDEN", ae.Code)
}

func TestErrNotFound(t *testing.T) {
	t.Parallel()
	err := ErrNotFound("user")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 404, ae.Status)
	assert.Contains(t, ae.Message, "user")
}

func TestErrInternalServer(t *testing.T) {
	t.Parallel()
	err := ErrInternalServer("oops")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 500, ae.Status)
	assert.Equal(t, "INTERNAL_SERVER_ERROR", ae.Code)
}

func TestErrServiceUnavailable(t *testing.T) {
	t.Parallel()
	err := ErrServiceUnavailable("down")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 503, ae.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", ae.Code)
}

func TestErrRateLimited(t *testing.T) {
	t.Parallel()
	err := ErrRateLimited()
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 429, ae.Status)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", ae.Code)
}

func TestErrInvalidParameter(t *testing.T) {
	t.Parallel()
	err := ErrInvalidParameter("id", "must be positive")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 400, ae.Status)
	assert.Contains(t, ae.Message, "id")
	assert.Contains(t, ae.Message, "must be positive")
}

func TestErrMissingParameter(t *testing.T) {
	t.Parallel()
	err := ErrMissingParameter("address")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 400, ae.Status)
	assert.Contains(t, ae.Message, "address")
}

func TestErrValidationFailed(t *testing.T) {
	t.Parallel()
	err := ErrValidationFailed("email", "invalid format")
	ae, ok := IsAPIError(err)
	assert.True(t, ok)
	assert.Equal(t, 400, ae.Status)
	assert.Contains(t, ae.Message, "email")
}

func TestMapErrorToAPIErrorWrappedSystemError(t *testing.T) {
	t.Parallel()
	se := core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeValidation, "bad", nil)
	wrapped := fmt.Errorf("context: %w", se)
	result := MapErrorToAPIError(wrapped)
	assert.Equal(t, 400, result.Status)
	assert.Equal(t, "VALIDATION_FAILED", result.Code)
}

func TestMapErrorToAPIErrorWrappedAPIError(t *testing.T) {
	t.Parallel()
	ae := &APIError{Code: "MY_CODE", Message: "my msg", Status: 418}
	wrapped := fmt.Errorf("context: %w", ae)
	result := MapErrorToAPIError(wrapped)
	assert.Equal(t, ae, result)
}
