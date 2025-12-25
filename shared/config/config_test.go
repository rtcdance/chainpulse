package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Test with default values
	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "https://mainnet.infura.io/v3/YOUR_PROJECT_ID", config.EthereumNodeURL)
	assert.Equal(t, "postgres://user:password@localhost:5432/chainpulse?sslmode=disable", config.PostgreSQLURL)
	assert.Equal(t, "redis://localhost:6379", config.RedisURL)
	assert.Equal(t, "8080", config.ServerPort)
	assert.Equal(t, "your-super-secret-jwt-key-change-in-production", config.JWTSecret)
	assert.Equal(t, 10, config.RateLimit)
	assert.Equal(t, 20, config.RateLimitBurst)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 5, config.FlushTimeout)
	assert.Equal(t, 10, config.MaxConcurrentWorkers)
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("ETHEREUM_NODE_URL", "https://test-node.example.com")
	os.Setenv("POSTGRESQL_URL", "postgres://test:test@localhost:5432/testdb")
	os.Setenv("REDIS_URL", "redis://localhost:6380")
	os.Setenv("PORT", "9090")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("RATE_LIMIT", "20")
	os.Setenv("RATE_LIMIT_BURST", "40")
	os.Setenv("BATCH_SIZE", "200")
	os.Setenv("FLUSH_TIMEOUT", "10")
	os.Setenv("MAX_CONCURRENT_WORKERS", "20")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("ETHEREUM_NODE_URL")
		os.Unsetenv("POSTGRESQL_URL")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("PORT")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("RATE_LIMIT")
		os.Unsetenv("RATE_LIMIT_BURST")
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("FLUSH_TIMEOUT")
		os.Unsetenv("MAX_CONCURRENT_WORKERS")
	}()

	// Load config with environment variables
	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "https://test-node.example.com", config.EthereumNodeURL)
	assert.Equal(t, "postgres://test:test@localhost:5432/testdb", config.PostgreSQLURL)
	assert.Equal(t, "redis://localhost:6380", config.RedisURL)
	assert.Equal(t, "9090", config.ServerPort)
	assert.Equal(t, "test-secret", config.JWTSecret)
	assert.Equal(t, 20, config.RateLimit)
	assert.Equal(t, 40, config.RateLimitBurst)
	assert.Equal(t, 200, config.BatchSize)
	assert.Equal(t, 10, config.FlushTimeout)
	assert.Equal(t, 20, config.MaxConcurrentWorkers)
}

func TestLoadConfigWithInvalidIntValues(t *testing.T) {
	// Set invalid integer environment variables
	os.Setenv("RATE_LIMIT", "invalid")
	os.Setenv("RATE_LIMIT_BURST", "invalid")
	os.Setenv("BATCH_SIZE", "invalid")
	os.Setenv("FLUSH_TIMEOUT", "invalid")
	os.Setenv("MAX_CONCURRENT_WORKERS", "invalid")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("RATE_LIMIT")
		os.Unsetenv("RATE_LIMIT_BURST")
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("FLUSH_TIMEOUT")
		os.Unsetenv("MAX_CONCURRENT_WORKERS")
	}()

	// Load config with invalid values (should use defaults)
	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, 10, config.RateLimit)      // default value
	assert.Equal(t, 20, config.RateLimitBurst) // default value
	assert.Equal(t, 100, config.BatchSize)     // default value
	assert.Equal(t, 5, config.FlushTimeout)    // default value
	assert.Equal(t, 10, config.MaxConcurrentWorkers) // default value
}