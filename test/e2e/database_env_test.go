package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// databaseProvisioned reports whether a real test database was provisioned
// via TEST_DATABASE_URL (decision D16(a)).
func databaseProvisioned() bool {
	return strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")) != ""
}

// fatalIfDatabaseBroken guards every database-dependent skip path: when
// TEST_DATABASE_URL is set but the database cannot be reached the test fails
// instead of skipping, so a broken provisioned database cannot fake-green CI.
// When the variable is unset the call is a no-op and the caller keeps its
// historical skip behavior.
func fatalIfDatabaseBroken(t *testing.T, ctx context.Context) {
	t.Helper()

	if !databaseProvisioned() {
		return
	}

	dm := NewDatabaseManager(testDatabaseURL(placeholderDatabaseURL))
	defer func() { _ = dm.Close() }()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := dm.Initialize(pingCtx); err != nil {
		t.Fatalf("TEST_DATABASE_URL is set but the database is unreachable: %v", err)
	}
}
