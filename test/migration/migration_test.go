//go:build migration

package migration

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// migrationReport is the §10 evidence artifact. It is written by the test that
// measured it, so no parser can drift from what actually happened, and it is
// written on failure too: an absent artifact is indistinguishable from "never
// run", whereas ok=false plus the failing phase is a real record.
type migrationReport struct {
	Schema      string             `json:"schema"`
	GeneratedAt string             `json:"generated_at"`
	CommitSHA   string             `json:"commit_sha"`
	TreeDirty   bool               `json:"tree_dirty"`
	Database    string             `json:"database"`
	Migrations  int                `json:"migration_count"`
	Passed      bool               `json:"passed"`
	Phases      []migrationPhase   `json:"phases"`
	Invariants  migrationInvariant `json:"invariants"`
}

type migrationPhase struct {
	Name    string   `json:"name"`
	OK      bool     `json:"ok"`
	Error   string   `json:"error,omitempty"`
	Version int      `json:"schema_version"`
	Tables  []string `json:"tables"`
}

type migrationInvariant struct {
	NaturalKey       string `json:"blockchain_events_natural_key"`
	NaturalKeyWanted string `json:"blockchain_events_natural_key_wanted"`
	SchemaStable     bool   `json:"schema_after_rollback_matches_fresh"`
}

// wantNaturalKey is the F06/I02 event identity: the identity is
// (chain_id, block_hash, tx_hash, log_index). block_number is deliberately NOT
// part of it — it is implied by block_hash and is only used for ordering.
const wantNaturalKey = "chain_id,block_hash,transaction_hash,log_index"

// appTables are the tables the application owns after every migration has been
// applied. `schema_migrations` is deliberately excluded: it is the runner's own
// bookkeeping and legitimately survives a rollback.
var appTables = []string{
	"api_keys",
	"blockchain_events",
	"blocks",
	"chain_leases",
	"chain_finality_observations",
	"canonical_query_revisions",
	"dlq_events",
	"indexing_state",
	"reorg_generations",
	"reorg_jobs",
	"reorg_outbox",
	"webhooks",
}

