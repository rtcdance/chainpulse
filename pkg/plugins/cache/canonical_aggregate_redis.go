package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// CanonicalAggregateRedisCache is a dedicated, bounded optional query cache.
// It never substitutes a local map when Redis fails. The runtime owns Close.
type CanonicalAggregateRedisCache struct {
	client *redis.Client
}

func NewCanonicalAggregateRedisCache(url string) (*CanonicalAggregateRedisCache, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("canonical Redis URL is required")
	}
	if !strings.Contains(url, "://") {
		url = "redis://" + url
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		// ParseURL errors may contain credentials; never return the input.
		return nil, fmt.Errorf("invalid canonical Redis URL")
	}
	options.ContextTimeoutEnabled = true
	options.DialTimeout = 50 * time.Millisecond
	options.ReadTimeout = 50 * time.Millisecond
	options.WriteTimeout = 50 * time.Millisecond
	options.PoolTimeout = 50 * time.Millisecond
	options.MaxRetries = -1
	options.PoolSize = 8
	options.MaxActiveConns = 8
	options.MinIdleConns = 0
	options.MaxIdleConns = 8
	return &CanonicalAggregateRedisCache{client: redis.NewClient(options)}, nil
}

func (c *CanonicalAggregateRedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	// GETRANGE bounds allocation even if a corrupt writer stored a huge value.
	value, err := c.client.GetRange(ctx, key, 0, 65536).Bytes()
	if err != nil {
		return nil, fmt.Errorf("read canonical aggregate Redis cache: %w", err)
	}
	if len(value) > 65536 {
		return nil, fmt.Errorf("canonical aggregate cache payload exceeds 64KiB")
	}
	if len(value) == 0 {
		return nil, nil
	}
	return value, nil
}

func (c *CanonicalAggregateRedisCache) Set(ctx context.Context, key string, value []byte, ttl int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ttl < 1 || ttl > 300 || len(value) > 65536 || key == "" {
		return fmt.Errorf("canonical aggregate cache requires key, TTL 1..300s and payload <=64KiB")
	}
	ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := c.client.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err(); err != nil {
		return fmt.Errorf("write canonical aggregate Redis cache: %w", err)
	}
	return nil
}

func (c *CanonicalAggregateRedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close canonical aggregate Redis cache: %w", err)
	}
	return nil
}
