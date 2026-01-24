package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
)

// ConsistencyCheckResult represents the result of a consistency check
type ConsistencyCheckResult struct {
	// CheckTime is when the check was performed
	CheckTime time.Time
	// EventCount is the total number of events
	EventCount int64
	// MetadataCount is the total number of metadata records
	MetadataCount int64
	// OrphanedMetadata is the number of metadata records without events
	OrphanedMetadata int64
	// MissingMetadata is the number of events without metadata
	MissingMetadata int64
	// CorruptedEvents is the number of events with data integrity issues
	CorruptedEvents int64
	// CorruptedMetadata is the number of metadata records with data integrity issues
	CorruptedMetadata int64
	// IsConsistent indicates if all checks passed
	IsConsistent bool
	// Issues contains a list of issues found
	Issues []string
}

// ConsistencyChecker checks data consistency between MongoDB and PostgreSQL
type ConsistencyChecker struct {
	eventStore    EventStore
	metadataStore EventMetadataStore
	logger        core.Logger
	metrics       core.MetricsCollector
	mu            sync.RWMutex
	initialized   bool
}

// NewConsistencyChecker creates a new consistency checker
func NewConsistencyChecker(eventStore EventStore, metadataStore EventMetadataStore, logger core.Logger, metrics core.MetricsCollector) *ConsistencyChecker {
	return &ConsistencyChecker{
		eventStore:    eventStore,
		metadataStore: metadataStore,
		logger:        logger,
		metrics:       metrics,
	}
}

// Initialize initializes the consistency checker
func (cc *ConsistencyChecker) Initialize(ctx context.Context) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.eventStore == nil {
		return fmt.Errorf("event store is nil")
	}

	if cc.metadataStore == nil {
		return fmt.Errorf("metadata store is nil")
	}

	cc.initialized = true
	cc.logger.Info("Consistency checker initialized")
	return nil
}

// CheckConsistency performs a full consistency check
func (cc *ConsistencyChecker) CheckConsistency(ctx context.Context) (*ConsistencyCheckResult, error) {
	cc.mu.RLock()
	if !cc.initialized {
		cc.mu.RUnlock()
		return nil, fmt.Errorf("consistency checker not initialized")
	}
	cc.mu.RUnlock()

	startTime := time.Now()
	result := &ConsistencyCheckResult{
		CheckTime: startTime,
		Issues:    []string{},
	}

	// Check event count consistency
	eventCount, err := cc.getEventCount(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to get event count: %v", err))
		cc.logger.Error("Failed to get event count", "error", err)
	}
	result.EventCount = eventCount

	// Check metadata count consistency
	metadataCount, err := cc.getMetadataCount(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to get metadata count: %v", err))
		cc.logger.Error("Failed to get metadata count", "error", err)
	}
	result.MetadataCount = metadataCount

	// Check for orphaned metadata
	orphanedCount, err := cc.checkOrphanedMetadata(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to check orphaned metadata: %v", err))
		cc.logger.Error("Failed to check orphaned metadata", "error", err)
	}
	result.OrphanedMetadata = orphanedCount
	if orphanedCount > 0 {
		result.Issues = append(result.Issues, fmt.Sprintf("Found %d orphaned metadata records", orphanedCount))
	}

	// Check for missing metadata
	missingCount, err := cc.checkMissingMetadata(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to check missing metadata: %v", err))
		cc.logger.Error("Failed to check missing metadata", "error", err)
	}
	result.MissingMetadata = missingCount
	if missingCount > 0 {
		result.Issues = append(result.Issues, fmt.Sprintf("Found %d events without metadata", missingCount))
	}

	// Check for data integrity issues
	corruptedEvents, err := cc.checkEventIntegrity(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to check event integrity: %v", err))
		cc.logger.Error("Failed to check event integrity", "error", err)
	}
	result.CorruptedEvents = corruptedEvents
	if corruptedEvents > 0 {
		result.Issues = append(result.Issues, fmt.Sprintf("Found %d events with data integrity issues", corruptedEvents))
	}

	// Determine if consistent
	result.IsConsistent = len(result.Issues) == 0 && orphanedCount == 0 && missingCount == 0 && corruptedEvents == 0

	// Record metrics
	cc.recordMetrics(result, time.Since(startTime))

	return result, nil
}

