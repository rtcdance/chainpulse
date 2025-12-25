package cache

import (
	"context"
	json "github.com/goccy/go-json"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	Client *redis.Client
}

func NewCache(redisURL string) (*Cache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	// Configure for high concurrency
	opts.PoolSize = 20
	opts.MinIdleConns = 10
	opts.MaxConnAge = 30 * time.Minute
	opts.PoolTimeout = 30 * time.Second
	opts.IdleTimeout = 5 * time.Minute
	opts.MaxRetries = 3

	client := redis.NewClient(opts)

	return &Cache{
		Client: client,
	}, nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Client.Set(ctx, key, data, expiration).Err()
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.Client.Del(ctx, key).Err()
}

func (c *Cache) Close() error {
	return c.Client.Close()
}

// Ping checks if the cache connection is alive
func (c *Cache) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}