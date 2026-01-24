package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BlockchainConfig represents configuration for a single blockchain
type BlockchainConfig struct {
	ChainID    string
	NodeURL    string
	StartBlock uint64
	ChainName  string
	Network    string
}

// DefaultConfigManager is the default implementation of ConfigManager
type DefaultConfigManager struct {
	config                Config
	mu                    sync.RWMutex
	logger                Logger
	configChangeListeners []func(Config)
	hotReloadEnabled      bool
	lastLoadTime          time.Time
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger Logger) *DefaultConfigManager {
	return &DefaultConfigManager{
		config: Config{
			FeatureFlags: make(map[string]bool),
		},
		logger:                logger,
		configChangeListeners: make([]func(Config), 0),
		hotReloadEnabled:      true,
		lastLoadTime:          time.Now(),
	}
}

// Load loads configuration from environment variables
func (cm *DefaultConfigManager) Load() (Config, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config := Config{
		// Data Puller Configuration
		DataPullerType:    getEnv("DATA_PULLER_TYPE", "https-jsonrpc"),
		BlockchainNodeURL: getEnv("BLOCKCHAIN_NODE_URL", "http://localhost:8545"),
		StartBlock:        getEnvUint64("START_BLOCK", 0),

		// Message Queue Configuration
		MQType:          getEnv("MQ_TYPE", "kafka"),
		MQConnectionURL: getEnv("MQ_CONNECTION_URL", "localhost:9092"),

		// Cache Configuration
		CacheType:          getEnv("CACHE_TYPE", "redis"),
		CacheConnectionURL: getEnv("CACHE_CONNECTION_URL", "localhost:6379"),
		CacheTTL:           getEnvInt("CACHE_TTL", DefaultCacheTTL),

		// Database Configuration
		DatabaseType: getEnv("DATABASE_TYPE", "postgres"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://localhost/chainpulse"),

		// API Configuration
		APIType: getEnv("API_TYPE", "rest"),
		APIPort: getEnvInt("API_PORT", DefaultAPIPort),

		// Processing Configuration
		WorkerPoolSize: getEnvInt("WORKER_POOL_SIZE", DefaultWorkerPoolSize),
		BatchSize:      getEnvInt("BATCH_SIZE", DefaultBatchSize),
		MaxRetries:     getEnvInt("MAX_RETRIES", DefaultMaxRetries),
		RetryBackoff:   getEnvInt("RETRY_BACKOFF", DefaultRetryBackoff),

		// Deployment Configuration
		DeploymentMode: getEnv("DEPLOYMENT_MODE", "monolithic"),
		ServiceName:    getEnv("SERVICE_NAME", "chainpulse"),

		// Logging Configuration
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// Feature Flags
		FeatureFlags: parseFeatureFlags(getEnv("FEATURE_FLAGS", "")),

		// Multi-blockchain Configuration
		Blockchains: make(map[string]BlockchainConfig),
		ActiveChains: make([]string, 0),
	}

	// Load multi-blockchain configuration
	chainsStr := getEnv("CHAINPULSE_CHAINS", "")
	if chainsStr != "" {
		chains := strings.Split(chainsStr, ",")
		for _, chain := range chains {
			chain = strings.TrimSpace(chain)
			if chain == "" {
				continue
			}

			// Load chain-specific configuration
			nodeURL := getEnv(fmt.Sprintf("CHAINPULSE_%s_NODE_URL", strings.ToUpper(chain)), "")
			chainID := getEnv(fmt.Sprintf("CHAINPULSE_%s_CHAIN_ID", strings.ToUpper(chain)), "")
			startBlockStr := getEnv(fmt.Sprintf("CHAINPULSE_%s_START_BLOCK", strings.ToUpper(chain)), "0")

			startBlock := uint64(0)
			if startBlockStr != "" {
				if block, err := strconv.ParseUint(startBlockStr, 10, 64); err == nil {
					startBlock = block
				}
			}

			if nodeURL != "" && chainID != "" {
				config.Blockchains[chain] = BlockchainConfig{
					ChainID:    chainID,
					NodeURL:    nodeURL,
					StartBlock: startBlock,
					ChainName:  chain,
					Network:    "devnet",
				}
				config.ActiveChains = append(config.ActiveChains, chain)
			}
		}
	}

	cm.config = config

	if cm.logger != nil {
		if len(config.ActiveChains) > 0 {
			cm.logger.Info("multi-blockchain configuration loaded",
				"active_chains", strings.Join(config.ActiveChains, ","),
				"chain_count", len(config.ActiveChains),
			)
		} else {
			cm.logger.Info("configuration loaded",
				"data_puller_type", config.DataPullerType,
				"mq_type", config.MQType,
				"cache_type", config.CacheType,
				"database_type", config.DatabaseType,
				"deployment_mode", config.DeploymentMode,
			)
		}
	}

	return config, nil
}

// Validate validates the configuration
func (cm *DefaultConfigManager) Validate(config Config) error {
	// Validate Data Puller Configuration
	if config.DataPullerType == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"DataPullerType is required",
			nil,
		)
	}

	validDataPullerTypes := []string{"https-jsonrpc", "websocket", "grpc"}
	if !contains(validDataPullerTypes, config.DataPullerType) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid DataPullerType: %s", config.DataPullerType),
			nil,
		)
	}

	if config.BlockchainNodeURL == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"BlockchainNodeURL is required",
			nil,
		)
	}

	// Validate Message Queue Configuration
	if config.MQType == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"MQType is required",
			nil,
		)
	}

	validMQTypes := []string{"kafka", "redis", "zeromq"}
	if !contains(validMQTypes, config.MQType) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid MQType: %s", config.MQType),
			nil,
		)
	}

	if config.MQConnectionURL == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"MQConnectionURL is required",
			nil,
		)
	}

	// Validate Cache Configuration
	if config.CacheType == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"CacheType is required",
			nil,
		)
	}

	validCacheTypes := []string{"redis", "memory"}
	if !contains(validCacheTypes, config.CacheType) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid CacheType: %s", config.CacheType),
			nil,
		)
	}

	if config.CacheTTL <= 0 {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"CacheTTL must be greater than 0",
			nil,
		)
	}

	// Validate Database Configuration
	if config.DatabaseType == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"DatabaseType is required",
			nil,
		)
	}

	validDatabaseTypes := []string{"postgres", "mongodb"}
	if !contains(validDatabaseTypes, config.DatabaseType) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid DatabaseType: %s", config.DatabaseType),
			nil,
		)
	}

	if config.DatabaseURL == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"DatabaseURL is required",
			nil,
		)
	}

	// Validate API Configuration
	if config.APIType == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"APIType is required",
			nil,
		)
	}

	validAPITypes := []string{"rest", "grpc", "websocket"}
	if !contains(validAPITypes, config.APIType) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid APIType: %s", config.APIType),
			nil,
		)
	}

	if config.APIPort <= 0 || config.APIPort > 65535 {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid APIPort: %d", config.APIPort),
			nil,
		)
	}

	// Validate Processing Configuration
	if config.WorkerPoolSize <= 0 {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"WorkerPoolSize must be greater than 0",
			nil,
		)
	}

	if config.BatchSize <= 0 {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"BatchSize must be greater than 0",
			nil,
		)
	}

	if config.MaxRetries < 0 {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"MaxRetries must be non-negative",
			nil,
		)
	}

	if config.RetryBackoff <= 0 {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"RetryBackoff must be greater than 0",
			nil,
		)
	}

	// Validate Deployment Configuration
	if config.DeploymentMode == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"DeploymentMode is required",
			nil,
		)
	}

	validDeploymentModes := []string{"monolithic", "microservice"}
	if !contains(validDeploymentModes, config.DeploymentMode) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid DeploymentMode: %s", config.DeploymentMode),
			nil,
		)
	}

	if config.ServiceName == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"ServiceName is required",
			nil,
		)
	}

	// Validate Log Level
	if config.LogLevel == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			"LogLevel is required",
			nil,
		)
	}

	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, config.LogLevel) {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeConfigError,
			fmt.Sprintf("invalid LogLevel: %s", config.LogLevel),
			nil,
		)
	}

	return nil
}

