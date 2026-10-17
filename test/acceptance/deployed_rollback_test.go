//go:build integration

package acceptance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// rollbackChainID is the chain the deployed fixture indexes. The Anvil node and
// the emitter are supplied by the environment; this test never deploys its own
// contract so it stays runnable against an already-provisioned isolated stack.
const rollbackChainID = "31337"

// schemaBeforeExpand is the migration version the previous release ships with
// (7636755 carries migrations 1..16); the current tree carries 1..21. Building
// the previous binary from that revision is what makes the 16->21 expand real
// instead of a no-op against two trees that are already at 21.
const schemaBeforeExpand = 16

// rollbackSchemaFloor is the version migration 20 installs; the canonical
// aggregate cache refuses to serve below it, which is what makes both the
// "stale schema" and the "interrupted migration" probes fail closed.
const rollbackSchemaFloor = 20

type rollbackProbeResult struct {
	Name       string `json:"name"`
	ExitCode   int    `json:"exit_code"`
	Message    string `json:"expected_message"`
	Rejected   bool   `json:"startup_rejected"`
	EvidenceAt string `json:"evidence"`
}

type rollbackStep struct {
	Name   string `json:"name"`
	Schema int    `json:"schema_version"`
	Rows   int    `json:"event_rows"`
	Events int    `json:"digest_events"`
	Digest string `json:"digest"`
}

type rollbackOutcome struct {
	Schema       int    `json:"schema_version"`
	HealthStatus int    `json:"health_status"`
	EventsStatus int    `json:"events_status"`
	EventsPath   string `json:"events_path"`
	PriorDigest  string `json:"prior_digest"`
	Digest       string `json:"digest"`
	Identical    bool   `json:"digest_identical"`
}

// rollbackReport is the E15 artifact. It is written on failure too: an absent
// artifact is indistinguishable from "never ran".
type rollbackReport struct {
	Schema           string                `json:"schema"`
	GeneratedAt      string                `json:"generated_at"`
	CommitSHA        string                `json:"commit_sha"`
	TreeDirty        bool                  `json:"tree_dirty"`
	Database         string                `json:"database"`
	PrevRef          string                `json:"prev_ref"`
	PrevBinarySHA256 string                `json:"prev_binary_sha256"`
	CurBinarySHA256  string                `json:"cur_binary_sha256"`
	Steps            []rollbackStep        `json:"steps"`
	Probes           []rollbackProbeResult `json:"probes"`
	Rollback         rollbackOutcome       `json:"rollback"`
	// PostRollbackIndexing is an observation, not a gate: §9 promises "the old
	// release is restored and the data digest is unchanged" and says nothing
	// about a previous release indexing blocks newer than its own schema. A
	// previous build that only reads still satisfies the scenario.
	PostRollbackIndexingObserved bool     `json:"post_rollback_indexing_observed"`
	Passed                       bool     `json:"passed"`
	Notes                        []string `json:"notes"`
}

