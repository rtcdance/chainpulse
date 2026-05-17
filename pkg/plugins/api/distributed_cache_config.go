package api

import (
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/env"
)

// DistributedCacheConfig holds configuration for distributed caching in the API layer
type DistributedCacheConfig struct {
	// Redis configuration
	RedisAddr           string
	RedisPassword       core.SecretString
	RedisDB             int
	PoolSize            int
	MinIdleConns        int
	MaxRetries          int
	DialTimeout         time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	HealthCheckInterval time.Duration
	MaxLocalCacheSize   int
	DefaultTTL          time.Duration
	FallbackEnabled     bool
	CacheKeyPrefix      string
}

// NewDistributedCacheConfig creates a new distributed cache configuration for the API layer
func NewDistributedCacheConfig() *DistributedCacheConfig {
	return &DistributedCacheConfig{
		RedisAddr:           env.Get("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       core.SecretString(env.Get("REDIS_PASSWORD", "")),
		RedisDB:             env.GetInt("REDIS_DB", 0),
		PoolSize:            env.GetInt("REDIS_POOL_SIZE", 10),
		MinIdleConns:        env.GetInt("REDIS_MIN_IDLE_CONNS", 5),
		MaxRetries:          env.GetInt("REDIS_MAX_RETRIES", 3),
		DialTimeout:         time.Duration(env.GetInt("REDIS_DIAL_TIMEOUT_MS", 5000)) * time.Millisecond,
		ReadTimeout:         time.Duration(env.GetInt("REDIS_READ_TIMEOUT_MS", 3000)) * time.Millisecond,
		WriteTimeout:        time.Duration(env.GetInt("REDIS_WRITE_TIMEOUT_MS", 3000)) * time.Millisecond,
		HealthCheckInterval: time.Duration(env.GetInt("REDIS_HEALTH_CHECK_INTERVAL_S", 60)) * time.Second,
		MaxLocalCacheSize:   env.GetInt("MAX_LOCAL_CACHE_SIZE", 10000),
		DefaultTTL:          time.Duration(env.GetInt("DEFAULT_CACHE_TTL_S", 3600)) * time.Second,
		FallbackEnabled:     env.GetBool("CACHE_FALLBACK_ENABLED", true),
		CacheKeyPrefix:      env.Get("CACHE_KEY_PREFIX", "api:"),
	}
}

// Validate validates the configuration
func (dcc *DistributedCacheConfig) Validate() error {
	if dcc.RedisAddr == "" {
		return ErrInvalidRequest("RedisAddr is required")
	}

	if dcc.PoolSize < 1 {
		return ErrInvalidRequest("PoolSize must be at least 1")
	}

	if dcc.MaxLocalCacheSize < 1 {
		return ErrInvalidRequest("MaxLocalCacheSize must be at least 1")
	}

	return nil
}
