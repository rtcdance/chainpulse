package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// CacheWarmer pre-populates cache with common queries
type CacheWarmer struct {
	cache              *CacheMiddleware
	config             *CacheConfig
	logger             core.Logger
	metrics            core.MetricsCollector
	dataProvider       DataProvider
	ticker             *time.Ticker
	done               chan struct{}
	isRunning          bool
	lastWarmingTime    time.Time
	warmingCount       int64
	failedWarmingCount int64
	mu                 sync.RWMutex
}

// DataProvider provides data for cache warming
type DataProvider interface {
	GetWarmingData(ctx context.Context, batchSize int) ([]WarmingData, error)
}

// WarmingData represents data to be cached
type WarmingData struct {
	Key        string
	Value      []byte
	StatusCode int
	TTL        time.Duration
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *CacheMiddleware, config *CacheConfig, logger core.Logger, metrics core.MetricsCollector) *CacheWarmer {
	return &CacheWarmer{
		cache:              cache,
		config:             config,
		logger:             logger,
		metrics:            metrics,
		done:               make(chan struct{}),
		isRunning:          false,
		lastWarmingTime:    time.Now(),
		warmingCount:       0,
		failedWarmingCount: 0,
	}
}

// SetDataProvider sets the data provider for cache warming
func (cw *CacheWarmer) SetDataProvider(provider DataProvider) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	cw.dataProvider = provider
}

// Start starts the cache warmer
func (cw *CacheWarmer) Start(ctx context.Context) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.isRunning {
		return fmt.Errorf("cache warmer already running")
	}

	if !cw.config.WarmingEnabled {
		return fmt.Errorf("cache warming is disabled")
	}

	if cw.dataProvider == nil {
		return fmt.Errorf("data provider not set")
	}

	cw.isRunning = true
	cw.ticker = time.NewTicker(cw.config.WarmingInterval)

	go cw.run(ctx)

	cw.logger.Info("Cache warmer started",
		"interval", cw.config.WarmingInterval,
		"batchSize", cw.config.WarmingBatchSize,
	)

	return nil
}

// run runs the cache warming loop
func (cw *CacheWarmer) run(ctx context.Context) {
	// Perform initial warming
	cw.warm(ctx)

	for {
		select {
		case <-cw.ticker.C:
			cw.warm(ctx)
		case <-cw.done:
			cw.mu.Lock()
			cw.isRunning = false
			cw.mu.Unlock()
			return
		case <-ctx.Done():
			cw.mu.Lock()
			cw.isRunning = false
			cw.mu.Unlock()
			return
		}
	}
}

// warm performs cache warming
func (cw *CacheWarmer) warm(ctx context.Context) {
	cw.mu.Lock()
	provider := cw.dataProvider
	cw.mu.Unlock()

	if provider == nil {
		return
	}

	start := time.Now()

	// Get warming data
	data, err := provider.GetWarmingData(ctx, cw.config.WarmingBatchSize)
	if err != nil {
		cw.mu.Lock()
		cw.failedWarmingCount++
		cw.mu.Unlock()

		cw.logger.Error("Failed to get warming data",
			"error", err,
		)
		cw.metrics.RecordCounter("cache_warming_failed", 1, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Cache the data
	successCount := 0
	for _, item := range data {
		cw.cache.Set(item.Key, item.Value, nil, item.StatusCode)
		successCount++
	}

	cw.mu.Lock()
	cw.warmingCount++
	cw.lastWarmingTime = time.Now()
	cw.mu.Unlock()

	duration := time.Since(start)

	cw.logger.Info("Cache warming completed",
		"itemsWarmed", successCount,
		"duration", duration,
	)

	cw.metrics.RecordCounter("cache_warming_completed", 1, map[string]string{
		"items": fmt.Sprintf("%d", successCount),
	})
	cw.metrics.RecordHistogram("cache_warming_duration_ms", float64(duration.Milliseconds()), map[string]string{})
}

// Stop stops the cache warmer
func (cw *CacheWarmer) Stop() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.isRunning {
		return fmt.Errorf("cache warmer not running")
	}

	cw.isRunning = false
	cw.ticker.Stop()
	close(cw.done)

	cw.logger.Info("Cache warmer stopped")

	return nil
}

// IsRunning returns whether the cache warmer is running
func (cw *CacheWarmer) IsRunning() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	return cw.isRunning
}

// GetStats returns cache warmer statistics
func (cw *CacheWarmer) GetStats() map[string]interface{} {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	return map[string]interface{}{
		"is_running":           cw.isRunning,
		"warming_count":        cw.warmingCount,
		"failed_warming_count": cw.failedWarmingCount,
		"last_warming_time":    cw.lastWarmingTime,
		"warming_interval":     cw.config.WarmingInterval,
		"warming_batch_size":   cw.config.WarmingBatchSize,
	}
}
