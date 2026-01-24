package api

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// CacheConfig holds cache configuration
type CacheConfig struct {
	// Cache settings
	Enabled            bool
	MaxSize            int           // Maximum number of entries
	DefaultTTL         time.Duration // Default time-to-live
	CleanupInterval    time.Duration // Interval for cleanup
	EvictionStrategy   string        // LRU, LFU, FIFO
	WarmingEnabled     bool
	WarmingInterval    time.Duration
	WarmingBatchSize   int
	EncryptionEnabled  bool
	CompressionEnabled bool
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:            true,
		MaxSize:            10000,
		DefaultTTL:         5 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		EvictionStrategy:   "LRU",
		WarmingEnabled:     false,
		WarmingInterval:    10 * time.Minute,
		WarmingBatchSize:   100,
		EncryptionEnabled:  false,
		CompressionEnabled: false,
	}
}

// LoadCacheConfig loads cache configuration from environment
func LoadCacheConfig() (*CacheConfig, error) {
	config := DefaultCacheConfig()

	// Load from environment variables
	if enabled := os.Getenv("CACHE_ENABLED"); enabled != "" {
		b, err := strconv.ParseBool(enabled)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_ENABLED: %w", err)
		}
		config.Enabled = b
	}

	if maxSize := os.Getenv("CACHE_MAX_SIZE"); maxSize != "" {
		size, err := strconv.Atoi(maxSize)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_MAX_SIZE: %w", err)
		}
		if size <= 0 {
			return nil, fmt.Errorf("CACHE_MAX_SIZE must be positive")
		}
		config.MaxSize = size
	}

	if ttl := os.Getenv("CACHE_DEFAULT_TTL"); ttl != "" {
		duration, err := time.ParseDuration(ttl)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_DEFAULT_TTL: %w", err)
		}
		if duration <= 0 {
			return nil, fmt.Errorf("CACHE_DEFAULT_TTL must be positive")
		}
		config.DefaultTTL = duration
	}

	if interval := os.Getenv("CACHE_CLEANUP_INTERVAL"); interval != "" {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_CLEANUP_INTERVAL: %w", err)
		}
		if duration <= 0 {
			return nil, fmt.Errorf("CACHE_CLEANUP_INTERVAL must be positive")
		}
		config.CleanupInterval = duration
	}

	if strategy := os.Getenv("CACHE_EVICTION_STRATEGY"); strategy != "" {
		if !isValidEvictionStrategy(strategy) {
			return nil, fmt.Errorf("invalid CACHE_EVICTION_STRATEGY: %s", strategy)
		}
		config.EvictionStrategy = strategy
	}

	if warming := os.Getenv("CACHE_WARMING_ENABLED"); warming != "" {
		b, err := strconv.ParseBool(warming)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_WARMING_ENABLED: %w", err)
		}
		config.WarmingEnabled = b
	}

	if interval := os.Getenv("CACHE_WARMING_INTERVAL"); interval != "" {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_WARMING_INTERVAL: %w", err)
		}
		if duration <= 0 {
			return nil, fmt.Errorf("CACHE_WARMING_INTERVAL must be positive")
		}
		config.WarmingInterval = duration
	}

	if batchSize := os.Getenv("CACHE_WARMING_BATCH_SIZE"); batchSize != "" {
		size, err := strconv.Atoi(batchSize)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_WARMING_BATCH_SIZE: %w", err)
		}
		if size <= 0 {
			return nil, fmt.Errorf("CACHE_WARMING_BATCH_SIZE must be positive")
		}
		config.WarmingBatchSize = size
	}

	if encryption := os.Getenv("CACHE_ENCRYPTION_ENABLED"); encryption != "" {
		b, err := strconv.ParseBool(encryption)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_ENCRYPTION_ENABLED: %w", err)
		}
		config.EncryptionEnabled = b
	}

	if compression := os.Getenv("CACHE_COMPRESSION_ENABLED"); compression != "" {
		b, err := strconv.ParseBool(compression)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_COMPRESSION_ENABLED: %w", err)
		}
		config.CompressionEnabled = b
	}

	return config, nil
}

// Validate validates the cache configuration
func (c *CacheConfig) Validate() error {
	if c.MaxSize <= 0 {
		return fmt.Errorf("MaxSize must be positive")
	}

	if c.DefaultTTL <= 0 {
		return fmt.Errorf("DefaultTTL must be positive")
	}

	if c.CleanupInterval <= 0 {
		return fmt.Errorf("CleanupInterval must be positive")
	}

	if !isValidEvictionStrategy(c.EvictionStrategy) {
		return fmt.Errorf("invalid EvictionStrategy: %s", c.EvictionStrategy)
	}

	if c.WarmingEnabled {
		if c.WarmingInterval <= 0 {
			return fmt.Errorf("WarmingInterval must be positive when warming is enabled")
		}
		if c.WarmingBatchSize <= 0 {
			return fmt.Errorf("WarmingBatchSize must be positive when warming is enabled")
		}
	}

	return nil
}

// isValidEvictionStrategy checks if the eviction strategy is valid
func isValidEvictionStrategy(strategy string) bool {
	switch strategy {
	case "LRU", "LFU", "FIFO":
		return true
	default:
		return false
	}
}
