// Package histogram provides a fixed-capacity ring buffer for float64 values.
package histogram

import "slices"

// Histogram is a fixed-capacity ring buffer for float64 values.
// It provides O(1) Record, and O(n) Percentile/Sum/Avg queries on the
// current window. When the buffer is full, new recordings overwrite the
// oldest entries (FIFO eviction), preventing unbounded memory growth.
type Histogram struct {
	buf      []float64
	head     int
	count    int
	capacity int
	sumVal   float64
}

// New creates a histogram with the given capacity.
func New(capacity int) *Histogram {
	if capacity <= 0 {
		capacity = 1024
	}
	return &Histogram{
		buf:      make([]float64, capacity),
		capacity: capacity,
	}
}

// Record adds a value to the histogram.
func (h *Histogram) Record(v float64) {
	if h.count >= h.capacity {
		h.sumVal -= h.buf[h.head]
	}
	h.sumVal += v
	h.buf[h.head] = v
	h.head = (h.head + 1) % h.capacity
	if h.count < h.capacity {
		h.count++
	}
}

// Count returns the number of recorded values.
func (h *Histogram) Count() int {
	return h.count
}

// Sum returns the sum of all values in the buffer.
func (h *Histogram) Sum() float64 {
	return h.sumVal
}

// Avg returns the arithmetic mean. Returns 0 if empty.
func (h *Histogram) Avg() float64 {
	if h.count == 0 {
		return 0
	}
	return h.sumVal / float64(h.count)
}

// Min returns the minimum value. Returns 0 if empty.
func (h *Histogram) Min() float64 {
	if h.count == 0 {
		return 0
	}
	all := h.All()
	return slices.Min(all)
}

// Max returns the maximum value. Returns 0 if empty.
func (h *Histogram) Max() float64 {
	if h.count == 0 {
		return 0
	}
	all := h.All()
	return slices.Max(all)
}

// Percentile returns the value at the given percentile (0-100).
func (h *Histogram) Percentile(p float64) float64 {
	if h.count == 0 {
		return 0
	}
	if p <= 0 {
		return h.Min()
	}
	if p >= 100 {
		return h.Max()
	}
	all := h.All()
	slices.Sort(all)
	rank := p / 100.0 * float64(len(all)-1)
	lower := int(rank)
	upper := lower + 1
	if upper >= len(all) {
		return all[len(all)-1]
	}
	frac := rank - float64(lower)
	return all[lower]*(1-frac) + all[upper]*frac
}

// All returns a copy of all values in insertion order.
func (h *Histogram) All() []float64 {
	if h.count == 0 {
		return nil
	}
	out := make([]float64, h.count)
	if h.count < h.capacity {
		copy(out, h.buf[:h.count])
	} else {
		copy(out, h.buf[h.head:])
		copy(out[h.capacity-h.head:], h.buf[:h.head])
	}
	return out
}

// Cap returns the maximum capacity of the histogram.
func (h *Histogram) Cap() int {
	return h.capacity
}

// Reset clears all recorded values.
func (h *Histogram) Reset() {
	h.head = 0
	h.count = 0
	h.sumVal = 0
}
