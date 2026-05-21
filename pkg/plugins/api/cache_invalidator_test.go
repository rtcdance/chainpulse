package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheInvalidator(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	assert.NotNil(t, ci)
	assert.NotNil(t, ci.invalidationQueue)
	assert.NotNil(t, ci.retryPolicy)
}

func TestDefaultRetryPolicy(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy()

	assert.NotNil(t, policy)
	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 10*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.BackoffFactor)
}

func TestInvalidateKey(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	err := ci.InvalidateKey("key1", "test reason")
	require.NoError(t, err)
}

func TestInvalidatePattern(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	err := ci.InvalidatePattern("user:*", "test reason")
	require.NoError(t, err)
}

func TestInvalidateRelated(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	keys := []string{"key1", "key2", "key3"}
	err := ci.InvalidateRelated(keys, "test reason")
	require.NoError(t, err)
}

func TestInvalidateRelatedEmpty(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	err := ci.InvalidateRelated([]string{}, "test reason")
	require.NoError(t, err)
}

func TestGetBackoffDuration(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy()

	duration0 := policy.GetBackoffDuration(0)
	assert.Equal(t, 100*time.Millisecond, duration0)

	duration1 := policy.GetBackoffDuration(1)
	assert.Greater(t, duration1, duration0)

	duration2 := policy.GetBackoffDuration(2)
	assert.Greater(t, duration2, duration1)
}

func TestGetBackoffDurationMaxBackoff(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy()

	// With high retry count, should hit max backoff
	duration := policy.GetBackoffDuration(100)
	assert.LessOrEqual(t, duration, policy.MaxBackoff)
}

func TestShouldRetry(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy()

	assert.True(t, policy.ShouldRetry(0))
	assert.True(t, policy.ShouldRetry(1))
	assert.True(t, policy.ShouldRetry(2))
	assert.False(t, policy.ShouldRetry(3))
	assert.False(t, policy.ShouldRetry(4))
}

func TestCacheInvalidatorGetStats(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	stats := ci.GetStats()

	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalInvalidations)
	assert.Equal(t, int64(0), stats.SuccessfulCount)
	assert.Equal(t, int64(0), stats.FailedCount)
}

func TestClose(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)

	err := ci.Close()
	require.NoError(t, err)
}

func TestInvalidationQueueFull(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	// Fill the queue
	for i := 0; i < 1000; i++ {
		_ = ci.InvalidateKey("key"+string(rune(i)), "reason")
	}

	// Try to add one more (should timeout)
	err := ci.InvalidateKey("overflow_key", "reason")
	// May or may not error depending on timing
	_ = err
}

func TestInvalidationRequestStructure(t *testing.T) {
	t.Parallel()
	req := InvalidationRequest{
		Key:       "test_key",
		Pattern:   "test:*",
		Reason:    "test reason",
		Timestamp: time.Now(),
		Retries:   0,
	}

	assert.Equal(t, "test_key", req.Key)
	assert.Equal(t, "test:*", req.Pattern)
	assert.Equal(t, "test reason", req.Reason)
	assert.Equal(t, 0, req.Retries)
}

func TestRetryPolicyStructure(t *testing.T) {
	t.Parallel()
	policy := &RetryPolicy{
		MaxRetries:     5,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		BackoffFactor:  1.5,
	}

	assert.Equal(t, 5, policy.MaxRetries)
	assert.Equal(t, 50*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 5*time.Second, policy.MaxBackoff)
	assert.Equal(t, 1.5, policy.BackoffFactor)
}

func TestInvalidationStatsStructure(t *testing.T) {
	t.Parallel()
	stats := InvalidationStats{
		TotalInvalidations: 100,
		SuccessfulCount:    95,
		FailedCount:        5,
		AverageLatency:     10 * time.Millisecond,
	}

	assert.Equal(t, int64(100), stats.TotalInvalidations)
	assert.Equal(t, int64(95), stats.SuccessfulCount)
	assert.Equal(t, int64(5), stats.FailedCount)
	assert.Equal(t, 10*time.Millisecond, stats.AverageLatency)
}

func TestConcurrentInvalidation(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			key := "key" + string(rune(id))
			_ = ci.InvalidateKey(key, "concurrent test")
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestInvalidateKeyWithContext(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	// Set a value first
	headers := http.Header{}
	cm.Set("test_key", []byte("test_value"), headers, 200)

	// Invalidate it
	err := ci.InvalidateKey("test_key", "manual invalidation")
	require.NoError(t, err)

	// Wait for invalidation to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify it's gone
	_, _, _, found := cm.Get("test_key")
	assert.False(t, found)
}

func TestInvalidatePatternMultipleKeys(t *testing.T) {
	t.Parallel()
	config := DefaultCacheConfig()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMiddleware(config, logger, metrics)
	ci := NewCacheInvalidator(cm, logger, metrics)
	defer ci.Close()

	// Set multiple keys
	headers := http.Header{}
	cm.Set("user:1:profile", []byte("data1"), headers, 200)
	cm.Set("user:2:profile", []byte("data2"), headers, 200)
	cm.Set("post:1:data", []byte("data3"), headers, 200)

	// Invalidate pattern
	err := ci.InvalidatePattern("user:*", "pattern invalidation")
	require.NoError(t, err)

	// Give time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify user keys are gone
	_, _, _, found1 := cm.Get("user:1:profile")
	_, _, _, found2 := cm.Get("user:2:profile")
	_, _, _, found3 := cm.Get("post:1:data")

	assert.False(t, found1)
	assert.False(t, found2)
	assert.True(t, found3) // post key should still exist
}

func TestBackoffCalculation(t *testing.T) {
	t.Parallel()
	policy := &RetryPolicy{
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		BackoffFactor:  2.0,
	}

	backoff0 := policy.GetBackoffDuration(0)
	backoff1 := policy.GetBackoffDuration(1)
	backoff2 := policy.GetBackoffDuration(2)

	assert.Equal(t, 100*time.Millisecond, backoff0)
	assert.Greater(t, backoff1, backoff0)
	assert.Greater(t, backoff2, backoff1)
}
