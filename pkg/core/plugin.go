package core

import (
	"context"
	"time"
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
	Subscribe(ctx context.Context, topic string, handler func(interface{})) error
	Unsubscribe(topic string, handler func(interface{})) error
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
	DataPullerType    string `json:"data_puller_type"`
	BlockchainNodeURL string `json:"blockchain_node_url"`
	StartBlock        uint64 `json:"start_block"`

	// Message Queue Configuration
	MQType          string `json:"mq_type"`
	MQConnectionURL string `json:"mq_connection_url"`

	// Cache Configuration
	CacheType         string `json:"cache_type"`
	CacheConnectionURL string `json:"cache_connection_url"`
	CacheTTL          int    `json:"cache_ttl"` // in seconds

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

	// Logging Configuration
	LogLevel string `json:"log_level"`

	// Feature Flags
	FeatureFlags map[string]bool `json:"feature_flags"`

	// Multi-blockchain Configuration
	Blockchains map[string]BlockchainConfig `json:"blockchains"`
	ActiveChains []string `json:"active_chains"`
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
}

// CacheStats represents cache statistics
type CacheStats struct {
	HitCount    int64   `json:"hit_count"`
	MissCount   int64   `json:"miss_count"`
	EvictionCount int64 `json:"eviction_count"`
	HitRate     float64 `json:"hit_rate"`
}

// DatabasePlugin manages database operations
type DatabasePlugin interface {
	Plugin
	StoreEvent(ctx context.Context, event interface{}) error
	GetEvent(ctx context.Context, id string) (*BlockchainEvent, error)
	QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error)
	BatchStoreEvents(ctx context.Context, events []interface{}) error
	GetAllEvents(ctx context.Context) ([]*BlockchainEvent, error)
	GetAllBlocks(ctx context.Context) ([]*Block, error)
	DeleteEvent(ctx context.Context, eventID string) error
	GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*BlockchainEvent, error)
	GetBlock(ctx context.Context, blockNumber uint64) (*Block, error)
	GetLatestBlock(ctx context.Context) (uint64, error)
	DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error)
	GetReorgStats(ctx context.Context) (*ReorgStats, error)
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