// TestDeployedReleaseRollback is §9's E15: wrong image / wrong configuration /
// interrupted migration must fail closed, and rolling back to the *previous*
// release on the migrated schema must restore service with the data digest
// byte-identical to the one the new release was serving.
//
// The whole scenario is opt-in because it replaces a real database and runs two
// real binaries. Targets must be isolated: the test creates and drops its own
// databases and starts its own processes.
func TestDeployedReleaseRollback(t *testing.T) {
	if os.Getenv("CHAINPULSE_RUN_DEPLOYED_ROLLBACK") != "1" {
		t.Skip("set CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1 with isolated deployed targets")
	}
	required := func(key string) string {
		t.Helper()
		value := os.Getenv(key)
		require.NotEmpty(t, value, key)
		return value
	}
	baseDSN := required("CHAINPULSE_DEPLOYED_ROLLBACK_DATABASE_URL")
	rpcURL := required("CHAINPULSE_DEPLOYED_ROLLBACK_RPC_URL")
	emitter := common.HexToAddress(required("CHAINPULSE_DEPLOYED_ROLLBACK_EMITTER"))
	prevRef := required("CHAINPULSE_DEPLOYED_ROLLBACK_PREV_REF")
	mongoURI := required("CHAINPULSE_DEPLOYED_MONGODB_URI")
	redisAddr := required("CHAINPULSE_DEPLOYED_REDIS_ADDR")
	kafkaAddr := required("CHAINPULSE_DEPLOYED_KAFKA_ADDR")
	artifacts := required("CHAINPULSE_ACCEPTANCE_OUTPUT")
	require.NoError(t, os.MkdirAll(artifacts, 0o755))

	projectRoot := acceptanceProjectRoot(t)
	report := &rollbackReport{
		Schema:      "chainpulse.rollback-report.v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		CommitSHA:   gitRevision(projectRoot),
		TreeDirty:   gitDirty(projectRoot),
		Database:    redactConnectionString(baseDSN),
		PrevRef:     prevRef,
	}
	defer writeRollbackReport(t, artifacts, report)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	httpClient := &http.Client{Timeout: 3 * time.Second}
	require.NoError(t, validateRollbackRPC(ctx, rpcURL, emitter))
	pool, err := rpc.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer pool.Close()
	eth := ethclient.NewClient(pool)

	// Three databases: the one the scenario migrates in place, one frozen at the
	// previous release's schema, and one left dirty by an interrupted migration.
	// The dirty one is deliberately at schema 21 rather than 16 so the failure is
	// attributable to the dirty ledger alone, not to a version floor that would
	// have refused the start anyway.
	suffix := time.Now().UnixNano()
	mainDB := fmt.Sprintf("chainpulse_e15_main_%d", suffix)
	staleDB := fmt.Sprintf("chainpulse_e15_stale_%d", suffix)
	dirtyDB := fmt.Sprintf("chainpulse_e15_dirty_%d", suffix)
	t.Cleanup(func() { rollbackDropDatabase(t, baseDSN, staleDB, dirtyDB, mainDB) })
	rollbackCreateDatabase(t, baseDSN, mainDB, staleDB, dirtyDB)
	mainDSN := rollbackDatabaseURL(t, baseDSN, mainDB)
	staleDSN := rollbackDatabaseURL(t, baseDSN, staleDB)
	dirtyDSN := rollbackDatabaseURL(t, baseDSN, dirtyDB)

	// Binaries: the current tree, and the previous release rebuilt from its own
	// source so the expand migration is a real 16->21 upgrade.
	binDir := t.TempDir()
	curBinary := filepath.Join(binDir, "chainpulse-current")
	rollbackRun(t, projectRoot, "go", "build", "-o", curBinary, "./cmd/monolithic/chainpulse")
	prevSource := t.TempDir()
	rollbackExtractRevision(t, projectRoot, prevRef, prevSource)
	prevBinary := filepath.Join(binDir, "chainpulse-previous")
	rollbackRun(t, prevSource, "go", "build", "-o", prevBinary, "./cmd/monolithic/chainpulse")
	report.CurBinarySHA256 = rollbackFileSHA256(t, curBinary)
	report.PrevBinarySHA256 = rollbackFileSHA256(t, prevBinary)

	// The previous release's migrations, then the previous release itself on a
	// schema the current tree will later expand.
	rollbackMigrate(t, filepath.Join(prevSource, "migrations"), mainDSN, schemaBeforeExpand)
	rollbackMigrate(t, filepath.Join(prevSource, "migrations"), staleDSN, schemaBeforeExpand)
	rollbackMigrate(t, filepath.Join(projectRoot, "migrations"), dirtyDSN, 21)
	rollbackSetLedgerDirty(t, dirtyDSN)

	adapter := newAdapterForDigest(t, mainDSN)
	db := openDB(t, mainDSN)

	// One port for both releases: the rollback must land on the same endpoint the
	// old release was serving before the deploy.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	apiPort := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	apiBase := fmt.Sprintf("http://127.0.0.1:%d", apiPort)

	// Both releases run as native processes (docker is used only for the corrupt
	// image probe). The environment is identical for prev and cur on purpose: the
	// only difference between the two starts is the binary, so a difference in
	// behaviour cannot be blamed on configuration drift.
	releaseEnv := func() map[string]string {
		return map[string]string{
			"CHAINS":                         rollbackChainID,
			"CHAIN_ID":                       rollbackChainID,
			"CHAINPULSE_CHAINS":              rollbackChainID,
			"BLOCKCHAIN_NODE_URL":            rpcURL,
			"BLOCKCHAIN_NODE_URLS":           rpcURL,
			"FINALITY_DEFAULT_CONFIRMATIONS": "0",
			"DATABASE_TYPE":                  "postgres",
			"DATABASE_URL":                   mainDSN,
			"DATABASE_SSLMODE":               "disable",
			"CACHE_TYPE":                     "redis",
			"CACHE_CONNECTION_URL":           redisAddr,
			"MONGODB_URI":                    mongoURI,
			"MQ_CONNECTION_URL":              kafkaAddr,
			"API_PORT":                       fmt.Sprint(apiPort),
		}
	}
	var process *exec.Cmd
	var logFile *os.File
	var logPath string
	var processExit chan error
	start := func(binary, label string) {
		t.Helper()
		logFile, err = os.CreateTemp(artifacts, "rollback-"+label+"-*.log")
		require.NoError(t, err)
		logPath = logFile.Name()
		report.Notes = append(report.Notes, label+" log: "+logPath)
		process = exec.Command(binary)
		process.Env = rollbackProcessEnv(releaseEnv())
		process.Stdout = logFile
		process.Stderr = logFile
		require.NoError(t, process.Start())
		processExit = make(chan error, 1)
		go func() { processExit <- process.Wait() }()
	}
	stop := func(label string) {
		t.Helper()
		if process == nil {
			return
		}
		// Wait may already have reaped the process (waitHealthy probes for an
		// early exit); signalling again would only produce a spurious error.
		if process.ProcessState == nil {
			_ = process.Process.Signal(syscall.SIGTERM)
			select {
			case err := <-processExit:
				require.NoError(t, err, "%s must exit cleanly on SIGTERM", label)
			case <-time.After(30 * time.Second):
				_ = process.Process.Kill()
				<-processExit
				t.Fatalf("%s shutdown exceeded deadline", label)
			}
		}
		require.NoError(t, logFile.Close())
		process = nil
	}
	t.Cleanup(func() { stop("cleanup") })
	waitHealthy := func(label string, limit time.Duration) {
		t.Helper()
		deadline := time.Now().Add(limit)
		for time.Now().Before(deadline) {
			select {
			case exitErr := <-processExit:
				t.Fatalf("%s exited before becoming healthy (%v); last log lines:\n%s", label, exitErr, rollbackLogTail(logPath))
			default:
			}
			response, err := httpClient.Get(apiBase + "/health")
			if err == nil {
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatalf("%s never served /health 200; last log lines:\n%s", label, rollbackLogTail(logPath))
	}
	rows := func() int {
		var count int
		require.NoError(t, db.QueryRowContext(ctx,
			"SELECT count(*) FROM blockchain_events WHERE chain_id=$1", rollbackChainID).Scan(&count))
		return count
	}
	// An emission is only evidence once the process has durably recorded the
	// block it landed in; comparing digests before that would race the writer.
	waitIndexed := func(label string, atLeastRows int, height uint64) {
		t.Helper()
		require.Eventually(t, func() bool {
			var count int
			var checkpoint uint64
			if err := db.QueryRowContext(ctx,
				"SELECT count(*) FROM blockchain_events WHERE chain_id=$1", rollbackChainID).Scan(&count); err != nil {
				return false
			}
			if err := db.QueryRowContext(ctx,
				"SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1", rollbackChainID).Scan(&checkpoint); err != nil {
				return false
			}
			return count >= atLeastRows && checkpoint >= height
		}, 2*time.Minute, 200*time.Millisecond,
			"%s did not index the emitted block; last log lines:\n%s", label, rollbackLogTail(logPath))
	}
	emit := func(value int64) uint64 {
		t.Helper()
		data := append(crypto.Keccak256([]byte("emitTransfer(address,uint256)"))[:4],
			common.LeftPadBytes(common.HexToAddress("0x000000000000000000000000000000000000dEaD").Bytes(), 32)...)
		data = append(data, common.LeftPadBytes(big.NewInt(value).Bytes(), 32)...)
		var hash common.Hash
		require.NoError(t, pool.CallContext(ctx, &hash, "eth_sendTransaction", map[string]any{
			"from":     "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			"to":       emitter.Hex(),
			"data":     hexutil.Encode(data),
			"gas":      "0x7a120",
			"gasPrice": "0x3b9aca00",
		}))
		receipt, err := rollbackWaitReceipt(ctx, eth, hash)
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status)
		return receipt.BlockNumber.Uint64()
	}
	recordStep := func(name string, dsn string) rollbackStep {
		t.Helper()
		digest, events := chainDigest(t, adapter, rollbackChainID)
		version, _ := rollbackSchemaVersion(t, dsn)
		step := rollbackStep{Name: name, Schema: version, Rows: rows(), Events: events, Digest: digest}
		report.Steps = append(report.Steps, step)
		t.Logf("step %s: schema=%d rows=%d digest_events=%d digest=%s", name, step.Schema, step.Rows, step.Events, step.Digest)
		return step
	}

	// --- previous release on its own schema ---------------------------------
	start(prevBinary, "previous-schema16")
	waitHealthy("previous release on schema16", 2*time.Minute)
	before := rows()
	heightBefore := emit(111)
	waitIndexed("previous release on schema16", before+1, heightBefore)
	require.Greater(t, rows(), before, "the previous release must index the existing chain")

	// --- probe A: invalid configuration -------------------------------------
	pending := releaseEnv()
	pending["API_PORT"] = "0"
	probe := rollbackRunApp(t, ctx, artifacts, "config", curBinary, pending, "API_PORT must be 1-65535")

	// The previous release is stopped before the digest is taken so the sample is
	// a quiescent state, not one a running writer can still move.
	stop("previous release on schema16")
	beforeExpand := recordStep("previous_release_schema16", mainDSN)

	// --- expand 16 -> 21 ----------------------------------------------------
	rollbackMigrate(t, filepath.Join(projectRoot, "migrations"), mainDSN, 21)
	version, dirty := rollbackSchemaVersion(t, mainDSN)
	require.Equal(t, 21, version, "the expand must land the main database on the current schema")
	require.False(t, dirty, "a completed expand must leave the ledger clean")

	// --- probes B/C/D: identity, stale schema, interrupted migration --------
	// B runs after the expand on purpose: on the old schema the canonical cache
	// gate fires first and the identity error would never be reached, so the
	// probe would be measuring the wrong refusal.
	identity := releaseEnv()
	identity["CHAINS"] = "1"
	identity["CHAIN_ID"] = "1"
	identity["CHAINPULSE_CHAINS"] = "1"
	probeB := rollbackRunApp(t, ctx, artifacts, "identity", curBinary, identity, "RPC chain ID mismatch")

	stale := releaseEnv()
	stale["DATABASE_URL"] = staleDSN
	probeC := rollbackRunApp(t, ctx, artifacts, "stale-schema", curBinary, stale, "requires clean schema"+fmt.Sprint(rollbackSchemaFloor))
	staleVersion, _ := rollbackSchemaVersion(t, staleDSN)
	require.Equal(t, schemaBeforeExpand, staleVersion, "probe C must be measured against the previous release's schema")

	dirtyEnv := releaseEnv()
	dirtyEnv["DATABASE_URL"] = dirtyDSN
	probeD := rollbackRunApp(t, ctx, artifacts, "dirty-ledger", curBinary, dirtyEnv, "requires clean schema"+fmt.Sprint(rollbackSchemaFloor))
	dirtyVersion, dirtyFlag := rollbackSchemaVersion(t, dirtyDSN)
	require.Equal(t, 21, dirtyVersion, "probe D must be attributable to the dirty ledger, not to a version floor")
	require.True(t, dirtyFlag)

	// --- probe E: corrupt image ---------------------------------------------
	require.NotEmpty(t, rollbackDockerBinary(t), "docker must be available while CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1; the corrupt-image probe is the only container-form evidence")
	probeE := rollbackRunBadImage(t, artifacts, projectRoot)

	// --- current release on the expanded schema -----------------------------
	start(curBinary, "current-schema21")
	waitHealthy("current release on schema21", 2*time.Minute)
	heightDeploy := emit(222)
	waitIndexed("current release on schema21", beforeExpand.Rows+1, heightDeploy)
	require.Greater(t, rows(), beforeExpand.Rows, "the current release must index on the expanded schema")
	stop("current release on schema21")
	afterDeploy := recordStep("current_release_schema21", mainDSN)
	require.NotEqual(t, beforeExpand.Digest, afterDeploy.Digest,
		"the emit must change the digest, otherwise the rollback comparison below is vacuous")

	// --- rollback: the previous release on the current schema ---------------
	start(prevBinary, "previous-rollback")
	waitHealthy("previous release after rollback", 2*time.Minute)
	health := rollbackHTTPStatus(t, httpClient, apiBase+"/health")
	eventsStatus, eventsPath := rollbackEventsStatus(t, httpClient, apiBase)
	require.Equal(t, http.StatusOK, health, "the previous release must serve /health after the rollback")
	require.Equal(t, http.StatusOK, eventsStatus, "the previous release must serve the events API after the rollback")
	restored := recordStep("rollback_to_previous_release", mainDSN)
	require.Equal(t, afterDeploy.Digest, restored.Digest,
		"the rollback must restore the previous release with a byte-identical data digest")
	require.Equal(t, afterDeploy.Rows, restored.Rows)

	report.Rollback = rollbackOutcome{
		Schema:       restored.Schema,
		HealthStatus: health,
		EventsStatus: eventsStatus,
		EventsPath:   eventsPath,
		PriorDigest:  afterDeploy.Digest,
		Digest:       restored.Digest,
		Identical:    afterDeploy.Digest == restored.Digest,
	}

	// --- observation: does the previous release still index? ----------------
	emit(333)
	indexed := false
	require.Eventually(t, func() bool {
		indexed = rows() > restored.Rows
		return indexed
	}, 2*time.Minute, 200*time.Millisecond, "post-rollback indexing observation timed out")
	report.PostRollbackIndexingObserved = indexed
	postStep := recordStep("previous_release_after_rollback_indexing", mainDSN)
	if indexed {
		t.Logf("previous release kept indexing after the rollback: rows=%d digest=%s", postStep.Rows, postStep.Digest)
	} else {
		t.Logf("previous release stayed read-only after the rollback; §9 only requires the digest to be unchanged")
	}
	stop("previous release after observation")

	report.Probes = []rollbackProbeResult{probe, probeB, probeC, probeD, probeE}
	report.Passed = true
	t.Logf("E15 passed: prev=%s cur=%s digests equal across rollback (%s)",
		shortDigest(report.PrevBinarySHA256), shortDigest(report.CurBinarySHA256), shortDigest(restored.Digest))
}

// rollbackWaitReceipt waits for a receipt without failing the test on a
// transient miss, so the caller can report which transaction never landed.
func rollbackWaitReceipt(ctx context.Context, eth *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		receipt, err := eth.TransactionReceipt(ctx, hash)
		if err == nil && receipt != nil {
			return receipt, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("receipt for %s never appeared: %w", hash.Hex(), lastErr)
}

// validateRollbackRPC fails early when the fixture's node is not the chain this
// scenario claims or the emitter is not deployed, rather than letting every
// later step fail with an unrelated error.
func validateRollbackRPC(ctx context.Context, rpcURL string, emitter common.Address) error {
	pool, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("dial %s: %w", rpcURL, err)
	}
	defer pool.Close()
	eth := ethclient.NewClient(pool)
	chainID, err := eth.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("read chain id from %s: %w", rpcURL, err)
	}
	if chainID.String() != rollbackChainID {
		return fmt.Errorf("fixture node serves chain %s, this scenario needs %s", chainID, rollbackChainID)
	}
	code, err := eth.CodeAt(ctx, emitter, nil)
	if err != nil {
		return fmt.Errorf("read emitter %s code: %w", emitter, err)
	}
	if len(code) == 0 {
		return fmt.Errorf("emitter %s has no code on chain %s", emitter, rollbackChainID)
	}
	return nil
}

