package e2e

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ConcurrentEventProcessor manages concurrent event processing for E2E testing
type ConcurrentEventProcessor interface {
	// Lifecycle management
	Startup(ctx context.Context) error
	Shutdown(ctx context.Context) error

	// Event generation
	GenerateEventsAsync(ctx context.Context, count int, concurrency int) error
	GenerateEventsWithDelay(ctx context.Context, count int, concurrency int, delay time.Duration) error

	// Race condition detection
	DetectRaceConditions(ctx context.Context) ([]RaceCondition, error)
	ValidateConcurrentOrdering(ctx context.Context) error

	// Database operations
	ConcurrentDatabaseWrites(ctx context.Context, count int, concurrency int) error
	ConcurrentDatabaseReads(ctx context.Context, count int, concurrency int) error

	// Metrics
	GetMetrics() ConcurrentMetrics
	Reset()
}

// RaceCondition represents a detected race condition
type RaceCondition struct {
	EventID1    string
	EventID2    string
	Timestamp1  time.Time
	Timestamp2  time.Time
	Description string
}

// ConcurrentMetrics tracks metrics for concurrent operations
type ConcurrentMetrics struct {
	TotalEventsGenerated   int64
	TotalEventsProcessed   int64
	ConcurrentOperations   int
	RaceConditionsDetected int
	OrderingViolations     int
	DatabaseWriteErrors    int
	DatabaseReadErrors     int
	AverageLatency         time.Duration
	MaxLatency             time.Duration
	MinLatency             time.Duration
	Throughput             float64 // events per second
	LastUpdated            time.Time
}

// DefaultConcurrentEventProcessor implements ConcurrentEventProcessor
type DefaultConcurrentEventProcessor struct {
	blockchainManager   *BlockchainManager
	mu                  sync.RWMutex
	eventsGenerated     int64
	eventsProcessed     int64
	raceConditions      []RaceCondition
	orderingViolations  int
	databaseWriteErrors int
	databaseReadErrors  int
	latencies           []time.Duration
	startTime           time.Time
	eventTimestamps     map[string]time.Time
	eventTimestampsMu   sync.RWMutex
}

// NewDefaultConcurrentEventProcessor creates a new concurrent event processor
func NewDefaultConcurrentEventProcessor(bm *BlockchainManager) *DefaultConcurrentEventProcessor {
	return &DefaultConcurrentEventProcessor{
		blockchainManager: bm,
		eventTimestamps:   make(map[string]time.Time),
		latencies:         make([]time.Duration, 0),
		startTime:         time.Now(),
	}
}

// Startup initializes the concurrent event processor
func (p *DefaultConcurrentEventProcessor) Startup(ctx context.Context) error {
	if p.blockchainManager == nil {
		return fmt.Errorf("blockchain manager not initialized")
	}
	return nil
}

// Shutdown cleans up resources
func (p *DefaultConcurrentEventProcessor) Shutdown(ctx context.Context) error {
	return nil
}