// Get retrieves a configuration value
func (cm *DefaultConfigManager) Get(key string) (interface{}, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	switch key {
	case "data_puller_type":
		return cm.config.DataPullerType, nil
	case "blockchain_node_url":
		return cm.config.BlockchainNodeURL, nil
	case "mq_type":
		return cm.config.MQType, nil
	case "cache_type":
		return cm.config.CacheType, nil
	case "database_type":
		return cm.config.DatabaseType, nil
	case "api_type":
		return cm.config.APIType, nil
	case "api_port":
		return cm.config.APIPort, nil
	case "worker_pool_size":
		return cm.config.WorkerPoolSize, nil
	case "batch_size":
		return cm.config.BatchSize, nil
	case "max_retries":
		return cm.config.MaxRetries, nil
	case "deployment_mode":
		return cm.config.DeploymentMode, nil
	case "log_level":
		return cm.config.LogLevel, nil
	default:
		return nil, NewSystemError(
			ErrorTypePermanent,
			ErrorCodeNotFound,
			fmt.Sprintf("configuration key not found: %s", key),
			nil,
		)
	}
}

// Set sets a configuration value
func (cm *DefaultConfigManager) Set(key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	switch key {
	case "data_puller_type":
		if v, ok := value.(string); ok {
			cm.config.DataPullerType = v
			return nil
		}
	case "mq_type":
		if v, ok := value.(string); ok {
			cm.config.MQType = v
			return nil
		}
	case "cache_type":
		if v, ok := value.(string); ok {
			cm.config.CacheType = v
			return nil
		}
	case "database_type":
		if v, ok := value.(string); ok {
			cm.config.DatabaseType = v
			return nil
		}
	case "api_type":
		if v, ok := value.(string); ok {
			cm.config.APIType = v
			return nil
		}
	case "api_port":
		if v, ok := value.(int); ok {
			cm.config.APIPort = v
			return nil
		}
	case "worker_pool_size":
		if v, ok := value.(int); ok {
			cm.config.WorkerPoolSize = v
			return nil
		}
	case "batch_size":
		if v, ok := value.(int); ok {
			cm.config.BatchSize = v
			return nil
		}
	case "max_retries":
		if v, ok := value.(int); ok {
			cm.config.MaxRetries = v
			return nil
		}
	case "deployment_mode":
		if v, ok := value.(string); ok {
			cm.config.DeploymentMode = v
			return nil
		}
	case "log_level":
		if v, ok := value.(string); ok {
			cm.config.LogLevel = v
			return nil
		}
	}

	return NewSystemError(
		ErrorTypePermanent,
		ErrorCodeValidation,
		fmt.Sprintf("invalid configuration key or value type: %s", key),
		nil,
	)
}

