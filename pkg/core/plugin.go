package core

//go:generate mockgen -destination=mock_plugins.go -package=core . Plugin,PluginRegistry,ConfigManager,EventBus,Logger,MetricsCollector,HealthChecker,CheckpointStore,DataPullerPlugin,MQPlugin,CachePlugin,EventReader,EventWriter,DatabasePlugin,APIPlugin,ProcessingPlugin

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Plugin is the base interface for all plugins
type Plugin interface {
	Name() string
	Version() string
	Initialize(config Config) error
	Start() error
	Stop() error
	Health() error
}

// ContextualStarter is an optional interface that plugins can implement to
// receive a context during startup. This enables cancellation, timeouts,
// and tracing through the plugin lifecycle. If a plugin does not implement
// this interface, Start() is called as a fallback.
type ContextualStarter interface {
	StartWithContext(ctx context.Context) error
}

// ContextualStopper is an optional interface that plugins can implement to
// receive a context during shutdown. This enables graceful shutdown with
// deadlines and cancellation. If a plugin does not implement this interface,
// Stop() is called as a fallback.
type ContextualStopper interface {
	StopWithContext(ctx context.Context) error
}

// StartPlugin starts a plugin, using StartWithContext if the plugin implements
// ContextualStarter, otherwise falling back to Start().
func StartPlugin(ctx context.Context, p Plugin) error {
	if cs, ok := p.(ContextualStarter); ok {
		return cs.StartWithContext(ctx)
	}
	return p.Start()
}

// StopPlugin stops a plugin, using StopWithContext if the plugin implements
// ContextualStopper, otherwise falling back to Stop().
func StopPlugin(ctx context.Context, p Plugin) error {
	if cs, ok := p.(ContextualStopper); ok {
		return cs.StopWithContext(ctx)
	}
	return p.Stop()
}

// PluginRegistry manages plugin lifecycle
type PluginRegistry interface {
	Register(plugin Plugin) error
	Unregister(name string) error
	Get(name string) (Plugin, error)
	List() []Plugin
	Start() error
	Stop() error
}

// ConfigManager manages configuration
type ConfigManager interface {
	Load() (Config, error)
	Validate(config Config) error
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
}

// EventBus provides pub-sub communication
type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
	Subscribe(ctx context.Context, topic string, handler func(interface{})) (uint64, error)
	Unsubscribe(subscriptionID uint64) error
}

// IdempotencyInvalidator clears idempotency entries for a block range.
// After a reorg, previously-processed events in the reorged range must be
// removed from the idempotency store so that re-indexed events are not
// incorrectly rejected as duplicates.
type IdempotencyInvalidator interface {
	InvalidateRange(fromBlock, toBlock uint64) int
}

// Logger provides structured logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	WithCorrelationID(id string) Logger
}

// MetricsCollector collects metrics
type MetricsCollector interface {
	RecordCounter(name string, value int64, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
	GetMetrics() map[string]interface{}
}

// HealthChecker checks system health
type HealthChecker interface {
	Check(ctx context.Context) (HealthStatus, error)
}

// CheckpointStore persists indexing progress across restarts
type CheckpointStore interface {
	GetLastIndexedBlock(ctx context.Context, chainID string) (uint64, string, error)
	SaveLastIndexedBlock(ctx context.Context, chainID string, blockNumber uint64, blockHash string) error
	GetBlockHash(ctx context.Context, chainID string, blockNumber uint64) (string, error)
}

// HealthStatus represents system health
type HealthStatus struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
}

// Config represents system configuration
type Config struct {
	// Data Puller Configuration
	DataPullerType    string   `json:"data_puller_type"`
	BlockchainNodeURL string   `json:"blockchain_node_url"`
	StartBlock        uint64   `json:"start_block"`
	ContractAddresses []string `json:"contract_addresses"`  // optional address filter for eth_getLogs
	EventSignatures    []string `json:"event_signatures"`     // optional topic0 hashes for eth_getLogs topics filter
	BlockChunkSize    int      `json:"block_chunk_size"`    // blocks per eth_getLogs request (default 1000)

	// Message Queue Configuration
	MQType          string `json:"mq_type"`
	MQConnectionURL string `json:"mq_connection_url"`

	// Cache Configuration
	CacheType          string `json:"cache_type"`
	CacheConnectionURL string `json:"cache_connection_url"`
	CacheTTL           int    `json:"cache_ttl"` // in seconds

	// Database Configuration
	DatabaseType string `json:"database_type"`
	DatabaseURL  string `json:"database_url"`

	// PostgreSQL Configuration
	PostgresHost     string `json:"postgres_host"`
	PostgresPort     string `json:"postgres_port"`
	PostgresUser     string `json:"postgres_user"`
	PostgresPassword string `json:"postgres_password"`
	PostgresDB       string `json:"postgres_db"`

	// API Configuration
	APIType string `json:"api_type"`
	APIPort int    `json:"api_port"`

	// Processing Configuration
	WorkerPoolSize int `json:"worker_pool_size"`
	BatchSize      int `json:"batch_size"`
	MaxRetries     int `json:"max_retries"`
	RetryBackoff   int `json:"retry_backoff"` // in milliseconds

	// Deployment Configuration
	DeploymentMode string `json:"deployment_mode"`
	ServiceName    string `json:"service_name"`
	ChainID        string `json:"chain_id"` // Per-chain identifier for multi-chain indexing
	Network        string `json:"network"`  // Chain network: "mainnet", "testnet", "devnet"

	// Idempotency Configuration
	IdempotencyRecordTTL       int `json:"idempotency_record_ttl"`       // in seconds, default 86400 (24h)
	IdempotencyCleanupInterval int `json:"idempotency_cleanup_interval"` // in seconds, default 600 (10m)

	// Logging Configuration
	LogLevel string `json:"log_level"`

	// Feature Flags
	FeatureFlags map[string]bool `json:"feature_flags"`

	// Multi-blockchain Configuration
	Blockchains      map[string]BlockchainConfig `json:"blockchains"`
	ActiveChains     []string                    `json:"active_chains"`
	SkipRemovedLogs  bool                        `json:"skip_removed_logs"` // skip log.Removed=true events from reorgs (default: false, publish with flag)
}

