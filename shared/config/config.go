package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	EthereumNodeURL string
	EthereumNodeWSURL string
	PostgreSQLURL   string
	RedisURL        string
	GRPCServerURL   string
	ServerPort      string
	JWTSecret       string
	RateLimit       int
	RateLimitBurst  int
	BatchSize       int
	FlushTimeout    int // in seconds
	MaxConcurrentWorkers int
}

func LoadConfig() (*Config, error) {
	return &Config{
		EthereumNodeURL: getEnv("ETHEREUM_NODE_URL", "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"),
		EthereumNodeWSURL: getEnv("ETHEREUM_NODE_WS_URL", "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID"),
		PostgreSQLURL:   getEnv("POSTGRESQL_URL", "postgres://user:password@localhost:5432/chainpulse?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		GRPCServerURL:   getEnv("GRPC_SERVER_URL", "localhost:50051"),
		ServerPort:      getEnv("PORT", "8080"),
		JWTSecret:       getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		RateLimit:       getEnvAsInt("RATE_LIMIT", 10), // 10 requests per second
		RateLimitBurst:  getEnvAsInt("RATE_LIMIT_BURST", 20), // Burst of 20 requests
		BatchSize:       getEnvAsInt("BATCH_SIZE", 100), // 100 events per batch
		FlushTimeout:    getEnvAsInt("FLUSH_TIMEOUT", 5), // 5 seconds timeout
		MaxConcurrentWorkers: getEnvAsInt("MAX_CONCURRENT_WORKERS", 10), // 10 concurrent workers
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// LoadSharedConfig loads shared configuration that can be used across services
func LoadSharedConfig() (*Config, error) {
	shared, err := loadSharedConfigDirectly()
	if err != nil {
		return nil, err
	}
	
	return &Config{
		EthereumNodeURL: shared.EthereumNodeURL,
		PostgreSQLURL:   shared.PostgreSQLURL,
		RedisURL:        shared.RedisURL,
		ServerPort:      shared.APIPort,
		JWTSecret:       shared.JWTSecret,
		BatchSize:       shared.BatchSize,
		FlushTimeout:    shared.FlushTimeout,
		// Default values for other fields that might not be in shared config
		RateLimit:       10,
		RateLimitBurst:  20,
		MaxConcurrentWorkers: 10,
	}, nil
}

// loadSharedConfigDirectly loads shared config directly without circular reference
func loadSharedConfigDirectly() (*SharedConfig, error) {
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