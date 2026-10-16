//go:build integration

package acceptance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestDeployedAPIRejectsMissingCacheAuthority(t *testing.T) {
	t.Parallel()
	if os.Getenv("CHAINPULSE_RUN_DEPLOYED_BLOCKS") != "1" {
		t.Skip("set CHAINPULSE_RUN_DEPLOYED_BLOCKS=1 with isolated native targets")
	}
	base := os.Getenv("CHAINPULSE_DEPLOYED_MICRO_DATABASE_URL")
	require.NotEmpty(t, base)
	db, err := sql.Open("postgres", base)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	name := fmt.Sprintf("chainpulse_cache_startup_%d", time.Now().UnixNano())
	_, err = db.Exec("CREATE DATABASE " + pq.QuoteIdentifier(name))
	require.NoError(t, err)
	defer func() { _, err := db.Exec("DROP DATABASE " + pq.QuoteIdentifier(name)); require.NoError(t, err) }()
	dsn, err := url.Parse(base)
	require.NoError(t, err)
	dsn.Path = "/" + name
	root, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)
	migrator, err := migrate.New("file://"+root, dsn.String())
	require.NoError(t, err)
	defer func() { a, b := migrator.Close(); require.NoError(t, a); require.NoError(t, b) }()
	require.NoError(t, migrator.Migrate(19))
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	binary := os.Getenv("CHAINPULSE_DEPLOYED_API_SERVICE_BINARY")
	require.NotEmpty(t, binary)
	command := exec.CommandContext(ctx, binary)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	command.Env = append(os.Environ(), "DATABASE_TYPE=postgres", "DATABASE_URL="+dsn.String(), "DATABASE_SSLMODE=disable",
		"MONGODB_URI="+os.Getenv("CHAINPULSE_DEPLOYED_MONGODB_URI"), "CHAINS=31337", "CHAIN_ID=31337", "CACHE_TYPE=redis",
		"CACHE_CONNECTION_URL="+os.Getenv("CHAINPULSE_DEPLOYED_REDIS_ADDR"), "API_SERVICE_PORT="+strconv.Itoa(port), "API_SERVICE_RATE_LIMIT_ENABLED=false")
	output, exitErr := command.CombinedOutput()
	artifacts := os.Getenv("CHAINPULSE_ACCEPTANCE_OUTPUT")
	require.NotEmpty(t, artifacts)
	require.NoError(t, os.MkdirAll(artifacts, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, "api-cache-schema19-"+name+".log"), output, 0o644))
	require.NoError(t, ctx.Err(), "a killed or timed-out process is not startup rejection")
	require.Error(t, exitErr)
	var processExit *exec.ExitError
	require.ErrorAs(t, exitErr, &processExit)
	require.Contains(t, string(output), "schema20")
	require.NotContains(t, string(output), "Status: Running")
	proof, err := json.MarshalIndent(map[string]any{"actualSchema": 19, "requiredSchema": 20, "startupRejected": true, "exitCode": processExit.ExitCode()}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, "api-cache-schema19-"+name+".json"), proof, 0o644))
}
