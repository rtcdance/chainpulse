package database

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestPostgreSQLRealConnection tests connection to real PostgreSQL
func TestPostgreSQLRealConnection(t *testing.T) {
	requirePostgresIntegration(t)

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: "chainpulse",
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)

	// Test initialization
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test start
	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify we can query
	var result int
	err = db.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if result != 1 {
		t.Fatalf("Expected 1, got %d", result)
	}

	t.Log("PostgreSQL connection test passed")
}
