package configmodel

import "fmt"

// Validate checks mandatory fields and applies reasonable bounds.
// Returns nil if config is valid for deployment; returns an error with
// a human-readable message listing all violations otherwise.
func (c *Config) Validate() error {
	var errs []string

	if c.WorkerPoolSize < 0 {
		errs = append(errs, "worker_pool_size must be >= 0")
	}
	if c.BatchSize < 0 {
		errs = append(errs, "batch_size must be >= 0")
	}
	if c.MaxRetries < 0 {
		errs = append(errs, "max_retries must be >= 0")
	}
	if c.RetryBackoff < 0 {
		errs = append(errs, "retry_backoff must be >= 0")
	}
	if c.BlockChunkSize < 0 {
		errs = append(errs, "block_chunk_size must be >= 0")
	}
	if c.ShutdownTimeout < 0 {
		errs = append(errs, "shutdown_timeout must be >= 0")
	}
	if c.ReorgThreshold == 0 && c.ConfirmationDepth == 0 && len(c.Blockchains) > 0 {
		errs = append(errs, "reorg_threshold or confirmation_depth required when indexing blockchains")
	}
	if c.ExpectedGenesisHash != "" && (len(c.ExpectedGenesisHash) != 66 || c.ExpectedGenesisHash[:2] != "0x") {
		errs = append(errs, "expected_genesis_hash must be 0x-prefixed 32-byte hex (66 chars)")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %v", errs)
	}
	return nil
}

// activeChainFields returns the chain-level config for the primary active chain,
// preferring multi-chain Blockchains entries over the legacy flat fields.
//
// Precedence:
//  1. Blockchains[chainID]  — multi-chain mode (per-chain config)
//  2. flat BlockchainNodeURL — legacy single-chain mode
func (c *Config) activeChainFields() *BlockchainConfig {
	for _, chainID := range c.ActiveChains {
		if bc, ok := c.Blockchains[chainID]; ok {
			bc.ChainID = chainID
			return &bc
		}
	}
	if c.ChainID == "" && c.BlockchainNodeURL == "" {
		return nil
	}
	return &BlockchainConfig{
		ChainID:            c.ChainID,
		NodeURL:            c.BlockchainNodeURL,
		StartBlock:         c.StartBlock,
		Network:            c.Network,
		EventSignatures:    c.EventSignatures,
		ConfirmationBlocks: c.ConfirmationDepth,
	}
}

type BlockchainConfig struct {
	ChainID            string
	NodeURL            string
	FallbackNodeURLs   []string
	StartBlock         uint64
	ChainName          string
	Network            string
	EventSignatures    []string
	ConfirmationBlocks uint64
}

