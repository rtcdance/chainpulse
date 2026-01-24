package deployment

import (
	"fmt"
	"os"
	"testing"
	"chainpulse/pkg/core"
)

// TestPropertyConfigurationValidation tests that configuration validation is consistent
// Feature: deployment-configuration, Property 25: Configuration Validation
// Validates: Requirements 6.2, 6.3
func TestPropertyConfigurationValidation(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Test that valid configurations always pass validation
	validConfigs := []core.Config{
		{
			DataPullerType:     "https-jsonrpc",
			BlockchainNodeURL:  "http://localhost:8545",
			MQType:             "kafka",
			MQConnectionURL:    "localhost:9092",
			CacheType:          "redis",
			CacheConnectionURL: "localhost:6379",
			CacheTTL:           3600,
			DatabaseType:       "postgres",
			DatabaseURL:        "postgres://localhost/chainpulse",
			APIType:            "rest",
			APIPort:            8080,
			WorkerPoolSize:     8,
			BatchSize:          100,
			MaxRetries:         3,
			RetryBackoff:       100,
			DeploymentMode:     "monolithic",
			ServiceName:        "chainpulse",
			LogLevel:           "info",
			FeatureFlags:       make(map[string]bool),
		},
		{
			DataPullerType:     "websocket",
			BlockchainNodeURL:  "ws://localhost:8546",
			MQType:             "redis",
			MQConnectionURL:    "localhost:6379",
			CacheType:          "memory",
			CacheConnectionURL: "localhost:6379",
			CacheTTL:           7200,
			DatabaseType:       "mongodb",
			DatabaseURL:        "mongodb://localhost:27017",
			APIType:            "grpc",
			APIPort:            50051,
			WorkerPoolSize:     16,
			BatchSize:          256,
			MaxRetries:         5,
			RetryBackoff:       200,
			DeploymentMode:     "microservice",
			ServiceName:        "event-processor",
			LogLevel:           "debug",
			FeatureFlags:       make(map[string]bool),
		},
		{
			DataPullerType:     "grpc",
			BlockchainNodeURL:  "grpc://localhost:50052",
			MQType:             "zeromq",
			MQConnectionURL:    "tcp://localhost:5555",
			CacheType:          "redis",
			CacheConnectionURL: "localhost:6379",
			CacheTTL:           1800,
			DatabaseType:       "postgres",
			DatabaseURL:        "postgres://localhost/chainpulse",
			APIType:            "websocket",
			APIPort:            8081,
			WorkerPoolSize:     4,
			BatchSize:          50,
			MaxRetries:         2,
			RetryBackoff:       50,
			DeploymentMode:     "monolithic",
			ServiceName:        "api-gateway",
			LogLevel:           "warn",
			FeatureFlags:       make(map[string]bool),
		},
	}

	for i, config := range validConfigs {
		err := cm.Validate(config)
		if err != nil {
			t.Errorf("valid config %d failed validation: %v", i, err)
		}
	}
}

// TestPropertyFeatureFlagConsistency tests that feature flags are consistent
// Feature: deployment-configuration, Property 27: Feature Flag Support
// Validates: Requirements 6.5
func TestPropertyFeatureFlagConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Test that setting and getting feature flags is consistent
	testFlags := map[string]bool{
		"enable_cache":    true,
		"enable_metrics":  false,
		"enable_tracing":  true,
		"enable_profiling": false,
	}

	for flag, value := range testFlags {
		err := cm.SetFeatureFlag(flag, value)
		if err != nil {
			t.Errorf("failed to set feature flag %s: %v", flag, err)
		}

		if cm.IsFeatureFlagEnabled(flag) != value {
			t.Errorf("feature flag %s: expected %v, got %v", flag, value, cm.IsFeatureFlagEnabled(flag))
		}
	}

	// Test that GetFeatureFlags returns all flags
	flags := cm.GetFeatureFlags()
	for flag, value := range testFlags {
		if flags[flag] != value {
			t.Errorf("GetFeatureFlags: flag %s expected %v, got %v", flag, value, flags[flag])
		}
	}
}

