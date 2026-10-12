package bootstrap

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/observability"
)

func TestProvideLogger(t *testing.T) {
	t.Parallel()
	l := provideLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestProvideMetrics(t *testing.T) {
	t.Parallel()
	m := provideMetrics()
	if m == nil {
		t.Fatal("expected non-nil metrics collector")
	}

	// The provider must hand out a GUARDED collector: an unbounded label such as
	// a tx hash must be redacted rather than creating one series per event.
	guard, ok := m.(*observability.MetricLabelGuard)
	if !ok {
		t.Fatalf("provideMetrics must return a guarded collector, got %T", m)
	}
	sanitized := guard.Sanitize(map[string]string{"tx_hash": "0xdeadbeef"})
	if sanitized["tx_hash"] != observability.RedactedLabelValue {
		t.Fatalf("tx_hash label was not redacted: %q", sanitized["tx_hash"])
	}
}
