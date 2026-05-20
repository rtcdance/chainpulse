package histogram

import (
	"testing"
)

func TestHistogramSmoke(t *testing.T) {
	h := New(10)
	h.Record(50.0)
	h.Record(0.0)
	h.Record(100.0)
	h.Record(101.0)
	h.Record(-1.0)

	p50 := h.Percentile(50)
	if p50 < 0 || p50 > 101 {
		t.Errorf("p50 out of range: %f", p50)
	}
	if h.Count() != 5 {
		t.Errorf("expected 5 records, got %d", h.Count())
	}
	if h.Min() != -1.0 {
		t.Errorf("expected min -1.0, got %f", h.Min())
	}
	if h.Max() > 101 {
		t.Errorf("max should be <= 101, got %f", h.Max())
	}
	if h.Avg() <= 0 {
		t.Errorf("avg should be > 0, got %f", h.Avg())
	}
	if h.Cap() != 10 {
		t.Errorf("expected cap 10, got %d", h.Cap())
	}
	all := h.All()
	if len(all) != 5 {
		t.Errorf("expected 5 values, got %d", len(all))
	}
	h.Reset()
	if h.Count() != 0 {
		t.Error("expected 0 after reset")
	}
}

func TestHistogramEmpty(t *testing.T) {
	h := New(5)
	if h.Min() != 0 {
		t.Error("expected 0 min for empty")
	}
	if h.Max() != 0 {
		t.Error("expected 0 max for empty")
	}
	if h.Avg() != 0 {
		t.Error("expected 0 avg for empty")
	}
}

func TestHistogramOverwrite(t *testing.T) {
	h := New(3)
	for i := 0; i < 10; i++ {
		h.Record(float64(i))
	}
	if h.Count() != 3 {
		t.Errorf("expected 3 (capacity), got %d", h.Count())
	}
}
