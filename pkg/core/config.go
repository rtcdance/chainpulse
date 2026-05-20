package core

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
	"github.com/rtcdance/chainpulse/pkg/env"
)

// BlockchainConfig is defined in pkg/configmodel.
type BlockchainConfig = configmodel.BlockchainConfig

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
		DataPullerType:    env.Get("DATA_PULLER_TYPE", "https-jsonrpc"),
		BlockchainNodeURL: env.Get("BLOCKCHAIN_NODE_URL", "http://localhost:8545"),
		StartBlock:        env.GetUint64("START_BLOCK", 0),

		// Message Queue Configuration
		MQType:          env.Get("MQ_TYPE", "kafka"),
		MQConnectionURL: env.Get("MQ_CONNECTION_URL", "localhost:9092"),

		// Cache Configuration
		CacheType:          env.Get("CACHE_TYPE", "redis"),
		CacheConnectionURL: env.Get("CACHE_CONNECTION_URL", "localhost:6379"),
		CacheTTL:           env.GetInt("CACHE_TTL", DefaultCacheTTL),

		// Database Configuration
		DatabaseType: env.Get("DATABASE_TYPE", "postgres"),
		DatabaseURL:  env.Get("DATABASE_URL", "postgres://localhost/chainpulse"),

		// API Configuration
		APIType: env.Get("API_TYPE", "rest"),
		APIPort: env.GetInt("API_PORT", DefaultAPIPort),

		// Processing Configuration
		WorkerPoolSize: env.GetInt("WORKER_POOL_SIZE", DefaultWorkerPoolSize),
		BatchSize:      env.GetInt("BATCH_SIZE", DefaultBatchSize),
		MaxRetries:     env.GetInt("MAX_RETRIES", DefaultMaxRetries),
		RetryBackoff:   env.GetInt("RETRY_BACKOFF", DefaultRetryBackoff),

		// Deployment Configuration
		DeploymentMode: env.Get("DEPLOYMENT_MODE", "monolithic"),
		ServiceName:    env.Get("SERVICE_NAME", "chainpulse"),
		ChainID:        env.Get("CHAIN_ID", ""),

		// MongoDB Configuration
		MongoDBConnectionString: configmodel.SecretString(env.Get("MONGODB_CONNECTION_STRING", "")),
		MongoDBHost:             env.Get("MONGODB_HOST", "localhost"),
		MongoDBPort:             env.Get("MONGODB_PORT", "27017"),
		MongoDBUser:             env.Get("MONGODB_USER", ""),
		MongoDBPassword:         configmodel.SecretString(env.Get("MONGODB_PASSWORD", "")),
		MongoDBDatabase:         env.Get("MONGODB_DATABASE", "chainpulse"),
		MongoDBCollection:       env.Get("MONGODB_COLLECTION", "events"),

		// SSL/TLS Configuration
		SslMode: env.Get("DATABASE_SSLMODE", "require"),

		// Idempotency Configuration
		IdempotencyRecordTTL:       env.GetInt("IDEMPOTENCY_RECORD_TTL", 86400),
		IdempotencyCleanupInterval: env.GetInt("IDEMPOTENCY_CLEANUP_INTERVAL", 600),

		// Logging Configuration
		LogLevel: env.Get("LOG_LEVEL", "info"),

		// Feature Flags
		FeatureFlags: parseFeatureFlags(env.Get("FEATURE_FLAGS", "")),

		// Multi-blockchain Configuration
		Blockchains:  make(map[string]BlockchainConfig),
		ActiveChains: make([]string, 0),
	}

	// Load multi-blockchain configuration
	chainsStr := env.Get("CHAINPULSE_CHAINS", "")
	if chainsStr != "" {
		chains := strings.Split(chainsStr, ",")
		for _, chain := range chains {
			chain = strings.TrimSpace(chain)
			if chain == "" {
				continue
			}

			nodeURL := env.Get(fmt.Sprintf("CHAINPULSE_%s_NODE_URL", strings.ToUpper(chain)), "")
			chainID := env.Get(fmt.Sprintf("CHAINPULSE_%s_CHAIN_ID", strings.ToUpper(chain)), "")
			startBlockStr := env.Get(fmt.Sprintf("CHAINPULSE_%s_START_BLOCK", strings.ToUpper(chain)), "0")
			network := env.Get(fmt.Sprintf("CHAINPULSE_%s_NETWORK", strings.ToUpper(chain)), "mainnet")
			confBlocksStr := env.Get(fmt.Sprintf("CHAINPULSE_%s_CONFIRMATION_BLOCKS", strings.ToUpper(chain)), "")

			startBlock := uint64(0)
			if startBlockStr != "" {
				if block, err := strconv.ParseUint(startBlockStr, 10, 64); err == nil {
					startBlock = block
				}
			}

			confBlocks := uint64(0)
			if confBlocksStr != "" {
				if blocks, err := strconv.ParseUint(confBlocksStr, 10, 64); err == nil {
					confBlocks = blocks
				}
			}

			if nodeURL != "" && chainID != "" {
				config.Blockchains[chain] = BlockchainConfig{
					ChainID:            chainID,
					NodeURL:            nodeURL,
					StartBlock:         startBlock,
					ChainName:          chain,
					Network:            network,
					ConfirmationBlocks: confBlocks,
				}
				config.ActiveChains = append(config.ActiveChains, chain)
			}
		}
	}

	cm.config = config

	if cm.logger != nil {
		if len(config.ActiveChains) > 0 {
			cm.logger.Info(
				"multi-blockchain configuration loaded",
				"active_chains", strings.Join(config.ActiveChains, ","),
				"chain_count", len(config.ActiveChains),
			)
		} else {
			cm.logger.Info(
				"configuration loaded",
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

// Validate checks the loaded configuration for minimum correctness.
func (cm *DefaultConfigManager) Validate() error {
	cm.mu.RLock()
	config := cm.config
	cm.mu.RUnlock()

	var errs []string

	if config.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if config.BlockchainNodeURL == "" && len(config.Blockchains) == 0 {
		errs = append(errs, "at least one of BLOCKCHAIN_NODE_URL or CHAINPULSE_CHAINS is required")
	}
	if config.APIPort <= 0 || config.APIPort > 65535 {
		errs = append(errs, fmt.Sprintf("API_PORT must be 1-65535, got %d", config.APIPort))
	}
	if config.WorkerPoolSize <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_POOL_SIZE must be > 0, got %d", config.WorkerPoolSize))
	}
	if config.BatchSize <= 0 {
		errs = append(errs, fmt.Sprintf("BATCH_SIZE must be > 0, got %d", config.BatchSize))
	}
	if config.MaxRetries < 0 {
		errs = append(errs, fmt.Sprintf("MAX_RETRIES must be >= 0, got %d", config.MaxRetries))
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Get retrieves a configuration value by key.
func (cm *DefaultConfigManager) Get(key string) (any, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	acc, ok := ConfigFields[key]
	if !ok {
		return nil, NewSystemError(
			ErrorTypePermanent,
			ErrorCodeNotFound,
			fmt.Sprintf("configuration key not found: %s", key),
			nil,
		)
	}
	return acc.Get(&cm.config)
}

// Set sets a configuration value by key.
func (cm *DefaultConfigManager) Set(key string, value any) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	acc, ok := ConfigFields[key]
	if !ok {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			fmt.Sprintf("invalid configuration key: %s", key),
			nil,
		)
	}
	return acc.Set(&cm.config, value)
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