// rollbackRunApp starts a release with an adversarial environment and requires
// the process to refuse to run: non-zero exit, the exact message, and never the
// "Status: Running" line that proves it reached steady state.
//
// A context deadline is explicitly NOT accepted as rejection: a hung process is
// a different defect from a fail-closed startup.
func rollbackRunApp(t *testing.T, ctx context.Context, artifacts, name, binary string, env map[string]string, wantMessage string) rollbackProbeResult {
	t.Helper()
	probeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	command := exec.CommandContext(probeCtx, binary)
	command.Env = rollbackProcessEnv(env)
	output, err := command.CombinedOutput()
	evidence := filepath.Join(artifacts, "rollback-probe-"+name+".log")
	require.NoError(t, os.WriteFile(evidence, output, 0o644))
	require.NoError(t, probeCtx.Err(), "probe %s was killed or timed out; that is not startup rejection", name)
	require.Error(t, err, "probe %s exited 0:\n%s", name, output)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Contains(t, string(output), wantMessage, "probe %s must refuse with the exact message", name)
	require.NotContains(t, string(output), "Status: Running", "probe %s must never reach steady state", name)
	t.Logf("probe %s rejected the start: exit=%d message=%q", name, exitErr.ExitCode(), wantMessage)
	return rollbackProbeResult{Name: name, ExitCode: exitErr.ExitCode(), Message: wantMessage, Rejected: true, EvidenceAt: evidence}
}

