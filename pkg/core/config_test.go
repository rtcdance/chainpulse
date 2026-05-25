package core

import (
	"os"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/env"
)

type testLogger struct{}

func (l *testLogger) Debug(msg string, keysAndValues ...any) {}
func (l *testLogger) Info(msg string, keysAndValues ...any)  {}
func (l *testLogger) Warn(msg string, keysAndValues ...any)  {}
func (l *testLogger) Error(msg string, keysAndValues ...any) {}
func (l *testLogger) Fatal(msg string, keysAndValues ...any) {}
func (l *testLogger) WithCorrelationID(id string) Logger     { return l }

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
	err := cm.Validate(config)
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
	err := cm.Validate(config)

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
	err := cm.Validate(config)

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
	err := cm.Validate(config)

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

func TestParseFeatureFlagsEdgeCases(t *testing.T) {
	flags := parseFeatureFlags("")
	if len(flags) != 0 {
		t.Errorf("expected empty flags for empty string, got %d", len(flags))
	}

	flags = parseFeatureFlags("key1=true,,key2=false")
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}

	flags = parseFeatureFlags("key_without_value")
	if len(flags) != 0 {
		t.Errorf("expected no flags for malformed entry, got %d", len(flags))
	}

	flags = parseFeatureFlags("  key1 = true , key2 = 0 ")
	if !flags["key1"] {
		t.Error("expected key1 to be true")
	}
	if flags["key2"] {
		t.Error("expected key2 to be false (0 != true/1)")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("expected contains 'a' to be true")
	}
	if !contains(slice, "c") {
		t.Error("expected contains 'c' to be true")
	}
	if contains(slice, "d") {
		t.Error("expected contains 'd' to be false")
	}
	if contains(nil, "x") {
		t.Error("expected contains nil to be false")
	}
	if contains([]string{}, "x") {
		t.Error("expected contains empty to be false")
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

func TestCheckRequired(t *testing.T) {
	if err := checkRequired("", "Field"); err == nil {
		t.Error("expected error for empty required field")
	}
	if err := checkRequired("value", "Field"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckOneOf(t *testing.T) {
	allowed := []string{"a", "b", "c"}
	if err := checkOneOf("b", allowed, "Field"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkOneOf("d", allowed, "Field"); err == nil {
		t.Error("expected error for value not in allowed list")
	}
}

func TestCheckPositive(t *testing.T) {
	if err := checkPositive(0, "Field"); err == nil {
		t.Error("expected error for zero value")
	}
	if err := checkPositive(-1, "Field"); err == nil {
		t.Error("expected error for negative value")
	}
	if err := checkPositive(1, "Field"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckPort(t *testing.T) {
	if err := checkPort(0, "Port"); err == nil {
		t.Error("expected error for port 0")
	}
	if err := checkPort(-1, "Port"); err == nil {
		t.Error("expected error for negative port")
	}
	if err := checkPort(65536, "Port"); err == nil {
		t.Error("expected error for port > 65535")
	}
	if err := checkPort(8080, "Port"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkPort(65535, "Port"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckURLScheme(t *testing.T) {
	if err := checkURLScheme("", "http", "URL"); err != nil {
		t.Errorf("unexpected error for empty URL: %v", err)
	}
	if err := checkURLScheme("not-a-valid-url://", "", "URL"); err == nil {
		t.Error("expected error for invalid URL")
	}
	if err := checkURLScheme("http://example.com", "https", "URL"); err == nil {
		t.Error("expected error for wrong scheme")
	}
	if err := checkURLScheme("http://example.com", "http", "URL"); err != nil {
		t.Errorf("unexpected error for valid URL: %v", err)
	}
	if err := checkURLScheme("http://", "", "URL"); err == nil {
		t.Error("expected error for URL with no host")
	}
}

func TestCheckFileExists(t *testing.T) {
	if err := checkFileExists("", "File"); err != nil {
		t.Errorf("unexpected error for empty path: %v", err)
	}
	if err := checkFileExists("/nonexistent/path/file.txt", "File"); err == nil {
		t.Error("expected error for nonexistent file")
	}

	tmpFile, err := os.CreateTemp("", "config_test_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if err := checkFileExists(tmpFile.Name(), "File"); err != nil {
		t.Errorf("unexpected error for existing file: %v", err)
	}

	tmpDir := t.TempDir()
	if err := checkFileExists(tmpDir, "File"); err == nil {
		t.Error("expected error for directory path")
	}
}

func TestCheckMinLength(t *testing.T) {
	if err := checkMinLength("ab", 3, "Field"); err == nil {
		t.Error("expected error for string shorter than min length")
	}
	if err := checkMinLength("abc", 3, "Field"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigManagerExtensions(t *testing.T) {
	cm := NewConfigManager(nil)

	cm.config.Blockchains = map[string]BlockchainConfig{
		"ethereum": {ChainID: "1"},
		"polygon":  {ChainID: "137"},
	}
	cm.config.ActiveChains = []string{"ethereum", "polygon"}
	cm.config.FeatureFlags = map[string]bool{"new_indexer": true}

	cfg, err := cm.GetBlockchainConfig("ethereum")
	if err != nil {
		t.Errorf("GetBlockchainConfig error: %v", err)
	}
	if cfg.ChainID != "1" {
		t.Errorf("ChainID = %s, want 1", cfg.ChainID)
	}
	_, err = cm.GetBlockchainConfig("unknown")
	if err == nil {
		t.Error("expected error for unknown chain")
	}

	if !cm.IsMultiChain() {
		t.Error("expected IsMultiChain to be true")
	}

	chains := cm.GetActiveChains()
	if len(chains) != 2 {
		t.Errorf("GetActiveChains len = %d", len(chains))
	}

	allCfgs := cm.GetAllBlockchainConfigs()
	if len(allCfgs) != 2 {
		t.Errorf("GetAllBlockchainConfigs len = %d", len(allCfgs))
	}

	if err := cm.SetFeatureFlag("", true); err == nil {
		t.Error("expected error for empty flag name")
	}
	if err := cm.SetFeatureFlag("debug_mode", true); err != nil {
		t.Errorf("SetFeatureFlag error: %v", err)
	}

	if !cm.IsFeatureFlagEnabled("debug_mode") {
		t.Error("expected debug_mode to be enabled")
	}

	flags := cm.GetFeatureFlags()
	if len(flags) != 2 {
		t.Errorf("GetFeatureFlags len = %d, want 2", len(flags))
	}

	cm.SetHotReloadEnabled(false)
	cm.SetHotReloadEnabled(true)

	loadTime := cm.GetLastLoadTime()
	if loadTime.IsZero() {
		t.Error("GetLastLoadTime should not be zero")
	}
}

func TestConfigValidate(t *testing.T) {
	cm := NewConfigManager(nil)
	cfg := Config{
		DataPullerType:    "https-jsonrpc",
		BlockchainNodeURL: "http://localhost:8545",
		MQType:            "kafka",
		MQConnectionURL:   "kafka://localhost:9092",
		CacheType:         "memory",
		CacheTTL:          3600,
		DatabaseType:      "postgres",
		DatabaseURL:       "postgres://localhost:5432/db",
		APIType:           "rest",
		APIPort:           8080,
		WorkerPoolSize:    10,
		BatchSize:         100,
		MaxRetries:        3,
		RetryBackoff:      100,
		DeploymentMode:    "monolithic",
		ServiceName:       "test-service",
		LogLevel:          "info",
	}
	if err := cm.Validate(cfg); err != nil {
		t.Errorf("Validate error for valid config: %v", err)
	}

	invalidCfg := cfg
	invalidCfg.APIPort = 0
	if err := cm.Validate(invalidCfg); err == nil {
		t.Error("expected validation error for invalid port")
	}
}

func TestGetConfig(t *testing.T) {
	cm := NewConfigManager(nil)
	cm.config.APIPort = 9090
	cfg := cm.GetConfig()
	if cfg.APIPort != 9090 {
		t.Errorf("APIPort = %d, want 9090", cfg.APIPort)
	}
}

func TestOnConfigChange(t *testing.T) {
	cm := NewConfigManager(nil)
	var called bool
	cm.OnConfigChange(func(c Config) {
		called = true
	})
	if cm.configChangeListeners == nil {
		t.Error("configChangeListeners should not be nil")
	}
	if len(cm.configChangeListeners) != 1 {
		t.Errorf("expected 1 listener, got %d", len(cm.configChangeListeners))
	}
	_ = called
}

func TestHotReloadDisabled(t *testing.T) {
	cm := NewConfigManager(nil)
	cm.SetHotReloadEnabled(false)
	cm.config.APIPort = 8080
	cfg, err := cm.HotReload()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.APIPort != 8080 {
		t.Errorf("APIPort = %d, want 8080", cfg.APIPort)
	}
}

func TestHotReloadEnabled(t *testing.T) {
	cm := NewConfigManager(&testLogger{})
	cm.SetHotReloadEnabled(true)
	cfg, err := cm.HotReload()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.DataPullerType == "" {
		t.Error("DataPullerType should not be empty")
	}
}

func TestValidateLoaded(t *testing.T) {
	cm := NewConfigManager(nil)

	cm.config.DatabaseURL = ""
	cm.config.BlockchainNodeURL = ""
	err := cm.ValidateLoaded()
	if err == nil {
		t.Error("expected validation error for empty config")
	}

	cm.config.DatabaseURL = "postgres://localhost/db"
	cm.config.BlockchainNodeURL = "http://localhost:8545"
	cm.config.APIPort = 8080
	cm.config.WorkerPoolSize = 10
	cm.config.BatchSize = 100
	cm.config.MaxRetries = 3
	err = cm.ValidateLoaded()
	if err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}
}

func TestErrTypeError(t *testing.T) {
	err := errTypeError("test_key", "int")
	if err == nil {
		t.Error("expected error")
	}
	code := ClassifyErrorCode(err)
	if code == "" {
		t.Error("expected non-empty error code")
	}
}
