package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisPlugin implements MQPlugin for Redis
type RedisPlugin struct {
	client           *redis.Client
	metricsCollector *MetricsCollector
	config           RedisConfig
}

// RedisConfig holds configuration for Redis connection
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewRedisPlugin creates a new Redis plugin instance
func NewRedisPlugin() *RedisPlugin {
	return &RedisPlugin{}
}

// Initialize initializes the Redis plugin with configuration
func (r *RedisPlugin) Initialize(config map[string]interface{}) error {
	addrInterface, exists := config["addr"]
	if !exists {
		return fmt.Errorf("addr configuration is required for Redis plugin")
	}

	addr, ok := addrInterface.(string)
	if !ok {
		return fmt.Errorf("addr must be a string")
	}

	password := ""
	if passwordInterface, exists := config["password"]; exists {
		if passwordStr, ok := passwordInterface.(string); ok {
			password = passwordStr
		}
	}

	db := 0
	if dbInterface, exists := config["db"]; exists {
		if dbFloat, ok := dbInterface.(float64); ok {
			db = int(dbFloat)
		} else if dbInt, ok := dbInterface.(int); ok {
			db = dbInt
		}
	}

	r.config = RedisConfig{
		Addr:     addr,
		Password: password,
		DB:       db,
	}

	// Create Redis client
	r.client = redis.NewClient(&redis.Options{
		Addr:     r.config.Addr,
		Password: r.config.Password,
		DB:       r.config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// GetName returns the name of the plugin
func (r *RedisPlugin) GetName() string {
	return "redis"
}

// SetMetricsCollector sets the metrics collector for the plugin
func (r *RedisPlugin) SetMetricsCollector(collector *MetricsCollector) {
	r.metricsCollector = collector
}

// Publish sends a message to the specified topic using Redis
func (r *RedisPlugin) Publish(topic string, message interface{}) error {
	startTime := time.Now()

	data, err := json.Marshal(message)
	if err != nil {
		if r.metricsCollector != nil {
			r.metricsCollector.RecordRequest("redis", time.Since(startTime), err)
		}
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use Redis list as a simple queue
	err = r.client.LPush(ctx, topic, data).Err()

	if r.metricsCollector != nil {
		r.metricsCollector.RecordRequest("redis", time.Since(startTime), err)
	}

	if err != nil {
		return fmt.Errorf("failed to publish message to Redis: %w", err)
	}

	return nil
}

// Consume reads messages from the specified topic and handles them using Redis
func (r *RedisPlugin) Consume(ctx context.Context, topic string, handler MessageHandler) error {
	// Create a worker pool for concurrent message processing
	const numWorkers = 5
	tasks := make(chan []byte, numWorkers*2)

	// Start worker goroutines
	var workersDone int
	workersDoneChan := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() {
				workersDone++
				workersDoneChan <- true
			}()

			for msg := range tasks {
				startTime := time.Now()
				err := handler(msg)

				if r.metricsCollector != nil {
					r.metricsCollector.RecordRequest("redis", time.Since(startTime), err)
				}

				if err != nil {
					log.Printf("Error handling message in worker %d: %v", workerID, err)
				}
			}
		}(i)
	}

	// Goroutine to fetch messages from Redis
	go func() {
		defer close(tasks)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Use BRPOP to block until a message is available
				ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
				result, err := r.client.BRPop(ctx, 1*time.Second, topic).Result()
				cancel()

				if err != nil && err != redis.Nil {
					log.Printf("Error fetching message from Redis: %v", err)
					time.Sleep(100 * time.Millisecond) // Brief pause before retrying
					continue
				}

				// If we got a result (message), send it to the workers
				if err == nil && len(result) > 1 {
					select {
					case tasks <- []byte(result[1]): // result[1] contains the value
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	// Wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		<-workersDoneChan
	}

	return ctx.Err()
}

// Close closes the Redis connection
func (r *RedisPlugin) Close() error {
	return r.client.Close()
}
