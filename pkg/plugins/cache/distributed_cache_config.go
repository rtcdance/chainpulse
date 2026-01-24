package cache

import (
	"os"
	"strconv"
	"time"
)

// DistributedCacheConfig holds configuration for distributed caching
type DistributedCacheConfig struct {
	// Redis configuration
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	PoolSize             int
	MinIdleConns         int
	MaxRetries           int
	DialTimeout          time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	HealthCheckInterval  time.Duration
	MaxLocalCacheSize    int
	DefaultTTL           time.Duration
	FallbackEnabled      bool
}

// NewDistributedCacheConfig creates a new distributed cache configuration
func NewDistributedCacheConfig() *DistributedCacheConfig {
	return &DistributedCacheConfig{
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnvInt("REDIS_DB", 0),
		PoolSize:            getEnvInt("REDIS_POOL_SIZE", 10),
		MinIdleConns:        getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
		MaxRetries:          getEnvInt("REDIS_MAX_RETRIES", 3),
		DialTimeout:         time.Duration(getEnvInt("REDIS_DIAL_TIMEOUT_MS", 5000)) * time.Millisecond,
		ReadTimeout:         time.Duration(getEnvInt("REDIS_READ_TIMEOUT_MS", 3000)) * time.Millisecond,
		WriteTimeout:        time.Duration(getEnvInt("REDIS_WRITE_TIMEOUT_MS", 3000)) * time.Millisecond,
		HealthCheckInterval: time.Duration(getEnvInt("REDIS_HEALTH_CHECK_INTERVAL_S", 60)) * time.Second,
		MaxLocalCacheSize:   getEnvInt("MAX_LOCAL_CACHE_SIZE", 10000),
		DefaultTTL:          time.Duration(getEnvInt("DEFAULT_CACHE_TTL_S", 3600)) * time.Second,
		FallbackEnabled:     getEnvBool("CACHE_FALLBACK_ENABLED", true),
	}
}

// Validate validates the configuration
func (dcc *DistributedCacheConfig) Validate() error {
	if dcc.RedisAddr == "" {
		return ErrInvalidConfig("RedisAddr is required")
	}

	if dcc.PoolSize < 1 {
		return ErrInvalidConfig("PoolSize must be at least 1")
	}

	if dcc.MaxLocalCacheSize < 1 {
		return ErrInvalidConfig("MaxLocalCacheSize must be at least 1")
	}

	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
