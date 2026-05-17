package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

// BenchmarkBatchInsert benchmarks batch insert performance
func BenchmarkBatchInsert(b *testing.B) {
	requirePostgresIntegration(b)

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: core.SecretString("chainpulse"),
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		b.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		if err := db.Stop(); err != nil {
			b.Logf("Failed to stop: %v", err)
		}
	}()

	// Create test events
	events := make([]core.BlockchainEvent, 1000)
	for i := 0; i < 1000; i++ {
		events[i] = core.BlockchainEvent{
			EventHash:       fmt.Sprintf("bench-hash-%d", i),
			BlockNumber:     uint64(i),
			TransactionHash: common.HexToHash(fmt.Sprintf("bench-tx-%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
			BlockTimestamp:  time.Now().Unix(),
		}
	}

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		err := db.WriteEvents(context.Background(), events)
		if err != nil {
			b.Fatalf("Failed to write events: %v", err)
		}
	}
}

// TestBatchInsertPerformance tests batch insert performance
func TestBatchInsertPerformance(t *testing.T) {
	requirePostgresIntegration(t)

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: core.SecretString("chainpulse"),
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Create test events
	batchSize := 1000
	events := make([]core.BlockchainEvent, batchSize)
	for i := 0; i < batchSize; i++ {
		events[i] = core.BlockchainEvent{
			EventHash:       fmt.Sprintf("perf-hash-%d-%d", time.Now().UnixNano(), i),
			BlockNumber:     uint64(i),
			TransactionHash: common.HexToHash(fmt.Sprintf("perf-tx-%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
			BlockTimestamp:  time.Now().Unix(),
		}
	}

	// Measure write time
	start := time.Now()
	err = db.WriteEvents(context.Background(), events)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to write events: %v", err)
	}

	// Calculate throughput
	throughput := float64(batchSize) / duration.Seconds()
	t.Logf("Throughput: %.0f events/second (duration: %v)", throughput, duration)

	// Target: 1000+ events/second
	if throughput < 1000 {
		t.Logf("WARNING: Throughput below target (%.0f < 1000)", throughput)
	} else {
		t.Logf("SUCCESS: Throughput meets target (%.0f >= 1000)", throughput)
	}
}

// TestBatchInsertVariousSizes tests performance with different batch sizes
func TestBatchInsertVariousSizes(t *testing.T) {
	requirePostgresIntegration(t)

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: core.SecretString("chainpulse"),
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Test different batch sizes
	batchSizes := []int{100, 500, 1000, 5000}

	for _, batchSize := range batchSizes {
		events := make([]core.BlockchainEvent, batchSize)
		for i := 0; i < batchSize; i++ {
			events[i] = core.BlockchainEvent{
				EventHash:       fmt.Sprintf("batch-size-test-%d-%d-%d", batchSize, time.Now().UnixNano(), i),
				BlockNumber:     uint64(i),
				TransactionHash: common.HexToHash(fmt.Sprintf("batch-tx-%d", i)),
				LogIndex:        uint64(i),
				ContractAddress: common.HexToAddress("0x123"),
				EventName:       "Transfer",
				EventData:       []byte("data1"),
				BlockTimestamp:  time.Now().Unix(),
			}
		}

		start := time.Now()
		err := db.WriteEvents(context.Background(), events)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to write events (batch size %d): %v", batchSize, err)
		}

		throughput := float64(batchSize) / duration.Seconds()
		t.Logf("Batch size %d: %.0f events/second (duration: %v)", batchSize, throughput, duration)
	}
}

// TestSingleEventPerformance tests single event write performance
func TestSingleEventPerformance(t *testing.T) {
	requirePostgresIntegration(t)

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: core.SecretString("chainpulse"),
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Write 100 single events and measure
	numEvents := 100
	start := time.Now()

	for i := 0; i < numEvents; i++ {
		event := &core.BlockchainEvent{
			EventHash:       fmt.Sprintf("single-event-%d-%d", time.Now().UnixNano(), i),
			BlockNumber:     uint64(i),
			TransactionHash: common.HexToHash(fmt.Sprintf("single-tx-%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
			BlockTimestamp:  time.Now().Unix(),
		}

		err := db.WriteEvent(context.Background(), event)
		if err != nil {
			t.Fatalf("Failed to write event %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	throughput := float64(numEvents) / duration.Seconds()

	t.Logf("Single event performance: %.0f events/second (duration: %v)", throughput, duration)
}

// TestQueryPerformance tests query performance
func TestQueryPerformance(t *testing.T) {
	requirePostgresIntegration(t)

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: core.SecretString("chainpulse"),
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		if err := db.Stop(); err != nil {
			t.Logf("Failed to stop: %v", err)
		}
	}()

	// Write test events
	numEvents := 1000
	events := make([]core.BlockchainEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = core.BlockchainEvent{
			EventHash:       fmt.Sprintf("query-perf-%d-%d", time.Now().UnixNano(), i),
			BlockNumber:     uint64(i),
			TransactionHash: common.HexToHash(fmt.Sprintf("query-tx-%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
			BlockTimestamp:  time.Now().Unix(),
		}
	}

	err = db.WriteEvents(context.Background(), events)
	if err != nil {
		t.Fatalf("Failed to write events: %v", err)
	}

	// Query performance
	filter := &core.EventFilter{
		Limit:  100,
		Offset: 0,
	}

	start := time.Now()
	result, err := db.QueryEvents(filter)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to query events: %v", err)
	}

	t.Logf("Query performance: %d events in %v (%.2f ms per event)", len(result.Events), duration, float64(duration.Milliseconds())/float64(len(result.Events)))
}
