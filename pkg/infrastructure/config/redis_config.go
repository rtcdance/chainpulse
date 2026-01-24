package config

import (
	"context"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// RedisCluster manages Redis cluster operations
type RedisCluster struct {
	config *RedisConfig
	Client *redisv9.Client
}

// NewRedisCluster creates a new Redis cluster manager
func NewRedisCluster(cfg *RedisConfig) (*RedisCluster, error) {
	if cfg == nil {
		cfg = &RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		}
	}

	client := redisv9.NewClient(&redisv9.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCluster{
		config: cfg,
		Client: client,
	}, nil
}

// Health checks Redis cluster health
func (r *RedisCluster) Health(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisCluster) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// WaitForRedis waits for Redis to be available
func WaitForRedis(cfg *RedisConfig, timeout time.Duration) error {
	cluster, err := NewRedisCluster(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := cluster.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for Redis")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := cluster.Health(ctx)
		cancel()

		if err == nil {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}
