package observability

import (
	"math"
	"sort"
	"sync"
	"time"
)

// latencyRing is a fixed-capacity ring buffer for latency measurements.
// It provides O(1) push and bounded memory usage, unlike append+re-slice
// which never frees the underlying array.
type latencyRing struct {
	mu   sync.RWMutex
	buf  []time.Duration
	head int // next write position
	size int // current number of elements
	cap  int // maximum capacity
}

// newLatencyRing creates a new ring buffer with the given capacity.
func newLatencyRing(capacity int) *latencyRing {
	if capacity <= 0 {
		capacity = 1
	}
	return &latencyRing{
		buf:  make([]time.Duration, capacity),
		head: 0,
		size: 0,
		cap:  capacity,
	}
}

// Push adds a latency measurement to the ring buffer.
// When the buffer is full, the oldest measurement is overwritten.
func (r *latencyRing) Push(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buf[r.head] = d
	r.head = (r.head + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
}

// Len returns the current number of elements in the ring buffer.
func (r *latencyRing) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}

// Avg computes the average latency of all measurements.
// Returns 0 if the buffer is empty.
func (r *latencyRing) Avg() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return 0
	}

	var total time.Duration
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.cap) % r.cap
		total += r.buf[idx]
	}
	return total / time.Duration(r.size)
}

// Max returns the maximum latency in the buffer.
// Returns 0 if the buffer is empty.
func (r *latencyRing) Max() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return 0
	}

	var max time.Duration
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.cap) % r.cap
		if r.buf[idx] > max {
			max = r.buf[idx]
		}
	}
	return max
}

// Min returns the minimum latency in the buffer.
// Returns 0 if the buffer is empty.
func (r *latencyRing) Min() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return 0
	}

	min := time.Duration(math.MaxInt64)
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.cap) % r.cap
		if r.buf[idx] < min {
			min = r.buf[idx]
		}
	}
	return min
}

// Percentile returns the latency at the given percentile (0.0 to 1.0).
// Uses linear interpolation between adjacent values.
// Returns 0 if the buffer is empty.
func (r *latencyRing) Percentile(p float64) time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return 0
	}

	// Clamp percentile
	p = math.Max(0, math.Min(1, p))

	// Collect and sort values
	values := make([]time.Duration, r.size)
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.cap) % r.cap
		values[i] = r.buf[idx]
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })

	// Compute index
	idx := p * float64(r.size-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))

	if upper >= r.size {
		return values[r.size-1]
	}
	if lower == upper {
		return values[lower]
	}

	// Linear interpolation
	frac := idx - float64(lower)
	return values[lower] + time.Duration(float64(values[upper]-values[lower])*frac)
}

// All returns a snapshot of all values in insertion order.
func (r *latencyRing) All() []time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]time.Duration, r.size)
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.cap) % r.cap
		result[i] = r.buf[idx]
	}
	return result
}

// Reset clears the ring buffer.
func (r *latencyRing) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.head = 0
	r.size = 0
}
