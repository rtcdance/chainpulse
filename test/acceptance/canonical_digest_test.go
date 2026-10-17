//go:build integration

package acceptance

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq" // postgres driver
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"

	"github.com/rtcdance/chainpulse/test/acceptance/tools/digest"
)

// The scenario covers invariants I01 (chain-scoped state), I02 (physical event
// identity) and I03 (one canonical block per chain and height) at the level the
// plan compares them: a canonical digest produced by the T02 oracle.
const (
	digestChainA = "digest-a"
	digestChainB = "digest-b"
	digestHeight = uint64(3)
)

// digestStep is one recorded state of the world.
type digestStep struct {
	Name string `json:"name"`
	Note string `json:"note"`
	// ChainA and ChainB are the canonical digests of each chain at this step.
	ChainA string `json:"chain_a"`
	ChainB string `json:"chain_b"`
	// EventsA/EventsB are the row counts the digests were computed over.
	EventsA int `json:"events_a"`
	EventsB int `json:"events_b"`
}

// canonicalDigestReport is the §10 artifact. It is written by the test that
// measured it and is written on failure too: an absent artifact is
// indistinguishable from "never ran", whereas a failing step is a record.
type canonicalDigestReport struct {
	Schema      string            `json:"schema"`
	GeneratedAt string            `json:"generated_at"`
	CommitSHA   string            `json:"commit_sha"`
	TreeDirty   bool              `json:"tree_dirty"`
	Database    string            `json:"database"`
	Oracle      string            `json:"oracle"`
	Scenario    string            `json:"scenario"`
	Passed      bool              `json:"passed"`
	Steps       []digestStep      `json:"steps"`
	Invariants  map[string]string `json:"invariants"`
}

// chainDigest composes the canonical digest of one chain exactly as the
// contracts suite does (postgres_chain_isolation_test.go): events carry their
// name and status so a rollback is visible, blocks carry (chain, height, hash),
// and the checkpoint is folded in as one more block record. Keeping the
// composition identical is what makes these digests comparable with the
// contract assertions rather than merely self-consistent.
func chainDigest(t *testing.T, adapter *database.PostgreSQLPluginAdapter, chainID string) (string, int) {
	t.Helper()
	ctx := context.Background()

	events, err := adapter.GetEventsByBlockRangeForChain(ctx, chainID, 0, 1<<40)
	require.NoError(t, err)

	digestEvents := make([]digest.Event, 0, len(events))
	for _, e := range events {
		digestEvents = append(digestEvents, digest.Event{
			Chain:     e.ChainID,
			BlockHash: e.BlockHash.Hex(),
			TxHash:    e.TransactionHash.Hex(),
			LogIndex:  uint32(e.LogIndex),
			Data:      e.EventName + ":" + string(e.Status),
		})
	}

	blocks, err := adapter.GetAllBlocksForChain(ctx, chainID)
	require.NoError(t, err)

	digestBlocks := make([]digest.Block, 0, len(blocks)+1)
	for _, b := range blocks {
		digestBlocks = append(digestBlocks, digest.Block{Chain: chainID, Height: b.Number, BlockHash: b.Hash.Hex()})
	}

	checkpointBlock, checkpointHash, err := adapter.GetCheckpointForChain(ctx, chainID)
	require.NoError(t, err)
	digestBlocks = append(digestBlocks, digest.Block{Chain: chainID, Height: checkpointBlock, BlockHash: checkpointHash})

	return digest.MustCanonicalEvents(digestEvents) + "|" + digest.MustCanonicalBlocks(digestBlocks), len(events)
}

func digestEvent(chainID string, height uint64, blockHash common.Hash, tag string) *blockchain.BlockchainEvent {
	event := &blockchain.BlockchainEvent{
		ChainID:         chainID,
		BlockNumber:     height,
		BlockHash:       blockHash,
		TransactionHash: common.BytesToHash([]byte(chainID + ":" + tag)),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000dd"),
		EventName:       "Transfer",
		BlockTimestamp:  1700000000,
	}
	event.EventHash = core.ComputeEventHash(event)
	event.ID = event.EventHash
	return event
}

