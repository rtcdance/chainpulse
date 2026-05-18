package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCacheMiddleware(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.metrics)
	assert.NotNil(t, cm.invalidator)
}

func TestCacheEntryIsExpired(t *testing.T) {
	t.Parallel()
	entry := &CacheEntry{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	assert.True(t, entry.IsExpired())
}

func TestCacheEntryNotExpired(t *testing.T) {
	t.Parallel()
	entry := &CacheEntry{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	assert.False(t, entry.IsExpired())
}

func TestCacheGet(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	// Set a value
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)

	// Get the value
	value, retrievedHeaders, statusCode, found := cm.Get("key1")

	assert.True(t, found)
	assert.Equal(t, []byte("value1"), value)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.NotNil(t, retrievedHeaders)
}

func TestCacheGetNotFound(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	value, _, _, found := cm.Get("nonexistent")

	assert.False(t, found)
	assert.Nil(t, value)
}

func TestCacheGetExpired(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	config.DefaultTTL = 1 * time.Millisecond
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)

	time.Sleep(2 * time.Millisecond)

	_, _, _, found := cm.Get("key1")

	assert.False(t, found)
}

func TestCacheSet(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	cm.Set("key1", []byte("value1"), headers, http.StatusOK)

	assert.Equal(t, 1, len(cm.cache))
}

func TestCacheInvalidate(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)

	assert.Equal(t, 1, len(cm.cache))

	cm.Invalidate("key1")

	assert.Equal(t, 0, len(cm.cache))
}

func TestCacheInvalidatePattern(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("user:1:profile", []byte("data1"), headers, http.StatusOK)
	cm.Set("user:2:profile", []byte("data2"), headers, http.StatusOK)
	cm.Set("post:1:data", []byte("data3"), headers, http.StatusOK)

	assert.Equal(t, 3, len(cm.cache))

	cm.InvalidatePattern("user:*")

	assert.Equal(t, 1, len(cm.cache))
}

func TestCacheClear(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)
	cm.Set("key2", []byte("value2"), headers, http.StatusOK)

	assert.Equal(t, 2, len(cm.cache))

	cm.Clear()

	assert.Equal(t, 0, len(cm.cache))
}

func TestCacheGetStats(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)

	stats := cm.GetStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "size")
	assert.Contains(t, stats, "max_size")
	assert.Contains(t, stats, "metrics")
}

func TestLRUEvictionStrategy(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	config.MaxSize = 2
	config.EvictionStrategy = "LRU"
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)
	cm.Set("key2", []byte("value2"), headers, http.StatusOK)

	// Access key1 to make it more recently used
	cm.Get("key1")

	// Add key3, should evict key2 (least recently used)
	cm.Set("key3", []byte("value3"), headers, http.StatusOK)

	_, _, _, found := cm.Get("key1")
	assert.True(t, found)

	_, _, _, found = cm.Get("key3")
	assert.True(t, found)

	_, _, _, found = cm.Get("key2")
	assert.False(t, found)
}

func TestLFUEvictionStrategy(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	config.MaxSize = 2
	config.EvictionStrategy = "LFU"
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)
	cm.Set("key2", []byte("value2"), headers, http.StatusOK)

	// Access key1 multiple times
	cm.Get("key1")
	cm.Get("key1")

	// Add key3, should evict key2 (least frequently used)
	cm.Set("key3", []byte("value3"), headers, http.StatusOK)

	_, _, _, found := cm.Get("key1")
	assert.True(t, found)

	_, _, _, found = cm.Get("key3")
	assert.True(t, found)

	_, _, _, found = cm.Get("key2")
	assert.False(t, found)
}

func TestFIFOEvictionStrategy(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	config.MaxSize = 2
	config.EvictionStrategy = "FIFO"
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)
	cm.Set("key2", []byte("value2"), headers, http.StatusOK)

	// Add key3, should evict key1 (oldest)
	cm.Set("key3", []byte("value3"), headers, http.StatusOK)

	_, _, _, found := cm.Get("key1")
	assert.False(t, found)

	_, _, _, found = cm.Get("key2")
	assert.True(t, found)

	_, _, _, found = cm.Get("key3")
	assert.True(t, found)
}

func TestMiddlewareGetRequest(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Wrap with cache middleware
	cachedHandler := cm.Middleware(handler)

	// First request
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	w1 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))

	// Second request (should be cached)
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	w2 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
}

func TestMiddlewarePostRequest(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cachedHandler := cm.Middleware(handler)

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()
	cachedHandler.ServeHTTP(w, req)

	// POST requests should not be cached
	assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
}

func TestGenerateCacheKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/test?param1=value1&param2=value2", nil)
	req.Header.Set("X-User-ID", "user123")

	key := generateCacheKey(req)

	assert.NotEmpty(t, key)
	assert.Len(t, key, 32) // MD5 hash length
}

func TestMatchPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		key     string
		pattern string
		match   bool
	}{
		{"user:1:profile", "user:*", true},
		{"user:1:profile", "post:*", false},
		{"user:1:profile", "*", true},
		{"user:1:profile", "user:1:profile", true},
		{"user:1:profile", "user:2:profile", false},
	}

	for _, tt := range tests {
		t.Run(tt.key+":"+tt.pattern, func(t *testing.T) {
			result := matchPattern(tt.key, tt.pattern)
			assert.Equal(t, tt.match, result)
		})
	}
}

func TestCacheMetricsRecording(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	cm.Set("key1", []byte("value1"), headers, http.StatusOK)

	// Cache hit
	cm.Get("key1")

	// Cache miss
	cm.Get("nonexistent")

	assert.Greater(t, metrics.counters["cache_hit"], int64(0))
	assert.Greater(t, metrics.counters["cache_miss"], int64(0))
}

func TestConcurrentCacheAccess(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			headers := http.Header{}
			key := "key" + string(rune(id))
			cm.Set(key, []byte("value"), headers, http.StatusOK)
			cm.Get(key)
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, len(cm.cache))
}

func TestResponseRecorder(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	rr := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           new(bytes.Buffer),
	}

	rr.WriteHeader(http.StatusOK)
	_, _ = rr.Write([]byte("test data"))

	assert.Equal(t, http.StatusOK, rr.statusCode)
	assert.Equal(t, "test data", rr.body.String())
}

func TestCacheMaxSizeEnforcement(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	config.MaxSize = 3
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	defer cm.Close()

	headers := http.Header{}
	for i := 0; i < 5; i++ {
		key := "key" + string(rune(i))
		cm.Set(key, []byte("value"), headers, http.StatusOK)
	}

	assert.LessOrEqual(t, len(cm.cache), 3)
}
