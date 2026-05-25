package bootstrap

import (
	"testing"
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
}
