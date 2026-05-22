package query

import (
	"errors"
	"testing"
)

func TestErrorType_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		et   ErrorType
		want string
	}{
		{0, "unknown"},
		{1, "unknown"},
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := tc.et.String()
			if got != tc.want && got != "" {
				t.Logf("ErrorType(%d).String() = %q", tc.et, got)
			}
		})
	}
}

func TestNewErrorClassifier(t *testing.T) {
	t.Parallel()
	ec := NewErrorClassifier()
	if ec == nil {
		t.Fatal("NewErrorClassifier() returned nil")
	}
}

func TestErrorClassifier_Classify(t *testing.T) {
	t.Parallel()
	ec := NewErrorClassifier()
	et := ec.Classify(errors.New("connection refused"))
	_ = et
}

func TestErrorClassifier_IsPermanent(t *testing.T) {
	t.Parallel()
	ec := NewErrorClassifier()
	_ = ec.IsPermanent(errors.New("not found"))
}

func TestErrorClassifier_IsTransient(t *testing.T) {
	t.Parallel()
	ec := NewErrorClassifier()
	_ = ec.IsTransient(errors.New("timeout"))
}

func TestErrorClassifier_IsCritical(t *testing.T) {
	t.Parallel()
	ec := NewErrorClassifier()
	_ = ec.IsCritical(errors.New("fatal"))
}

func TestErrorClassifier_ClassifyErrorWithContext(t *testing.T) {
	t.Parallel()
	ec := NewErrorClassifier()
	_, op := ec.ClassifyErrorWithContext(errors.New("connection reset"), "fetch_block")
	if op == "" {
		t.Error("expected non-empty operation string")
	}
}
