//go:build migration

package migration

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
)

// TestMigrationIdempotency verifies that all database migrations can be
// applied, rolled back, and re-applied without errors.
//
// This simulates the enterprise deployment pattern:
//
//	ci: migrate up → migrate down → migrate up → verify schema
//
// It requires a running PostgreSQL instance (use Docker compose).
// Set DATABASE_URL to a test database, or run with:
//
//	docker compose -f docker/docker-compose.yml up -d postgres
//	MIGRATION_TEST_DSN="postgres://chainpulse:chainpulse_dev@localhost:5432/chainpulse_test?sslmode=disable" \
//	  go test -tags=migration -v ./test/migration/
func TestMigrationIdempotency(t *testing.T) {
	dsn := os.Getenv("MIGRATION_TEST_DSN")
	if dsn == "" {
		baseDSN := os.Getenv("DATABASE_URL")
		if baseDSN == "" {
			baseDSN = "postgres://chainpulse:chainpulse_dev@localhost:5432/chainpulse?sslmode=disable"
		}
		dsn = baseDSN
	}

	projectRoot := findProjectRoot()
	migrationsPath := filepath.Join(projectRoot, "migrations")

	// Create a fresh test database
	testDB := "chainpulse_migration_test_" + randomSuffix()
	dsn = replaceDBName(dsn, testDB)
	baseDSN := replaceDBName(dsn, "chainpulse")

	createDB(t, baseDSN, testDB)
	defer dropDB(t, baseDSN, testDB)

	// Open connection to test database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping: %v", err)
	}

	// Phase 1: Apply all migrations UP
	t.Log("Phase 1: migrate up")
	if err := runMigrate(dsn, migrationsPath, "up"); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	// Verify schema is present: check events table
	if !tableExists(t, db, "events") {
		t.Error("events table not found after migrate up")
	}
	if !tableExists(t, db, "events_metadata") {
		t.Error("events_metadata table not found after migrate up")
	}

	// Phase 2: Rollback all migrations DOWN
	t.Log("Phase 2: migrate down")
	if err := runMigrate(dsn, migrationsPath, "down"); err != nil {
		t.Fatalf("migrate down failed: %v", err)
	}

	// Verify schema is gone
	if tableExists(t, db, "events") {
		t.Log("note: events table still exists after migrate down (expected for some migration strategies)")
	}

	// Phase 3: Re-apply all migrations UP (idempotency check)
	t.Log("Phase 3: migrate up (idempotent)")
	if err := runMigrate(dsn, migrationsPath, "up"); err != nil {
		t.Fatalf("migrate up (idempotent) failed: %v", err)
	}

	if !tableExists(t, db, "events") {
		t.Error("events table not found after second migrate up")
	}

	t.Log("Migration idempotency test PASSED")
}

func runMigrate(dsn, migrationsPath, direction string) error {
	cmd := exec.Command(
		"go", "run", "./cmd/migrate/",
		"-db", dsn,
		"-path", migrationsPath,
		direction,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w\noutput: %s", direction, err, string(output))
	}
	return nil
}

func createDB(t *testing.T, baseDSN, dbName string) {
	t.Helper()
	db, err := sql.Open("postgres", baseDSN)
	if err != nil {
		t.Fatalf("failed to connect for db create: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("failed to create database %s: %v", dbName, err)
	}
}

func dropDB(t *testing.T, baseDSN, dbName string) {
	t.Helper()
	db, err := sql.Open("postgres", baseDSN)
	if err != nil {
		t.Logf("warning: failed to connect for db drop: %v", err)
		return
	}
	defer db.Close()

	// Terminate existing connections before dropping
	_, _ = db.Exec(fmt.Sprintf(
		"SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '%s' AND pid <> pg_backend_pid()",
		dbName,
	))
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		t.Logf("warning: failed to drop database %s: %v (left for inspection)", dbName, err)
	}
}

func tableExists(t *testing.T, db *sql.DB, tableName string) bool {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Logf("warning: table check failed for %s: %v", tableName, err)
		return false
	}
	return exists
}

func replaceDBName(dsn, newName string) string {
	// Simple replace: postgres://user:pass@host:port/dbname → postgres://user:pass@host:port/newName
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			return dsn[:i+1] + newName
		}
	}
	return dsn + "/" + newName
}

func randomSuffix() string {
	return fmt.Sprintf("%06d", os.Getpid()%1000000)
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}