// rollbackRunBadImage builds the prebuilt image from a build context whose
// binary is not an executable and requires the container to fail. This is the
// only probe that has to be container-shaped: "a corrupt image does not start"
// cannot be demonstrated by a configuration error in a process.
func rollbackRunBadImage(t *testing.T, artifacts, projectRoot string) rollbackProbeResult {
	t.Helper()
	contextDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "chainpulse-linux"), []byte("not-an-executable"), 0o644))
	dockerfile, err := os.ReadFile(filepath.Join(projectRoot, "docker", "Dockerfile.prebuilt"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "Dockerfile.prebuilt"), dockerfile, 0o644))
	image := fmt.Sprintf("chainpulse-rollback-bad:%d", time.Now().UnixNano())
	t.Cleanup(func() { rollbackDocker(t, "rmi", "-f", image) })
	rollbackDocker(t, "build", "-q", "-f", filepath.Join(contextDir, "Dockerfile.prebuilt"), "-t", image, contextDir)
	probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(probeCtx, "docker", "run", "--rm", image)
	output, err := command.CombinedOutput()
	evidence := filepath.Join(artifacts, "rollback-probe-corrupt-image.log")
	require.NoError(t, os.WriteFile(evidence, output, 0o644))
	require.NoError(t, probeCtx.Err(), "the corrupt image probe was killed or timed out")
	require.Error(t, err, "a corrupt image must not start:\n%s", output)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.NotContains(t, string(output), "Status: Running")
	t.Logf("probe corrupt-image failed to run: exit=%d", exitErr.ExitCode())
	return rollbackProbeResult{Name: "corrupt-image", ExitCode: exitErr.ExitCode(), Message: "container exit non-zero", Rejected: true, EvidenceAt: evidence}
}

