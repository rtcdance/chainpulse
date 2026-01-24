package processing

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewIdempotencyService tests service creation
func TestNewIdempotencyService(t *testing.T) {
	ttl := 1 * time.Hour
	service := NewIdempotencyService(ttl)

	assert.NotNil(t, service)
	assert.NotNil(t, service.processedEvents)
	assert.Equal(t, ttl, service.recordTTL)
	assert.Equal(t, 5*time.Minute, service.cleanupInterval)
	assert.Equal(t, int64(0), service.duplicateCount)
	assert.Equal(t, int64(0), service.checkCount)
}

// TestIsDuplicateNotProcessed tests checking non-processed event
func TestIsDuplicateNotProcessed(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	isDuplicate, err := service.IsDuplicate(ctx, "event-hash-1")

	assert.NoError(t, err)
	assert.False(t, isDuplicate)
}

// TestIsDuplicateEmptyHash tests checking with empty hash
func TestIsDuplicateEmptyHash(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_, err := service.IsDuplicate(ctx, "")

	assert.Error(t, err)
}

// TestMarkProcessed tests marking event as processed
func TestMarkProcessed(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	err := service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), service.GetProcessedCount())
}

// TestMarkProcessedEmptyHash tests marking with empty hash
func TestMarkProcessedEmptyHash(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	err := service.MarkProcessed(ctx, "", "ethereum", "tx-1", "success")

	assert.Error(t, err)
}

// TestMarkProcessedDefaultStatus tests marking with default status
func TestMarkProcessedDefaultStatus(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	err := service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "")

	assert.NoError(t, err)

	record, err := service.GetProcessedRecord(ctx, "event-hash-1")
	assert.NoError(t, err)
	assert.Equal(t, "success", record.Status)
}

// TestIsDuplicateAfterMarkProcessed tests duplicate detection after marking
func TestIsDuplicateAfterMarkProcessed(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	isDuplicate, err := service.IsDuplicate(ctx, "event-hash-1")

	assert.NoError(t, err)
	assert.True(t, isDuplicate)
}

// TestGetProcessedRecord tests retrieving processed record
func TestGetProcessedRecord(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	record, err := service.GetProcessedRecord(ctx, "event-hash-1")

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "event-hash-1", record.EventHash)
	assert.Equal(t, "ethereum", record.ChainID)
	assert.Equal(t, "tx-1", record.TransactionID)
	assert.Equal(t, "success", record.Status)
}

// TestGetProcessedRecordEmptyHash tests retrieving with empty hash
func TestGetProcessedRecordEmptyHash(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_, err := service.GetProcessedRecord(ctx, "")

	assert.Error(t, err)
}

// TestGetProcessedRecordNotFound tests retrieving non-existent record
func TestGetProcessedRecordNotFound(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_, err := service.GetProcessedRecord(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestGetProcessedRecordExpired tests retrieving expired record
func TestGetProcessedRecordExpired(t *testing.T) {
	service := NewIdempotencyService(100 * time.Millisecond)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, err := service.GetProcessedRecord(ctx, "event-hash-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestGetDuplicateCount tests getting duplicate count
func TestGetDuplicateCount(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	// First check - is duplicate
	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, int64(1), service.GetDuplicateCount())

	// Second check - is duplicate
	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, int64(2), service.GetDuplicateCount())
}

// TestGetCheckCount tests getting check count
func TestGetCheckCount(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	assert.Equal(t, int64(0), service.GetCheckCount())

	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, int64(1), service.GetCheckCount())

	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, int64(2), service.GetCheckCount())
}

// TestGetProcessedCountIdempotency tests getting processed count
func TestGetProcessedCountIdempotency(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	assert.Equal(t, int64(0), service.GetProcessedCount())

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")
	assert.Equal(t, int64(1), service.GetProcessedCount())

	_ = service.MarkProcessed(ctx, "event-hash-2", "ethereum", "tx-2", "success")
	assert.Equal(t, int64(2), service.GetProcessedCount())
}

// TestGetDuplicateRate tests getting duplicate rate
func TestGetDuplicateRate(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	// No checks yet
	assert.Equal(t, 0.0, service.GetDuplicateRate())

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	// 1 check, 1 duplicate (event was already processed)
	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, 1.0, service.GetDuplicateRate())

	// 2 checks, 2 duplicates
	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, 1.0, service.GetDuplicateRate())

	// 3 checks, 3 duplicates
	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	assert.Equal(t, 1.0, service.GetDuplicateRate())
}

// TestReset tests resetting service
func TestReset(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")
	_, _ = service.IsDuplicate(ctx, "event-hash-1")
	_, _ = service.IsDuplicate(ctx, "event-hash-1")

	assert.Equal(t, int64(2), service.GetDuplicateCount())
	assert.Equal(t, int64(2), service.GetCheckCount())
	assert.Equal(t, int64(1), service.GetProcessedCount())

	service.Reset()

	assert.Equal(t, int64(0), service.GetDuplicateCount())
	assert.Equal(t, int64(0), service.GetCheckCount())
	assert.Equal(t, int64(0), service.GetProcessedCount())
}

