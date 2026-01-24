package database

import (
	"os"
	"testing"
	"time"
)

// TestLoadConfigDefaults tests loading configuration with default values
func TestLoadConfigDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.MongoDBURI != "mongodb://localhost:27017" {
		t.Errorf("expected default MongoDBURI, got %s", config.MongoDBURI)
	}

	if config.PostgresURL != "postgres://localhost:5432/chainpulse" {
		t.Errorf("expected default PostgresURL, got %s", config.PostgresURL)
	}

	if config.PoolSize != 10 {
		t.Errorf("expected default PoolSize of 10, got %d", config.PoolSize)
	}

	if config.TimeoutMS != 5000 {
		t.Errorf("expected default TimeoutMS of 5000, got %d", config.TimeoutMS)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("expected default RetryAttempts of 3, got %d", config.RetryAttempts)
	}

	if config.RetryDelayMS != 100 {
		t.Errorf("expected default RetryDelayMS of 100, got %d", config.RetryDelayMS)
	}

	if config.CacheTTLSeconds != 3600 {
		t.Errorf("expected default CacheTTLSeconds of 3600, got %d", config.CacheTTLSeconds)
	}

	if config.EventTTLDays != 30 {
		t.Errorf("expected default EventTTLDays of 30, got %d", config.EventTTLDays)
	}

	if config.EventBatchSize != 100 {
		t.Errorf("expected default EventBatchSize of 100, got %d", config.EventBatchSize)
	}
}

// TestLoadConfigFromEnvironment tests loading configuration from environment variables
func TestLoadConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("MONGODB_URI", "mongodb://custom:27017"); err != nil {
		t.Fatalf("failed to set MONGODB_URI: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", "postgres://custom:5432/db"); err != nil {
		t.Fatalf("failed to set DATABASE_URL: %v", err)
	}
	if err := os.Setenv("DB_POOL_SIZE", "20"); err != nil {
		t.Fatalf("failed to set DB_POOL_SIZE: %v", err)
	}
	if err := os.Setenv("DB_TIMEOUT_MS", "10000"); err != nil {
		t.Fatalf("failed to set DB_TIMEOUT_MS: %v", err)
	}
	if err := os.Setenv("DB_RETRY_ATTEMPTS", "5"); err != nil {
		t.Fatalf("failed to set DB_RETRY_ATTEMPTS: %v", err)
	}
	if err := os.Setenv("DB_RETRY_DELAY_MS", "200"); err != nil {
		t.Fatalf("failed to set DB_RETRY_DELAY_MS: %v", err)
	}
	if err := os.Setenv("CACHE_TTL_SECONDS", "7200"); err != nil {
		t.Fatalf("failed to set CACHE_TTL_SECONDS: %v", err)
	}
	if err := os.Setenv("EVENT_TTL_DAYS", "60"); err != nil {
		t.Fatalf("failed to set EVENT_TTL_DAYS: %v", err)
	}
	if err := os.Setenv("EVENT_BATCH_SIZE", "500"); err != nil {
		t.Fatalf("failed to set EVENT_BATCH_SIZE: %v", err)
	}

	defer os.Clearenv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.MongoDBURI != "mongodb://custom:27017" {
		t.Errorf("expected custom MongoDBURI, got %s", config.MongoDBURI)
	}

	if config.PostgresURL != "postgres://custom:5432/db" {
		t.Errorf("expected custom PostgresURL, got %s", config.PostgresURL)
	}

	if config.PoolSize != 20 {
		t.Errorf("expected PoolSize of 20, got %d", config.PoolSize)
	}

	if config.TimeoutMS != 10000 {
		t.Errorf("expected TimeoutMS of 10000, got %d", config.TimeoutMS)
	}

	if config.RetryAttempts != 5 {
		t.Errorf("expected RetryAttempts of 5, got %d", config.RetryAttempts)
	}

	if config.RetryDelayMS != 200 {
		t.Errorf("expected RetryDelayMS of 200, got %d", config.RetryDelayMS)
	}

	if config.CacheTTLSeconds != 7200 {
		t.Errorf("expected CacheTTLSeconds of 7200, got %d", config.CacheTTLSeconds)
	}

	if config.EventTTLDays != 60 {
		t.Errorf("expected EventTTLDays of 60, got %d", config.EventTTLDays)
	}

	if config.EventBatchSize != 500 {
		t.Errorf("expected EventBatchSize of 500, got %d", config.EventBatchSize)
	}
}