// Config represents system configuration.
type Config struct {
	// Data Puller Configuration
	DataPullerType    string   `json:"data_puller_type"`
	BlockchainNodeURL string   `json:"blockchain_node_url"`
	StartBlock        uint64   `json:"start_block"`
	ContractAddresses []string `json:"contract_addresses"`
	EventSignatures   []string `json:"event_signatures"`
	BlockChunkSize    int      `json:"block_chunk_size"`

	// Message Queue Configuration
	MQType          string `json:"mq_type"`
	MQConnectionURL string `json:"mq_connection_url"`

	// Cache Configuration
	CacheType          string `json:"cache_type"`
	CacheConnectionURL string `json:"cache_connection_url"`
	CacheTTL           int    `json:"cache_ttl"`

	// Database Configuration
	DatabaseType string `json:"database_type"`
	DatabaseURL  string `json:"database_url"`

	// PostgreSQL Configuration
	PostgresHost     string       `json:"postgres_host"`
	PostgresPort     string       `json:"postgres_port"`
	PostgresUser     string       `json:"postgres_user"`
	PostgresPassword SecretString `json:"postgres_password"`
	PostgresDB       string       `json:"postgres_db"`

	// MongoDB Configuration
	MongoDBConnectionString SecretString `json:"mongodb_connection_string"`
	MongoDBHost             string       `json:"mongodb_host"`
	MongoDBPort             string       `json:"mongodb_port"`
	MongoDBUser             string       `json:"mongodb_user"`
	MongoDBPassword         SecretString `json:"mongodb_password"`
	MongoDBDatabase         string       `json:"mongodb_database"`
	MongoDBCollection       string       `json:"mongodb_collection"`

	// SSL/TLS Configuration
	SslMode string `json:"ssl_mode"`

	// API Configuration
	APIType string `json:"api_type"`
	APIPort int    `json:"api_port"`

	// CORS configuration (D5). AllowedOrigins empty means same-origin only:
	// no Access-Control-Allow-Origin header is emitted, so browsers refuse
	// cross-origin reads. There is deliberately no wildcard default.
	CORSAllowedOrigins []string `json:"cors_allowed_origins"`

	// Processing Configuration
	WorkerPoolSize  int `json:"worker_pool_size"`
	BatchSize       int `json:"batch_size"`
	MaxRetries      int `json:"max_retries"`
	RetryBackoff    int `json:"retry_backoff"`
	ShutdownTimeout int `json:"shutdown_timeout"`

	// Deployment Configuration
	DeploymentMode string `json:"deployment_mode"`
	ServiceName    string `json:"service_name"`
	ChainID        string `json:"chain_id"`
	// ExpectedGenesisHash, when non-empty, pins an EVM RPC provider to one
	// specific chain: Start fetches eth_getBlockByNumber(0) and refuses the
	// provider when the hash disagrees. Expected format: 0x + 64 hex chars.
	ExpectedGenesisHash string `json:"expected_genesis_hash"`
	Network             string `json:"network"`
	Version             string `json:"version"`

	// Idempotency Configuration
	IdempotencyRecordTTL       int `json:"idempotency_record_ttl"`
	IdempotencyCleanupInterval int `json:"idempotency_cleanup_interval"`

	// Reorg Configuration
	ReorgThreshold    uint64 `json:"reorg_threshold"`
	MaxRollbackDepth  uint64 `json:"max_rollback_depth"`
	ConfirmationDepth uint64 `json:"confirmation_depth"`

	// Logging Configuration
	LogLevel string `json:"log_level"`

	// Feature Flags
	FeatureFlags map[string]bool `json:"feature_flags"`

	// Multi-blockchain Configuration
	Blockchains     map[string]BlockchainConfig `json:"blockchains"`
	ActiveChains    []string                    `json:"active_chains"`
	SkipRemovedLogs bool                        `json:"skip_removed_logs"`

	// Alerting Configuration (D6). Rules is the config-driven input to the
	// log-based AlertManager; empty means no rules are armed.
	Alerts AlertsConfig `json:"alerts"`

	// Finality Configuration (D7). DefaultConfirmations is the confirmation
	// depth applied to EVM chains with no explicit per-chain entry;
	// Strategies maps chain ID to "N", "latest", "tag:safe" or "tag:finalized".
	Finality FinalityConfig `json:"finality"`
}

// AlertsConfig drives the monolithic AlertManager with a log sink.
type AlertsConfig struct {
	Enabled bool              `json:"enabled"`
	Rules   []AlertRuleConfig `json:"rules"`
}

// AlertRuleConfig is one config-driven alert rule (no full DSL).
type AlertRuleConfig struct {
	Name            string  `json:"name"`
	Metric          string  `json:"metric"`
	Operator        string  `json:"operator"`
	Threshold       float64 `json:"threshold"`
	DurationSeconds int     `json:"duration_seconds"`
	Severity        string  `json:"severity"`
	Message         string  `json:"message"`
}

// FinalityConfig selects the per-chain finality strategy.
type FinalityConfig struct {
	DefaultConfirmations uint64            `json:"default_confirmations"`
	Strategies           map[string]string `json:"strategies"`
}
