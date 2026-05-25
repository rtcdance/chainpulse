package api

import (
	"testing"
)

func TestNewDistributedCacheConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDistributedCacheConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("expected RedisAddr=localhost:6379, got %s", cfg.RedisAddr)
	}
	if cfg.PoolSize != 10 {
		t.Errorf("expected PoolSize=10, got %d", cfg.PoolSize)
	}
	if cfg.MaxLocalCacheSize != 10000 {
		t.Errorf("expected MaxLocalCacheSize=10000, got %d", cfg.MaxLocalCacheSize)
	}
	if cfg.CacheKeyPrefix != "api:" {
		t.Errorf("expected CacheKeyPrefix=api:, got %s", cfg.CacheKeyPrefix)
	}
	if !cfg.FallbackEnabled {
		t.Error("expected FallbackEnabled=true")
	}
}

func TestDistributedCacheConfig_Validate_Success(t *testing.T) {
	t.Parallel()

	dcc := &DistributedCacheConfig{
		RedisAddr:         "localhost:6379",
		PoolSize:          10,
		MaxLocalCacheSize: 1000,
	}

	err := dcc.Validate()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDistributedCacheConfig_Validate_EmptyRedisAddr(t *testing.T) {
	t.Parallel()

	dcc := &DistributedCacheConfig{
		RedisAddr:         "",
		PoolSize:          10,
		MaxLocalCacheSize: 1000,
	}

	err := dcc.Validate()
	if err == nil {
		t.Error("expected error for empty RedisAddr")
	}
}

func TestDistributedCacheConfig_Validate_ZeroPoolSize(t *testing.T) {
	t.Parallel()

	dcc := &DistributedCacheConfig{
		RedisAddr:         "localhost:6379",
		PoolSize:          0,
		MaxLocalCacheSize: 1000,
	}

	err := dcc.Validate()
	if err == nil {
		t.Error("expected error for zero PoolSize")
	}
}

func TestDistributedCacheConfig_Validate_NegativePoolSize(t *testing.T) {
	t.Parallel()

	dcc := &DistributedCacheConfig{
		RedisAddr:         "localhost:6379",
		PoolSize:          -1,
		MaxLocalCacheSize: 1000,
	}

	err := dcc.Validate()
	if err == nil {
		t.Error("expected error for negative PoolSize")
	}
}

func TestDistributedCacheConfig_Validate_ZeroMaxLocalCacheSize(t *testing.T) {
	t.Parallel()

	dcc := &DistributedCacheConfig{
		RedisAddr:         "localhost:6379",
		PoolSize:          10,
		MaxLocalCacheSize: 0,
	}

	err := dcc.Validate()
	if err == nil {
		t.Error("expected error for zero MaxLocalCacheSize")
	}
}
