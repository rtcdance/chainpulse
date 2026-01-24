package api

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 10000, config.MaxSize)
	assert.Equal(t, 5*time.Minute, config.DefaultTTL)
	assert.Equal(t, 1*time.Minute, config.CleanupInterval)
	assert.Equal(t, "LRU", config.EvictionStrategy)
	assert.False(t, config.WarmingEnabled)
	assert.Equal(t, 10*time.Minute, config.WarmingInterval)
	assert.Equal(t, 100, config.WarmingBatchSize)
	assert.False(t, config.EncryptionEnabled)
	assert.False(t, config.CompressionEnabled)
}

func TestLoadCacheConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("CACHE_ENABLED", "false")
	_ = os.Setenv("CACHE_MAX_SIZE", "5000")
	_ = os.Setenv("CACHE_DEFAULT_TTL", "10m")
	_ = os.Setenv("CACHE_CLEANUP_INTERVAL", "2m")
	_ = os.Setenv("CACHE_EVICTION_STRATEGY", "LFU")
	_ = os.Setenv("CACHE_WARMING_ENABLED", "true")
	_ = os.Setenv("CACHE_WARMING_INTERVAL", "5m")
	_ = os.Setenv("CACHE_WARMING_BATCH_SIZE", "200")
	_ = os.Setenv("CACHE_ENCRYPTION_ENABLED", "true")
	_ = os.Setenv("CACHE_COMPRESSION_ENABLED", "true")

	defer func() {
		_ = os.Unsetenv("CACHE_ENABLED")
		_ = os.Unsetenv("CACHE_MAX_SIZE")
		_ = os.Unsetenv("CACHE_DEFAULT_TTL")
		_ = os.Unsetenv("CACHE_CLEANUP_INTERVAL")
		_ = os.Unsetenv("CACHE_EVICTION_STRATEGY")
		_ = os.Unsetenv("CACHE_WARMING_ENABLED")
		_ = os.Unsetenv("CACHE_WARMING_INTERVAL")
		_ = os.Unsetenv("CACHE_WARMING_BATCH_SIZE")
		_ = os.Unsetenv("CACHE_ENCRYPTION_ENABLED")
		_ = os.Unsetenv("CACHE_COMPRESSION_ENABLED")
	}()

	config, err := LoadCacheConfig()
	require.NoError(t, err)

	assert.False(t, config.Enabled)
	assert.Equal(t, 5000, config.MaxSize)
	assert.Equal(t, 10*time.Minute, config.DefaultTTL)
	assert.Equal(t, 2*time.Minute, config.CleanupInterval)
	assert.Equal(t, "LFU", config.EvictionStrategy)
	assert.True(t, config.WarmingEnabled)
	assert.Equal(t, 5*time.Minute, config.WarmingInterval)
	assert.Equal(t, 200, config.WarmingBatchSize)
	assert.True(t, config.EncryptionEnabled)
	assert.True(t, config.CompressionEnabled)
}

func TestLoadCacheConfigInvalidMaxSize(t *testing.T) {
	_ = os.Setenv("CACHE_MAX_SIZE", "invalid")
	defer func() { _ = os.Unsetenv("CACHE_MAX_SIZE") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CACHE_MAX_SIZE")
}

func TestLoadCacheConfigNegativeMaxSize(t *testing.T) {
	_ = os.Setenv("CACHE_MAX_SIZE", "-100")
	defer func() { _ = os.Unsetenv("CACHE_MAX_SIZE") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestLoadCacheConfigInvalidTTL(t *testing.T) {
	_ = os.Setenv("CACHE_DEFAULT_TTL", "invalid")
	defer func() { _ = os.Unsetenv("CACHE_DEFAULT_TTL") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CACHE_DEFAULT_TTL")
}

func TestLoadCacheConfigNegativeTTL(t *testing.T) {
	_ = os.Setenv("CACHE_DEFAULT_TTL", "-5m")
	defer func() { _ = os.Unsetenv("CACHE_DEFAULT_TTL") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestLoadCacheConfigInvalidEvictionStrategy(t *testing.T) {
	_ = os.Setenv("CACHE_EVICTION_STRATEGY", "INVALID")
	defer func() { _ = os.Unsetenv("CACHE_EVICTION_STRATEGY") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CACHE_EVICTION_STRATEGY")
}

func TestValidateCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()
	err := config.Validate()
	assert.NoError(t, err)
}

func TestValidateCacheConfigInvalidMaxSize(t *testing.T) {
	config := DefaultCacheConfig()
	config.MaxSize = 0

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxSize must be positive")
}

func TestValidateCacheConfigInvalidTTL(t *testing.T) {
	config := DefaultCacheConfig()
	config.DefaultTTL = 0

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DefaultTTL must be positive")
}

func TestValidateCacheConfigInvalidCleanupInterval(t *testing.T) {
	config := DefaultCacheConfig()
	config.CleanupInterval = 0

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CleanupInterval must be positive")
}

func TestValidateCacheConfigInvalidEvictionStrategy(t *testing.T) {
	config := DefaultCacheConfig()
	config.EvictionStrategy = "INVALID"

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid EvictionStrategy")
}

func TestValidateCacheConfigWarmingWithoutInterval(t *testing.T) {
	config := DefaultCacheConfig()
	config.WarmingEnabled = true
	config.WarmingInterval = 0

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WarmingInterval must be positive")
}

func TestValidateCacheConfigWarmingWithoutBatchSize(t *testing.T) {
	config := DefaultCacheConfig()
	config.WarmingEnabled = true
	config.WarmingBatchSize = 0

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WarmingBatchSize must be positive")
}

func TestIsValidEvictionStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		valid    bool
	}{
		{"LRU", true},
		{"LFU", true},
		{"FIFO", true},
		{"INVALID", false},
		{"", false},
		{"lru", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			result := isValidEvictionStrategy(tt.strategy)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestLoadCacheConfigInvalidBool(t *testing.T) {
	_ = os.Setenv("CACHE_ENABLED", "maybe")
	defer func() { _ = os.Unsetenv("CACHE_ENABLED") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CACHE_ENABLED")
}

func TestLoadCacheConfigInvalidCleanupInterval(t *testing.T) {
	_ = os.Setenv("CACHE_CLEANUP_INTERVAL", "invalid")
	defer func() { _ = os.Unsetenv("CACHE_CLEANUP_INTERVAL") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CACHE_CLEANUP_INTERVAL")
}

func TestLoadCacheConfigInvalidWarmingBatchSize(t *testing.T) {
	_ = os.Setenv("CACHE_WARMING_BATCH_SIZE", "invalid")
	defer func() { _ = os.Unsetenv("CACHE_WARMING_BATCH_SIZE") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CACHE_WARMING_BATCH_SIZE")
}

func TestLoadCacheConfigNegativeWarmingBatchSize(t *testing.T) {
	_ = os.Setenv("CACHE_WARMING_BATCH_SIZE", "-50")
	defer func() { _ = os.Unsetenv("CACHE_WARMING_BATCH_SIZE") }()

	_, err := LoadCacheConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestCacheConfigAllEvictionStrategies(t *testing.T) {
	strategies := []string{"LRU", "LFU", "FIFO"}

	for _, strategy := range strategies {
		config := DefaultCacheConfig()
		config.EvictionStrategy = strategy

		err := config.Validate()
		assert.NoError(t, err, "strategy %s should be valid", strategy)
	}
}
