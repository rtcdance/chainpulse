//go:build integration

package acceptance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// This scenario starts the real monolithic entry point, writes via a real Anvil
// and compares canonical SQL effects with eth_getLogs. Targets are opt-in and
// must be isolated: it intentionally replaces local chain branches.
func TestDeployedReorgCanonicalDigestAndRestart(t *testing.T) {
	if os.Getenv("CHAINPULSE_RUN_DEPLOYED_REORG") != "1" {
		t.Skip("set CHAINPULSE_RUN_DEPLOYED_REORG=1 with isolated deployed targets")
	}
	required := func(key string) string { t.Helper(); v := os.Getenv(key); require.NotEmpty(t, v, key); return v }
	rpcURL := required("CHAINPULSE_DEPLOYED_RPC_URL")
	dsn := required("CHAINPULSE_DEPLOYED_DATABASE_URL")
	binary := required("CHAINPULSE_DEPLOYED_BINARY")
	emitter := common.HexToAddress(required("CHAINPULSE_DEPLOYED_EMITTER"))
	apiPort := required("CHAINPULSE_DEPLOYED_API_PORT")
	artifacts := required("CHAINPULSE_ACCEPTANCE_OUTPUT")
	require.NoError(t, os.MkdirAll(artifacts, 0o755))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	client, err := rpc.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer client.Close()
	eth := ethclient.NewClient(client)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	// The parent scenario owns an isolated target database. Its restart checks
	// preserve data within this run, not checkpoints belonging to a previous
	// independently created chain with the same numeric ID.
	_, err = db.ExecContext(ctx, "TRUNCATE blockchain_events,indexing_state,blocks,chain_leases,reorg_jobs,reorg_outbox CASCADE")
	require.NoError(t, err)

	var process *exec.Cmd
	var logFile *os.File
	start := func(t *testing.T) {
		t.Helper()
		logFile, err = os.CreateTemp(artifacts, "monolith-*.log")
		require.NoError(t, err)
		process = exec.Command(binary)
		process.Env = append(os.Environ(), "CHAINS=31337", "CHAIN_ID=31337", "CHAINPULSE_CHAINS=31337", "BLOCKCHAIN_NODE_URL="+rpcURL, "BLOCKCHAIN_NODE_URLS="+rpcURL, "FINALITY_DEFAULT_CONFIRMATIONS=0", "DATABASE_URL="+dsn, "DATABASE_TYPE=postgres", "DATABASE_SSLMODE=disable", "API_PORT="+apiPort, "MONGODB_URI="+required("CHAINPULSE_DEPLOYED_MONGODB_URI"), "CACHE_CONNECTION_URL="+required("CHAINPULSE_DEPLOYED_REDIS_ADDR"), "MQ_CONNECTION_URL="+required("CHAINPULSE_DEPLOYED_KAFKA_ADDR"))
		process.Stdout = logFile
		process.Stderr = logFile
		require.NoError(t, process.Start())
	}
	stop := func(t *testing.T) {
		t.Helper()
		if process == nil {
			return
		}
		require.NoError(t, process.Process.Signal(syscall.SIGTERM))
		done := make(chan error, 1)
		go func() { done <- process.Wait() }()
		select {
		case err := <-done:
			require.NoError(t, err, "SIGTERM must exit cleanly")
		case <-time.After(25 * time.Second):
			_ = process.Process.Kill()
			<-done
			t.Fatal("monolith shutdown exceeded deadline")
		}
		require.NoError(t, logFile.Close())
		process = nil
	}
	t.Cleanup(func() { stop(t) })
	start(t)
	waitCheckpoint := func(t *testing.T, target uint64) {
		t.Helper()
		require.Eventually(t, func() bool {
			var height uint64
			err := db.QueryRowContext(ctx, "SELECT last_indexed_block FROM indexing_state WHERE chain_id='31337'").Scan(&height)
			return err == nil && height >= target
		}, 30*time.Second, 100*time.Millisecond, "durable checkpoint did not catch up")
	}
	emit := func(t *testing.T) uint64 {
		t.Helper()
		data := append(crypto.Keccak256([]byte("emitBatch(uint256)"))[:4], common.LeftPadBytes(big.NewInt(2).Bytes(), 32)...)
		var hash common.Hash
		require.NoError(t, client.CallContext(ctx, &hash, "eth_sendTransaction", map[string]any{"from": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266", "to": emitter.Hex(), "data": hexutil.Encode(data), "gas": "0x7a120", "gasPrice": "0x3b9aca00"}))
		var height uint64
		require.Eventually(t, func() bool {
			receipt, err := eth.TransactionReceipt(ctx, hash)
			if err != nil {
				return false
			}
			height = receipt.BlockNumber.Uint64()
			return receipt.Status == 1
		}, 5*time.Second, 50*time.Millisecond)
		return height
	}
	for _, depth := range []int{1, 2, 12} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			var snapshot string
			require.NoError(t, client.CallContext(ctx, &snapshot, "evm_snapshot"))
			var tip uint64
			for range depth {
				tip = emit(t)
			}
			waitCheckpoint(t, tip)
			oldBlock, err := eth.BlockByNumber(ctx, new(big.Int).SetUint64(tip))
			require.NoError(t, err)
			require.Len(t, oldBlock.Transactions(), 1)
			if depth == 2 {
				stop(t)
			}
			var reverted bool
			require.NoError(t, client.CallContext(ctx, &reverted, "evm_revert", snapshot))
			require.True(t, reverted)
			var advanced json.RawMessage
			require.NoError(t, client.CallContext(ctx, &advanced, "evm_increaseTime", 3600))
			var newTip uint64
			for range depth {
				newTip = emit(t)
			}
			require.Equal(t, tip, newTip, "replacement branch must reach exactly the same height")
			newBlock, err := eth.BlockByNumber(ctx, new(big.Int).SetUint64(newTip))
			require.NoError(t, err)
			require.Len(t, newBlock.Transactions(), 1)
			require.Equal(t, oldBlock.Transactions()[0].Hash(), newBlock.Transactions()[0].Hash(), "same transaction must be re-included")
			require.NotEqual(t, oldBlock.Hash(), newBlock.Hash(), "same-height branch identity must change")
			if depth == 2 {
				start(t)
			}
			started := time.Now()
			var observedDigest string
			require.Eventually(t, func() bool {
				logs, err := eth.FilterLogs(ctx, ethereum.FilterQuery{Addresses: []common.Address{emitter}})
				if err != nil {
					return false
				}
				var expected []string
				for _, log := range logs {
					expected = append(expected, fmt.Sprintf("%s/%s/%d/%s", log.BlockHash.Hex(), log.TxHash.Hex(), log.Index, base64.StdEncoding.EncodeToString(log.Data)))
				}
				rows, err := db.QueryContext(ctx, "SELECT block_hash,transaction_hash,log_index,event_data->>'raw' FROM blockchain_events WHERE chain_id='31337' AND lower(contract_address)=lower($1) AND status<>'reorged'", emitter.Hex())
				if err != nil {
					return false
				}
				var actual []string
				for rows.Next() {
					var block, tx, raw string
					var index uint64
					if rows.Scan(&block, &tx, &index, &raw) != nil {
						_ = rows.Close()
						return false
					}
					actual = append(actual, fmt.Sprintf("%s/%s/%d/%s", block, tx, index, raw))
				}
				rowErr := rows.Err()
				_ = rows.Close()
				if rowErr != nil {
					return false
				}
				sort.Strings(expected)
				sort.Strings(actual)
				if strings.Join(expected, "\n") != strings.Join(actual, "\n") {
					return false
				}
				var pending int
				err = db.QueryRowContext(ctx, "SELECT count(*) FROM reorg_jobs WHERE chain_id='31337' AND status<>'done'").Scan(&pending)
				if err != nil || pending != 0 {
					return false
				}
				sum := sha256.Sum256([]byte(strings.Join(actual, "\n")))
				observedDigest = hex.EncodeToString(sum[:])
				return true
			}, 30*time.Second, 100*time.Millisecond, "canonical effects must match RPC with no pending replay")
			waitCheckpoint(t, tip)
			t.Logf("depth=%d canonical_sha256=%s recovery_ms=%d", depth, observedDigest, time.Since(started).Milliseconds())
		})
	}
	stop(t)
}