// GetConfig returns the current configuration
func (cm *DefaultConfigManager) GetConfig() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config
}

// OnConfigChange registers a listener for configuration changes
func (cm *DefaultConfigManager) OnConfigChange(listener func(Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.configChangeListeners = append(cm.configChangeListeners, listener)
}

// HotReload reloads configuration from environment variables
func (cm *DefaultConfigManager) HotReload() (Config, error) {
	if !cm.hotReloadEnabled {
		return cm.GetConfig(), nil
	}

	newConfig, err := cm.Load()
	if err != nil {
		return Config{}, err
	}

	// Notify listeners of configuration change
	cm.mu.RLock()
	listeners := make([]func(Config), len(cm.configChangeListeners))
	copy(listeners, cm.configChangeListeners)
	cm.mu.RUnlock()

	for _, listener := range listeners {
		listener(newConfig)
	}

	return newConfig, nil
}

// SetFeatureFlag sets a feature flag value
func (cm *DefaultConfigManager) SetFeatureFlag(flag string, enabled bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if flag == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"feature flag name cannot be empty",
			nil,
		)
	}

	cm.config.FeatureFlags[flag] = enabled

	if cm.logger != nil {
		cm.logger.Info("feature flag updated",
			"flag", flag,
			"enabled", enabled,
		)
	}

	return nil
}

// IsFeatureFlagEnabled checks if a feature flag is enabled
func (cm *DefaultConfigManager) IsFeatureFlagEnabled(flag string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.FeatureFlags[flag]
}

// GetFeatureFlags returns all feature flags
func (cm *DefaultConfigManager) GetFeatureFlags() map[string]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	flags := make(map[string]bool)
	for k, v := range cm.config.FeatureFlags {
		flags[k] = v
	}
	return flags
}

// SetHotReloadEnabled enables or disables hot reload
func (cm *DefaultConfigManager) SetHotReloadEnabled(enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.hotReloadEnabled = enabled

	if cm.logger != nil {
		cm.logger.Info("hot reload configuration",
			"enabled", enabled,
		)
	}
}

// GetLastLoadTime returns the last time configuration was loaded
func (cm *DefaultConfigManager) GetLastLoadTime() time.Time {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.lastLoadTime
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvUint64(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if uint64Val, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uint64Val
		}
	}
	return defaultValue
}

func parseFeatureFlags(flagsStr string) map[string]bool {
	flags := make(map[string]bool)
	if flagsStr == "" {
		return flags
	}

	parts := strings.Split(flagsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			flags[key] = value == "true" || value == "1"
		}
	}

	return flags
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


// GetBlockchainConfig returns configuration for a specific blockchain
func (cm *DefaultConfigManager) GetBlockchainConfig(chainName string) (BlockchainConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cfg, exists := cm.config.Blockchains[chainName]; exists {
		return cfg, nil
	}
	return BlockchainConfig{}, NewSystemError(
		ErrorTypePermanent,
		ErrorCodeNotFound,
		fmt.Sprintf("blockchain %s not configured", chainName),
		nil,
	)
}

// IsMultiChain returns true if multiple blockchains are configured
func (cm *DefaultConfigManager) IsMultiChain() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.config.Blockchains) > 1
}

// GetActiveChains returns the list of active blockchain chains
func (cm *DefaultConfigManager) GetActiveChains() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	chains := make([]string, len(cm.config.ActiveChains))
	copy(chains, cm.config.ActiveChains)
	return chains
}

// GetAllBlockchainConfigs returns all blockchain configurations
func (cm *DefaultConfigManager) GetAllBlockchainConfigs() map[string]BlockchainConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make(map[string]BlockchainConfig)
	for k, v := range cm.config.Blockchains {
		configs[k] = v
	}
	return configs
}
