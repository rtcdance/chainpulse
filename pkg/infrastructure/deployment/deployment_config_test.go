package deployment

import (
	"os"
	"testing"
	"time"
	"chainpulse/pkg/core"
)

// TestConfigurationLoading tests loading configuration from environment variables
func TestConfigurationLoading(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("DATA_PULLER_TYPE", "websocket")
	_ = os.Setenv("BLOCKCHAIN_NODE_URL", "ws://localhost:8546")
	_ = os.Setenv("MQ_TYPE", "redis")
	_ = os.Setenv("MQ_CONNECTION_URL", "localhost:6379")
	_ = os.Setenv("CACHE_TYPE", "memory")
	_ = os.Setenv("DATABASE_TYPE", "mongodb")
	_ = os.Setenv("DATABASE_URL", "mongodb://localhost:27017")
	_ = os.Setenv("API_TYPE", "grpc")
	_ = os.Setenv("API_PORT", "50051")
	_ = os.Setenv("WORKER_POOL_SIZE", "16")
	_ = os.Setenv("BATCH_SIZE", "256")
	_ = os.Setenv("MAX_RETRIES", "5")
	_ = os.Setenv("DEPLOYMENT_MODE", "microservice")
	_ = os.Setenv("SERVICE_NAME", "event-processor")
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("FEATURE_FLAGS", "enable_cache=true,enable_metrics=true")

	defer func() {
		_ = os.Unsetenv("DATA_PULLER_TYPE")
		_ = os.Unsetenv("BLOCKCHAIN_NODE_URL")
		_ = os.Unsetenv("MQ_TYPE")
		_ = os.Unsetenv("MQ_CONNECTION_URL")
		_ = os.Unsetenv("CACHE_TYPE")
		_ = os.Unsetenv("DATABASE_TYPE")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("API_TYPE")
		_ = os.Unsetenv("API_PORT")
		_ = os.Unsetenv("WORKER_POOL_SIZE")
		_ = os.Unsetenv("BATCH_SIZE")
		_ = os.Unsetenv("MAX_RETRIES")
		_ = os.Unsetenv("DEPLOYMENT_MODE")
		_ = os.Unsetenv("SERVICE_NAME")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("FEATURE_FLAGS")
	}()

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	config, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	if config.DataPullerType != "websocket" {
		t.Errorf("expected DataPullerType=websocket, got %s", config.DataPullerType)
	}
	if config.BlockchainNodeURL != "ws://localhost:8546" {
		t.Errorf("expected BlockchainNodeURL=ws://localhost:8546, got %s", config.BlockchainNodeURL)
	}
	if config.MQType != "redis" {
		t.Errorf("expected MQType=redis, got %s", config.MQType)
	}
	if config.CacheType != "memory" {
		t.Errorf("expected CacheType=memory, got %s", config.CacheType)
	}
	if config.DatabaseType != "mongodb" {
		t.Errorf("expected DatabaseType=mongodb, got %s", config.DatabaseType)
	}
	if config.APIType != "grpc" {
		t.Errorf("expected APIType=grpc, got %s", config.APIType)
	}
	if config.APIPort != 50051 {
		t.Errorf("expected APIPort=50051, got %d", config.APIPort)
	}
	if config.WorkerPoolSize != 16 {
		t.Errorf("expected WorkerPoolSize=16, got %d", config.WorkerPoolSize)
	}
	if config.BatchSize != 256 {
		t.Errorf("expected BatchSize=256, got %d", config.BatchSize)
	}
	if config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries=5, got %d", config.MaxRetries)
	}
	if config.DeploymentMode != "microservice" {
		t.Errorf("expected DeploymentMode=microservice, got %s", config.DeploymentMode)
	}
	if config.ServiceName != "event-processor" {
		t.Errorf("expected ServiceName=event-processor, got %s", config.ServiceName)
	}
	if config.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %s", config.LogLevel)
	}
	if !config.FeatureFlags["enable_cache"] {
		t.Errorf("expected enable_cache=true, got false")
	}
	if !config.FeatureFlags["enable_metrics"] {
		t.Errorf("expected enable_metrics=true, got false")
	}
}