// GenerateEventsAsync generates events concurrently
func (p *DefaultConcurrentEventProcessor) GenerateEventsAsync(ctx context.Context, count int, concurrency int) error {
	if concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive, got %d", concurrency)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, concurrency)
	eventsPerWorker := count / concurrency

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerWorker; j++ {
				eventID := fmt.Sprintf("event-w%d-e%d", workerID, j)
				contractAddr := fmt.Sprintf("0x%x", workerID*1000+j)
				eventName := fmt.Sprintf("Event%d", j)
				blockNumber, err := nonNegativeIntToUint64(j)
				if err != nil {
					errChan <- fmt.Errorf("worker %d: %w", workerID, err)
					return
				}
				params := map[string]any{
					"blockNumber":      blockNumber,
					"transactionIndex": blockNumber,
				}

				start := time.Now()
				if _, err := p.blockchainManager.EmitEvent(ctx, contractAddr, eventName, params); err != nil {
					errChan <- fmt.Errorf("worker %d: %w", workerID, err)
					return
				}
				latency := time.Since(start)

				p.recordEvent(eventID, latency)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// GenerateEventsWithDelay generates events with delays between them
func (p *DefaultConcurrentEventProcessor) GenerateEventsWithDelay(ctx context.Context, count int, concurrency int, delay time.Duration) error {
	var wg sync.WaitGroup
	errChan := make(chan error, concurrency)
	eventsPerWorker := count / concurrency

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerWorker; j++ {
				time.Sleep(delay)

				eventID := fmt.Sprintf("event-delayed-w%d-e%d", workerID, j)
				contractAddr := fmt.Sprintf("0x%x", workerID*1000+j)
				eventName := fmt.Sprintf("Event%d", j)
				blockNumber, err := nonNegativeIntToUint64(j)
				if err != nil {
					errChan <- fmt.Errorf("worker %d: %w", workerID, err)
					return
				}
				params := map[string]any{
					"blockNumber":      blockNumber,
					"transactionIndex": blockNumber,
				}

				start := time.Now()
				if _, err := p.blockchainManager.EmitEvent(ctx, contractAddr, eventName, params); err != nil {
					errChan <- fmt.Errorf("worker %d: %w", workerID, err)
					return
				}
				latency := time.Since(start)

				p.recordEvent(eventID, latency)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// DetectRaceConditions detects race conditions in event processing
func (p *DefaultConcurrentEventProcessor) DetectRaceConditions(ctx context.Context) ([]RaceCondition, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	raceConditions := make([]RaceCondition, 0)

	// Check for events with same timestamp (potential race condition)
	timestampMap := make(map[time.Time][]string)
	p.eventTimestampsMu.RLock()
	for eventID, ts := range p.eventTimestamps {
		timestampMap[ts] = append(timestampMap[ts], eventID)
	}
	p.eventTimestampsMu.RUnlock()

	// Events with same timestamp might indicate race condition
	for ts, eventIDs := range timestampMap {
		if len(eventIDs) > 1 {
			for i := 0; i < len(eventIDs)-1; i++ {
				raceConditions = append(raceConditions, RaceCondition{
					EventID1:    eventIDs[i],
					EventID2:    eventIDs[i+1],
					Timestamp1:  ts,
					Timestamp2:  ts,
					Description: "Events processed at same timestamp",
				})
			}
		}
	}

	return raceConditions, nil
}

func nonNegativeIntToUint64(value int) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("negative value %d cannot be converted to uint64", value)
	}
	return uint64(value), nil
}

// ValidateConcurrentOrdering validates that concurrent operations maintain ordering
func (p *DefaultConcurrentEventProcessor) ValidateConcurrentOrdering(ctx context.Context) error {
	p.eventTimestampsMu.RLock()
	defer p.eventTimestampsMu.RUnlock()

	var lastTimestamp time.Time
	for _, ts := range p.eventTimestamps {
		if !lastTimestamp.IsZero() && ts.Before(lastTimestamp) {
			p.mu.Lock()
			p.orderingViolations++
			p.mu.Unlock()
			return fmt.Errorf("ordering violation detected: event timestamp %v before previous %v", ts, lastTimestamp)
		}
		lastTimestamp = ts
	}

	return nil
}

// ConcurrentDatabaseWrites performs concurrent database writes
func (p *DefaultConcurrentEventProcessor) ConcurrentDatabaseWrites(ctx context.Context, count int, concurrency int) error {
	var wg sync.WaitGroup
	var writeErrors int64
	writesPerWorker := count / concurrency

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < writesPerWorker; j++ {
				// Simulate database write
				if j%100 == 0 {
					// Simulate occasional errors
					atomic.AddInt64(&writeErrors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	p.mu.Lock()
	p.databaseWriteErrors += int(writeErrors)
	p.mu.Unlock()

	return nil
}

// ConcurrentDatabaseReads performs concurrent database reads
func (p *DefaultConcurrentEventProcessor) ConcurrentDatabaseReads(ctx context.Context, count int, concurrency int) error {
	var wg sync.WaitGroup
	var readErrors int64
	readsPerWorker := count / concurrency

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < readsPerWorker; j++ {
				// Simulate database read
				if j%100 == 0 {
					// Simulate occasional errors
					atomic.AddInt64(&readErrors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	p.mu.Lock()
	p.databaseReadErrors += int(readErrors)
	p.mu.Unlock()

	return nil
}

// GetMetrics returns metrics for concurrent operations
func (p *DefaultConcurrentEventProcessor) GetMetrics() ConcurrentMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgLatency := time.Duration(0)
	minLatency := time.Duration(0)
	maxLatency := time.Duration(0)

	if len(p.latencies) > 0 {
		total := time.Duration(0)
		minLatency = p.latencies[0]
		maxLatency = p.latencies[0]

		for _, lat := range p.latencies {
			total += lat
			if lat < minLatency {
				minLatency = lat
			}
			if lat > maxLatency {
				maxLatency = lat
			}
		}
		avgLatency = total / time.Duration(len(p.latencies))
	}

	elapsed := time.Since(p.startTime).Seconds()
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(p.eventsGenerated) / elapsed
	}

	return ConcurrentMetrics{
		TotalEventsGenerated:   p.eventsGenerated,
		TotalEventsProcessed:   p.eventsProcessed,
		RaceConditionsDetected: len(p.raceConditions),
		OrderingViolations:     p.orderingViolations,
		DatabaseWriteErrors:    p.databaseWriteErrors,
		DatabaseReadErrors:     p.databaseReadErrors,
		AverageLatency:         avgLatency,
		MaxLatency:             maxLatency,
		MinLatency:             minLatency,
		Throughput:             throughput,
		LastUpdated:            time.Now(),
	}
}

// Reset resets all metrics
func (p *DefaultConcurrentEventProcessor) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.eventsGenerated = 0
	p.eventsProcessed = 0
	p.raceConditions = make([]RaceCondition, 0)
	p.orderingViolations = 0
	p.databaseWriteErrors = 0
	p.databaseReadErrors = 0
	p.latencies = make([]time.Duration, 0)
	p.startTime = time.Now()
	p.eventTimestamps = make(map[string]time.Time)
}

// Helper methods

func (p *DefaultConcurrentEventProcessor) recordEvent(eventID string, latency time.Duration) {
	p.mu.Lock()
	p.eventsGenerated++
	p.eventsProcessed++
	p.latencies = append(p.latencies, latency)
	p.mu.Unlock()

	p.eventTimestampsMu.Lock()
	p.eventTimestamps[eventID] = time.Now()
	p.eventTimestampsMu.Unlock()
}