// TestGetMetricsIdempotency tests getting metrics
func TestGetMetricsIdempotency(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")
	_, _ = service.IsDuplicate(ctx, "event-hash-1")

	metrics := service.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics["processed_count"])
	assert.Equal(t, int64(1), metrics["duplicate_count"])
	assert.Equal(t, int64(1), metrics["check_count"])
	assert.Contains(t, metrics, "duplicate_rate")
	assert.Contains(t, metrics, "record_ttl")
	assert.Contains(t, metrics, "cleanup_interval")
}

// TestValidateIdempotencyNilEvent tests validation with nil event
func TestValidateIdempotencyNilEvent(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	err := service.ValidateIdempotency(ctx, nil)

	assert.Error(t, err)
}

// TestValidateIdempotencyEmptyHash tests validation with empty hash
func TestValidateIdempotencyEmptyHash(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	event := &Event{EventHash: ""}

	err := service.ValidateIdempotency(ctx, event)

	assert.Error(t, err)
}

// TestValidateIdempotencyNewEvent tests validation with new event
func TestValidateIdempotencyNewEvent(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	event := &Event{EventHash: "event-hash-1"}

	err := service.ValidateIdempotency(ctx, event)

	assert.NoError(t, err)
}

// TestValidateIdempotencyDuplicateEvent tests validation with duplicate event
func TestValidateIdempotencyDuplicateEvent(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	event := &Event{EventHash: "event-hash-1"}

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	err := service.ValidateIdempotency(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

// TestBatchValidateIdempotencyEmpty tests batch validation with empty list
func TestBatchValidateIdempotencyEmpty(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	results, err := service.BatchValidateIdempotency(ctx, []*Event{})

	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

// TestBatchValidateIdempotencyNewEvents tests batch validation with new events
func TestBatchValidateIdempotencyNewEvents(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	events := []*Event{
		{EventHash: "event-hash-1"},
		{EventHash: "event-hash-2"},
		{EventHash: "event-hash-3"},
	}

	results, err := service.BatchValidateIdempotency(ctx, events)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))
	assert.False(t, results[0])
	assert.False(t, results[1])
	assert.False(t, results[2])
}

// TestBatchValidateIdempotencyMixedEvents tests batch validation with mixed events
func TestBatchValidateIdempotencyMixedEvents(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")
	_ = service.MarkProcessed(ctx, "event-hash-3", "ethereum", "tx-3", "success")

	events := []*Event{
		{EventHash: "event-hash-1"},
		{EventHash: "event-hash-2"},
		{EventHash: "event-hash-3"},
	}

	results, err := service.BatchValidateIdempotency(ctx, events)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))
	assert.True(t, results[0])
	assert.False(t, results[1])
	assert.True(t, results[2])
}

// TestBatchValidateIdempotencyNilEvents tests batch validation with nil events
func TestBatchValidateIdempotencyNilEvents(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	events := []*Event{
		{EventHash: "event-hash-1"},
		nil,
		{EventHash: "event-hash-3"},
	}

	results, err := service.BatchValidateIdempotency(ctx, events)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))
	assert.False(t, results[0])
	assert.False(t, results[1])
	assert.False(t, results[2])
}

// TestMultipleProcessedRecords tests handling multiple processed records
func TestMultipleProcessedRecords(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		hash := "event-hash-" + string(rune(48+i))
		_ = service.MarkProcessed(ctx, hash, "ethereum", "tx-"+string(rune(48+i)), "success")
	}

	assert.Equal(t, int64(10), service.GetProcessedCount())

	for i := 1; i <= 10; i++ {
		hash := "event-hash-" + string(rune(48+i))
		isDuplicate, err := service.IsDuplicate(ctx, hash)
		assert.NoError(t, err)
		assert.True(t, isDuplicate)
	}
}

// TestConcurrentMarkProcessed tests concurrent marking
func TestConcurrentMarkProcessed(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			hash := "event-hash-" + string(rune(48+(id%10)))
			_ = service.MarkProcessed(ctx, hash, "ethereum", "tx-"+string(rune(48+id)), "success")
			atomic.AddInt32(&counter, 1)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&counter))
}

// TestConcurrentIsDuplicate tests concurrent duplicate checking
func TestConcurrentIsDuplicate(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	var wg sync.WaitGroup
	var duplicateCount int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			isDuplicate, err := service.IsDuplicate(ctx, "event-hash-1")
			if err == nil && isDuplicate {
				atomic.AddInt32(&duplicateCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&duplicateCount))
}

