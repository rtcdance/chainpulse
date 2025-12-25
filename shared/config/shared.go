// pkg/config/shared.go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// SharedConfig contains configuration that can be shared across services
type SharedConfig struct {
	PostgreSQLURL    string
	RedisURL         string
	EthereumNodeURL  string
	JWTSecret        string
	BatchSize        int
	FlushTimeout     int
	MaxRetries       int
	RetryDelay       int
	APIPort          string
	GRPCPort         string
	MetricsPort      string
	LogLevel         string
}

// LoadSharedConfig loads configuration from environment variables
func LoadSharedConfig() (*SharedConfig, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &SharedConfig{
		PostgreSQLURL:   getEnv("POSTGRES_URL", "postgres://user:password@localhost:5432/chainpulse?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		EthereumNodeURL: getEnv("ETHEREUM_NODE_URL", "ws://localhost:8545"),
		JWTSecret:       getEnv("JWT_SECRET", "default_secret_key"),
		BatchSize:       getEnvAsInt("BATCH_SIZE", 100),
		FlushTimeout:    getEnvAsInt("FLUSH_TIMEOUT", 30),
		MaxRetries:      getEnvAsInt("MAX_RETRIES", 3),
		RetryDelay:      getEnvAsInt("RETRY_DELAY", 5),
		APIPort:         getEnv("API_PORT", "8080"),
		GRPCPort:        getEnv("GRPC_PORT", "9090"),
		MetricsPort:     getEnv("METRICS_PORT", "9091"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		fmt.Sscanf(value, "%d", &result)
		return result
	}
	return defaultValue
}