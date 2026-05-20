// Package ports defines the primary interfaces (ports) for the ChainPulse
// hexagonal architecture. All external actors (plugins, services, infrastructure)
// depend on these interfaces, never on concrete implementations in pkg/core.
//
// Migration note: these interfaces were historically defined in pkg/core and
// still have type aliases there. New code should import pkg/ports directly.
package ports

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/configmodel"
)

// Config is a type alias for the canonical config model.
type Config = configmodel.Config

// Plugin is the minimal base interface for all plugins.
// Specific capabilities (lifecycle, health, configuration) are expressed
// through separate optional interfaces — see LifecyclePlugin, HealthPlugin,
// ConfigurablePlugin, and the composite interfaces below.
// This follows the Interface Segregation Principle: no plugin is forced to
// depend on methods it does not use.
type Plugin interface {
	Name() string
}

// LifecyclePlugin manages plugin startup and shutdown lifecycle.
type LifecyclePlugin interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HealthPlugin provides health checking capability.
type HealthPlugin interface {
	Name() string
	Health(ctx context.Context) error
}

// ConfigurablePlugin supports runtime configuration updates.
type ConfigurablePlugin interface {
	Initialize(ctx context.Context, config Config) error
}

// LivenessChecker checks if a component is alive (process is running).
type LivenessChecker interface {
	Liveness() error
}

// ReadinessChecker checks if a component is ready to serve traffic.
type ReadinessChecker interface {
	LivenessChecker
	Readiness() error
}

// PluginRegistry manages plugin lifecycle
type PluginRegistry interface {
	Register(plugin Plugin) error
	Unregister(name string) error
	Get(name string) (Plugin, error)
	List() []Plugin
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HotReloadablePlugin extends Plugin with hot-reload capability.
type HotReloadablePlugin interface {
	Plugin
	Reload(ctx context.Context, cfg Config) (success bool, err error)
	IsReloadable() bool
}

// DataPullerPlugin pulls events from blockchain sources.
type DataPullerPlugin interface {
	Plugin
	ChainID() string
	PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]blockchain.BlockchainEvent, error)
	GetLatestBlock(ctx context.Context) (uint64, error)
	SubscribeToEvents(ctx context.Context, handler func(blockchain.BlockchainEvent)) error
	GetStats() map[string]any
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
	TotalSize     int64   `json:"total_size,omitempty"`
	EntryCount    int64   `json:"entry_count,omitempty"`
}

// EventReader provides read-only access to blockchain events.
type EventReader interface {
	GetEvent(ctx context.Context, id string) (*blockchain.BlockchainEvent, error)
	QueryEvents(ctx context.Context, filter any) ([]any, error)
	GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error)
	GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error)
}

// EventWriter provides write access to blockchain events.
type EventWriter interface {
	StoreEvent(ctx context.Context, event any) error
	BatchStoreEvents(ctx context.Context, events []any) error
	DeleteEvent(ctx context.Context, eventID string) error
	DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error)
	MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error)
}

// BlockReader provides read-only access to blockchain blocks.
type BlockReader interface {
	GetBlock(ctx context.Context, blockNumber uint64) (*blockchain.Block, error)
	GetLatestBlock(ctx context.Context) (uint64, error)
	GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error)
}

// ReorgStatsProvider provides chain reorganization statistics.
type ReorgStatsProvider interface {
	GetReorgStats(ctx context.Context) (*ReorgStats, error)
}

// BlockHashProvider returns the canonical chain block hash for a given block number.
type BlockHashProvider interface {
	GetBlockHash(ctx context.Context, blockNumber uint64) (common.Hash, error)
}

// DatabasePlugin provides full database access for blockchain data.
type DatabasePlugin interface {
	Plugin
	EventReader
	EventWriter
	BlockReader
	ReorgStatsProvider
}

// Transactioner exposes transactional operations for atomic writes.
type Transactioner interface {
	BeginTx(ctx context.Context) (Tx, error)
}

// Tx represents an active database transaction.
type Tx interface {
	StoreEvent(ctx context.Context, event any) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// APIPlugin serves data to clients
type APIPlugin interface {
	Plugin
	RegisterHandler(path string, handler any) error
	StartServer(port int) error
	StopServer() error
}

// ProcessingPlugin handles event processing
type ProcessingPlugin interface {
	Plugin
	Process(ctx context.Context, event any) error
}

// DependentPlugin declares startup ordering dependencies.
type DependentPlugin interface {
	Dependencies() []string
}
