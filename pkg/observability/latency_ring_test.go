package observability

import (
	"testing"
	"time"
)

func TestLatencyRing_PushAndLen(t *testing.T) {
	r := newLatencyRing(5)

	if r.Len() != 0 {
		t.Errorf("Len() = %d, want 0", r.Len())
	}

	r.Push(10 * time.Millisecond)
	r.Push(20 * time.Millisecond)

	if r.Len() != 2 {
		t.Errorf("Len() = %d, want 2", r.Len())
	}
}

func TestLatencyRing_Overwrite(t *testing.T) {
	r := newLatencyRing(3)

	r.Push(10 * time.Millisecond)
	r.Push(20 * time.Millisecond)
	r.Push(30 * time.Millisecond)
	r.Push(40 * time.Millisecond) // overwrites first

	if r.Len() != 3 {
		t.Errorf("Len() = %d, want 3", r.Len())
	}

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All() length = %d, want 3", len(all))
	}

	// Should contain 20, 30, 40
	if all[0] != 20*time.Millisecond || all[1] != 30*time.Millisecond || all[2] != 40*time.Millisecond {
		t.Errorf("All() = %v, want [20ms, 30ms, 40ms]", all)
	}
}

func TestLatencyRing_Avg(t *testing.T) {
	r := newLatencyRing(100)

	if avg := r.Avg(); avg != 0 {
		t.Errorf("Avg() on empty = %v, want 0", avg)
	}

	r.Push(10 * time.Millisecond)
	r.Push(20 * time.Millisecond)
	r.Push(30 * time.Millisecond)

	avg := r.Avg()
	if avg != 20*time.Millisecond {
		t.Errorf("Avg() = %v, want 20ms", avg)
	}
}

func TestLatencyRing_Max(t *testing.T) {
	r := newLatencyRing(100)

	if max := r.Max(); max != 0 {
		t.Errorf("Max() on empty = %v, want 0", max)
	}

	r.Push(10 * time.Millisecond)
	r.Push(50 * time.Millisecond)
	r.Push(30 * time.Millisecond)

	if max := r.Max(); max != 50*time.Millisecond {
		t.Errorf("Max() = %v, want 50ms", max)
	}
}

func TestLatencyRing_Min(t *testing.T) {
	r := newLatencyRing(100)

	r.Push(10 * time.Millisecond)
	r.Push(50 * time.Millisecond)
	r.Push(30 * time.Millisecond)

	if min := r.Min(); min != 10*time.Millisecond {
		t.Errorf("Min() = %v, want 10ms", min)
	}
}

func TestLatencyRing_Percentile(t *testing.T) {
	r := newLatencyRing(100)

	if p := r.Percentile(0.5); p != 0 {
		t.Errorf("Percentile() on empty = %v, want 0", p)
	}

	// Add 10 values: 10, 20, 30, ..., 100 ms
	for i := 1; i <= 10; i++ {
		r.Push(time.Duration(i) * 10 * time.Millisecond)
	}

	tests := []struct {
		p    float64
		want time.Duration
	}{
		{0.0, 10 * time.Millisecond},
		{0.5, 55 * time.Millisecond},   // index 4.5: interpolated between 50ms and 60ms
		{1.0, 100 * time.Millisecond},
		{0.9, 91 * time.Millisecond},   // index 8.1: interpolated between 90ms and 100ms
	}

	for _, tt := range tests {
		got := r.Percentile(tt.p)
		if got != tt.want {
			t.Errorf("Percentile(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestLatencyRing_PercentileClamp(t *testing.T) {
	r := newLatencyRing(5)
	r.Push(10 * time.Millisecond)

	// Out-of-range percentiles should clamp
	if p := r.Percentile(-1); p != 10*time.Millisecond {
		t.Errorf("Percentile(-1) = %v, want 10ms", p)
	}
	if p := r.Percentile(2.0); p != 10*time.Millisecond {
		t.Errorf("Percentile(2.0) = %v, want 10ms", p)
	}
}

func TestLatencyRing_Reset(t *testing.T) {
	r := newLatencyRing(5)
	r.Push(10 * time.Millisecond)
	r.Push(20 * time.Millisecond)

	r.Reset()

	if r.Len() != 0 {
		t.Errorf("Len() after reset = %d, want 0", r.Len())
	}
	if avg := r.Avg(); avg != 0 {
		t.Errorf("Avg() after reset = %v, want 0", avg)
	}
}

func TestLatencyRing_All(t *testing.T) {
	r := newLatencyRing(5)
	r.Push(10 * time.Millisecond)
	r.Push(20 * time.Millisecond)

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() length = %d, want 2", len(all))
	}
	if all[0] != 10*time.Millisecond || all[1] != 20*time.Millisecond {
		t.Errorf("All() = %v, want [10ms, 20ms]", all)
	}
}

func TestLatencyRing_ZeroCapacity(t *testing.T) {
	// Should default to capacity 1
	r := newLatencyRing(0)
	r.Push(10 * time.Millisecond)
	if r.Len() != 1 {
		t.Errorf("Len() = %d, want 1", r.Len())
	}
}
