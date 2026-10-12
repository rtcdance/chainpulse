//go:build integration

package acceptance

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// The shared helpers in sigterm_test.go are scoped to that scenario's chain, so
// this rehearsal carries its own chain-scoped variants.

func txTestEventForChain(chainID string, block uint64) *blockchain.BlockchainEvent {
	event := &blockchain.BlockchainEvent{
		ChainID:         chainID,
		BlockNumber:     block,
		BlockHash:       common.HexToHash(fmt.Sprintf("0x%064x", block+1)),
		TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", block*10+1)),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000ee"),
		EventName:       "Transfer",
		BlockTimestamp:  1700000500,
	}
	event.EventHash = core.ComputeEventHash(event)
	event.ID = event.EventHash
	return event
}

func readCheckpointFor(t *testing.T, dsn, chainID string) (uint64, bool) {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var block uint64
	err = db.QueryRow(
		"SELECT last_indexed_block FROM indexing_state WHERE chain_id = $1", chainID,
	).Scan(&block)
	if err == sql.ErrNoRows {
		return 0, false
	}
	require.NoError(t, err)
	return block, true
}

func countEventsFor(t *testing.T, dsn, chainID string) int64 {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var n int64
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM blockchain_events WHERE chain_id = $1", chainID,
	).Scan(&n))
	return n
}

// forwarderName is the socat container that publishes the database on a port
// this sandbox can reach. Stopping it is how the rehearsal makes the database
// unreachable without touching the database itself.
func forwarderName() string {
	if name := os.Getenv("CHAINPULSE_PG_FORWARDER"); name != "" {
		return name
	}
	return "cp-pg-fwd"
}

// dockerLifecycle runs `docker <action> <name>`, skipping the test when docker
// is unavailable rather than reporting a false pass.
func dockerLifecycle(t *testing.T, action, name string) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available; the fault rehearsal needs it to toggle the forwarder")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", action, name).CombinedOutput()
	require.NoError(t, err, "docker %s %s: %s", action, name, out)
}

func waitForDatabase(t *testing.T, dsn string, within time.Duration) {
	t.Helper()

	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		a := database.NewPostgreSQLPluginAdapter(
			testhelpers.NewTestLogger(),
			testhelpers.NewTestMetricsCollector(),
		)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := a.Initialize(ctx, core.Config{DatabaseURL: dsn})
		cancel()
		if err == nil {
			_ = a.Stop(context.Background())
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("database did not become reachable within %s", within)
}

// TestDatabaseDisconnectIsBackpressure is the §9 E09 rehearsal (DB fault).
//
// It asserts the properties the plan names for that scenario:
//
//   - the failure is BACKPRESSURE, not data loss: a batch that cannot reach the
//     database fails instead of half-committing;
//   - the checkpoint stays at the last block whose effects are durable, so the
//     checkpoint and the durable effects agree;
//   - once the database is reachable again, the workload RESUMES from that
//     checkpoint — nothing is skipped and nothing is applied twice.
func TestDatabaseDisconnectIsBackpressure(t *testing.T) {
	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	resetState(t, dsn)

	adapter := database.NewPostgreSQLPluginAdapter(
		testhelpers.NewTestLogger(),
		testhelpers.NewTestMetricsCollector(),
	)
	require.NoError(t, adapter.Initialize(ctx, core.Config{DatabaseURL: dsn}))
	require.NoError(t, adapter.Start(ctx))
	t.Cleanup(func() { _ = adapter.Stop(context.Background()) })

	const chainID = "e09-db-fault"

	committer, err := processor.NewBatchCommitter(adapter, nil, chainID, "worker-A", time.Minute)
	require.NoError(t, err)

	commit := func(block uint64) error {
		event := txTestEventForChain(chainID, block)
		return committer.CommitBlock(ctx, block, event.BlockHash.Hex(), []*blockchain.BlockchainEvent{event})
	}

	// 1. Healthy: three blocks commit and the checkpoint follows them.
	for block := uint64(0); block < 3; block++ {
		require.NoError(t, commit(block), "block %d must commit while the database is healthy", block)
	}
	checkpoint, ok := readCheckpointFor(t, dsn, chainID)
	require.True(t, ok)
	require.Equal(t, uint64(2), checkpoint)
	require.Equal(t, int64(3), countEventsFor(t, dsn, chainID))

	// 2. Cut the database off.
	dockerLifecycle(t, "stop", forwarderName())
	t.Cleanup(func() {
		// Always restore the environment, even if an assertion below fails.
		_ = exec.Command("docker", "start", forwarderName()).Run()
	})

	// 3. The next batch must fail rather than half-commit.
	err = commit(3)
	require.Error(t, err, "a batch must fail while the database is unreachable")

	// 4. Restore the database. The assertions about what the failed batch left
	//    behind have to wait until it is reachable again — querying during the
	//    outage would only prove that the outage was real.
	dockerLifecycle(t, "start", forwarderName())
	waitForDatabase(t, dsn, 60*time.Second)

	// 5. Checkpoint and durable effects agree: block 3 left nothing behind.
	afterFault, ok := readCheckpointFor(t, dsn, chainID)
	require.True(t, ok)
	assert.Equal(t, uint64(2), afterFault,
		"the checkpoint must not advance past the last durably committed block")
	assert.Equal(t, int64(3), countEventsFor(t, dsn, chainID),
		"a failed batch must not leave partial rows behind")

	// 6. Resume from the checkpoint.

	// A fresh adapter, as a restarted process would have.
	resumed := database.NewPostgreSQLPluginAdapter(
		testhelpers.NewTestLogger(),
		testhelpers.NewTestMetricsCollector(),
	)
	require.NoError(t, resumed.Initialize(ctx, core.Config{DatabaseURL: dsn}))
	require.NoError(t, resumed.Start(ctx))
	t.Cleanup(func() { _ = resumed.Stop(context.Background()) })

	resumedCommitter, err := processor.NewBatchCommitter(resumed, nil, chainID, "worker-A", time.Minute)
	require.NoError(t, err)

	event := txTestEventForChain(chainID, 3)
	require.NoError(t,
		resumedCommitter.CommitBlock(ctx, 3, event.BlockHash.Hex(), []*blockchain.BlockchainEvent{event}),
		"the workload must resume from the checkpoint once the database is back")

	finalCheckpoint, ok := readCheckpointFor(t, dsn, chainID)
	require.True(t, ok)
	assert.Equal(t, uint64(3), finalCheckpoint, "no block may be skipped after recovery")
	assert.Equal(t, int64(4), countEventsFor(t, dsn, chainID), "and none may be applied twice")
}
