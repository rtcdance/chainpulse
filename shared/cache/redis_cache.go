package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache implements a caching layer using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(addr, password string, db int) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,     // Redis server address
		Password: password, // Redis password (empty if no password)
		DB:       db,       // Redis database number
		PoolSize: 20,       // Connection pool size for high concurrency
		MinIdleConns: 10,   // Minimum number of idle connections
		MaxConnAge: 30 * time.Minute, // Maximum connection age
		PoolTimeout: 30 * time.Second, // Connection pool timeout
		IdleTimeout: 5 * time.Minute,  // Connection idle timeout
		MaxRetries: 3, // Number of retries for failed commands
	})

	return &RedisCache{
		client: rdb,
	}
}

// Get retrieves a value from the cache
func (rc *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key %s not found in cache", key)
		}
		return fmt.Errorf("error getting from cache: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("error unmarshaling cached data: %w", err)
	}

	return nil
}

// Set stores a value in the cache with expiration
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshaling data for cache: %w", err)
	}

	if err := rc.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("error setting cache: %w", err)
	}

	return nil
}

// SetWithoutExpiration stores a value in the cache without expiration
func (rc *RedisCache) SetWithoutExpiration(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshaling data for cache: %w", err)
	}

	if err := rc.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("error setting cache: %w", err)
	}

	return nil
}

// Delete removes a key from the cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("error deleting from cache: %w", err)
	}

	return nil
}

// Exists checks if a key exists in the cache
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("error checking if key exists in cache: %w", err)
	}

	return count > 0, nil
}

// Ping tests the Redis connection
func (rc *RedisCache) Ping(ctx context.Context) error {
	if err := rc.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error pinging Redis: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// InvalidatePrefix deletes all keys with the given prefix
func (rc *RedisCache) InvalidatePrefix(ctx context.Context, prefix string) error {
	iter := rc.client.Scan(ctx, 0, prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if err := rc.client.Del(ctx, key).Err(); err != nil {
			log.Printf("Error deleting key %s from cache: %v", key, err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("error scanning keys with prefix %s: %w", prefix, err)
	}
	return nil
}