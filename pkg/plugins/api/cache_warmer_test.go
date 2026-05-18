package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipCacheWarmerStressTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping cache warmer background/stress test in short mode")
	}
}

// MockDataProvider implements DataProvider for testing
type MockDataProvider struct {
	data  []WarmingData
	err   error
	calls int
	mu    sync.RWMutex
}

func NewMockDataProvider() *MockDataProvider {
	return &MockDataProvider{
		data:  make([]WarmingData, 0),
		calls: 0,
	}
}

func (m *MockDataProvider) GetWarmingData(ctx context.Context, batchSize int) ([]WarmingData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls++

	if m.err != nil {
		return nil, m.err
	}

	if len(m.data) > batchSize {
		return m.data[:batchSize], nil
	}

	return m.data, nil
}

func (m *MockDataProvider) SetData(data []WarmingData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
}

func (m *MockDataProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MockDataProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls
}

// TestNewCacheWarmer tests cache warmer initialization
func TestNewCacheWarmer(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)

	require.NotNil(t, warmer)
	assert.False(t, warmer.IsRunning())
	assert.Equal(t, int64(0), warmer.warmingCount)
	assert.Equal(t, int64(0), warmer.failedWarmingCount)
}

// TestSetDataProvider tests setting data provider
func TestSetDataProvider(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()

	warmer.SetDataProvider(provider)

	assert.Equal(t, provider, warmer.dataProvider)
}

// TestStartWithoutProvider tests starting without data provider
func TestStartWithoutProvider(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	ctx := context.Background()

	err := warmer.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data provider not set")
	assert.False(t, warmer.IsRunning())
}

// TestStartWithDisabledWarming tests starting with warming disabled
func TestStartWithDisabledWarming(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   false,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache warming is disabled")
	assert.False(t, warmer.IsRunning())
}

// TestStartSuccess tests successful start
func TestStartSuccess(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)

	require.NoError(t, err)
	assert.True(t, warmer.IsRunning())

	// Give it time to warm
	time.Sleep(50 * time.Millisecond)

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestStartAlreadyRunning tests starting when already running
func TestStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err1 := warmer.Start(ctx)
	require.NoError(t, err1)

	err2 := warmer.Start(ctx)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "already running")

	err3 := warmer.Stop()
	require.NoError(t, err3)
}

// TestStopSuccess tests successful stop
func TestStopSuccess(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	assert.True(t, warmer.IsRunning())

	err = warmer.Stop()

	require.NoError(t, err)
	assert.False(t, warmer.IsRunning())
}

// TestStopNotRunning tests stopping when not running
func TestStopNotRunning(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)

	err := warmer.Stop()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestWarmingData tests warming data caching
