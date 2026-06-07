package core

import (
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestErrorTypeConstants tests error type constants
func TestErrorTypeConstants(t *testing.T) {
	assert.Equal(t, ErrorType("transient"), ErrorTypeTransient)
	assert.Equal(t, ErrorType("permanent"), ErrorTypePermanent)
	assert.Equal(t, ErrorType("critical"), ErrorTypeCritical)
}

// TestSystemErrorCreation tests SystemError creation
func TestSystemErrorCreation(t *testing.T) {
	details := map[string]any{
		"service": "test-service",
		"code":    500,
	}

	err := &SystemError{
		Type:    ErrorTypeTransient,
		Message: "Test error",
		Code:    "TEST_ERROR",
		Details: details,
	}

	assert.Equal(t, ErrorTypeTransient, err.Type)
	assert.Equal(t, "Test error", err.Message)
	assert.Equal(t, "TEST_ERROR", err.Code)
	assert.Equal(t, details, err.Details)
}

// TestSystemErrorTypes tests different error types
func TestSystemErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
	}{
		{"Transient", ErrorTypeTransient},
		{"Permanent", ErrorTypePermanent},
		{"Critical", ErrorTypeCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &SystemError{
				Type:    tt.errorType,
				Message: "Test",
				Code:    "TEST",
			}
			assert.Equal(t, tt.errorType, err.Type)
		})
	}
}

// TestCacheEntryCreation tests CacheEntry creation
func TestCacheEntryCreation(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	entry := &CacheEntry{
		Key:       "test-key",
		Value:     []byte("test-value"),
		HitCount:  5,
		TTL:       3600,
		ExpiresAt: expiresAt,
	}

	assert.Equal(t, "test-key", entry.Key)
	assert.Equal(t, []byte("test-value"), entry.Value)
	assert.Equal(t, int64(5), entry.HitCount)
	assert.Equal(t, 3600, entry.TTL)
	assert.Equal(t, expiresAt, entry.ExpiresAt)
}

// TestCacheEntryExpiration tests cache entry expiration
func TestCacheEntryExpiration(t *testing.T) {
	now := time.Now()

	// Expired entry
	expiredEntry := &CacheEntry{
		Key:       "expired",
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	// Valid entry
	validEntry := &CacheEntry{
		Key:       "valid",
		ExpiresAt: now.Add(1 * time.Hour),
	}

	assert.True(t, expiredEntry.ExpiresAt.Before(now))
	assert.True(t, validEntry.ExpiresAt.After(now))
}

// TestQueryResultCreation tests QueryResult creation
func TestQueryResultCreation(t *testing.T) {
	events := []blockchain.BlockchainEvent{
		{
			ID:        "event-1",
			EventName: "Transfer",
		},
		{
			ID:        "event-2",
			EventName: "Approval",
		},
	}

	result := &QueryResult{
		Events:       events,
		Total:        2,
		CacheHit:     true,
		ResponseTime: 100,
	}

	assert.Equal(t, 2, len(result.Events))
	assert.Equal(t, int64(2), result.Total)
	assert.True(t, result.CacheHit)
	assert.Equal(t, int64(100), result.ResponseTime)
}

// TestQueryResultEmpty tests empty QueryResult
func TestQueryResultEmpty(t *testing.T) {
	result := &QueryResult{
		Events:       []blockchain.BlockchainEvent{},
		Total:        0,
		CacheHit:     false,
		ResponseTime: 50,
	}

	assert.Equal(t, 0, len(result.Events))
	assert.Equal(t, int64(0), result.Total)
	assert.False(t, result.CacheHit)
}

// TestReorgStatsCreation tests ReorgStats creation
func TestReorgStatsCreation(t *testing.T) {
	now := time.Now()

	stats := &ReorgStats{
		TotalReorgsDetected:   5,
		TotalBlocksRolledBack: 100,
		AverageReorgSize:      20.0,
		LastReorgTime:         now,
		LastReorgBlock:        12345,
	}

	assert.Equal(t, uint64(5), stats.TotalReorgsDetected)
	assert.Equal(t, uint64(100), stats.TotalBlocksRolledBack)
	assert.Equal(t, 20.0, stats.AverageReorgSize)
	assert.Equal(t, now, stats.LastReorgTime)
	assert.Equal(t, uint64(12345), stats.LastReorgBlock)
}

// TestReorgStatsZero tests ReorgStats with zero values
func TestReorgStatsZero(t *testing.T) {
	stats := &ReorgStats{}

	assert.Equal(t, uint64(0), stats.TotalReorgsDetected)
	assert.Equal(t, uint64(0), stats.TotalBlocksRolledBack)
	assert.Equal(t, 0.0, stats.AverageReorgSize)
	assert.True(t, stats.LastReorgTime.IsZero())
	assert.Equal(t, uint64(0), stats.LastReorgBlock)
}

// TestCacheEntryHitCount tests cache entry hit count
func TestCacheEntryHitCount(t *testing.T) {
	entry := &CacheEntry{
		Key:      "test",
		HitCount: 0,
	}

	assert.Equal(t, int64(0), entry.HitCount)

	entry.HitCount++
	assert.Equal(t, int64(1), entry.HitCount)

	entry.HitCount += 10
	assert.Equal(t, int64(11), entry.HitCount)
}

// TestSystemErrorWithWrappedError tests SystemError with wrapped error
func TestSystemErrorWithWrappedError(t *testing.T) {
	originalErr := assert.AnError
	err := &SystemError{
		Type:    ErrorTypeCritical,
		Message: "Wrapped error",
		Code:    "WRAPPED",
		Err:     originalErr,
	}

	assert.Equal(t, originalErr, err.Err)
	assert.Equal(t, ErrorTypeCritical, err.Type)
}

// TestQueryResultCacheHit tests QueryResult cache hit scenarios
func TestQueryResultCacheHit(t *testing.T) {
	tests := []struct {
		name         string
		cacheHit     bool
		responseTime int64
	}{
		{"Cache Hit", true, 10},
		{"Cache Miss", false, 100},
		{"Cache Hit Fast", true, 5},
		{"Cache Miss Slow", false, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &QueryResult{
				CacheHit:     tt.cacheHit,
				ResponseTime: tt.responseTime,
			}
			assert.Equal(t, tt.cacheHit, result.CacheHit)
			assert.Equal(t, tt.responseTime, result.ResponseTime)
		})
	}
}
