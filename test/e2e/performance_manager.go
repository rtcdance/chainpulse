package e2e

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceManager manages performance testing
type PerformanceManager struct {
	mu              sync.RWMutex
	metrics         map[string]*PerformanceMetrics
	latencies       map[string][]time.Duration
	startTime       time.Time
	operationCounts map[string]int64
}

// NewPerformanceManager creates a new performance manager
func NewPerformanceManager() *PerformanceManager {
	return &PerformanceManager{
		metrics:         make(map[string]*PerformanceMetrics),
		latencies:       make(map[string][]time.Duration),
		startTime:       time.Now(),
		operationCounts: make(map[string]int64),
	}
}

// RecordOperation records a single operation's latency
func (pm *PerformanceManager) RecordOperation(name string, duration time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.latencies[name]; !exists {
		pm.latencies[name] = make([]time.Duration, 0)
		pm.metrics[name] = &PerformanceMetrics{
			OperationName:       name,
			TotalOperations:     0,
			SuccessfulOps:       0,
			FailedOps:           0,
			TotalDuration:       0,
			AverageDuration:     0,
			MinDuration:         0,
			MaxDuration:         0,
			ThroughputOpsPerSec: 0,
		}
	}

	pm.latencies[name] = append(pm.latencies[name], duration)
	pm.operationCounts[name]++

	metric := pm.metrics[name]
	metric.TotalOperations++
	if success {
		metric.SuccessfulOps++
	} else {
		metric.FailedOps++
	}
	metric.TotalDuration += duration

	// Update min/max
	if metric.MinDuration == 0 || duration < metric.MinDuration {
		metric.MinDuration = duration
	}
	if duration > metric.MaxDuration {
		metric.MaxDuration = duration
	}
}

// MeasureOperation measures the duration of an operation
func (pm *PerformanceManager) MeasureOperation(ctx context.Context, name string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	pm.RecordOperation(name, duration, err == nil)
	return err
}

// CalculateMetrics calculates final metrics for all operations
func (pm *PerformanceManager) CalculateMetrics() map[string]*PerformanceMetrics {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, metric := range pm.metrics {
		latencies := pm.latencies[name]
		if len(latencies) > 0 {
			// Calculate average duration
			metric.AverageDuration = metric.TotalDuration / time.Duration(len(latencies))

			// Calculate throughput
			elapsed := time.Since(pm.startTime)
			if elapsed > 0 {
				metric.ThroughputOpsPerSec = float64(len(latencies)) / elapsed.Seconds()
			}
		}
	}

	return pm.metrics
}

// GetMetrics returns metrics for a specific operation
func (pm *PerformanceManager) GetMetrics(name string) *PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if metric, exists := pm.metrics[name]; exists {
		return metric
	}
	return nil
}

// BenchmarkEventProcessing benchmarks event processing performance
func (pm *PerformanceManager) BenchmarkEventProcessing(ctx context.Context, eventCount int, concurrency int) error {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var successCount int64

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			err := pm.MeasureOperation(ctx, "event_processing", func() error {
				// Simulate event processing
				time.Sleep(time.Millisecond)
				return nil
			})

			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	return nil
}

// BenchmarkDatabaseOperations benchmarks database operation performance
func (pm *PerformanceManager) BenchmarkDatabaseOperations(ctx context.Context, opCount int, concurrency int) error {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < opCount; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			_ = pm.MeasureOperation(ctx, "database_write", func() error {
				// Simulate database write
				time.Sleep(time.Millisecond * 2)
				return nil
			})

			_ = pm.MeasureOperation(ctx, "database_read", func() error {
				// Simulate database read
				time.Sleep(time.Millisecond)
				return nil
			})
		}(i)
	}

	wg.Wait()
	return nil
}

// BenchmarkAPIQueries benchmarks API query performance
func (pm *PerformanceManager) BenchmarkAPIQueries(ctx context.Context, queryCount int, concurrency int) error {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < queryCount; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			_ = pm.MeasureOperation(ctx, "api_query", func() error {
				// Simulate API query
				time.Sleep(time.Millisecond * 5)
				return nil
			})
		}(i)
	}

	wg.Wait()
	return nil
}

// BenchmarkEndToEndLatency benchmarks end-to-end latency
func (pm *PerformanceManager) BenchmarkEndToEndLatency(ctx context.Context, iterations int) error {
	for i := 0; i < iterations; i++ {
		err := pm.MeasureOperation(ctx, "end_to_end_latency", func() error {
			// Simulate full pipeline: event -> processing -> storage -> query
			time.Sleep(time.Millisecond * 10)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// PrintMetrics prints formatted metrics
func (pm *PerformanceManager) PrintMetrics() {
	metrics := pm.CalculateMetrics()

	fmt.Println("\n=== Performance Metrics ===")
	for name, metric := range metrics {
		fmt.Printf("\nOperation: %s\n", name)
		fmt.Printf("  Total Operations: %d\n", metric.TotalOperations)
		fmt.Printf("  Successful: %d\n", metric.SuccessfulOps)
		fmt.Printf("  Failed: %d\n", metric.FailedOps)
		fmt.Printf("  Average Duration: %v\n", metric.AverageDuration)
		fmt.Printf("  Min Duration: %v\n", metric.MinDuration)
		fmt.Printf("  Max Duration: %v\n", metric.MaxDuration)
		fmt.Printf("  Throughput: %.2f ops/sec\n", metric.ThroughputOpsPerSec)
	}
}