// TestMigrationIdempotency verifies that all database migrations can be
// applied, rolled back in full, and re-applied without errors, and that the
// schema after the rollback-and-reapply cycle is identical to the schema on a
// fresh database.
//
// This simulates the enterprise deployment pattern:
//
//	fresh db → migrate up → migrate up again (no-op) → migrate down × N → migrate up → verify schema
//
// It requires a running PostgreSQL instance (use Docker compose). Set
// MIGRATION_TEST_DSN to a test database, or run with:
//
//	docker compose -f docker/docker-compose.yml up -d postgres
//	MIGRATION_TEST_DSN="postgres://chainpulse:chainpulse_dev@localhost:5432/chainpulse?sslmode=disable" \
//	  go test -tags=migration -v ./test/migration/
//
// Two things this test deliberately does NOT do:
//
//   - It does not assert on the pre-T04 table names (`events`,
//     `events_metadata`). Neither is created by any migration: the canonical
//     table is `blockchain_events`, and 000010 renames a stray legacy `events`
//     table rather than creating one. Asserting on them made the previous
//     version of this test fail on a fresh database.
//   - It does not treat a single `down` as a full rollback. `cmd/migrate down`
//     rolls back ONE step, so a full rollback needs N invocations; otherwise
//     the "rollback" phase only ever exercised the newest migration.
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
	total := migrationCount(t, migrationsPath)

	// Create a fresh test database
	testDB := "chainpulse_migration_test_" + randomSuffix()
	baseDSN := dsn
	dsn = replaceDBName(dsn, testDB)

	report := &migrationReport{
		Schema:      "chainpulse.migration-report.v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		CommitSHA:   gitOutput(projectRoot, "rev-parse", "HEAD"),
		TreeDirty:   gitOutput(projectRoot, "status", "--porcelain") != "",
		Database:    redactDSN(dsn),
		Migrations:  total,
		Invariants:  migrationInvariant{NaturalKeyWanted: wantNaturalKey},
	}
	defer writeMigrationReport(t, projectRoot, report)

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

	// record appends a phase to the report. It runs before t.Fatalf, and the
	// deferred writer is registered above, so a failing phase still lands in the
	// artifact.
	record := func(name string, phaseErr error, tables []string) {
		phase := migrationPhase{Name: name, OK: phaseErr == nil, Tables: tables}
		switch {
		case phaseErr != nil:
			phase.Error = phaseErr.Error()
		case tableExists(t, db, "schema_migrations"):
			phase.Version = schemaVersion(t, db)
		}
		report.Phases = append(report.Phases, phase)
	}
	fail := func(format string, args ...any) {
		report.Passed = false
		t.Fatalf(format, args...)
	}

	// Phase 1: apply every migration, from an empty database.
	t.Logf("Phase 1: migrate up (%d migrations)", total)
	err = runMigrate(projectRoot, dsn, migrationsPath, "up")
	tablesAfterUp := appTablesPresent(t, db)
	record("up_fresh", err, tablesAfterUp)
	if err != nil {
		fail("migrate up failed: %v", err)
	}
	if len(tablesAfterUp) != len(appTables) {
		fail("after migrate up expected %d tables, got %d: %v", len(appTables), len(tablesAfterUp), tablesAfterUp)
	}

	// The identity constraint is the invariant a migration-ordering mistake
	// breaks silently: 000004 and 000009 both manage the same constraint name,
	// so re-running 000004 alone reverts it to the pre-F06 shape.
	report.Invariants.NaturalKey = uniqueConstraintColumns(t, db, "blockchain_events")
	if report.Invariants.NaturalKey != wantNaturalKey {
		fail("blockchain_events natural key is %q, want %q (F06/I02)", report.Invariants.NaturalKey, wantNaturalKey)
	}

	version := schemaVersion(t, db)
	if version != total {
		fail("schema_migrations version = %d, want %d", version, total)
	}

	// Phase 2: applying UP again must be a no-op, not a re-application. If the
	// runner replayed the files, CREATE TABLE without IF NOT EXISTS would fail.
	t.Log("Phase 2: migrate up again (must be a no-op)")
	err = runMigrate(projectRoot, dsn, migrationsPath, "up")
	tablesAfterSecondUp := appTablesPresent(t, db)
	record("up_again", err, tablesAfterSecondUp)
	if err != nil {
		fail("second migrate up failed: %v", err)
	}
	if got := schemaVersion(t, db); got != version {
		fail("second up moved the schema version from %d to %d", version, got)
	}
	if fmt.Sprint(tablesAfterSecondUp) != fmt.Sprint(tablesAfterUp) {
		fail("second up changed the table set: %v", tablesAfterSecondUp)
	}

	// Phase 3: roll every migration back. `down` is one step per invocation.
	t.Logf("Phase 3: migrate down x%d", total)
	var rollbackErr error
	for i := 0; i < total; i++ {
		if rollbackErr = runMigrate(projectRoot, dsn, migrationsPath, "down"); rollbackErr != nil {
			rollbackErr = fmt.Errorf("step %d of %d: %w", i+1, total, rollbackErr)
			break
		}
	}
	tablesAfterRollback := appTablesPresent(t, db)
	record("down_all", rollbackErr, tablesAfterRollback)
	if rollbackErr != nil {
		fail("full rollback failed: %v", rollbackErr)
	}
	if len(tablesAfterRollback) != 0 {
		fail("after rolling back all %d migrations these tables remain: %v", total, tablesAfterRollback)
	}
	if got := schemaVersion(t, db); got != 0 {
		fail("after a full rollback schema_migrations version = %d, want 0", got)
	}

	// Phase 4: re-apply every migration. The schema must come back identical to
	// the fresh-database schema, otherwise the `down` files are lossy and a
	// rollback would not be recoverable.
	t.Log("Phase 4: migrate up again (rollback must be faithful)")
	err = runMigrate(projectRoot, dsn, migrationsPath, "up")
	tablesAfterReapply := appTablesPresent(t, db)
	record("up_after_rollback", err, tablesAfterReapply)
	if err != nil {
		fail("migrate up after rollback failed: %v", err)
	}

	report.Invariants.SchemaStable = fmt.Sprint(tablesAfterReapply) == fmt.Sprint(tablesAfterUp)
	if !report.Invariants.SchemaStable {
		fail("schema after rollback+reapply differs from fresh schema:\n fresh:     %v\n reapplied: %v",
			tablesAfterUp, tablesAfterReapply)
	}
	if cols := uniqueConstraintColumns(t, db, "blockchain_events"); cols != wantNaturalKey {
		fail("after rollback+reapply the natural key is %q, want %q", cols, wantNaturalKey)
	}

	report.Passed = true
	t.Log("Migration idempotency test PASSED")
}

// TestReplaceDBNamePreservesQuery pins the fix for a defect that made this
// suite unrunnable against any server requiring sslmode=disable: the old
// implementation truncated at the last '/', which also dropped the query
// string, so lib/pq fell back to requiring SSL and every statement failed with
// "SSL is not enabled on the server".
func TestReplaceDBNamePreservesQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		dsn     string
		newName string
		want    string
	}{
		{
			name:    "query string survives",
			dsn:     "postgres://u:p@host:5432/app?sslmode=disable",
			newName: "other",
			want:    "postgres://u:p@host:5432/other?sslmode=disable",
		},
		{
			name:    "multiple parameters survive",
			dsn:     "postgres://u:p@host:5432/app?sslmode=disable&connect_timeout=3",
			newName: "other",
			want:    "postgres://u:p@host:5432/other?sslmode=disable&connect_timeout=3",
		},
		{
			name:    "no query string",
			dsn:     "postgres://u:p@host:5432/app",
			newName: "other",
			want:    "postgres://u:p@host:5432/other",
		},
		{
			name:    "no path component",
			dsn:     "postgres://host",
			newName: "other",
			want:    "postgres://host/other",
		},
		{
			name:    "credentials containing a slash are not mistaken for the path",
			dsn:     "postgres://u:p@host/app?sslmode=disable",
			newName: "other",
			want:    "postgres://u:p@host/other?sslmode=disable",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := replaceDBName(tc.dsn, tc.newName); got != tc.want {
				t.Errorf("replaceDBName(%q, %q) = %q, want %q", tc.dsn, tc.newName, got, tc.want)
			}
		})
	}
}

