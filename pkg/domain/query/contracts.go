package query

import (
	"context"
	"time"

	"chainpulse/pkg/core"
)

// Request defines query input at the domain boundary.
type Request struct {
	QueryType  string
	Collection string
	Filter     map[string]interface{}
	Limit      int64
	Offset     int64
	CacheKey   string
	CacheTTL   time.Duration
	Sort       map[string]int
}

// Result defines query output at the domain boundary.
type Result struct {
	Events       []core.BlockchainEvent
	Total        int64
	CacheHit     bool
	ResponseTime int64
	Source       string
}

// Service is the domain-facing query contract.
type Service interface {
	Query(ctx context.Context, req *Request) (*Result, error)
	QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error)
	InvalidateCache(ctx context.Context, key string) error
	Health(ctx context.Context) *core.HealthStatus
}