// TestPropertyConfigurationImmutability tests that configuration changes don't affect other instances
// Feature: deployment-configuration, Property 24: Configuration Loading
// Validates: Requirements 6.1
func TestPropertyConfigurationImmutability(t *testing.T) {
	if err := os.Setenv("API_PORT", "8080"); err != nil {
		t.Fatalf("failed to set API_PORT: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("API_PORT"); err != nil {
			t.Logf("failed to unset API_PORT: %v", err)
		}
	}()

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm1 := core.NewConfigManager(logger)
	cm2 := core.NewConfigManager(logger)

	// Load configuration in cm1
	config1, err := cm1.Load()
	if err != nil {
		t.Fatalf("failed to load configuration in cm1: %v", err)
	}

	// Load configuration in cm2
	config2, err := cm2.Load()
	if err != nil {
		t.Fatalf("failed to load configuration in cm2: %v", err)
	}

	// Both should have the same API port
	if config1.APIPort != config2.APIPort {
		t.Errorf("config1.APIPort=%d, config2.APIPort=%d, expected equal", config1.APIPort, config2.APIPort)
	}

	// Change API port in cm1
	err = cm1.Set("api_port", 9090)
	if err != nil {
		t.Fatalf("failed to set api_port in cm1: %v", err)
	}

	// cm2 should not be affected
	config2Updated := cm2.GetConfig()
	if config2Updated.APIPort != config1.APIPort {
		// This is expected - cm2 should still have the original value
		if config2Updated.APIPort != 8080 {
			t.Errorf("cm2 config should not be affected by cm1 changes")
		}
	}
}

