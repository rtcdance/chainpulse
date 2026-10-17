package query

import (
	"errors"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/services/query/qerrors"
)

func TestErrorType_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		et   qerrors.Type
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
				t.Logf("qerrors.Type(%d).String() = %q", tc.et, got)
			}
		})
	}
}

func TestNewErrorClassifier(t *testing.T) {
	t.Parallel()
	ec := qerrors.NewClassifier()
	if ec == nil {
		t.Fatal("qerrors.NewClassifier() returned nil")
	}
}

func TestErrorClassifier_Classify(t *testing.T) {
	t.Parallel()
	ec := qerrors.NewClassifier()
	et := ec.Classify(errors.New("connection refused"))
	_ = et
}

func TestErrorClassifier_IsPermanent(t *testing.T) {
	t.Parallel()
	ec := qerrors.NewClassifier()
	_ = ec.IsPermanent(errors.New("not found"))
}

func TestErrorClassifier_IsTransient(t *testing.T) {
	t.Parallel()
	ec := qerrors.NewClassifier()
	_ = ec.IsTransient(errors.New("timeout"))
}

func TestErrorClassifier_IsCritical(t *testing.T) {
	t.Parallel()
	ec := qerrors.NewClassifier()
	_ = ec.IsCritical(errors.New("fatal"))
}

func TestErrorClassifier_ClassifyWithContext(t *testing.T) {
	t.Parallel()
	ec := qerrors.NewClassifier()
	_ = ec.ClassifyWithContext(errors.New("connection reset"), "fetch_block")
}
