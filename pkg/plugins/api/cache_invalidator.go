package api

import (
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// CacheInvalidator handles cache invalidation logic
type CacheInvalidator struct {
	cache             *CacheMiddleware
	logger            core.Logger
	metrics           core.MetricsCollector
	invalidationQueue chan InvalidationRequest
	retryPolicy       *RetryPolicy
	mu                sync.RWMutex
	closeOnce         sync.Once
}

// InvalidationRequest represents a cache invalidation request
type InvalidationRequest struct {
	Key       string
	Pattern   string
	Reason    string
	Timestamp time.Time
	Retries   int
}

// RetryPolicy defines retry behavior for failed invalidations
type RetryPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryPolicy returns default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
	}
}

// NewCacheInvalidator creates a new cache invalidator
func NewCacheInvalidator(cache *CacheMiddleware, logger core.Logger, metrics core.MetricsCollector) *CacheInvalidator {
	ci := &CacheInvalidator{
		cache:             cache,
		logger:            logger,
		metrics:           metrics,
		invalidationQueue: make(chan InvalidationRequest, 1000),
		retryPolicy:       DefaultRetryPolicy(),
	}

	// Start invalidation worker
	go ci.processInvalidations()

	return ci
}

// InvalidateKey invalidates a specific cache key
func (ci *CacheInvalidator) InvalidateKey(key string, reason string) error {
	req := InvalidationRequest{
		Key:       key,
		Reason:    reason,
		Timestamp: time.Now(),
		Retries:   0,
	}

	select {
	case ci.invalidationQueue <- req:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("invalidation queue full")
	}
}

// InvalidatePattern invalidates all keys matching a pattern
func (ci *CacheInvalidator) InvalidatePattern(pattern string, reason string) error {
	req := InvalidationRequest{
		Pattern:   pattern,
		Reason:    reason,
		Timestamp: time.Now(),
		Retries:   0,
	}

	select {
	case ci.invalidationQueue <- req:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("invalidation queue full")
	}
}

// InvalidateRelated invalidates related cache entries
func (ci *CacheInvalidator) InvalidateRelated(keys []string, reason string) error {
	for _, key := range keys {
		if err := ci.InvalidateKey(key, reason); err != nil {
			ci.logger.Error(
				"Failed to invalidate related key",
				"key", key,
				"reason", reason,
				"error", err,
			)
			return err
		}
	}
	return nil
}

// processInvalidations processes invalidation requests
func (ci *CacheInvalidator) processInvalidations() {
	for req := range ci.invalidationQueue {
		ci.processInvalidation(req)
	}
}

// processInvalidation processes a single invalidation request
func (ci *CacheInvalidator) processInvalidation(req InvalidationRequest) {
	if req.Key != "" {
		ci.cache.Invalidate(req.Key)
		ci.logger.Info(
			"Cache key invalidated",
			"key", req.Key,
			"reason", req.Reason,
		)
		ci.metrics.RecordCounter("cache_invalidation_key", 1, map[string]string{
			"reason": req.Reason,
		})
	} else if req.Pattern != "" {
		ci.cache.InvalidatePattern(req.Pattern)
		ci.logger.Info(
			"Cache pattern invalidated",
			"pattern", req.Pattern,
			"reason", req.Reason,
		)
		ci.metrics.RecordCounter("cache_invalidation_pattern", 1, map[string]string{
			"pattern": req.Pattern,
			"reason":  req.Reason,
		})
	}
}

// GetBackoffDuration calculates backoff duration for retry
func (rp *RetryPolicy) GetBackoffDuration(retries int) time.Duration {
	if retries <= 0 {
		return rp.InitialBackoff
	}

	backoff := time.Duration(float64(rp.InitialBackoff) * (rp.BackoffFactor * float64(retries)))
	if backoff > rp.MaxBackoff {
		backoff = rp.MaxBackoff
	}

	return backoff
}

// ShouldRetry checks if a request should be retried
func (rp *RetryPolicy) ShouldRetry(retries int) bool {
	return retries < rp.MaxRetries
}

// InvalidationStats represents invalidation statistics
type InvalidationStats struct {
	TotalInvalidations int64
	SuccessfulCount    int64
	FailedCount        int64
	AverageLatency     time.Duration
}

// GetStats returns invalidation statistics
func (ci *CacheInvalidator) GetStats() InvalidationStats {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	return InvalidationStats{
		TotalInvalidations: 0,
		SuccessfulCount:    0,
		FailedCount:        0,
		AverageLatency:     0,
	}
}

// Close closes the invalidator
func (ci *CacheInvalidator) Close() error {
	ci.closeOnce.Do(func() {
		close(ci.invalidationQueue)
	})
	return nil
}
