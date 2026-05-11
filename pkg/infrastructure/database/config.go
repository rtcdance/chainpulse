package database

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents database configuration
type Config struct {
	MongoDBURI      string
	PostgresURL     string
	PostgresSSLMode string
	PoolSize        int
	TimeoutMS       int
	RetryAttempts   int
	RetryDelayMS    int
	CacheTTLSeconds int
	EventTTLDays    int
	EventBatchSize  int
}

// LoadConfig loads database configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		MongoDBURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		PostgresURL:     getEnv("DATABASE_URL", "postgres://localhost:5432/chainpulse"),
		PostgresSSLMode: getEnv("DATABASE_SSLMODE", "disable"),
		PoolSize:        getEnvInt("DB_POOL_SIZE", 10),
		TimeoutMS:       getEnvInt("DB_TIMEOUT_MS", 5000),
		RetryAttempts:   getEnvInt("DB_RETRY_ATTEMPTS", 3),
		RetryDelayMS:    getEnvInt("DB_RETRY_DELAY_MS", 100),
		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 3600),
		EventTTLDays:    getEnvInt("EVENT_TTL_DAYS", 30),
		EventBatchSize:  getEnvInt("EVENT_BATCH_SIZE", 100),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the database configuration
func (c *Config) Validate() error {
	if c.MongoDBURI == "" {
		return fmt.Errorf("MONGODB_URI is required")
	}

	if c.PostgresURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.PoolSize < 1 || c.PoolSize > 100 {
		return fmt.Errorf("DB_POOL_SIZE must be between 1 and 100, got %d", c.PoolSize)
	}

	if c.TimeoutMS < 100 || c.TimeoutMS > 60000 {
		return fmt.Errorf("DB_TIMEOUT_MS must be between 100 and 60000, got %d", c.TimeoutMS)
	}

	if c.RetryAttempts < 0 || c.RetryAttempts > 10 {
		return fmt.Errorf("DB_RETRY_ATTEMPTS must be between 0 and 10, got %d", c.RetryAttempts)
	}

	if c.RetryDelayMS < 0 || c.RetryDelayMS > 10000 {
		return fmt.Errorf("DB_RETRY_DELAY_MS must be between 0 and 10000, got %d", c.RetryDelayMS)
	}

	if c.CacheTTLSeconds < 0 || c.CacheTTLSeconds > 86400 {
		return fmt.Errorf("CACHE_TTL_SECONDS must be between 0 and 86400, got %d", c.CacheTTLSeconds)
	}

	if c.EventTTLDays < 1 || c.EventTTLDays > 365 {
		return fmt.Errorf("EVENT_TTL_DAYS must be between 1 and 365, got %d", c.EventTTLDays)
	}

	if c.EventBatchSize < 1 || c.EventBatchSize > 10000 {
		return fmt.Errorf("EVENT_BATCH_SIZE must be between 1 and 10000, got %d", c.EventBatchSize)
	}

	return nil
}

// GetTimeout returns the timeout as a time.Duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.TimeoutMS) * time.Millisecond
}

// GetRetryDelay returns the retry delay as a time.Duration
func (c *Config) GetRetryDelay() time.Duration {
	return time.Duration(c.RetryDelayMS) * time.Millisecond
}

// GetCacheTTL returns the cache TTL as a time.Duration
func (c *Config) GetCacheTTL() time.Duration {
	return time.Duration(c.CacheTTLSeconds) * time.Second
}

// GetEventTTL returns the event TTL as a time.Duration
func (c *Config) GetEventTTL() time.Duration {
	return time.Duration(c.EventTTLDays) * 24 * time.Hour
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
