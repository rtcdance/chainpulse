package core

import (
	"context"
	"time"
)

// ErrorType represents error classification
type ErrorType string

const (
	ErrorTypeTransient ErrorType = "transient"
	ErrorTypePermanent ErrorType = "permanent"
	ErrorTypeCritical  ErrorType = "critical"
)

// SystemError represents a system error with classification
type SystemError struct {
	Type    ErrorType      `json:"type"`
	Message string         `json:"message"`
	Code    string         `json:"code"`
	Details map[string]any `json:"details"`
	Err     error          `json:"-"`
}

// CacheEntry represents a cached value
type CacheEntry struct {
	Key       string
	Value     []byte
	HitCount  int64
	TTL       int       // Time to live in seconds
	ExpiresAt time.Time // Expiration time
}

// QueryRequest defines query input at the domain boundary.
// Moved from pkg/domain/query to core so query contracts can live in the
// dependency-free core package.
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
	Events       []BlockchainEvent
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
// Moved from pkg/domain/query to core for dependency-free layering.
type QueryService interface {
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)
	QueryByHash(ctx context.Context, hash string) (*BlockchainEvent, error)
	InvalidateCache(ctx context.Context, key string) error
	Health(ctx context.Context) *HealthStatus
}
