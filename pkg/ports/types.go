package ports

import (
	"context"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// CacheEntry represents a cached value
type CacheEntry struct {
	Key       string
	Value     []byte
	HitCount  int64
	TTL       int       // Time to live in seconds
	ExpiresAt time.Time // Expiration time
}

// QueryRequest defines query input at the domain boundary.
type QueryRequest struct {
	QueryType  string
	Collection string
	Filter     map[string]any
	Limit      int64
	Offset     int64
	CacheKey   string
	CacheTTL   time.Duration
	Sort       map[string]int
}

// QueryResult represents a query result
type QueryResult struct {
	Events       []blockchain.BlockchainEvent
	Total        int64
	CacheHit     bool
	ResponseTime int64
	Source       string
}

// ReorgStats tracks reorg statistics
type ReorgStats struct {
	TotalReorgsDetected   uint64
	TotalBlocksRolledBack uint64
	AverageReorgSize      float64
	LastReorgTime         time.Time
	LastReorgBlock        uint64
}

// ReorgRollbackEvent is published after a reorg rollback to trigger re-indexing
type ReorgRollbackEvent struct {
	ChainID    string
	FromBlock  uint64
	ToBlock    uint64
	DetectedAt time.Time
}

// QueryService is the domain-facing query contract.
type QueryService interface {
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)
	QueryByHash(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error)
	InvalidateCache(ctx context.Context, key string) error
	Health(ctx context.Context) *HealthStatus
}
