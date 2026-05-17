package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCriticalError_ImplementsError(t *testing.T) {
	t.Parallel()
	ce := &CriticalError{
		Type:        CriticalErrorTypeDataCorruption,
		Message:     "block hash mismatch",
		Timestamp:   time.Now(),
		Component:   "reorg_detector",
		Recoverable: false,
	}

	// Must satisfy the error interface
	var err error = ce
	if err == nil {
		t.Fatal("CriticalError should satisfy error interface")
	}

	msg := err.Error()
	if msg == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestCriticalError_ErrorFormat(t *testing.T) {
	t.Parallel()
	ce := &CriticalError{
		Type:      CriticalErrorTypeSystemFailure,
		Message:   "out of memory",
		Component: "indexer",
	}
	msg := ce.Error()
	if msg != "[system_failure] out of memory" {
		t.Errorf("Error() = %q, want [system_failure] out of memory", msg)
	}
}

func TestCriticalError_ErrorWithWrap(t *testing.T) {
	t.Parallel()
	inner := errors.New("disk full")
	ce := &CriticalError{
		Type:    CriticalErrorTypeResourceExhaustion,
		Message: "storage exhausted",
		Err:     inner,
	}
	msg := ce.Error()
	if msg == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestCriticalError_Unwrap(t *testing.T) {
	t.Parallel()
	inner := errors.New("connection refused")
	ce := &CriticalError{
		Type:    CriticalErrorTypeSystemFailure,
		Message: "db unreachable",
		Err:     inner,
	}

	if !errors.Is(ce, inner) {
		t.Error("errors.Is should traverse CriticalError chain")
	}

	if unwrapped := ce.Unwrap(); unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestCriticalError_UnwrapNil(t *testing.T) {
	t.Parallel()
	ce := &CriticalError{
		Type:    CriticalErrorTypeSecurityBreach,
		Message: "unauthorized access",
	}
	if err := ce.Unwrap(); err != nil {
		t.Errorf("Unwrap() should return nil when no inner error, got %v", err)
	}
}

func TestCriticalError_ErrorsAs(t *testing.T) {
	t.Parallel()
	inner := errors.New("disk full")
	ce := &CriticalError{
		Type:    CriticalErrorTypeDataCorruption,
		Message: "write failed",
		Err:     inner,
	}

	var target *CriticalError
	if !errors.As(ce, &target) {
		t.Error("errors.As should extract CriticalError")
	}
	if target.Type != CriticalErrorTypeDataCorruption {
		t.Errorf("extracted Type = %q, want %q", target.Type, CriticalErrorTypeDataCorruption)
	}
}