func rollbackDocker(t *testing.T, args ...string) {
	t.Helper()
	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "docker %v: %s", args, output)
}

func rollbackDockerBinary(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("docker")
	if err != nil {
		return ""
	}
	return path
}

func rollbackEventsStatus(t *testing.T, client *http.Client, base string) (int, string) {
	t.Helper()
	for _, path := range []string{"/api/v1/events?limit=5", "/events?chain_id=" + rollbackChainID + "&limit=5"} {
		status := rollbackHTTPStatus(t, client, base+path)
		if status == http.StatusOK {
			return status, path
		}
		if status != http.StatusNotFound {
			return status, path
		}
	}
	return http.StatusNotFound, "/api/v1/events?limit=5"
}

func rollbackHTTPStatus(t *testing.T, client *http.Client, endpoint string) int {
	t.Helper()
	response, err := client.Get(endpoint)
	if err != nil {
		return 0
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode
}

func rollbackSchemaVersion(t *testing.T, dsn string) (int, bool) {
	t.Helper()
	db := openDB(t, dsn)
	var version int
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	require.NoError(t, err)
	return version, dirty
}

func rollbackSetLedgerDirty(t *testing.T, dsn string) {
	t.Helper()
	db := openDB(t, dsn)
	_, err := db.Exec("UPDATE schema_migrations SET dirty=true")
	require.NoError(t, err)
}

func rollbackMigrate(t *testing.T, migrationsDir, dsn string, version uint) {
	t.Helper()
	root, err := filepath.Abs(migrationsDir)
	require.NoError(t, err)
	migrator, err := migrate.New("file://"+root, dsn)
	require.NoError(t, err)
	defer func() {
		sourceErr, databaseErr := migrator.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, databaseErr)
	}()
	require.NoError(t, migrator.Migrate(version), "migrate %s to %d", migrationsDir, version)
}