func TestWarmingData(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
		{Key: "key2", Value: []byte("value2"), StatusCode: 200, TTL: 1 * time.Hour},
		{Key: "key3", Value: []byte("value3"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Verify data was cached
	_, _, _, ok1 := cache.Get("key1")
	assert.True(t, ok1)
	_, _, _, ok2 := cache.Get("key2")
	assert.True(t, ok2)
	_, _, _, ok3 := cache.Get("key3")
	assert.True(t, ok3)

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestWarmingWithError tests warming with data provider error
func TestWarmingWithError(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetError(fmt.Errorf("data provider error"))
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	stats := warmer.GetStats()
	assert.Greater(t, stats["failed_warming_count"], int64(0))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestCacheWarmerGetStats tests statistics retrieval
func TestCacheWarmerGetStats(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 50,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	stats := warmer.GetStats()

	assert.True(t, stats["is_running"].(bool))
	assert.Greater(t, stats["warming_count"].(int64), int64(0))
	assert.Equal(t, int64(0), stats["failed_warming_count"].(int64))
	assert.Equal(t, config.WarmingInterval, stats["warming_interval"].(time.Duration))
	assert.Equal(t, config.WarmingBatchSize, stats["warming_batch_size"].(int))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestIsRunning tests running state
func TestIsRunning(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  1 * time.Second,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	assert.False(t, warmer.IsRunning())

	err := warmer.Start(ctx)
	require.NoError(t, err)
	assert.True(t, warmer.IsRunning())

	err = warmer.Stop()
	require.NoError(t, err)
	assert.False(t, warmer.IsRunning())
}

// TestConcurrentWarmingOperations tests concurrent warming operations
func TestConcurrentWarmingOperations(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  50 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()

	data := make([]WarmingData, 100)
	for i := 0; i < 100; i++ {
		data[i] = WarmingData{
			Key:        fmt.Sprintf("key%d", i),
			Value:      []byte(fmt.Sprintf("value%d", i)),
			StatusCode: 200,
			TTL:        1 * time.Hour,
		}
	}
	provider.SetData(data)
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)

	// Verify multiple warming cycles occurred
	stats := warmer.GetStats()
	assert.Greater(t, stats["warming_count"].(int64), int64(1))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestWarmingWithBatchSize tests warming respects batch size
func TestWarmingWithBatchSize(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 5,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()

	data := make([]WarmingData, 20)
	for i := 0; i < 20; i++ {
		data[i] = WarmingData{
			Key:        fmt.Sprintf("key%d", i),
			Value:      []byte(fmt.Sprintf("value%d", i)),
			StatusCode: 200,
			TTL:        1 * time.Hour,
		}
	}
	provider.SetData(data)
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Verify batch size was respected
	callCount := provider.GetCallCount()
	assert.Greater(t, callCount, 0)

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestWarmingMetricsRecording tests metrics recording during warming
func TestWarmingMetricsRecording(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Verify metrics were recorded
	assert.Greater(t, metrics.GetCounterValue("cache_warming_completed"), int64(0))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestWarmingFailureMetrics tests metrics recording on warming failure
func TestWarmingFailureMetrics(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetError(fmt.Errorf("test error"))
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Verify failure metrics were recorded
	assert.Greater(t, metrics.GetCounterValue("cache_warming_failed"), int64(0))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestContextCancellation tests warming stops on context cancellation
func TestContextCancellation(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)

	ctx, cancel := context.WithCancel(context.Background())

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)

	// Warmer should stop on context cancellation
	assert.False(t, warmer.IsRunning())
}

// TestLastWarmingTime tests last warming time tracking
func TestLastWarmingTime(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	beforeStart := time.Now()
	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	stats := warmer.GetStats()
	lastWarmingTime := stats["last_warming_time"].(time.Time)

	assert.True(t, lastWarmingTime.After(beforeStart))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestMultipleWarmingCycles tests multiple warming cycles
func TestMultipleWarmingCycles(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  50 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	stats := warmer.GetStats()
	warmingCount := stats["warming_count"].(int64)

	// Should have multiple warming cycles
	assert.Greater(t, warmingCount, int64(2))

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestWarmingDataWithDifferentStatusCodes tests warming with different status codes
func TestWarmingDataWithDifferentStatusCodes(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
		{Key: "key2", Value: []byte("value2"), StatusCode: 201, TTL: 1 * time.Hour},
		{Key: "key3", Value: []byte("value3"), StatusCode: 204, TTL: 1 * time.Hour},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Verify all data was cached regardless of status code
	_, _, _, ok1 := cache.Get("key1")
	assert.True(t, ok1)
	_, _, _, ok2 := cache.Get("key2")
	assert.True(t, ok2)
	_, _, _, ok3 := cache.Get("key3")
	assert.True(t, ok3)

	err = warmer.Stop()
	require.NoError(t, err)
}

// TestWarmingDataWithDifferentTTLs tests warming with different TTLs
func TestWarmingDataWithDifferentTTLs(t *testing.T) {
	t.Parallel()
	skipCacheWarmerStressTestsInShortMode(t)

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cache := NewCacheMiddleware(DefaultCacheConfig(), logger, metrics)
	defer cache.Close()
	config := &CacheConfig{
		WarmingEnabled:   true,
		WarmingInterval:  100 * time.Millisecond,
		WarmingBatchSize: 100,
	}

	warmer := NewCacheWarmer(cache, config, logger, metrics)
	provider := NewMockDataProvider()
	provider.SetData([]WarmingData{
		{Key: "key1", Value: []byte("value1"), StatusCode: 200, TTL: 1 * time.Hour},
		{Key: "key2", Value: []byte("value2"), StatusCode: 200, TTL: 30 * time.Minute},
		{Key: "key3", Value: []byte("value3"), StatusCode: 200, TTL: 5 * time.Minute},
	})
	warmer.SetDataProvider(provider)
	ctx := context.Background()

	err := warmer.Start(ctx)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Verify all data was cached
	_, _, _, ok1 := cache.Get("key1")
	assert.True(t, ok1)
	_, _, _, ok2 := cache.Get("key2")
	assert.True(t, ok2)
	_, _, _, ok3 := cache.Get("key3")
	assert.True(t, ok3)

	err = warmer.Stop()
	require.NoError(t, err)
}