func writeChainBlock(t *testing.T, adapter *database.PostgreSQLPluginAdapter, chainID string, height uint64, hash common.Hash, tag string) {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, adapter.WriteBlock(ctx, chainID, &blockchain.Block{Number: height, Hash: hash}))
	require.NoError(t, adapter.WriteEvent(ctx, digestEvent(chainID, height, hash, tag)))

	tx, err := adapter.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.SaveCheckpoint(ctx, chainID, height, hash.Hex()))
	require.NoError(t, tx.Commit(ctx))
}

// TestCanonicalDigestsAcrossRollbackAndRedelivery is the digest-level acceptance
// for I01/I02/I03 and the producer of the §10 `canonical-digests.json` artifact.
//
// The steps are chosen so the assertions cannot pass vacuously:
//
//	1 initial          two chains, same height, distinct content
//	2 rollback A       A must CHANGE; B must be byte-identical (I01)
//	3 replacement A    A must CHANGE again (a new branch is a new identity, I02);
//	                   B still identical
//	4 duplicate write  A must be IDENTICAL to step 3 — the natural key folds the
//	                   re-delivery, which is the at-least-once story from §4.2
func TestCanonicalDigestsAcrossRollbackAndRedelivery(t *testing.T) {
	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	projectRoot := acceptanceProjectRoot(t)
	report := &canonicalDigestReport{
		Schema:      "chainpulse.canonical-digests.v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		CommitSHA:   gitRevision(projectRoot),
		TreeDirty:   gitDirty(projectRoot),
		Database:    redactConnectionString(dsn),
		Oracle:      "test/acceptance/tools/digest (T02 canonical oracle; order-independent, dedup by I02 identity)",
		Scenario:    "two chains at the same height, rollback + replacement on A, then duplicate re-delivery",
		Invariants: map[string]string{
			"I01": "a rollback on chain A leaves chain B's digest byte-identical",
			"I02": "a replacement branch is a new physical identity; a duplicate delivery folds into one effect",
			"I03": "one canonical block per (chain, height)",
		},
	}
	defer writeCanonicalDigestReport(t, projectRoot, report)

	truncateAll(t, dsn)
	adapter := newAdapterForDigest(t, dsn)

	ctx := context.Background()
	record := func(name, note string) digestStep {
		digestA, countA := chainDigest(t, adapter, digestChainA)
		digestB, countB := chainDigest(t, adapter, digestChainB)
		step := digestStep{Name: name, Note: note, ChainA: digestA, ChainB: digestB, EventsA: countA, EventsB: countB}
		report.Steps = append(report.Steps, step)
		return step
	}
	fail := func(format string, args ...any) {
		report.Passed = false
		t.Fatalf(format, args...)
	}

	// --- step 1: two chains, same height ---
	for height := uint64(1); height <= digestHeight; height++ {
		writeChainBlock(t, adapter, digestChainA, height, common.BigToHash(big.NewInt(int64(1000+height))), "initial")
		writeChainBlock(t, adapter, digestChainB, height, common.BigToHash(big.NewInt(int64(2000+height))), "initial")
	}
	initial := record("initial", "two chains indexed to the same height with distinct content")
	if initial.EventsA != int(digestHeight) || initial.EventsB != int(digestHeight) {
		fail("expected %d events per chain, got A=%d B=%d", digestHeight, initial.EventsA, initial.EventsB)
	}

	// --- step 2: roll back block 3 on chain A only ---
	affected, err := adapter.MarkEventsAsReorgedForChain(ctx, digestChainA, digestHeight, digestHeight)
	if err != nil {
		fail("MarkEventsAsReorgedForChain: %v", err)
	}
	if affected != 1 {
		fail("rollback must affect exactly chain A's row, affected %d", affected)
	}
	rolledBack := record("after_rollback_chain_a", "block 3 marked reorged on A only")

	if rolledBack.ChainA == initial.ChainA {
		fail("chain A's digest did not change after a rollback — the assertion below would be vacuous")
	}
	if rolledBack.ChainB != initial.ChainB {
		fail("I01 violated: chain B's digest changed after a rollback on chain A")
	}

	// --- step 3: re-index block 3 on A with a new branch ---
	replacementHash := common.BigToHash(big.NewInt(9999))
	writeChainBlock(t, adapter, digestChainA, digestHeight, replacementHash, "replacement")
	replaced := record("after_replacement_chain_a", "block 3 re-indexed on a new branch (new block hash)")

	if replaced.ChainA == rolledBack.ChainA {
		fail("a replacement branch must produce a different canonical digest (I02)")
	}
	if replaced.ChainB != initial.ChainB {
		fail("I01 violated: chain B's digest changed while repairing chain A")
	}

	// --- step 4: deliver the identical replacement again ---
	writeChainBlock(t, adapter, digestChainA, digestHeight, replacementHash, "replacement")
	redelivered := record("after_duplicate_delivery", "the identical replacement delivered a second time")

	if redelivered.ChainA != replaced.ChainA {
		fail("I02 violated: a duplicate delivery changed the canonical digest (%s -> %s)",
			replaced.ChainA, redelivered.ChainA)
	}
	if redelivered.EventsA != replaced.EventsA {
		fail("a duplicate delivery must not add rows: %d -> %d", replaced.EventsA, redelivered.EventsA)
	}
	if redelivered.ChainB != initial.ChainB {
		fail("I01 violated: chain B's digest changed across the whole scenario")
	}

	report.Passed = true
	t.Logf("digest steps: %d; chain B digest stable across all steps: %v",
		len(report.Steps), redelivered.ChainB == initial.ChainB)
}