func rollbackCreateDatabase(t *testing.T, base string, names ...string) {
	t.Helper()
	db, err := sql.Open("postgres", base)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	for _, name := range names {
		_, err := db.Exec("CREATE DATABASE " + pq.QuoteIdentifier(name))
		require.NoError(t, err, "create database %s", name)
	}
}

// rollbackDropDatabase is best-effort on purpose: it runs from a cleanup, after
// the scenario's real failure has already been reported, and losing that reason
// to a teardown error would be worse than leaking a scratch database.
func rollbackDropDatabase(t *testing.T, base string, names ...string) {
	t.Helper()
	db, err := sql.Open("postgres", base)
	if err != nil {
		t.Logf("cleanup: cannot open %s: %v", redactConnectionString(base), err)
		return
	}
	defer func() { _ = db.Close() }()
	for _, name := range names {
		if _, err := db.Exec("DROP DATABASE IF EXISTS " + pq.QuoteIdentifier(name) + " WITH (FORCE)"); err != nil {
			t.Logf("cleanup: could not drop %s: %v", name, err)
		}
	}
}

func rollbackDatabaseURL(t *testing.T, base, name string) string {
	t.Helper()
	parsed, err := url.Parse(base)
	require.NoError(t, err)
	parsed.Path = "/" + name
	return parsed.String()
}

