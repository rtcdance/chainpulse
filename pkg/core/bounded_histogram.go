package core

import "slices"

// boundedHistogram is a fixed-capacity ring buffer for float64 values.
// It provides O(1) Record, and O(n) Percentile/Sum/Avg queries on the
// current window. When the buffer is full, new recordings overwrite the
// oldest entries (FIFO eviction), preventing unbounded memory growth.
type boundedHistogram struct {
	buf    []float64
	head   int // next write position
	count  int // number of entries recorded (capped at cap)
	capacity int // capacity (exported name avoided)
	sumVal float64
}

// newBoundedHistogram creates a histogram with the given capacity.
func newBoundedHistogram(capacity int) *boundedHistogram {
	if capacity <= 0 {
		capacity = 1024
	}
	return &boundedHistogram{
		buf:  make([]float64, capacity),
		capacity: capacity,
	}
}

// Record adds a value to the histogram. If the buffer is full, the oldest
// value is evicted (FIFO).
func (h *boundedHistogram) Record(v float64) {
	if h.count >= h.capacity {
		// Evict oldest value at head position
		h.sumVal -= h.buf[h.head]
	}
	h.sumVal += v
	h.buf[h.head] = v
	h.head = (h.head + 1) % h.capacity
	if h.count < h.capacity {
		h.count++
	}
}

// Count returns the number of recorded values currently in the buffer.
func (h *boundedHistogram) Count() int {
	return h.count
}

// Sum returns the sum of all values in the buffer.
func (h *boundedHistogram) Sum() float64 {
	return h.sumVal
}

// Avg returns the arithmetic mean of all values. Returns 0 if empty.
func (h *boundedHistogram) Avg() float64 {
	if h.count == 0 {
		return 0
	}
	return h.sumVal / float64(h.count)
}

// Min returns the minimum value. Returns 0 if empty.
func (h *boundedHistogram) Min() float64 {
	if h.count == 0 {
		return 0
	}
	all := h.All()
	return slices.Min(all)
}

// Max returns the maximum value. Returns 0 if empty.
func (h *boundedHistogram) Max() float64 {
	if h.count == 0 {
		return 0
	}
	all := h.All()
	return slices.Max(all)
}

// Percentile returns the value at the given percentile (0-100) using
// linear interpolation. Returns 0 if empty.
func (h *boundedHistogram) Percentile(p float64) float64 {
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

	// Linear interpolation
	rank := p / 100.0 * float64(len(all)-1)
	lower := int(rank)
	upper := lower + 1
	if upper >= len(all) {
		return all[len(all)-1]
	}
	frac := rank - float64(lower)
	return all[lower]*(1-frac) + all[upper]*frac
}

// All returns a copy of all values in the buffer in insertion order.
func (h *boundedHistogram) All() []float64 {
	if h.count == 0 {
		return nil
	}
	out := make([]float64, h.count)
	if h.count < h.capacity {
		copy(out, h.buf[:h.count])
	} else {
		// Ring buffer is full: head..end + start..head
		copy(out, h.buf[h.head:])
		copy(out[h.capacity-h.head:], h.buf[:h.head])
	}
	return out
}

// Reset clears all recorded values.
func (h *boundedHistogram) Reset() {
	h.head = 0
	h.count = 0
	h.sumVal = 0
}