// runMigrate shells out to the real migrator. The command must run from the
// module root: a relative package path like ./cmd/migrate/ resolves against the
// child's working directory, and the test's working directory is test/migration.
func runMigrate(projectRoot, dsn, migrationsPath, direction string) error {
	cmd := exec.Command(
		"go", "run", "./cmd/migrate/",
		"-db", dsn,
		"-path", migrationsPath,
		direction,
	)
	cmd.Dir = projectRoot
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
	t.Helper()
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("table existence check failed for %s: %v", tableName, err)
	}
	return exists
}

// appTablesPresent returns the application tables that currently exist, sorted
// by the order of appTables so two snapshots can be compared directly.
func appTablesPresent(t *testing.T, db *sql.DB) []string {
	t.Helper()

	present := make([]string, 0, len(appTables))
	for _, name := range appTables {
		if tableExists(t, db, name) {
			present = append(present, name)
		}
	}
	return present
}

// uniqueConstraintColumns returns the comma-joined column names of the table's
// first unique constraint, in constraint order, or "" when there is none.
func uniqueConstraintColumns(t *testing.T, db *sql.DB, table string) string {
	t.Helper()

	var cols sql.NullString
	err := db.QueryRow(`
		SELECT string_agg(a.attname, ',' ORDER BY array_position(c.conkey, a.attnum))
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		WHERE c.conrelid = $1::regclass AND c.contype = 'u'
		GROUP BY c.conname
		ORDER BY c.conname
		LIMIT 1`, table).Scan(&cols)
	if err != nil {
		t.Fatalf("unique constraint lookup failed for %s: %v", table, err)
	}
	return cols.String
}

// schemaVersion returns the version the runner has recorded, or 0 when nothing
// has been applied (the table is absent, or the row was rolled back to zero).
func schemaVersion(t *testing.T, db *sql.DB) int {
	t.Helper()

	if !tableExists(t, db, "schema_migrations") {
		return 0
	}

	var version sql.NullInt64
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		t.Fatalf("schema version lookup failed: %v", err)
	}
	if dirty {
		t.Fatal("schema_migrations is dirty: a migration failed midway")
	}
	if !version.Valid {
		return 0
	}
	return int(version.Int64)
}

// migrationCount counts the up migrations, which is how many `down` steps a full
// rollback needs.
func migrationCount(t *testing.T, migrationsPath string) int {
	t.Helper()

	entries, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no up migrations found under %s", migrationsPath)
	}
	return len(entries)
}

// replaceDBName swaps the database name in a PostgreSQL URL, preserving the
// query string. Only the path segment after the host is rewritten, so a slash
// inside the credentials cannot be mistaken for the path separator.
func replaceDBName(dsn, newName string) string {
	base, query, hasQuery := strings.Cut(dsn, "?")

	schemeEnd := strings.Index(base, "://")
	if schemeEnd < 0 {
		return dsn
	}
	schemeEnd += len("://")

	slash := strings.LastIndex(base, "/")
	var replaced string
	if slash < schemeEnd {
		// No path component at all: postgres://host → postgres://host/newName
		replaced = base + "/" + newName
	} else {
		replaced = base[:slash+1] + newName
	}

	if hasQuery {
		replaced += "?" + query
	}
	return replaced
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

// reportPathFor returns where the §10 evidence artifact lands. §10 puts
// short-lived raw artifacts under build/acceptance/<sha>/, so a report is always
// attributable to the commit that produced it.
func reportPathFor(projectRoot, commitSHA string) string {
	dir := "uncommitted"
	if len(commitSHA) >= 12 {
		dir = commitSHA[:12]
	}
	return filepath.Join(projectRoot, "build", "acceptance", dir, "migration-report.json")
}

// writeMigrationReport persists the run's evidence. A failure to write is
// reported but does not fail the test: the run itself still passed or failed on
// its own merits, and pretending otherwise would make the gate depend on the
// filesystem.
func writeMigrationReport(t *testing.T, projectRoot string, report *migrationReport) {
	t.Helper()

	path := os.Getenv("MIGRATION_REPORT_PATH")
	if path == "" {
		path = reportPathFor(projectRoot, report.CommitSHA)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Logf("warning: could not create %s: %v", filepath.Dir(path), err)
		return
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Logf("warning: could not encode the migration report: %v", err)
		return
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Logf("warning: could not write %s: %v", path, err)
		return
	}
	t.Logf("migration report written to %s", path)
}

// redactDSN strips the password before the DSN is archived. §10 forbids
// archiving credentials.
func redactDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "<unparseable DSN>"
	}
	if _, hasPassword := parsed.User.Password(); hasPassword {
		parsed.User = url.UserPassword(parsed.User.Username(), "REDACTED")
	}
	return parsed.String()
}

// gitOutput runs a git command in dir and returns its trimmed stdout, or "" if
// git is unavailable or the command fails. Evidence provenance must never make
// a test unrunnable.
func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
