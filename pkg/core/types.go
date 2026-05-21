package core

import (
	"github.com/rtcdance/chainpulse/pkg/ports"
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

// Deprecated: Type aliases for backward compatibility. New code should import
// pkg/ports directly. These aliases will be removed in a future major version.
type (
	CacheEntry        = ports.CacheEntry
	QueryRequest      = ports.QueryRequest
	QueryResult       = ports.QueryResult
	ReorgStats        = ports.ReorgStats
	ReorgRollbackEvent = ports.ReorgRollbackEvent
	QueryService      = ports.QueryService
)
