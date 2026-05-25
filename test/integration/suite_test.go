// Package integration contains integration tests that verify component
// interactions across layers. Most tests use in-memory implementations
// and run without external infrastructure.
//
// Tests that require PostgreSQL, Kafka, or Redis check environment variables
// and skip if unavailable. To run the full suite:
//
//	# Start infrastructure (requires Docker)
//	docker compose -f docker/docker-compose.dev.yml up -d
//
//	# Run all integration tests
//	go test ./test/integration/ -v
//
// When testcontainers-go is approved (see docs/project/DEPENDENCY_APPROVAL.md),
// this file will be updated to programmatically manage containers.
package integration

import (
	"fmt"
	"os"
	"testing"
)

// TestMain is the integration test suite entry point.
// It performs environment validation before any test runs.
func TestMain(m *testing.M) {
	if !hasDocker() && hasInfraEnv() {
		fmt.Fprintln(os.Stderr, "WARNING: infra env vars set but Docker unavailable — tests that need external services may fail")
	}
	os.Exit(m.Run())
}

// hasInfraEnv returns true when at least one known external service URL
// is explicitly configured via environment — indicating the test environment
// has its own infrastructure (CI pipeline, dedicated test cluster, etc.).
func hasInfraEnv() bool {
	return os.Getenv("DATABASE_URL") != "" ||
		os.Getenv("KAFKA_BROKERS") != "" ||
		os.Getenv("REDIS_URL") != ""
}

// requireInfra skips the test when the required infrastructure is not available.
// Tests that call this MUST be explicitly requested or run in CI.
//
// Usage:
//
//	func TestWithPostgres(t *testing.T) {
//	    requireInfra(t)
//	    // ... test with DATABASE_URL ...
//	}
func requireInfra(t *testing.T) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("KAFKA_BROKERS") == "" && os.Getenv("REDIS_URL") == "" {
		t.Skip("integration test skipped: set DATABASE_URL, KAFKA_BROKERS, or REDIS_URL to run")
	}
}

// requirePostgres skips the test when PostgreSQL is unavailable.
func requirePostgres(t *testing.T) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("set DATABASE_URL (e.g. postgres://localhost:5432/chainpulse_test) to run this test")
	}
}

// requireKafka skips the test when Kafka is unavailable.
func requireKafka(t *testing.T) {
	t.Helper()
	if os.Getenv("KAFKA_BROKERS") == "" {
		t.Skip("set KAFKA_BROKERS (e.g. localhost:9092) to run this test")
	}
}

// TODO: When testcontainers-go is approved (DEPENDENCY_APPROVAL.md), replace
// the manual infra checks above with:
//
//	import (
//	    "github.com/testcontainers/testcontainers-go"
//	    "github.com/testcontainers/testcontainers-go/modules/postgres"
//	    "github.com/testcontainers/testcontainers-go/modules/redis"
//	)
//
//	func setupPostgresContainer(t *testing.T) (connStr string, cleanup func()) {
//	    ctx := context.Background()
//	    ctr, err := postgres.Run(ctx, "postgres:15-alpine",
//	        postgres.WithDatabase("chainpulse_test"),
//	    )
//	    require.NoError(t, err)
//	    connStr, _ = ctr.ConnectionString(ctx)
//	    return connStr, func() { ctr.Terminate(ctx) }
//	}