// TestValidatePoolSize tests pool size validation
func TestValidatePoolSize(t *testing.T) {
	tests := []struct {
		name      string
		poolSize  int
		shouldErr bool
	}{
		{"valid small pool", 1, false},
		{"valid medium pool", 10, false},
		{"valid large pool", 100, false},
		{"invalid zero pool", 0, true},
		{"invalid negative pool", -1, true},
		{"invalid too large pool", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				MongoDBURI:      "mongodb://localhost:27017",
				PostgresURL:     "postgres://localhost:5432",
				PoolSize:        tt.poolSize,
				TimeoutMS:       5000,
				RetryAttempts:   3,
				RetryDelayMS:    100,
				CacheTTLSeconds: 3600,
				EventTTLDays:    30,
				EventBatchSize:  100,
			}

			err := config.Validate()
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got %v", tt.shouldErr, err)
			}
		})
	}
}

// TestValidateTimeout tests timeout validation
func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name      string
		timeout   int
		shouldErr bool
	}{
		{"valid small timeout", 100, false},
		{"valid medium timeout", 5000, false},
		{"valid large timeout", 60000, false},
		{"invalid too small timeout", 99, true},
		{"invalid too large timeout", 60001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				MongoDBURI:      "mongodb://localhost:27017",
				PostgresURL:     "postgres://localhost:5432",
				PoolSize:        10,
				TimeoutMS:       tt.timeout,
				RetryAttempts:   3,
				RetryDelayMS:    100,
				CacheTTLSeconds: 3600,
				EventTTLDays:    30,
				EventBatchSize:  100,
			}

			err := config.Validate()
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got %v", tt.shouldErr, err)
			}
		})
	}
}

// TestGetTimeout tests timeout conversion
func TestGetTimeout(t *testing.T) {
	config := &Config{TimeoutMS: 5000}
	timeout := config.GetTimeout()

	expected := 5 * time.Second
	if timeout != expected {
		t.Errorf("expected %v, got %v", expected, timeout)
	}
}

// TestGetRetryDelay tests retry delay conversion
func TestGetRetryDelay(t *testing.T) {
	config := &Config{RetryDelayMS: 100}
	delay := config.GetRetryDelay()

	expected := 100 * time.Millisecond
	if delay != expected {
		t.Errorf("expected %v, got %v", expected, delay)
	}
}

// TestGetCacheTTL tests cache TTL conversion
func TestGetCacheTTL(t *testing.T) {
	config := &Config{CacheTTLSeconds: 3600}
	ttl := config.GetCacheTTL()

	expected := time.Hour
	if ttl != expected {
		t.Errorf("expected %v, got %v", expected, ttl)
	}
}

// TestGetEventTTL tests event TTL conversion
func TestGetEventTTL(t *testing.T) {
	config := &Config{EventTTLDays: 30}
	ttl := config.GetEventTTL()

	expected := 30 * 24 * time.Hour
	if ttl != expected {
		t.Errorf("expected %v, got %v", expected, ttl)
	}
}

// TestValidateEventTTLDays tests event TTL days validation
func TestValidateEventTTLDays(t *testing.T) {
	tests := []struct {
		name      string
		ttlDays   int
		shouldErr bool
	}{
		{"valid minimum", 1, false},
		{"valid medium", 30, false},
		{"valid maximum", 365, false},
		{"invalid zero", 0, true},
		{"invalid negative", -1, true},
		{"invalid too large", 366, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				MongoDBURI:      "mongodb://localhost:27017",
				PostgresURL:     "postgres://localhost:5432",
				PoolSize:        10,
				TimeoutMS:       5000,
				RetryAttempts:   3,
				RetryDelayMS:    100,
				CacheTTLSeconds: 3600,
				EventTTLDays:    tt.ttlDays,
				EventBatchSize:  100,
			}

			err := config.Validate()
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got %v", tt.shouldErr, err)
			}
		})
	}
}

// TestValidateEventBatchSize tests event batch size validation
func TestValidateEventBatchSize(t *testing.T) {
	tests := []struct {
		name      string
		batchSize int
		shouldErr bool
	}{
		{"valid minimum", 1, false},
		{"valid medium", 100, false},
		{"valid maximum", 10000, false},
		{"invalid zero", 0, true},
		{"invalid negative", -1, true},
		{"invalid too large", 10001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				MongoDBURI:      "mongodb://localhost:27017",
				PostgresURL:     "postgres://localhost:5432",
				PoolSize:        10,
				TimeoutMS:       5000,
				RetryAttempts:   3,
				RetryDelayMS:    100,
				CacheTTLSeconds: 3600,
				EventTTLDays:    30,
				EventBatchSize:  tt.batchSize,
			}

			err := config.Validate()
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got %v", tt.shouldErr, err)
			}
		})
	}
}