// GetString retrieves a string configuration value
func (c *Config) GetString(key, defaultValue string) string {
	switch key {
	case "POSTGRES_HOST":
		if c.PostgresHost != "" {
			return c.PostgresHost
		}
	case "POSTGRES_PORT":
		if c.PostgresPort != "" {
			return c.PostgresPort
		}
	case "POSTGRES_USER":
		if c.PostgresUser != "" {
			return c.PostgresUser
		}
	case "POSTGRES_PASSWORD":
		if c.PostgresPassword != "" {
			return c.PostgresPassword
		}
	case "POSTGRES_DB":
		if c.PostgresDB != "" {
			return c.PostgresDB
		}
	case "POSTGRES_CONNECTION_STRING":
		if c.DatabaseURL != "" {
			return c.DatabaseURL
		}
	}
	return defaultValue
}

// DataPullerPlugin pulls events from blockchain sources
type DataPullerPlugin interface {
	Plugin
	PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]BlockchainEvent, error)
	GetLatestBlock(ctx context.Context) (uint64, error)
	SubscribeToEvents(ctx context.Context, handler func(BlockchainEvent)) error
	GetStats() map[string]interface{}
}

// MQPlugin manages message queue operations
type MQPlugin interface {
	Plugin
	Publish(ctx context.Context, topic string, message []byte) error
	Subscribe(ctx context.Context, topic string, handler func([]byte)) error
	GetQueueDepth(ctx context.Context, topic string) (int64, error)
}

// CachePlugin manages caching operations
type CachePlugin interface {
	Plugin
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl int) error
	Delete(ctx context.Context, key string) error
	GetStats() CacheStats
	HealthCheck(ctx context.Context) error
}

// CacheStats represents cache statistics
type CacheStats struct {
	HitCount      int64   `json:"hit_count"`
	MissCount     int64   `json:"miss_count"`
	EvictionCount int64   `json:"eviction_count"`
	HitRate       float64 `json:"hit_rate"`
}

// DatabasePlugin manages database operations
// EventReader provides read-only access to blockchain events.
// Consumers that only need to query events should depend on this interface
// rather than the full DatabasePlugin, following the Interface Segregation Principle.
type EventReader interface {
	GetEvent(ctx context.Context, id string) (*BlockchainEvent, error)
	QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error)
	GetAllEvents(ctx context.Context) ([]*BlockchainEvent, error)
	GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*BlockchainEvent, error)
}

// EventWriter provides write access to blockchain events.
type EventWriter interface {
	StoreEvent(ctx context.Context, event interface{}) error
	BatchStoreEvents(ctx context.Context, events []interface{}) error
	DeleteEvent(ctx context.Context, eventID string) error
	DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error)
	MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error)
}

// BlockReader provides read-only access to blockchain blocks.
type BlockReader interface {
	GetBlock(ctx context.Context, blockNumber uint64) (*Block, error)
	GetLatestBlock(ctx context.Context) (uint64, error)
	GetAllBlocks(ctx context.Context) ([]*Block, error)
}

// ReorgStatsProvider provides chain reorganization statistics.
type ReorgStatsProvider interface {
	GetReorgStats(ctx context.Context) (*ReorgStats, error)
}

// BlockHashProvider returns the canonical chain block hash for a given block number.
// Implementations may query an RPC node (production) or a local database (testing).
// This is used by reorg detection to compare locally-indexed hashes against
// the live canonical chain — comparing against a local database is insufficient
// because both sources contain old-chain data after a reorg.
type BlockHashProvider interface {
	GetBlockHash(ctx context.Context, blockNumber uint64) (common.Hash, error)
}

// DatabasePlugin provides full database access for blockchain data.
// It composes fine-grained interfaces for event reading, event writing,
// block reading, and reorg statistics. Consumers should depend on the
// smallest interface that satisfies their needs (e.g., EventReader for
// query-only services).
type DatabasePlugin interface {
	Plugin
	EventReader
	EventWriter
	BlockReader
	ReorgStatsProvider
}

// APIPlugin serves data to clients
type APIPlugin interface {
	Plugin
	RegisterHandler(path string, handler interface{}) error
	StartServer(port int) error
	StopServer() error
}

// ProcessingPlugin handles event processing
type ProcessingPlugin interface {
	Plugin
	Process(ctx context.Context, event interface{}) error
}