// TestPropertyHotReloadIdempotence tests that hot reload is idempotent
// Feature: deployment-configuration, Property 26: Deployment Mode Support
// Validates: Requirements 6.4
func TestPropertyHotReloadIdempotence(t *testing.T) {
	if err := os.Setenv("DATA_PULLER_TYPE", "https-jsonrpc"); err != nil {
		t.Fatalf("failed to set DATA_PULLER_TYPE: %v", err)
	}
	if err := os.Setenv("BLOCKCHAIN_NODE_URL", "http://localhost:8545"); err != nil {
		t.Fatalf("failed to set BLOCKCHAIN_NODE_URL: %v", err)
	}
	if err := os.Setenv("MQ_TYPE", "kafka"); err != nil {
		t.Fatalf("failed to set MQ_TYPE: %v", err)
	}
	if err := os.Setenv("MQ_CONNECTION_URL", "localhost:9092"); err != nil {
		t.Fatalf("failed to set MQ_CONNECTION_URL: %v", err)
	}
	if err := os.Setenv("CACHE_TYPE", "redis"); err != nil {
		t.Fatalf("failed to set CACHE_TYPE: %v", err)
	}
	if err := os.Setenv("DATABASE_TYPE", "postgres"); err != nil {
		t.Fatalf("failed to set DATABASE_TYPE: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", "postgres://localhost/chainpulse"); err != nil {
		t.Fatalf("failed to set DATABASE_URL: %v", err)
	}
	if err := os.Setenv("API_TYPE", "rest"); err != nil {
		t.Fatalf("failed to set API_TYPE: %v", err)
	}
	if err := os.Setenv("API_PORT", "8080"); err != nil {
		t.Fatalf("failed to set API_PORT: %v", err)
	}
	if err := os.Setenv("WORKER_POOL_SIZE", "8"); err != nil {
		t.Fatalf("failed to set WORKER_POOL_SIZE: %v", err)
	}
	if err := os.Setenv("BATCH_SIZE", "100"); err != nil {
		t.Fatalf("failed to set BATCH_SIZE: %v", err)
	}
	if err := os.Setenv("MAX_RETRIES", "3"); err != nil {
		t.Fatalf("failed to set MAX_RETRIES: %v", err)
	}
	if err := os.Setenv("DEPLOYMENT_MODE", "monolithic"); err != nil {
		t.Fatalf("failed to set DEPLOYMENT_MODE: %v", err)
	}
	if err := os.Setenv("SERVICE_NAME", "chainpulse"); err != nil {
		t.Fatalf("failed to set SERVICE_NAME: %v", err)
	}
	if err := os.Setenv("LOG_LEVEL", "info"); err != nil {
		t.Fatalf("failed to set LOG_LEVEL: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("DATA_PULLER_TYPE"); err != nil {
			t.Logf("failed to unset DATA_PULLER_TYPE: %v", err)
		}
		if err := os.Unsetenv("BLOCKCHAIN_NODE_URL"); err != nil {
			t.Logf("failed to unset BLOCKCHAIN_NODE_URL: %v", err)
		}
		if err := os.Unsetenv("MQ_TYPE"); err != nil {
			t.Logf("failed to unset MQ_TYPE: %v", err)
		}
		if err := os.Unsetenv("MQ_CONNECTION_URL"); err != nil {
			t.Logf("failed to unset MQ_CONNECTION_URL: %v", err)
		}
		if err := os.Unsetenv("CACHE_TYPE"); err != nil {
			t.Logf("failed to unset CACHE_TYPE: %v", err)
		}
		if err := os.Unsetenv("DATABASE_TYPE"); err != nil {
			t.Logf("failed to unset DATABASE_TYPE: %v", err)
		}
		if err := os.Unsetenv("DATABASE_URL"); err != nil {
			t.Logf("failed to unset DATABASE_URL: %v", err)
		}
		if err := os.Unsetenv("API_TYPE"); err != nil {
			t.Logf("failed to unset API_TYPE: %v", err)
		}
		if err := os.Unsetenv("API_PORT"); err != nil {
			t.Logf("failed to unset API_PORT: %v", err)
		}
		if err := os.Unsetenv("WORKER_POOL_SIZE"); err != nil {
			t.Logf("failed to unset WORKER_POOL_SIZE: %v", err)
		}
		if err := os.Unsetenv("BATCH_SIZE"); err != nil {
			t.Logf("failed to unset BATCH_SIZE: %v", err)
		}
		if err := os.Unsetenv("MAX_RETRIES"); err != nil {
			t.Logf("failed to unset MAX_RETRIES: %v", err)
		}
		if err := os.Unsetenv("DEPLOYMENT_MODE"); err != nil {
			t.Logf("failed to unset DEPLOYMENT_MODE: %v", err)
		}
		if err := os.Unsetenv("SERVICE_NAME"); err != nil {
			t.Logf("failed to unset SERVICE_NAME: %v", err)
		}
		if err := os.Unsetenv("LOG_LEVEL"); err != nil {
			t.Logf("failed to unset LOG_LEVEL: %v", err)
		}
	}()

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Hot reload multiple times without changing environment
	config1, err := cm.HotReload()
	if err != nil {
		t.Fatalf("first hot reload failed: %v", err)
	}

	config2, err := cm.HotReload()
	if err != nil {
		t.Fatalf("second hot reload failed: %v", err)
	}

	config3, err := cm.HotReload()
	if err != nil {
		t.Fatalf("third hot reload failed: %v", err)
	}

	// All configurations should be identical
	if config1.DataPullerType != config2.DataPullerType || config2.DataPullerType != config3.DataPullerType {
		t.Errorf("hot reload is not idempotent: DataPullerType changed")
	}
	if config1.APIPort != config2.APIPort || config2.APIPort != config3.APIPort {
		t.Errorf("hot reload is not idempotent: APIPort changed")
	}
}

// TestPropertyConfigurationEnvironmentVariableParsing tests that environment variables are parsed correctly
// Feature: deployment-configuration, Property 24: Configuration Loading
// Validates: Requirements 6.1
func TestPropertyConfigurationEnvironmentVariableParsing(t *testing.T) {
	testCases := []struct {
		envVar   string
		envValue string
		key      string
		expected interface{}
	}{
		{"API_PORT", "8080", "api_port", 8080},
		{"API_PORT", "50051", "api_port", 50051},
		{"WORKER_POOL_SIZE", "16", "worker_pool_size", 16},
		{"BATCH_SIZE", "256", "batch_size", 256},
		{"MAX_RETRIES", "5", "max_retries", 5},
		{"DATA_PULLER_TYPE", "websocket", "data_puller_type", "websocket"},
		{"MQ_TYPE", "redis", "mq_type", "redis"},
		{"CACHE_TYPE", "memory", "cache_type", "memory"},
		{"DATABASE_TYPE", "mongodb", "database_type", "mongodb"},
		{"DEPLOYMENT_MODE", "microservice", "deployment_mode", "microservice"},
	}

	for _, tc := range testCases {
		if err := os.Setenv(tc.envVar, tc.envValue); err != nil {
			t.Logf("failed to set %s: %v", tc.envVar, err)
		}
		defer func(envVar string) {
			if err := os.Unsetenv(envVar); err != nil {
				t.Logf("failed to unset %s: %v", envVar, err)
			}
		}(tc.envVar)

		// Set required environment variables
		if err := os.Setenv("BLOCKCHAIN_NODE_URL", "http://localhost:8545"); err != nil {
			t.Logf("failed to set BLOCKCHAIN_NODE_URL: %v", err)
		}
		if err := os.Setenv("MQ_CONNECTION_URL", "localhost:9092"); err != nil {
			t.Logf("failed to set MQ_CONNECTION_URL: %v", err)
		}
		if err := os.Setenv("CACHE_CONNECTION_URL", "localhost:6379"); err != nil {
			t.Logf("failed to set CACHE_CONNECTION_URL: %v", err)
		}
		if err := os.Setenv("DATABASE_URL", "postgres://localhost/chainpulse"); err != nil {
			t.Logf("failed to set DATABASE_URL: %v", err)
		}
		if err := os.Setenv("SERVICE_NAME", "chainpulse"); err != nil {
			t.Logf("failed to set SERVICE_NAME: %v", err)
		}
		if err := os.Setenv("LOG_LEVEL", "info"); err != nil {
			t.Logf("failed to set LOG_LEVEL: %v", err)
		}

		defer func() {
			if err := os.Unsetenv("BLOCKCHAIN_NODE_URL"); err != nil {
				t.Logf("failed to unset BLOCKCHAIN_NODE_URL: %v", err)
			}
			if err := os.Unsetenv("MQ_CONNECTION_URL"); err != nil {
				t.Logf("failed to unset MQ_CONNECTION_URL: %v", err)
			}
			if err := os.Unsetenv("CACHE_CONNECTION_URL"); err != nil {
				t.Logf("failed to unset CACHE_CONNECTION_URL: %v", err)
			}
			if err := os.Unsetenv("DATABASE_URL"); err != nil {
				t.Logf("failed to unset DATABASE_URL: %v", err)
			}
			if err := os.Unsetenv("SERVICE_NAME"); err != nil {
				t.Logf("failed to unset SERVICE_NAME: %v", err)
			}
			if err := os.Unsetenv("LOG_LEVEL"); err != nil {
				t.Logf("failed to unset LOG_LEVEL: %v", err)
			}
		}()

		logger := core.NewDefaultLogger(core.LogLevelInfo)
		cm := core.NewConfigManager(logger)

		_, err := cm.Load()
		if err != nil {
			t.Fatalf("failed to load configuration: %v", err)
		}

		val, err := cm.Get(tc.key)
		if err != nil {
			t.Fatalf("failed to get configuration value for %s: %v", tc.key, err)
		}

		if val != tc.expected {
			t.Errorf("environment variable %s=%s: expected %v, got %v", tc.envVar, tc.envValue, tc.expected, val)
		}
	}
}

// TestPropertyConfigurationDefaultValues tests that default values are used when environment variables are not set
// Feature: deployment-configuration, Property 24: Configuration Loading
// Validates: Requirements 6.1
func TestPropertyConfigurationDefaultValues(t *testing.T) {
	// Clear all environment variables
	_ = os.Unsetenv("DATA_PULLER_TYPE")
	_ = os.Unsetenv("BLOCKCHAIN_NODE_URL")
	_ = os.Unsetenv("MQ_TYPE")
	_ = os.Unsetenv("MQ_CONNECTION_URL")
	_ = os.Unsetenv("CACHE_TYPE")
	_ = os.Unsetenv("CACHE_CONNECTION_URL")
	_ = os.Unsetenv("DATABASE_TYPE")
	_ = os.Unsetenv("DATABASE_URL")
	_ = os.Unsetenv("API_TYPE")
	_ = os.Unsetenv("API_PORT")
	_ = os.Unsetenv("WORKER_POOL_SIZE")
	_ = os.Unsetenv("BATCH_SIZE")
	_ = os.Unsetenv("MAX_RETRIES")
	_ = os.Unsetenv("RETRY_BACKOFF")
	_ = os.Unsetenv("DEPLOYMENT_MODE")
	_ = os.Unsetenv("SERVICE_NAME")
	_ = os.Unsetenv("LOG_LEVEL")

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	config, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Check default values
	if config.DataPullerType != "https-jsonrpc" {
		t.Errorf("expected default DataPullerType=https-jsonrpc, got %s", config.DataPullerType)
	}
	if config.MQType != "kafka" {
		t.Errorf("expected default MQType=kafka, got %s", config.MQType)
	}
	if config.CacheType != "redis" {
		t.Errorf("expected default CacheType=redis, got %s", config.CacheType)
	}
	if config.DatabaseType != "postgres" {
		t.Errorf("expected default DatabaseType=postgres, got %s", config.DatabaseType)
	}
	if config.APIType != "rest" {
		t.Errorf("expected default APIType=rest, got %s", config.APIType)
	}
	if config.APIPort != core.DefaultAPIPort {
		t.Errorf("expected default APIPort=%d, got %d", core.DefaultAPIPort, config.APIPort)
	}
	if config.WorkerPoolSize != core.DefaultWorkerPoolSize {
		t.Errorf("expected default WorkerPoolSize=%d, got %d", core.DefaultWorkerPoolSize, config.WorkerPoolSize)
	}
	if config.BatchSize != core.DefaultBatchSize {
		t.Errorf("expected default BatchSize=%d, got %d", core.DefaultBatchSize, config.BatchSize)
	}
	if config.MaxRetries != core.DefaultMaxRetries {
		t.Errorf("expected default MaxRetries=%d, got %d", core.DefaultMaxRetries, config.MaxRetries)
	}
	if config.DeploymentMode != "monolithic" {
		t.Errorf("expected default DeploymentMode=monolithic, got %s", config.DeploymentMode)
	}
	if config.ServiceName != "chainpulse" {
		t.Errorf("expected default ServiceName=chainpulse, got %s", config.ServiceName)
	}
	if config.LogLevel != "info" {
		t.Errorf("expected default LogLevel=info, got %s", config.LogLevel)
	}
}

// TestPropertyConfigurationThreadSafety tests that configuration is thread-safe
// Feature: deployment-configuration, Property 25: Configuration Validation
// Validates: Requirements 6.2, 6.3
func TestPropertyConfigurationThreadSafety(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Concurrent reads and writes
	done := make(chan bool, 10)

	// Readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = cm.GetConfig()
				_, _ = cm.Get("api_port")
				_ = cm.IsFeatureFlagEnabled("test_flag")
			}
			done <- true
		}()
	}

	// Writers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				_ = cm.SetFeatureFlag(fmt.Sprintf("flag_%d", id), j%2 == 0)
				_ = cm.Set("api_port", 8080+j%100)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestPropertyConfigurationValidationErrorMessages tests that validation errors have clear messages
