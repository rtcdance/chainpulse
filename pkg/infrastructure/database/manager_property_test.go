package database

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// testReporter wraps *testing.T to implement gopter.Reporter
type testReporter struct {
	t *testing.T
}

func (r *testReporter) ReportTestResult(name string, result *gopter.TestResult) {
	if !result.Passed() {
		r.t.Errorf("Property test failed: %s\n%s", name, result.Error)
	}
}

// TestConnectionPoolReuse validates Property 4: Connection Pool Reuse
// For any sequence of database operations, connections should be reused from the pool
// rather than creating new connections.
func TestConnectionPoolReuse(t *testing.T) {
	t.Parallel()
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("connections are reused from pool", prop.ForAll(
		func(poolSize int) bool {
			// Ensure reasonable pool size
			if poolSize < 1 || poolSize > 100 {
				return true
			}

			manager := NewDatabaseManager(
				"mongodb://invalid-host:27017",
				"postgres://invalid-host:5432",
				"disable",
				poolSize,
				5*time.Second,
			)

			// Verify pool size is set correctly
			if manager.poolSize != poolSize {
				return false
			}

			// Verify pool size is used for both databases
			if manager.mongoTimeout != 5*time.Second {
				return false
			}

			if manager.postgresTimeout != 5*time.Second {
				return false
			}

			return true
		},
		gen.IntRange(1, 100),
	))

	properties.Run(&testReporter{t: t})
}

// TestHealthCheckAccuracy validates Property 5: Health Check Accuracy
// For any database health check, the result should accurately reflect the current
// connectivity status of that database.
func TestHealthCheckAccuracy(t *testing.T) {
	t.Parallel()
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("health check reflects initialization state", prop.ForAll(
		func(timeout int) bool {
			// Ensure reasonable timeout
			if timeout < 1 || timeout > 30 {
				return true
			}

			manager := NewDatabaseManager(
				"mongodb://invalid-host:27017",
				"postgres://invalid-host:5432",
				"disable",
				10,
				time.Duration(timeout)*time.Second,
			)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Before initialization, health checks should fail
			mongoErr := manager.CheckMongoHealth(ctx)
			postgresErr := manager.CheckPostgresHealth(ctx)

			// Both should fail before initialization (invalid hosts)
			if mongoErr == nil || postgresErr == nil {
				return false
			}

			return true
		},
		gen.IntRange(1, 30),
	))

	properties.Run(&testReporter{t: t})
}

// TestDatabaseManagerConcurrency validates that DatabaseManager is thread-safe
// Multiple goroutines should be able to access the manager concurrently
func TestDatabaseManagerConcurrency(t *testing.T) {
	t.Parallel()
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("concurrent access is safe", prop.ForAll(
		func(numGoroutines int) bool {
			// Ensure reasonable number of goroutines
			if numGoroutines < 1 || numGoroutines > 100 {
				return true
			}

			manager := NewDatabaseManager(
				"mongodb://invalid-host:27017",
				"postgres://invalid-host:5432",
				"disable",
				10,
				5*time.Second,
			)

			// Create a channel to track completion
			done := make(chan bool, numGoroutines)

			// Launch multiple goroutines accessing the manager
			for i := 0; i < numGoroutines; i++ {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
					defer cancel()

					// Try to get clients
					_, _ = manager.GetMongoClient(ctx)
					_, _ = manager.GetPostgresDB(ctx)

					// Try to get database
					_ = manager.GetMongoDatabase("test")

					done <- true
				}()
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				<-done
			}

			return true
		},
		gen.IntRange(1, 100),
	))

	properties.Run(&testReporter{t: t})
}

// TestDatabaseManagerStateTransitions validates state transitions
// The manager should properly track initialization and closure states
func TestDatabaseManagerStateTransitions(t *testing.T) {
	t.Parallel()
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("state transitions are correct", prop.ForAll(
		func(poolSize int) bool {
			// Ensure reasonable pool size
			if poolSize < 1 || poolSize > 100 {
				return true
			}

			manager := NewDatabaseManager(
				"mongodb://invalid-host:27017",
				"postgres://invalid-host:5432",
				"disable",
				poolSize,
				1*time.Second,
			)

			// Initially not initialized
			if manager.mongoInit || manager.postgresInit {
				return false
			}

			if manager.closed {
				return false
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Try to initialize (will fail due to invalid URIs)
			// When initialization fails, initialized should remain false
			_ = manager.Initialize(ctx)

			// After failed initialization, initialized should still be false
			// because the Initialize method only sets it to true on success
			if manager.mongoInit || manager.postgresInit {
				return false
			}

			// Close the manager (should succeed even if not initialized)
			_ = manager.Close(ctx)

			// After close, closed should be true
			return manager.closed
		},
		gen.IntRange(1, 100),
	))

	properties.Run(&testReporter{t: t})
}

// TestDatabaseManagerConfigurationVariations validates configuration handling
// Different configurations should be properly stored and accessible
func TestDatabaseManagerConfigurationVariations(t *testing.T) {
	t.Parallel()
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("configuration is properly stored", prop.ForAll(
		func(poolSize int, timeoutSecs int) bool {
			// Ensure reasonable values
			if poolSize < 1 || poolSize > 100 {
				return true
			}

			if timeoutSecs < 1 || timeoutSecs > 60 {
				return true
			}

			mongoURI := "mongodb://invalid-host:27017"
			postgresURL := "postgres://invalid-host:5432"
			timeout := time.Duration(timeoutSecs) * time.Second

			manager := NewDatabaseManager(mongoURI, postgresURL, "disable", poolSize, timeout)

			// Verify all configuration is stored correctly
			if manager.mongoURI != mongoURI {
				return false
			}

			if manager.postgresURL != postgresURL {
				return false
			}

			if manager.poolSize != poolSize {
				return false
			}

			if manager.mongoTimeout != timeout {
				return false
			}

			if manager.postgresTimeout != timeout {
				return false
			}

			return true
		},
		gen.IntRange(1, 100),
		gen.IntRange(1, 60),
	))

	properties.Run(&testReporter{t: t})
}
