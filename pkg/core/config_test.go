package core

import (
	"os"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/env"
)

// TestNewConfigManager verifies config manager creation
func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager(nil)

	if cm == nil {
		t.Errorf("expected config manager to be created")
	}
}

// TestLoadConfiguration verifies configuration loading
func TestLoadConfiguration(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("DATA_PULLER_TYPE", "websocket")
	_ = os.Setenv("BLOCKCHAIN_NODE_URL", "ws://localhost:8546")
	_ = os.Setenv("MQ_TYPE", "redis")
	_ = os.Setenv("CACHE_TYPE", "memory")
	_ = os.Setenv("DATABASE_TYPE", "mongodb")
	_ = os.Setenv("API_TYPE", "grpc")
	_ = os.Setenv("API_PORT", "9090")
	_ = os.Setenv("WORKER_POOL_SIZE", "20")
	_ = os.Setenv("BATCH_SIZE", "200")
	_ = os.Setenv("MAX_RETRIES", "5")
	_ = os.Setenv("DEPLOYMENT_MODE", "microservice")
	_ = os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		_ = os.Unsetenv("DATA_PULLER_TYPE")
		_ = os.Unsetenv("BLOCKCHAIN_NODE_URL")
		_ = os.Unsetenv("MQ_TYPE")
		_ = os.Unsetenv("CACHE_TYPE")
		_ = os.Unsetenv("DATABASE_TYPE")
		_ = os.Unsetenv("API_TYPE")
		_ = os.Unsetenv("API_PORT")
		_ = os.Unsetenv("WORKER_POOL_SIZE")
		_ = os.Unsetenv("BATCH_SIZE")
		_ = os.Unsetenv("MAX_RETRIES")
		_ = os.Unsetenv("DEPLOYMENT_MODE")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	cm := NewConfigManager(nil)
	config, err := cm.Load()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if config.DataPullerType != "websocket" {
		t.Errorf("expected DataPullerType to be 'websocket', got %s", config.DataPullerType)
	}

	if config.APIPort != 9090 {
		t.Errorf("expected APIPort to be 9090, got %d", config.APIPort)
	}

	if config.WorkerPoolSize != 20 {
		t.Errorf("expected WorkerPoolSize to be 20, got %d", config.WorkerPoolSize)
	}
}

// TestValidateConfigurationValid verifies valid configuration passes validation
func TestValidateConfigurationValid(t *testing.T) {
	config := Config{
		DataPullerType:     "https-jsonrpc",
		BlockchainNodeURL:  "http://localhost:8545",
		MQType:             "kafka",
		MQConnectionURL:    "kafka://localhost:9092",
		CacheType:          "redis",
		CacheConnectionURL: "localhost:6379",
		CacheTTL:           3600,
		DatabaseType:       "postgres",
		DatabaseURL:        "postgres://localhost/chainpulse",
		APIType:            "rest",
		APIPort:            8080,
		WorkerPoolSize:     10,
		BatchSize:          100,
		MaxRetries:         3,
		RetryBackoff:       100,
		DeploymentMode:     "monolithic",
		ServiceName:        "chainpulse",
		LogLevel:           "info",
		FeatureFlags:       make(map[string]bool),
	}

	cm := NewConfigManager(nil)
	err := cm.ValidateConfig(config)
	if err != nil {
		t.Errorf("expected valid configuration to pass validation, got error %v", err)
	}
}

// TestValidateConfigurationInvalidDataPullerType verifies invalid data puller type
func TestValidateConfigurationInvalidDataPullerType(t *testing.T) {
	config := Config{
		DataPullerType:     "invalid",
		BlockchainNodeURL:  "http://localhost:8545",
		MQType:             "kafka",
		MQConnectionURL:    "kafka://localhost:9092",
		CacheType:          "redis",
		CacheConnectionURL: "localhost:6379",
		CacheTTL:           3600,
		DatabaseType:       "postgres",
		DatabaseURL:        "postgres://localhost/chainpulse",
		APIType:            "rest",
		APIPort:            8080,
		WorkerPoolSize:     10,
		BatchSize:          100,
		MaxRetries:         3,
		RetryBackoff:       100,
		DeploymentMode:     "monolithic",
		ServiceName:        "chainpulse",
		LogLevel:           "info",
		FeatureFlags:       make(map[string]bool),
	}

	cm := NewConfigManager(nil)
	err := cm.ValidateConfig(config)

	if err == nil {
		t.Errorf("expected validation to fail for invalid DataPullerType")
	}
}

