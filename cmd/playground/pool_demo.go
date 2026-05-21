package main

import (
	"net/http"
	"sync"
	"sync/atomic"
)

// eventPool is a sync.Pool for reusing BlockchainEvent slices.
// sync.Pool is Go's zero-cost object pool — it reduces GC pressure by
// recycling objects instead of allocating new ones on every request.
//
// For Web3 devs: this is like having a pre-allocated memory arena
// instead of malloc'ing every time. Useful for high-throughput indexing.
var eventPool = sync.Pool{
	New: func() any {
		return make([]*eventRecord, 0, 100)
	},
}

type eventRecord struct {
	ID          string
	EventName   string
	BlockNumber uint64
	Network     string
}

// poolStats tracks sync.Pool usage for educational visibility.
var poolStats struct {
	gets   atomic.Int64
	puts   atomic.Int64
	misses atomic.Int64
}

func (p *playground) handlePoolDemo(w http.ResponseWriter, r *http.Request) {
	// Get a slice from the pool (or allocate if none available)
	events := eventPool.Get().([]*eventRecord)

	// Track if this was a miss (new allocation)
	if cap(events) == 0 {
		poolStats.misses.Add(1)
	}
	poolStats.gets.Add(1)

	// Use the slice
	events = events[:0] // reset length, keep capacity
	stored, _ := p.db.GetAllEvents(r.Context())
	for i, ev := range stored {
		if i >= 100 {
			break
		}
		events = append(events, &eventRecord{
			ID:          ev.ID,
			EventName:   ev.EventName,
			BlockNumber: ev.BlockNumber,
			Network:     ev.Network,
		})
	}

	// Return to pool for reuse
	eventPool.Put(events)
	poolStats.puts.Add(1)

	getCount := poolStats.gets.Load()
	missCount := poolStats.misses.Load()
	missRate := 0.0
	if getCount > 0 {
		missRate = float64(missCount) / float64(getCount) * 100
	}

	writeJSON(w, map[string]any{
		"demo":         "sync.Pool in action",
		"events_count": len(events),
		"pool_stats": map[string]any{
			"gets":      getCount,
			"puts":      poolStats.puts.Load(),
			"misses":    missCount,
			"miss_rate": missRate,
		},
		"concept": "sync.Pool recycles objects to reduce GC — like a pre-allocated memory arena for Go",
	})
}
