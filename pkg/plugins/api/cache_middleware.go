package api

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api/shared"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	Key         string
	Value       []byte
	Headers     http.Header
	StatusCode  int
	ExpiresAt   time.Time
	CreatedAt   time.Time
	AccessedAt  time.Time
	AccessCount int64
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.ExpiresAt)
}

// CacheMiddleware implements HTTP caching middleware
type CacheMiddleware struct {
	cache            map[string]*CacheEntry
	config           *CacheConfig
	metrics          *CacheMetrics
	logger           core.Logger
	invalidator      *CacheInvalidator
	evictionStrategy EvictionStrategy
	lastCleanup      time.Time
	mu               sync.RWMutex
}

// EvictionStrategy defines how cache entries are evicted
type EvictionStrategy interface {
	SelectForEviction(entries map[string]*CacheEntry) string
}

// LRUEvictionStrategy evicts least recently used entries
type LRUEvictionStrategy struct{}

// SelectForEviction selects the least recently used entry
func (s *LRUEvictionStrategy) SelectForEviction(entries map[string]*CacheEntry) string {
	var lruKey string
	var lruTime time.Time

	for key, entry := range entries {
		if lruKey == "" || entry.AccessedAt.Before(lruTime) {
			lruKey = key
			lruTime = entry.AccessedAt
		}
	}

	return lruKey
}

// LFUEvictionStrategy evicts least frequently used entries
type LFUEvictionStrategy struct{}

// SelectForEviction selects the least frequently used entry
func (s *LFUEvictionStrategy) SelectForEviction(entries map[string]*CacheEntry) string {
	var lfuKey string
	var lfuCount int64

	for key, entry := range entries {
		if lfuKey == "" || entry.AccessCount < lfuCount {
			lfuKey = key
			lfuCount = entry.AccessCount
		}
	}

	return lfuKey
}

// FIFOEvictionStrategy evicts oldest entries first
type FIFOEvictionStrategy struct{}

// SelectForEviction selects the oldest entry
func (s *FIFOEvictionStrategy) SelectForEviction(entries map[string]*CacheEntry) string {
	var fifoKey string
	var fifoTime time.Time

	for key, entry := range entries {
		if fifoKey == "" || entry.CreatedAt.Before(fifoTime) {
			fifoKey = key
			fifoTime = entry.CreatedAt
		}
	}

	return fifoKey
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(config *CacheConfig, logger core.Logger, metrics core.MetricsCollector) *CacheMiddleware {
	var strategy EvictionStrategy
	switch config.EvictionStrategy {
	case "LFU":
		strategy = &LFUEvictionStrategy{}
	case "FIFO":
		strategy = &FIFOEvictionStrategy{}
	default:
		strategy = &LRUEvictionStrategy{}
	}

	cm := &CacheMiddleware{
		cache:            make(map[string]*CacheEntry),
		config:           config,
		metrics:          NewCacheMetrics(logger, metrics),
		logger:           logger,
		evictionStrategy: strategy,
		lastCleanup:      time.Now(),
	}

	cm.invalidator = NewCacheInvalidator(cm, logger, metrics)

	return cm
}

// Get retrieves a value from cache
func (cm *CacheMiddleware) Get(key string) ([]byte, http.Header, int, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	start := time.Now()

	entry, ok := cm.cache[key]
	if !ok {
		cm.metrics.RecordMiss(key, time.Since(start))
		return nil, nil, 0, false
	}

	if entry.IsExpired() {
		cm.metrics.RecordMiss(key, time.Since(start))
		return nil, nil, 0, false
	}

	entry.AccessedAt = time.Now()
	entry.AccessCount++

	cm.metrics.RecordHit(key, time.Since(start))
	return entry.Value, entry.Headers, entry.StatusCode, true
}

// Set stores a value in cache
func (cm *CacheMiddleware) Set(key string, value []byte, headers http.Header, statusCode int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Perform cleanup if needed
	if time.Since(cm.lastCleanup) > cm.config.CleanupInterval {
		cm.cleanup()
		cm.lastCleanup = time.Now()
	}

	// Check if cache is full
	if len(cm.cache) >= cm.config.MaxSize {
		cm.evict()
	}

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		Headers:     headers.Clone(),
		StatusCode:  statusCode,
		ExpiresAt:   time.Now().Add(cm.config.DefaultTTL),
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
	}

	cm.cache[key] = entry
}

