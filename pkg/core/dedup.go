package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type DedupEntry struct {
	expiresAt time.Time
}

type RequestDeduplicator struct {
	mu      sync.Mutex
	entries map[string]DedupEntry
	ttl     time.Duration
}

func NewRequestDeduplicator(ttl time.Duration) *RequestDeduplicator {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &RequestDeduplicator{
		entries: make(map[string]DedupEntry),
		ttl:     ttl,
	}
}

func (d *RequestDeduplicator) hash(method string, params string) string {
	h := sha256.Sum256([]byte(method + "|" + params))
	return hex.EncodeToString(h[:16])
}

func (d *RequestDeduplicator) IsDuplicate(method, params string) bool {
	key := d.hash(method, params)
	d.mu.Lock()
	defer d.mu.Unlock()

	d.evictExpired()

	entry, exists := d.entries[key]
	if exists && time.Now().Before(entry.expiresAt) {
		return true
	}

	d.entries[key] = DedupEntry{expiresAt: time.Now().Add(d.ttl)}
	return false
}

func (d *RequestDeduplicator) evictExpired() {
	now := time.Now()
	for key, entry := range d.entries {
		if now.After(entry.expiresAt) {
			delete(d.entries, key)
		}
	}
}

func (d *RequestDeduplicator) Size() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.evictExpired()
	return len(d.entries)
}

func (d *RequestDeduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries = make(map[string]DedupEntry)
}