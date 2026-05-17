package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

// TestDatabaseIntegrationSuite runs comprehensive integration tests
func TestDatabaseIntegrationSuite(t *testing.T) {
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

	// Test 1: Initialize
	t.Run("Initialize", func(t *testing.T) {
		err := db.Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
	})

	// Test 2: Start
	t.Run("Start", func(t *testing.T) {
		err := db.Start()
		if err != nil {
			t.Fatalf("Failed to start: %v", err)
		}
	})
	defer func() {
		if err := db.Stop(); err != nil {
			t.Logf("Failed to stop: %v", err)
		}
	}()

	// Test 3: Write single event
	t.Run("WriteSingleEvent", func(t *testing.T) {
		event := &core.BlockchainEvent{
			ID:              "suite-test-hash-1",
			BlockNumber:     1,
			TransactionHash: common.HexToHash("0x1234567890abcdef"),
			LogIndex:        0,
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
		}

		err := db.WriteEvent(context.Background(), event)
		if err != nil {
			t.Fatalf("Failed to write event: %v", err)
		}
	})

	// Test 4: Write batch events
	t.Run("WriteBatchEvents", func(t *testing.T) {
		events := make([]core.BlockchainEvent, 100)
		for i := 0; i < 100; i++ {
			events[i] = core.BlockchainEvent{
				ID:              fmt.Sprintf("suite-batch-hash-%d", i),
				BlockNumber:     uint64(i),
				TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", i)),
				LogIndex:        uint64(i),
				ContractAddress: common.HexToAddress("0x123"),
				EventName:       "Transfer",
				EventData:       []byte("data1"),
			}
		}

		err := db.WriteEvents(context.Background(), events)
		if err != nil {
			t.Fatalf("Failed to write events: %v", err)
		}
	})

	// Test 5: Query events
	t.Run("QueryEvents", func(t *testing.T) {
		filter := &core.EventFilter{
			Limit:  10,
			Offset: 0,
		}

		result, err := db.QueryEvents(filter)
		if err != nil {
			t.Fatalf("Failed to query events: %v", err)
		}

		if result == nil || len(result.Events) == 0 {
			t.Fatal("No events returned")
		}
	})

	// Test 6: Get event by hash
	t.Run("GetEventByHash", func(t *testing.T) {
		event, err := db.GetEventByHash("suite-test-hash-1")
		if err != nil {
			t.Fatalf("Failed to get event: %v", err)
		}

		if event == nil {
			t.Fatal("Event not found")
		}
	})

	// Test 7: Delete event
	t.Run("DeleteEvent", func(t *testing.T) {
		err := db.DeleteEvent(context.Background(), "suite-test-hash-1")
		if err != nil {
			t.Fatalf("Failed to delete event: %v", err)
		}

		// Verify deletion
		event, err := db.GetEventByHash("suite-test-hash-1")
		if err != nil {
			t.Fatalf("Failed to verify deletion: %v", err)
		}

		if event != nil {
			t.Fatal("Event should have been deleted")
		}
	})

	// Test 8: Performance
	t.Run("Performance", func(t *testing.T) {
		events := make([]core.BlockchainEvent, 1000)
		for i := 0; i < 1000; i++ {
			events[i] = core.BlockchainEvent{
				ID:              fmt.Sprintf("suite-perf-hash-%d-%d", time.Now().UnixNano(), i),
				BlockNumber:     uint64(i),
				TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", i)),
				LogIndex:        uint64(i),
				ContractAddress: common.HexToAddress("0x123"),
				EventName:       "Transfer",
				EventData:       []byte("data1"),
			}
		}

		start := time.Now()
		err := db.WriteEvents(context.Background(), events)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to write events: %v", err)
		}

		throughput := float64(len(events)) / duration.Seconds()
		t.Logf("Throughput: %.0f events/second", throughput)

		if throughput < 1000 {
			t.Logf("WARNING: Throughput below target (%.0f < 1000)", throughput)
		} else {
			t.Logf("SUCCESS: Throughput meets target (%.0f >= 1000)", throughput)
		}
	})

	// Test 9: Stats
	t.Run("GetStats", func(t *testing.T) {
		stats := db.GetStats()
		if stats == nil {
			t.Fatal("Stats is nil")
		}

		t.Logf("Stats: %+v", stats)
	})

	// Test 10: Health check
	t.Run("HealthCheck", func(t *testing.T) {
		hc := NewHealthChecker(db)
		hc.Start(1 * time.Second)
		defer hc.Stop()

		// Wait for health check
		time.Sleep(2 * time.Second)

		if !hc.IsHealthy() {
			t.Fatal("Database should be healthy")
		}

		status := hc.GetStatus()
		t.Logf("Health status: %+v", status)
	})
}

// TestDatabaseIntegrationWithErrors tests error handling
func TestDatabaseIntegrationWithErrors(t *testing.T) {
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

	// Test 1: Write nil event
	t.Run("WriteNilEvent", func(t *testing.T) {
		err := db.WriteEvent(context.Background(), nil)
		if err == nil {
			t.Fatal("Expected error for nil event")
		}
	})

	// Test 2: Get event with empty hash
	t.Run("GetEventWithEmptyHash", func(t *testing.T) {
		_, err := db.GetEventByHash("")
		if err == nil {
			t.Fatal("Expected error for empty hash")
		}
	})

	// Test 3: Delete event with empty hash
	t.Run("DeleteEventWithEmptyHash", func(t *testing.T) {
		err := db.DeleteEvent(context.Background(), "")
		if err == nil {
			t.Fatal("Expected error for empty hash")
		}
	})

	// Test 4: Query with nil filter
	t.Run("QueryWithNilFilter", func(t *testing.T) {
		_, err := db.QueryEvents(nil)
		if err == nil {
			t.Fatal("Expected error for nil filter")
		}
	})
}

// TestDatabaseIntegrationConcurrency tests concurrent operations
func TestDatabaseIntegrationConcurrency(t *testing.T) {
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

	// Run concurrent operations
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			event := &core.BlockchainEvent{
				ID:              fmt.Sprintf("concurrent-hash-%d", id),
				BlockNumber:     uint64(id),
				TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", id)),
				LogIndex:        uint64(id),
				ContractAddress: common.HexToAddress("0x123"),
				EventName:       "Transfer",
				EventData:       []byte("data1"),
			}

			err := db.WriteEvent(context.Background(), event)
			if err != nil {
				errors <- err
			}

			retrieved, err := db.GetEventByHash(event.ID)
			if err != nil {
				errors <- err
			}

			if retrieved == nil {
				errors <- fmt.Errorf("event not found for id %d", id)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for errors
	select {
	case err := <-errors:
		t.Fatalf("Concurrent operation failed: %v", err)
	default:
		t.Log("All concurrent operations completed successfully")
	}
}