// Feature: deployment-configuration, Property 25: Configuration Validation
// Validates: Requirements 6.2, 6.3
func TestPropertyConfigurationValidationErrorMessages(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	invalidConfigs := []struct {
		name   string
		config core.Config
	}{
		{
			name: "empty data puller type",
			config: core.Config{
				DataPullerType:     "",
				BlockchainNodeURL:  "http://localhost:8545",
				MQType:             "kafka",
				MQConnectionURL:    "localhost:9092",
				CacheType:          "redis",
				CacheConnectionURL: "localhost:6379",
				CacheTTL:           3600,
				DatabaseType:       "postgres",
				DatabaseURL:        "postgres://localhost/chainpulse",
				APIType:            "rest",
				APIPort:            8080,
				WorkerPoolSize:     8,
				BatchSize:          100,
				MaxRetries:         3,
				RetryBackoff:       100,
				DeploymentMode:     "monolithic",
				ServiceName:        "chainpulse",
				LogLevel:           "info",
				FeatureFlags:       make(map[string]bool),
			},
		},
		{
			name: "invalid api port",
			config: core.Config{
				DataPullerType:     "https-jsonrpc",
				BlockchainNodeURL:  "http://localhost:8545",
				MQType:             "kafka",
				MQConnectionURL:    "localhost:9092",
				CacheType:          "redis",
				CacheConnectionURL: "localhost:6379",
				CacheTTL:           3600,
				DatabaseType:       "postgres",
				DatabaseURL:        "postgres://localhost/chainpulse",
				APIType:            "rest",
				APIPort:            99999,
				WorkerPoolSize:     8,
				BatchSize:          100,
				MaxRetries:         3,
				RetryBackoff:       100,
				DeploymentMode:     "monolithic",
				ServiceName:        "chainpulse",
				LogLevel:           "info",
				FeatureFlags:       make(map[string]bool),
			},
		},
	}

	for _, tc := range invalidConfigs {
		err := cm.Validate(tc.config)
		if err == nil {
			t.Errorf("expected validation error for %s, got nil", tc.name)
		}

		// Check that error message is not empty
		if err.Error() == "" {
			t.Errorf("validation error for %s has empty message", tc.name)
		}
	}
}
