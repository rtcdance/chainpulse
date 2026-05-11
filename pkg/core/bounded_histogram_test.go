package core

import (
	"math"
	"testing"
)

func TestBoundedHistogram_RecordAndCount(t *testing.T) {
	h := newBoundedHistogram(5)

	if h.Count() != 0 {
		t.Errorf("initial count = %d, want 0", h.Count())
	}

	h.Record(1.0)
	h.Record(2.0)
	h.Record(3.0)

	if h.Count() != 3 {
		t.Errorf("count after 3 records = %d, want 3", h.Count())
	}
}

func TestBoundedHistogram_CapacityEviction(t *testing.T) {
	h := newBoundedHistogram(3)

	h.Record(1.0)
	h.Record(2.0)
	h.Record(3.0)
	// Buffer is full. Next record evicts oldest (1.0).
	h.Record(4.0)

	if h.Count() != 3 {
		t.Errorf("count = %d, want 3 (capped at capacity)", h.Count())
	}

	all := h.All()
	if len(all) != 3 {
		t.Fatalf("All() = %d entries, want 3", len(all))
	}

	// Should contain [2, 3, 4]
	for i, want := range []float64{2, 3, 4} {
		if all[i] != want {
			t.Errorf("all[%d] = %v, want %v", i, all[i], want)
		}
	}
}

func TestBoundedHistogram_Sum(t *testing.T) {
	h := newBoundedHistogram(10)

	h.Record(10.0)
	h.Record(20.0)
	h.Record(30.0)

	if s := h.Sum(); s != 60.0 {
		t.Errorf("Sum() = %v, want 60", s)
	}
}

func TestBoundedHistogram_SumWithEviction(t *testing.T) {
	h := newBoundedHistogram(2)

	h.Record(10.0)
	h.Record(20.0)
	h.Record(30.0) // evicts 10.0

	if s := h.Sum(); s != 50.0 {
		t.Errorf("Sum() after eviction = %v, want 50", s)
	}
}

func TestBoundedHistogram_Avg(t *testing.T) {
	h := newBoundedHistogram(10)

	if a := h.Avg(); a != 0 {
		t.Errorf("Avg() of empty = %v, want 0", a)
	}

	h.Record(10.0)
	h.Record(20.0)
	h.Record(30.0)

	if a := h.Avg(); a != 20.0 {
		t.Errorf("Avg() = %v, want 20", a)
	}
}

func TestBoundedHistogram_MinMax(t *testing.T) {
	h := newBoundedHistogram(10)

	if m := h.Min(); m != 0 {
		t.Errorf("Min() of empty = %v, want 0", m)
	}
	if m := h.Max(); m != 0 {
		t.Errorf("Max() of empty = %v, want 0", m)
	}

	h.Record(5.0)
	h.Record(1.0)
	h.Record(10.0)
	h.Record(3.0)

	if m := h.Min(); m != 1.0 {
		t.Errorf("Min() = %v, want 1", m)
	}
	if m := h.Max(); m != 10.0 {
		t.Errorf("Max() = %v, want 10", m)
	}
}

func TestBoundedHistogram_Percentile(t *testing.T) {
	h := newBoundedHistogram(100)

	if p := h.Percentile(50); p != 0 {
		t.Errorf("Percentile(50) of empty = %v, want 0", p)
	}

	// Record 1..10
	for i := 1; i <= 10; i++ {
		h.Record(float64(i) * 10.0) // 10, 20, ..., 100
	}

	tests := []struct {
		p    float64
		want float64
	}{
		{0, 10},
		{50, 55},   // interpolated: index 4.5 → (50+60)/2 = 55
		{100, 100},
	}

	for _, tt := range tests {
		got := h.Percentile(tt.p)
		if math.Abs(got-tt.want) > 0.01 {
			t.Errorf("Percentile(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestBoundedHistogram_Reset(t *testing.T) {
	h := newBoundedHistogram(5)

	h.Record(1.0)
	h.Record(2.0)
	h.Record(3.0)
	h.Reset()

	if h.Count() != 0 {
		t.Errorf("Count() after reset = %d, want 0", h.Count())
	}
	if h.Sum() != 0 {
		t.Errorf("Sum() after reset = %v, want 0", h.Sum())
	}
}

func TestBoundedHistogram_DefaultCapacity(t *testing.T) {
	h := newBoundedHistogram(0)
	if h.capacity != 1024 {
		t.Errorf("capacity with 0 = %d, want 1024", h.capacity)
	}

	h = newBoundedHistogram(-1)
	if h.capacity != 1024 {
		t.Errorf("capacity with -1 = %d, want 1024", h.capacity)
	}
}

func TestBoundedHistogram_AllReturnsCopy(t *testing.T) {
	h := newBoundedHistogram(5)

	h.Record(30.0)
	h.Record(10.0)
	h.Record(20.0)

	all := h.All()
	if len(all) != 3 {
		t.Fatalf("All() = %d entries, want 3", len(all))
	}

	// All() returns values in insertion order (not sorted)
	// Verify all values are present
	sum := 0.0
	for _, v := range all {
		sum += v
	}
	if sum != 60.0 {
		t.Errorf("sum of All() = %v, want 60", sum)
	}
}