// rollbackExtractRevision materialises a past revision with `git archive`, so
// the previous release is compiled from its own source rather than from the
// current tree with a different tag.
func rollbackExtractRevision(t *testing.T, projectRoot, revision, destination string) {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "revision.tar")
	rollbackRun(t, projectRoot, "git", "archive", "--format=tar", "-o", archive, revision)
	rollbackRun(t, projectRoot, "tar", "-x", "-f", archive, "-C", destination)
	entries, err := os.ReadDir(filepath.Join(destination, "migrations"))
	require.NoError(t, err, "revision %s has no migrations directory", revision)
	var ups int
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			ups++
		}
	}
	require.Equal(t, schemaBeforeExpand, ups,
		"revision %s must be the release before the expand migration (%d up migrations)", revision, schemaBeforeExpand)
}

func rollbackRun(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	goproxy := os.Getenv("GOPROXY")
	if goproxy == "" {
		goproxy = "https://goproxy.cn,direct"
	}
	command.Env = rollbackProcessEnv(map[string]string{"GOPROXY": goproxy})
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s %v (cwd %s): %s", name, args, dir, output)
	return string(output)
}

func rollbackFileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func rollbackLogTail(path string) string {
	if path == "" {
		return "<no log>"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "<unreadable log " + path + ">"
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > 25 {
		lines = lines[len(lines)-25:]
	}
	return strings.Join(lines, "\n")
}

// rollbackProcessEnv replaces, rather than appends to, inherited variables. A
// CI runner can already export DATABASE_URL or MONGODB_URI; appending the
// scenario's value second would leave the winner up to the platform, so the
// overrides are removed from the inherited environment first.
func rollbackProcessEnv(overrides map[string]string) []string {
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key := entry
		if index := strings.Index(entry, "="); index > 0 {
			key = entry[:index]
		}
		if _, replaced := overrides[key]; replaced {
			continue
		}
		env = append(env, entry)
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func shortDigest(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func writeRollbackReport(t *testing.T, artifacts string, report *rollbackReport) {
	t.Helper()
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Errorf("could not encode the rollback report: %v", err)
		return
	}
	path := filepath.Join(artifacts, "rollback-report.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		t.Errorf("could not write %s: %v", path, err)
		return
	}
	t.Logf("rollback report written to %s", path)
}
