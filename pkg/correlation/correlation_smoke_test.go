package correlation

import (
	"context"
	"testing"
)

func TestCorrelationSmoke(t *testing.T) {
	ctx := context.Background()

	ctx2, id := FromContextOrNew(ctx, "test-id")
	got := FromContext(ctx2)
	if got != id {
		t.Errorf("FromContext = %q, want %q", got, id)
	}

	ctx3 := WithID(ctx2, id)
	got2 := FromContext(ctx3)
	if got2 != id {
		t.Errorf("FromContext = %q, want %q", got2, id)
	}
}

func TestCorrelationNoID(t *testing.T) {
	ctx := context.Background()
	got := FromContext(ctx)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	ctx2, newID := FromContextOrNew(ctx, "")
	if newID == "" {
		t.Error("expected non-empty new ID")
	}
	if FromContext(ctx2) != newID {
		t.Error("FromContext should return the new ID")
	}
}
