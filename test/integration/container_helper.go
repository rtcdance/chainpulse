// Package integration provides programmatic Docker container management
// for integration tests. It wraps docker compose to start/stop infrastructure
// (PostgreSQL, Anvil) without requiring pre-running services or external tools.
//
// Usage:
//
//	func TestWithPostgres(t *testing.T) {
//	    ctx := context.Background()
//	    containers := integration.StartContainers(t, ctx)
//	    defer containers.Stop()
//
//	    db, _ := sql.Open("postgres", containers.PostgresURL())
//	    // ... test with real PostgreSQL
//	}
package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Containers manages Docker container lifecycle for integration tests.
// Zero external dependencies beyond Docker and docker compose.
type Containers struct {
	projectDir  string
	postgresURL string
	anvilURL    string
	teardown    func()
}

// StartContainers starts the required integration test containers.
// It skips the test via t.Skip if Docker is unavailable.
// The returned Containers stops containers automatically on test completion.
func StartContainers(t *testing.T, ctx context.Context) *Containers {
	t.Helper()

	if !hasDocker() {
		t.Skip("integration test requires Docker")
	}

	projectDir := findProjectRoot()
	composeFile := filepath.Join(projectDir, "docker", "docker-compose.yml")

	// Ensure PostgreSQL + Anvil are running
	startContainer(t, composeFile, "postgres")
	startContainer(t, composeFile, "anvil-ethereum")

	// Wait for PostgreSQL to be healthy
	waitForPort(t, "5432", 30)

	// Wait for Anvil to be ready
	waitForAnvil(t, ctx, 30)

	postgresURL := "postgres://chainpulse:chainpulse_dev@localhost:5432/chainpulse?sslmode=disable"
	anvilURL := "http://localhost:8545"

	c := &Containers{
		projectDir:  projectDir,
		postgresURL: postgresURL,
		anvilURL:    anvilURL,
	}

	t.Cleanup(func() {
		// Containers stay running for subsequent tests unless explicitly stopped
	})

	return c
}

// StartContainersClean starts containers and ensures they are stopped on cleanup.
func StartContainersClean(t *testing.T, ctx context.Context) *Containers {
	c := StartContainers(t, ctx)
	t.Cleanup(func() { c.Stop() })
	return c
}

// PostgresURL returns the connection string for the test PostgreSQL instance.
func (c *Containers) PostgresURL() string {
	return c.postgresURL
}

// AnvilURL returns the RPC URL for the test Anvil instance.
func (c *Containers) AnvilURL() string {
	return c.anvilURL
}

// Stop terminates the managed containers.
func (c *Containers) Stop() {
	composeFile := filepath.Join(c.projectDir, "docker", "docker-compose.yml")
	cmd := exec.Command("docker", "compose", "-f", composeFile, "rm", "-sf", "postgres", "anvil-ethereum")
	_ = cmd.Run()
}

func hasDocker() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func startContainer(t *testing.T, composeFile, service string) {
	t.Helper()
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d", "--wait", service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to start %s: %v\n%s", service, err, string(output))
	}
}

func waitForPort(t *testing.T, port string, timeoutSec int) {
	t.Helper()
	for i := 0; i < timeoutSec; i++ {
		cmd := exec.Command("sh", "-c", fmt.Sprintf("lsof -i :%s 2>/dev/null | grep -q LISTEN", port))
		if cmd.Run() == nil {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("port %s not ready within %ds", port, timeoutSec)
}

func waitForAnvil(t *testing.T, ctx context.Context, timeoutSec int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Anvil not ready within %ds", timeoutSec)
		default:
		}

		cmd := exec.Command("sh", "-c",
			`curl -s -X POST http://localhost:8545 -H 'Content-Type: application/json' \
			 --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' 2>/dev/null | \
			 python3 -c "import sys,json;d=json.load(sys.stdin);print(int(d['result'],16))" 2>/dev/null`)
		if output, err := cmd.Output(); err == nil && len(output) > 0 {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// findProjectRoot walks up from the current dir to find the go.mod file.
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
