package bootstrap

import (
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
)

// CoreConfigOverrides defines additive deployment-mode overrides.
type CoreConfigOverrides struct {
	APIType      *string
	APIPort      *int
	FeatureFlags map[string]bool
}

// NewAPIServiceCoreConfig creates default core config for api-service mode.
func NewAPIServiceCoreConfig(port int, logLevel string) core.Config {
	return core.Config{
		APIType:      "service",
		APIPort:      port,
		LogLevel:     logLevel,
		FeatureFlags: make(map[string]bool),
	}
}

// NewMonolithicCoreConfig creates default core config for monolithic mode,
// reading all required fields from environment variables with fallback defaults.
func NewMonolithicCoreConfig(logLevel, databaseType, databaseURL, cacheType, dataPullerType, blockchainNodeURL string) core.Config {
	if dataPullerType == "" {
		dataPullerType = env.Get("DATA_PULLER_TYPE", "https-jsonrpc")
	}
	if blockchainNodeURL == "" {
		blockchainNodeURL = env.Get("BLOCKCHAIN_NODE_URL", "http://localhost:8545")
	}

	cfg := core.Config{
		APIType:           env.Get("API_TYPE", "rest"),
		APIPort:           env.GetInt("API_PORT", 8080),
		LogLevel:          logLevel,
		DatabaseType:      databaseType,
		DatabaseURL:       databaseURL,
		CacheType:         cacheType,
		DataPullerType:    dataPullerType,
		BlockchainNodeURL: blockchainNodeURL,
		FeatureFlags:      make(map[string]bool),
	}

	// Populate remaining required fields from env vars
	cfg.MQType = env.Get("MQ_TYPE", "redis")
	cfg.MQConnectionURL = env.Get("MQ_CONNECTION_URL", "localhost:6379")
	cfg.CacheConnectionURL = env.Get("CACHE_CONNECTION_URL", "localhost:6379")
	cfg.DeploymentMode = env.Get("DEPLOYMENT_MODE", "monolithic")
	cfg.ServiceName = env.Get("SERVICE_NAME", "chainpulse")
	cfg.ChainID = env.Get("CHAIN_ID", "1")
	cfg.CacheTTL = env.GetInt("CACHE_TTL", 3600)
	cfg.WorkerPoolSize = env.GetInt("WORKER_POOL_SIZE", 8)
	cfg.BatchSize = env.GetInt("BATCH_SIZE", 100)
	cfg.MaxRetries = env.GetInt("MAX_RETRIES", 3)
	cfg.RetryBackoff = env.GetInt("RETRY_BACKOFF", 100)
	cfg.IdempotencyRecordTTL = env.GetInt("IDEMPOTENCY_RECORD_TTL", 86400)
	cfg.IdempotencyCleanupInterval = env.GetInt("IDEMPOTENCY_CLEANUP_INTERVAL", 600)
	cfg.StartBlock = uint64(env.GetInt("START_BLOCK", 0))

	return cfg
}

// ApplyCoreConfigOverrides applies deployment-specific overrides on top of a base config.
func ApplyCoreConfigOverrides(base core.Config, overrides CoreConfigOverrides) core.Config {
	cfg := base

	if cfg.FeatureFlags == nil {
		cfg.FeatureFlags = make(map[string]bool)
	}

	if overrides.APIType != nil {
		cfg.APIType = *overrides.APIType
	}

	if overrides.APIPort != nil {
		cfg.APIPort = *overrides.APIPort
	}

	for key, value := range overrides.FeatureFlags {
		cfg.FeatureFlags[key] = value
	}

	return cfg
}