// getEventCount returns the total number of events
func (cc *ConsistencyChecker) getEventCount(ctx context.Context) (int64, error) {
	// Query all events with high limit to get count
	// This is a simplified implementation - in production, you'd want a dedicated count method
	events, err := cc.eventStore.GetEventsByChain(ctx, 0, 1000000, 0)
	if err != nil {
		return 0, err
	}
	return int64(len(events)), nil
}

// getMetadataCount returns the total number of metadata records
func (cc *ConsistencyChecker) getMetadataCount(ctx context.Context) (int64, error) {
	// Query all metadata with high limit to get count
	// This is a simplified implementation - in production, you'd want a dedicated count method
	metadata, err := cc.metadataStore.GetMetadataByChain(ctx, 0, 1000000, 0)
	if err != nil {
		return 0, err
	}
	return int64(len(metadata)), nil
}

// checkOrphanedMetadata checks for metadata records without corresponding events
func (cc *ConsistencyChecker) checkOrphanedMetadata(ctx context.Context) (int64, error) {
	// Get all metadata
	metadata, err := cc.metadataStore.GetMetadataByChain(ctx, 0, 1000000, 0)
	if err != nil {
		return 0, err
	}

	var orphanedCount int64
	for _, m := range metadata {
		// Check if event exists
		_, err := cc.eventStore.GetEvent(ctx, m.EventID)
		if err != nil {
			orphanedCount++
		}
	}

	return orphanedCount, nil
}

// checkMissingMetadata checks for events without corresponding metadata
func (cc *ConsistencyChecker) checkMissingMetadata(ctx context.Context) (int64, error) {
	// Get all events
	events, err := cc.eventStore.GetEventsByChain(ctx, 0, 1000000, 0)
	if err != nil {
		return 0, err
	}

	var missingCount int64
	for _, e := range events {
		// Check if metadata exists
		_, err := cc.metadataStore.GetMetadata(ctx, e.ID)
		if err != nil {
			missingCount++
		}
	}

	return missingCount, nil
}

// checkEventIntegrity checks for data integrity issues in events
func (cc *ConsistencyChecker) checkEventIntegrity(ctx context.Context) (int64, error) {
	// Get all events
	events, err := cc.eventStore.GetEventsByChain(ctx, 0, 1000000, 0)
	if err != nil {
		return 0, err
	}

	var corruptedCount int64
	for _, e := range events {
		// Check for required fields
		if e.ID == "" || e.ChainID == "" || e.ContractAddress == (common.Address{}) {
			corruptedCount++
		}
	}

	return corruptedCount, nil
}

// recordMetrics records consistency check metrics
func (cc *ConsistencyChecker) recordMetrics(result *ConsistencyCheckResult, duration time.Duration) {
	cc.metrics.RecordHistogram("consistency_check_duration_ms", float64(duration.Milliseconds()), nil)
	cc.metrics.RecordCounter("consistency_checks_total", 1, nil)

	if result.IsConsistent {
		cc.metrics.RecordCounter("consistency_checks_passed", 1, nil)
	} else {
		cc.metrics.RecordCounter("consistency_checks_failed", 1, nil)
	}

	cc.metrics.RecordGauge("consistency_orphaned_metadata", float64(result.OrphanedMetadata), nil)
	cc.metrics.RecordGauge("consistency_missing_metadata", float64(result.MissingMetadata), nil)
	cc.metrics.RecordGauge("consistency_corrupted_events", float64(result.CorruptedEvents), nil)
}

// Health returns the health status of the consistency checker
func (cc *ConsistencyChecker) Health(ctx context.Context) *core.HealthStatus {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if !cc.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Consistency checker not initialized",
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "Consistency checker is healthy",
	}
}

// Close closes the consistency checker
func (cc *ConsistencyChecker) Close(ctx context.Context) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.initialized = false
	cc.logger.Info("Consistency checker closed")
	return nil
}