// TestRecordExpiration tests record expiration
func TestRecordExpiration(t *testing.T) {
	service := NewIdempotencyService(100 * time.Millisecond)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	// Should be duplicate immediately
	isDuplicate, err := service.IsDuplicate(ctx, "event-hash-1")
	assert.NoError(t, err)
	assert.True(t, isDuplicate)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be duplicate after expiration
	isDuplicate, err = service.IsDuplicate(ctx, "event-hash-1")
	assert.NoError(t, err)
	assert.False(t, isDuplicate)
}

// TestProcessedRecordFields tests processed record fields
func TestProcessedRecordFields(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	chainID := "ethereum"
	txID := "tx-123"
	status := "success"

	_ = service.MarkProcessed(ctx, "event-hash-1", chainID, txID, status)

	record, err := service.GetProcessedRecord(ctx, "event-hash-1")

	assert.NoError(t, err)
	assert.Equal(t, "event-hash-1", record.EventHash)
	assert.Equal(t, chainID, record.ChainID)
	assert.Equal(t, txID, record.TransactionID)
	assert.Equal(t, status, record.Status)
	assert.NotZero(t, record.ProcessedAt)
	assert.NotZero(t, record.ExpiresAt)
	assert.True(t, record.ExpiresAt.After(record.ProcessedAt))
}

// TestDifferentStatuses tests different event statuses
func TestDifferentStatuses(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	statuses := []string{"success", "failed", "pending"}

	for i, status := range statuses {
		hash := "event-hash-" + string(rune(49+i))
		_ = service.MarkProcessed(ctx, hash, "ethereum", "tx-"+string(rune(49+i)), status)
	}

	for i, status := range statuses {
		hash := "event-hash-" + string(rune(49+i))
		record, err := service.GetProcessedRecord(ctx, hash)
		assert.NoError(t, err)
		assert.Equal(t, status, record.Status)
	}
}

// TestCleanupExpiredRecords tests cleanup of expired records
func TestCleanupExpiredRecords(t *testing.T) {
	service := NewIdempotencyService(100 * time.Millisecond)
	ctx := context.Background()

	// Mark multiple events
	for i := 1; i <= 5; i++ {
		hash := "event-hash-" + string(rune(48+i))
		_ = service.MarkProcessed(ctx, hash, "ethereum", "tx-"+string(rune(48+i)), "success")
	}

	assert.Equal(t, int64(5), service.GetProcessedCount())

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup by calling cleanupExpiredRecords
	// (normally triggered by MarkProcessed after cleanupInterval)
	service.mu.Lock()
	service.cleanupExpiredRecords()
	service.mu.Unlock()

	// Old records should be cleaned up
	assert.Equal(t, int64(0), service.GetProcessedCount())
}

// TestDuplicateRateCalculation tests duplicate rate calculation
func TestDuplicateRateCalculation(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	// Mark 3 events
	for i := 1; i <= 3; i++ {
		hash := "event-hash-" + strconv.Itoa(i)
		_ = service.MarkProcessed(ctx, hash, "ethereum", "tx-"+strconv.Itoa(i), "success")
	}

	// Check each event twice (both checks are duplicates since record exists)
	for i := 1; i <= 3; i++ {
		hash := "event-hash-" + strconv.Itoa(i)
		_, _ = service.IsDuplicate(ctx, hash)
		_, _ = service.IsDuplicate(ctx, hash)
	}

	// 6 checks total, 6 duplicates (all checks find existing records)
	assert.Equal(t, int64(6), service.GetCheckCount())
	assert.Equal(t, int64(6), service.GetDuplicateCount())
	// Duplicate rate should be 6/6 = 1.0
	assert.InDelta(t, 1.0, service.GetDuplicateRate(), 0.01)
}

// TestProcessedRecordTimestamps tests processed record timestamps
func TestProcessedRecordTimestamps(t *testing.T) {
	service := NewIdempotencyService(1 * time.Hour)
	ctx := context.Background()

	beforeMark := time.Now()
	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")
	afterMark := time.Now()

	record, err := service.GetProcessedRecord(ctx, "event-hash-1")

	assert.NoError(t, err)
	assert.True(t, record.ProcessedAt.After(beforeMark) || record.ProcessedAt.Equal(beforeMark))
	assert.True(t, record.ProcessedAt.Before(afterMark) || record.ProcessedAt.Equal(afterMark))
	assert.True(t, record.ExpiresAt.After(record.ProcessedAt))
}

// TestBatchValidateIdempotencyExpiredRecords tests batch validation with expired records
func TestBatchValidateIdempotencyExpiredRecords(t *testing.T) {
	service := NewIdempotencyService(100 * time.Millisecond)
	ctx := context.Background()

	_ = service.MarkProcessed(ctx, "event-hash-1", "ethereum", "tx-1", "success")

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	events := []*Event{
		{EventHash: "event-hash-1"},
	}

	results, err := service.BatchValidateIdempotency(ctx, events)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.False(t, results[0])
}
