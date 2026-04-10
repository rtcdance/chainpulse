package core

import "time"

// ErrorType represents error classification
type ErrorType string

const (
	ErrorTypeTransient ErrorType = "transient"
	ErrorTypePermanent ErrorType = "permanent"
	ErrorTypeCritical  ErrorType = "critical"
)

// SystemError represents a system error with classification
type SystemError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details"`
	Err     error                  `json:"-"`
}

// CacheEntry represents a cached value
type CacheEntry struct {
	Key       string
	Value     []byte
	HitCount  int64
	TTL       int       // Time to live in seconds
	ExpiresAt time.Time // Expiration time
}

// QueryResult represents a query result
type QueryResult struct {
	Events       []BlockchainEvent
	Total        int64
	CacheHit     bool
	ResponseTime int64
}

// ReorgStats tracks reorg statistics
type ReorgStats struct {
	TotalReorgsDetected   uint64
	TotalBlocksRolledBack uint64
	AverageReorgSize      float64
	LastReorgTime         time.Time
	LastReorgBlock        uint64
}