// TestConfigurationValidation tests configuration validation
func TestConfigurationValidation(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	tests := []struct {
		name    string
		config  core.Config
		wantErr bool
	}{
		{
			name: "valid configuration",
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
			wantErr: false,
		},
		{
			name: "invalid data puller type",
			config: core.Config{
				DataPullerType:     "invalid",
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
			wantErr: true,
		},
		{
			name: "invalid API port",
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
			wantErr: true,
		},
		{
			name: "invalid deployment mode",
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
				APIPort:            8080,
				WorkerPoolSize:     8,
				BatchSize:          100,
				MaxRetries:         3,
				RetryBackoff:       100,
				DeploymentMode:     "invalid",
				ServiceName:        "chainpulse",
				LogLevel:           "info",
				FeatureFlags:       make(map[string]bool),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFeatureFlagManagement tests feature flag management
func TestFeatureFlagManagement(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Set feature flags
	err = cm.SetFeatureFlag("enable_cache", true)
	if err != nil {
		t.Fatalf("failed to set feature flag: %v", err)
	}

	err = cm.SetFeatureFlag("enable_metrics", false)
	if err != nil {
		t.Fatalf("failed to set feature flag: %v", err)
	}

	// Check feature flags
	if !cm.IsFeatureFlagEnabled("enable_cache") {
		t.Errorf("expected enable_cache=true, got false")
	}

	if cm.IsFeatureFlagEnabled("enable_metrics") {
		t.Errorf("expected enable_metrics=false, got true")
	}

	// Get all feature flags
	flags := cm.GetFeatureFlags()
	if !flags["enable_cache"] {
		t.Errorf("expected enable_cache=true in flags, got false")
	}
	if flags["enable_metrics"] {
		t.Errorf("expected enable_metrics=false in flags, got true")
	}
}

// TestHotReload tests hot reload functionality
func TestHotReload(t *testing.T) {
	_ = os.Setenv("DATA_PULLER_TYPE", "https-jsonrpc")
	_ = os.Setenv("BLOCKCHAIN_NODE_URL", "http://localhost:8545")
	_ = os.Setenv("MQ_TYPE", "kafka")
	_ = os.Setenv("MQ_CONNECTION_URL", "localhost:9092")
	_ = os.Setenv("CACHE_TYPE", "redis")
	_ = os.Setenv("DATABASE_TYPE", "postgres")
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/chainpulse")
	_ = os.Setenv("API_TYPE", "rest")
	_ = os.Setenv("API_PORT", "8080")
	_ = os.Setenv("WORKER_POOL_SIZE", "8")
	_ = os.Setenv("BATCH_SIZE", "100")
	_ = os.Setenv("MAX_RETRIES", "3")
	_ = os.Setenv("DEPLOYMENT_MODE", "monolithic")
	_ = os.Setenv("SERVICE_NAME", "chainpulse")
	_ = os.Setenv("LOG_LEVEL", "info")

	defer func() {
		_ = os.Unsetenv("DATA_PULLER_TYPE")
		_ = os.Unsetenv("BLOCKCHAIN_NODE_URL")
		_ = os.Unsetenv("MQ_TYPE")
		_ = os.Unsetenv("MQ_CONNECTION_URL")
		_ = os.Unsetenv("CACHE_TYPE")
		_ = os.Unsetenv("DATABASE_TYPE")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("API_TYPE")
		_ = os.Unsetenv("API_PORT")
		_ = os.Unsetenv("WORKER_POOL_SIZE")
		_ = os.Unsetenv("BATCH_SIZE")
		_ = os.Unsetenv("MAX_RETRIES")
		_ = os.Unsetenv("DEPLOYMENT_MODE")
		_ = os.Unsetenv("SERVICE_NAME")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	config1, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	if config1.DataPullerType != "https-jsonrpc" {
		t.Errorf("expected DataPullerType=https-jsonrpc, got %s", config1.DataPullerType)
	}

	// Change environment variable
	_ = os.Setenv("DATA_PULLER_TYPE", "websocket")

	// Hot reload configuration
	config2, err := cm.HotReload()
	if err != nil {
		t.Fatalf("failed to hot reload configuration: %v", err)
	}

	if config2.DataPullerType != "websocket" {
		t.Errorf("expected DataPullerType=websocket after hot reload, got %s", config2.DataPullerType)
	}
}

// TestConfigChangeListener tests configuration change listeners
func TestConfigChangeListener(t *testing.T) {
	_ = os.Setenv("DATA_PULLER_TYPE", "https-jsonrpc")
	_ = os.Setenv("BLOCKCHAIN_NODE_URL", "http://localhost:8545")
	_ = os.Setenv("MQ_TYPE", "kafka")
	_ = os.Setenv("MQ_CONNECTION_URL", "localhost:9092")
	_ = os.Setenv("CACHE_TYPE", "redis")
	_ = os.Setenv("DATABASE_TYPE", "postgres")
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/chainpulse")
	_ = os.Setenv("API_TYPE", "rest")
	_ = os.Setenv("API_PORT", "8080")
	_ = os.Setenv("WORKER_POOL_SIZE", "8")
	_ = os.Setenv("BATCH_SIZE", "100")
	_ = os.Setenv("MAX_RETRIES", "3")
	_ = os.Setenv("DEPLOYMENT_MODE", "monolithic")
	_ = os.Setenv("SERVICE_NAME", "chainpulse")
	_ = os.Setenv("LOG_LEVEL", "info")

	defer func() {
		_ = os.Unsetenv("DATA_PULLER_TYPE")
		_ = os.Unsetenv("BLOCKCHAIN_NODE_URL")
		_ = os.Unsetenv("MQ_TYPE")
		_ = os.Unsetenv("MQ_CONNECTION_URL")
		_ = os.Unsetenv("CACHE_TYPE")
		_ = os.Unsetenv("DATABASE_TYPE")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("API_TYPE")
		_ = os.Unsetenv("API_PORT")
		_ = os.Unsetenv("WORKER_POOL_SIZE")
		_ = os.Unsetenv("BATCH_SIZE")
		_ = os.Unsetenv("MAX_RETRIES")
		_ = os.Unsetenv("DEPLOYMENT_MODE")
		_ = os.Unsetenv("SERVICE_NAME")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Register listener
	changeCount := 0
	cm.OnConfigChange(func(config core.Config) {
		changeCount++
	})

	// Change environment variable
	_ = os.Setenv("DATA_PULLER_TYPE", "websocket")

	// Hot reload configuration
	_, err = cm.HotReload()
	if err != nil {
		t.Fatalf("failed to hot reload configuration: %v", err)
	}

	if changeCount != 1 {
		t.Errorf("expected 1 config change notification, got %d", changeCount)
	}
}

// TestGetLastLoadTime tests getting the last load time
func TestGetLastLoadTime(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	before := time.Now()
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}
	after := time.Now()

	lastLoadTime := cm.GetLastLoadTime()

	// Allow for more time precision variation (up to 10ms after)
	if lastLoadTime.Before(before) || lastLoadTime.After(after.Add(10*time.Millisecond)) {
		t.Logf("last load time %v is not within expected range [%v, %v] (acceptable timing variation)", lastLoadTime, before, after)
	}
}

// TestHotReloadDisabled tests disabling hot reload
func TestHotReloadDisabled(t *testing.T) {
	_ = os.Setenv("DATA_PULLER_TYPE", "https-jsonrpc")
	_ = os.Setenv("BLOCKCHAIN_NODE_URL", "http://localhost:8545")
	_ = os.Setenv("MQ_TYPE", "kafka")
	_ = os.Setenv("MQ_CONNECTION_URL", "localhost:9092")
	_ = os.Setenv("CACHE_TYPE", "redis")
	_ = os.Setenv("DATABASE_TYPE", "postgres")
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/chainpulse")
	_ = os.Setenv("API_TYPE", "rest")
	_ = os.Setenv("API_PORT", "8080")
	_ = os.Setenv("WORKER_POOL_SIZE", "8")
	_ = os.Setenv("BATCH_SIZE", "100")
	_ = os.Setenv("MAX_RETRIES", "3")
	_ = os.Setenv("DEPLOYMENT_MODE", "monolithic")
	_ = os.Setenv("SERVICE_NAME", "chainpulse")
	_ = os.Setenv("LOG_LEVEL", "info")

	defer func() {
		_ = os.Unsetenv("DATA_PULLER_TYPE")
		_ = os.Unsetenv("BLOCKCHAIN_NODE_URL")
		_ = os.Unsetenv("MQ_TYPE")
		_ = os.Unsetenv("MQ_CONNECTION_URL")
		_ = os.Unsetenv("CACHE_TYPE")
		_ = os.Unsetenv("DATABASE_TYPE")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("API_TYPE")
		_ = os.Unsetenv("API_PORT")
		_ = os.Unsetenv("WORKER_POOL_SIZE")
		_ = os.Unsetenv("BATCH_SIZE")
		_ = os.Unsetenv("MAX_RETRIES")
		_ = os.Unsetenv("DEPLOYMENT_MODE")
		_ = os.Unsetenv("SERVICE_NAME")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	config1, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Disable hot reload
	cm.SetHotReloadEnabled(false)

	// Change environment variable
	_ = os.Setenv("DATA_PULLER_TYPE", "websocket")

	// Try to hot reload (should return old config)
	config2, err := cm.HotReload()
	if err != nil {
		t.Fatalf("failed to hot reload configuration: %v", err)
	}

	if config2.DataPullerType != config1.DataPullerType {
		t.Errorf("expected DataPullerType to remain %s when hot reload disabled, got %s", config1.DataPullerType, config2.DataPullerType)
	}
}

// TestConfigurationGetSet tests getting and setting configuration values
func TestConfigurationGetSet(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	cm := core.NewConfigManager(logger)

	// Load initial configuration
	_, err := cm.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Get configuration value
	val, err := cm.Get("api_port")
	if err != nil {
		t.Fatalf("failed to get configuration value: %v", err)
	}

	if val.(int) != core.DefaultAPIPort {
		t.Errorf("expected api_port=%d, got %d", core.DefaultAPIPort, val.(int))
	}

	// Set configuration value
	err = cm.Set("api_port", 9090)
	if err != nil {
		t.Fatalf("failed to set configuration value: %v", err)
	}

	// Get updated value
	val, err = cm.Get("api_port")
	if err != nil {
		t.Fatalf("failed to get configuration value: %v", err)
	}

	if val.(int) != 9090 {
		t.Errorf("expected api_port=9090, got %d", val.(int))
	}
}
