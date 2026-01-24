package e2e

import (
	"context"
	"time"
)

// Logger defines logging interface
type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

// BlockchainManagerInterface defines the interface for blockchain managers
type BlockchainManagerInterface interface {
	// Startup initializes the blockchain manager
	Startup(ctx context.Context) error

	// Shutdown stops the blockchain manager
	Shutdown(ctx context.Context) error

	// StartAnvil starts the Anvil test node
	StartAnvil(ctx context.Context) error

	// StopAnvil stops the Anvil instance
	StopAnvil(ctx context.Context) error

	// DeployContract deploys a smart contract
	DeployContract(ctx context.Context, contract ContractDefinition) (*DeployedContract, error)

	// EmitEvent triggers an event emission
	EmitEvent(ctx context.Context, contractAddr string, eventName string, params map[string]interface{}) (*EventEmission, error)

	// GetBlockNumber returns current block number
	GetBlockNumber(ctx context.Context) (uint64, error)

	// CreateSnapshot creates a blockchain state snapshot
	CreateSnapshot(ctx context.Context) (string, error)

	// RestoreSnapshot restores blockchain to a previous state
	RestoreSnapshot(ctx context.Context, snapshotID string) error

	// GetContractABI returns the ABI for a deployed contract
	GetContractABI(contractAddr string) (string, error)
}



// IndexerManager manages ChainPulse indexer components
type IndexerManager interface {
	// StartIndexer starts the indexer service
	StartIndexer(ctx context.Context, config IndexerConfig) error

	// StopIndexer stops the indexer
	StopIndexer(ctx context.Context) error

	// WaitForIndexing waits for events to be indexed
	WaitForIndexing(ctx context.Context, expectedCount int, timeout time.Duration) error

	// GetIndexedEvents returns indexed events
	GetIndexedEvents(ctx context.Context, filter EventFilter) ([]*IndexedEvent, error)

	// GetIndexerMetrics returns indexer performance metrics
	GetIndexerMetrics(ctx context.Context) IndexerMetrics
}

// ValidationManager validates test results and assertions
type ValidationManager interface {
	// ValidateEventCollection validates that all events were collected
	ValidateEventCollection(ctx context.Context, emitted []*EventEmission, indexed []*IndexedEvent) error

	// ValidateEventDecoding validates event decoding accuracy
	ValidateEventDecoding(ctx context.Context, event *IndexedEvent) error

	// ValidateEventOrdering validates event ordering
	ValidateEventOrdering(ctx context.Context, events []*IndexedEvent) error

	// ValidateAPIResponse validates API query response
	ValidateAPIResponse(ctx context.Context, response *APIResponse, expectedEvents []*IndexedEvent) error

	// ValidatePerformance validates performance metrics
	ValidatePerformance(ctx context.Context, metrics PerformanceMetrics) error
}

// ContractDefinition defines a smart contract for deployment
type ContractDefinition struct {
	Name        string
	Bytecode    string
	ABI         string
	Constructor []interface{}
}

// DeployedContract represents a deployed smart contract
type DeployedContract struct {
	Address     string
	ABI         string
	TxHash      string
	Code        string
	DeployedAt  time.Time
	BlockNumber uint64
}

// EventEmission represents an event emitted on the blockchain
type EventEmission struct {
	ID              string
	ContractAddress string
	EventName       string
	TxHash          string
	BlockNumber     uint64
	LogIndex        uint32
	Parameters      map[string]interface{}
	Timestamp       time.Time
}

// IndexedEvent represents an event after indexing
type IndexedEvent struct {
	ID              string
	ContractAddress string
	EventName       string
	TxHash          string
	BlockNumber     uint64
	LogIndex        uint32
	DecodedData     map[string]interface{}
	IndexedAt       time.Time
	ChainID         string
}

// EventFilter filters events for queries
type EventFilter struct {
	ContractAddress string
	EventName       string
	BlockRange      *BlockRange
	ChainID         string
	Limit           int
	Offset          int
}

// BlockRange specifies a range of blocks
type BlockRange struct {
	Start uint64
	End   uint64
}

// IndexerConfig configures the indexer
type IndexerConfig struct {
	Port              int
	DatabaseURL       string
	BlockchainRPC     string
	ContractAddresses []string
	LogLevel          string
	Timeout           time.Duration
}

// IndexerMetrics contains indexer performance metrics
type IndexerMetrics struct {
	EventsProcessed   int64
	EventsIndexed     int64
	ErrorCount        int64
	AverageLatency    time.Duration
	Throughput        float64
	MemoryUsage       int64
	CPUUsage          float64
}

// TestMetrics contains collected test metrics
type TestMetrics struct {
	CollectionLatency []int64
	ProcessingLatency []int64
	QueryLatency      []int64
	Throughput        float64
	ErrorCount        int
	MemoryUsage       int64
	CPUUsage          float64
}

// APIResponse represents an API query response
type APIResponse struct {
	Events []*IndexedEvent
	Total  int
	Limit  int
	Offset int
}

// TestAccount represents a test account with funds
type TestAccount struct {
	Address string
	Balance string
	Key     string
}

// AnvilConfig configures Anvil instance
type AnvilConfig struct {
	Port          int
	BlockTime     int
	Deterministic bool
	ChainID       int
	Accounts      int
	Balance       string
}

// ErrorSimulation simulates errors for testing
type ErrorSimulation struct {
	Type     string
	Duration time.Duration
	Retries  int
}

// Contract represents a smart contract for multi-chain operations
type Contract struct {
	Name     string
	Bytecode string
	ABI      string
}

// Event represents a blockchain event for multi-chain operations
type Event struct {
	ID                 string
	ContractAddress    string
	EventName          string
	Parameters         map[string]interface{}
	ChainID            string
	BlockNumber        uint64
	TransactionIndex   uint32
	TransactionHash    string
}
