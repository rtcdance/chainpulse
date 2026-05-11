package processing

import (
	"errors"
	"testing"
)

func TestEventValidationError_ImplementsError(t *testing.T) {
	eve := &EventValidationError{
		EventID: "evt-123",
		Reason:  "invalid signature",
	}

	// Must satisfy the error interface
	var err error = eve
	if err == nil {
		t.Fatal("EventValidationError should satisfy error interface")
	}

	msg := err.Error()
	if msg == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestEventValidationError_ErrorFormat(t *testing.T) {
	eve := &EventValidationError{
		EventID: "evt-456",
		Reason:  "block number is zero",
	}
	msg := eve.Error()
	expected := "validation failed for event evt-456: block number is zero"
	if msg != expected {
		t.Errorf("Error() = %q, want %q", msg, expected)
	}
}

func TestEventValidationError_Unwrap(t *testing.T) {
	inner := errors.New("abi decoding failed")
	eve := &EventValidationError{
		EventID: "evt-789",
		Reason:  "decode error",
		Err:     inner,
	}

	if !errors.Is(eve, inner) {
		t.Error("errors.Is should traverse EventValidationError chain")
	}

	if unwrapped := eve.Unwrap(); unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestEventValidationError_UnwrapNil(t *testing.T) {
	eve := &EventValidationError{
		EventID: "evt-000",
		Reason:  "missing field",
	}
	if err := eve.Unwrap(); err != nil {
		t.Errorf("Unwrap() should return nil when no inner error, got %v", err)
	}
}