// Invalidate removes a key from cache
func (cm *CacheMiddleware) Invalidate(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.cache[key]; ok {
		delete(cm.cache, key)
		cm.metrics.RecordInvalidation(key, "manual")
	}
}

// InvalidatePattern removes all keys matching a pattern
func (cm *CacheMiddleware) InvalidatePattern(pattern string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for key := range cm.cache {
		if matchPattern(key, pattern) {
			delete(cm.cache, key)
			cm.metrics.RecordInvalidation(key, "pattern")
		}
	}
}

// cleanup removes expired entries
func (cm *CacheMiddleware) cleanup() {
	now := time.Now()
	for key, entry := range cm.cache {
		if now.After(entry.ExpiresAt) {
			delete(cm.cache, key)
			cm.metrics.RecordEviction(key, "expired")
		}
	}
}

// evict removes an entry based on eviction strategy
func (cm *CacheMiddleware) evict() {
	if len(cm.cache) == 0 {
		return
	}

	keyToEvict := cm.evictionStrategy.SelectForEviction(cm.cache)
	if keyToEvict != "" {
		delete(cm.cache, keyToEvict)
		cm.metrics.RecordEviction(keyToEvict, "size_limit")
	}
}

// GetStats returns cache statistics
func (cm *CacheMiddleware) GetStats() map[string]any {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]any{
		"size":     len(cm.cache),
		"max_size": cm.config.MaxSize,
		"metrics":  cm.metrics.GetStats(),
	}
}

// Clear clears all cache entries
func (cm *CacheMiddleware) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache = make(map[string]*CacheEntry)
	cm.metrics.Reset()
}

// Close closes the cache middleware and its invalidator
func (cm *CacheMiddleware) Close() error {
	if cm.invalidator != nil {
		return cm.invalidator.Close()
	}
	return nil
}

// Middleware wraps an HTTP handler with caching
func (cm *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests
		if r.Method != http.MethodGet {
			w.Header().Set("X-Cache", "MISS")
			next.ServeHTTP(w, r)
			return
		}

		// Generate cache key
		key := generateCacheKey(r)

		// Check cache
		if value, headers, statusCode, ok := cm.Get(key); ok {
			// Return cached response
			for k, v := range headers {
				w.Header()[k] = v
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(statusCode)
			if _, err := w.Write(value); err != nil {
				cm.logger.Error("Failed to write cached response", "error", err.Error())
			}
			return
		}

		// Capture response
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           new(bytes.Buffer),
		}

		// Call next handler
		next.ServeHTTP(recorder, r)

		// Cache response if successful
		if recorder.statusCode >= 200 && recorder.statusCode < 300 {
			cm.Set(key, recorder.body.Bytes(), recorder.Header(), recorder.statusCode)
		}

		// Set cache header
		w.Header().Set("X-Cache", "MISS")
	})
}

// responseRecorder captures HTTP response
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// Header returns the header map
func (rr *responseRecorder) Header() http.Header {
	return rr.ResponseWriter.Header()
}

// WriteHeader records the status code
func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response body
func (rr *responseRecorder) Write(b []byte) (int, error) {
	if _, err := rr.body.Write(b); err != nil {
		return 0, err
	}
	return rr.ResponseWriter.Write(b)
}

// generateCacheKey generates a cache key from the request
func generateCacheKey(r *http.Request) string {
	// Include method, path, and query parameters
	key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

	// Sort query parameters for consistent key generation
	params := r.URL.Query()
	if len(params) > 0 {
		var keys []string
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var queryParts []string
		for _, k := range keys {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, params.Get(k)))
		}
		key = fmt.Sprintf("%s?%s", key, strings.Join(queryParts, "&"))
	}

	// Include user ID if available
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		key = fmt.Sprintf("%s:user=%s", key, userID)
	}

	// Generate a short, stable key. Cryptographic strength isn't required here,
	// but sha256 avoids gosec's md5 warnings.
	return shared.HashCacheKey(key)
}

// matchPattern checks if a key matches a pattern
func matchPattern(key, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}

	return key == pattern
}