func newAdapterForDigest(t *testing.T, dsn string) *database.PostgreSQLPluginAdapter {
	t.Helper()

	adapter := database.NewPostgreSQLPluginAdapter(
		testhelpers.NewTestLogger(),
		testhelpers.NewTestMetricsCollector(),
	)
	require.NoError(t, adapter.Initialize(context.Background(), core.Config{DatabaseURL: dsn}))
	require.NoError(t, adapter.Start(context.Background()))
	t.Cleanup(func() { _ = adapter.Stop(context.Background()) })
	return adapter
}

func truncateAll(t *testing.T, dsn string) {
	t.Helper()
	db := openDB(t, dsn)
	for _, table := range []string{"blockchain_events", "blocks", "indexing_state", "chain_leases"} {
		_, err := db.Exec("TRUNCATE " + table)
		require.NoError(t, err)
	}
}

func acceptanceProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	return root
}

func gitRevision(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitDirty(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// redactConnectionString strips the password before the DSN is archived (§10
// forbids archiving credentials).
func redactConnectionString(dsn string) string {
	at := strings.LastIndex(dsn, "@")
	scheme := strings.Index(dsn, "://")
	if at < 0 || scheme < 0 || at < scheme {
		return "<unparseable DSN>"
	}
	credentials := dsn[scheme+3 : at]
	if colon := strings.Index(credentials, ":"); colon >= 0 {
		credentials = credentials[:colon] + ":REDACTED"
	}
	return dsn[:scheme+3] + credentials + dsn[at:]
}

func writeCanonicalDigestReport(t *testing.T, projectRoot string, report *canonicalDigestReport) {
	t.Helper()

	dir := "uncommitted"
	if len(report.CommitSHA) >= 12 {
		dir = report.CommitSHA[:12]
	}
	path := filepath.Join(projectRoot, "build", "acceptance", dir, "canonical-digests.json")

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Logf("warning: could not create %s: %v", filepath.Dir(path), err)
		return
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Logf("warning: could not encode the digest report: %v", err)
		return
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Logf("warning: could not write %s: %v", path, err)
		return
	}
	t.Logf("canonical digest report written to %s", path)
}
