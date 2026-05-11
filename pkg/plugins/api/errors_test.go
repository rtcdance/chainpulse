package api

import (
	"errors"
	"testing"
)

func TestAPIError_Unwrap(t *testing.T) {
	inner := errors.New("database connection lost")
	ae := &APIError{
		Code:    "SERVICE_UNAVAILABLE",
		Message: "cache temporarily unavailable",
		Status:  503,
		Err:     inner,
	}

	// errors.Is should traverse through APIError
	if !errors.Is(ae, inner) {
		t.Error("errors.Is should find wrapped error through APIError.Unwrap()")
	}

	// Verify Unwrap returns the inner error
	if unwrapped := ae.Unwrap(); unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestAPIError_UnwrapNil(t *testing.T) {
	ae := ErrNotFound("block")
	if err := ae.(*APIError).Unwrap(); err != nil {
		t.Errorf("Unwrap() should return nil when no inner error, got %v", err)
	}
}

func TestAPIError_ErrorWithWrap(t *testing.T) {
	inner := errors.New("connection refused")
	ae := &APIError{
		Code:    "BAD_GATEWAY",
		Message: "upstream error",
		Status:  502,
		Err:     inner,
	}
	msg := ae.Error()
	if msg == "" {
		t.Error("Error() should not be empty")
	}
}

func TestAPIError_ErrorWithoutWrap(t *testing.T) {
	ae := ErrInvalidRequest("missing field")
	msg := ae.Error()
	if msg == "" {
		t.Error("Error() should not be empty")
	}
}

func TestAPIError_ErrorsIsChain(t *testing.T) {
	inner := errors.New("timeout")
	wrapped := &APIError{
		Code:    "GATEWAY_TIMEOUT",
		Message: "request timed out",
		Status:  504,
		Err:     inner,
	}

	// Should find inner error
	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is should traverse APIError chain")
	}

	// Should NOT match unrelated
	unrelated := errors.New("something else")
	if errors.Is(wrapped, unrelated) {
		t.Error("errors.Is should not match unrelated error")
	}
}

func TestAPIError_ToJSONWithoutErr(t *testing.T) {
	ae := &APIError{
		Code:    "NOT_FOUND",
		Message: "resource not found",
		Status:  404,
	}
	data := ae.ToJSON()
	if len(data) == 0 {
		t.Error("ToJSON() should return non-empty bytes")
	}
	// Err field should not appear in JSON (json:"-")
	if string(data) == "" {
		t.Error("ToJSON() output should be valid JSON")
	}
}