// TestValidateConfigurationMissingBlockchainNodeURL verifies missing blockchain node URL
func TestValidateConfigurationMissingBlockchainNodeURL(t *testing.T) {
	config := Config{
		DataPullerType:     "https-jsonrpc",
		BlockchainNodeURL:  "",
		MQType:             "kafka",
		MQConnectionURL:    "kafka://localhost:9092",
		CacheType:          "redis",
		CacheConnectionURL: "localhost:6379",
		CacheTTL:           3600,
		DatabaseType:       "postgres",
		DatabaseURL:        "postgres://localhost/chainpulse",
		APIType:            "rest",
		APIPort:            8080,
		WorkerPoolSize:     10,
		BatchSize:          100,
		MaxRetries:         3,
		RetryBackoff:       100,
		DeploymentMode:     "monolithic",
		ServiceName:        "chainpulse",
		LogLevel:           "info",
		FeatureFlags:       make(map[string]bool),
	}

	cm := NewConfigManager(nil)
	err := cm.ValidateConfig(config)

	if err == nil {
		t.Errorf("expected validation to fail for missing BlockchainNodeURL")
	}
}

// TestValidateConfigurationInvalidAPIPort verifies invalid API port
func TestValidateConfigurationInvalidAPIPort(t *testing.T) {
	config := Config{
		DataPullerType:     "https-jsonrpc",
		BlockchainNodeURL:  "http://localhost:8545",
		MQType:             "kafka",
		MQConnectionURL:    "kafka://localhost:9092",
		CacheType:          "redis",
		CacheConnectionURL: "localhost:6379",
		CacheTTL:           3600,
		DatabaseType:       "postgres",
		DatabaseURL:        "postgres://localhost/chainpulse",
		APIType:            "rest",
		APIPort:            99999,
		WorkerPoolSize:     10,
		BatchSize:          100,
		MaxRetries:         3,
		RetryBackoff:       100,
		DeploymentMode:     "monolithic",
		ServiceName:        "chainpulse",
		LogLevel:           "info",
		FeatureFlags:       make(map[string]bool),
	}

	cm := NewConfigManager(nil)
	err := cm.ValidateConfig(config)

	if err == nil {
		t.Errorf("expected validation to fail for invalid APIPort")
	}
}

// TestGetConfigurationValue verifies getting configuration values
func TestGetConfigurationValue(t *testing.T) {
	config := Config{
		DataPullerType:     "https-jsonrpc",
		BlockchainNodeURL:  "http://localhost:8545",
		MQType:             "kafka",
		MQConnectionURL:    "kafka://localhost:9092",
		CacheType:          "redis",
		CacheConnectionURL: "localhost:6379",
		CacheTTL:           3600,
		DatabaseType:       "postgres",
		DatabaseURL:        "postgres://localhost/chainpulse",
		APIType:            "rest",
		APIPort:            8080,
		WorkerPoolSize:     10,
		BatchSize:          100,
		MaxRetries:         3,
		RetryBackoff:       100,
		DeploymentMode:     "monolithic",
		ServiceName:        "chainpulse",
		LogLevel:           "info",
		FeatureFlags:       make(map[string]bool),
	}

	cm := NewConfigManager(nil)
	cm.config = config

	value, err := cm.Get("api_port")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if value != 8080 {
		t.Errorf("expected api_port to be 8080, got %v", value)
	}
}

// TestSetConfigurationValue verifies setting configuration values
func TestSetConfigurationValue(t *testing.T) {
	cm := NewConfigManager(nil)
	cm.config = Config{
		APIPort:      8080,
		FeatureFlags: make(map[string]bool),
	}

	err := cm.Set("api_port", 9090)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	value, _ := cm.Get("api_port")
	if value != 9090 {
		t.Errorf("expected api_port to be 9090, got %v", value)
	}
}

// TestParseFeatureFlags verifies feature flag parsing
func TestParseFeatureFlags(t *testing.T) {
	flags := parseFeatureFlags("feature1=true,feature2=false,feature3=1")

	if !flags["feature1"] {
		t.Errorf("expected feature1 to be true")
	}

	if flags["feature2"] {
		t.Errorf("expected feature2 to be false")
	}

	if !flags["feature3"] {
		t.Errorf("expected feature3 to be true")
	}
}

// TestGetEnvWithDefault verifies environment variable retrieval with default
func TestGetEnvWithDefault(t *testing.T) {
	_ = os.Setenv("TEST_VAR", "test_value")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()

	value := env.Get("TEST_VAR", "default")
	if value != "test_value" {
		t.Errorf("expected 'test_value', got %s", value)
	}

	value = env.Get("NONEXISTENT_VAR", "default")
	if value != "default" {
		t.Errorf("expected 'default', got %s", value)
	}
}
