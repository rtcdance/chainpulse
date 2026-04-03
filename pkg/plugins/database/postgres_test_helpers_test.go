package database

import (
	"os"
	"testing"
)

const postgresIntegrationEnv = "CHAINPULSE_RUN_POSTGRES_INTEGRATION"

func requirePostgresIntegration(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}
	if os.Getenv(postgresIntegrationEnv) != "1" {
		t.Skip("Skipping PostgreSQL integration test; set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 to enable")
	}
}
