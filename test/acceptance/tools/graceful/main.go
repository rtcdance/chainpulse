// Command graceful is the workload behind the T05 / F24 acceptance test.
//
// It repeatedly commits a batch of events together with the checkpoint advance
// for that batch in ONE transaction (the T05 unit of work), printing
// "committed block <n>" after each commit. On SIGTERM it stops accepting new
// batches, lets the in-flight one finish, and exits 0.
//
// The property under test: after SIGTERM the checkpoint must still be a batch
// boundary and the event count must match it exactly — no half-commit, no gap.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

const defaultBatchSize = 3

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "graceful: %v\n", err)
	os.Exit(1)
}

// startBlock returns the first block this process should commit.
//
// By default it starts at 0, which is what the SIGTERM test wants: a fresh
// process replaying blocks 0..N proves that re-delivery folds into one effect.
// With CHAINPULSE_GRACEFUL_START_FROM_CHECKPOINT=1 it instead resumes after the
// durable checkpoint, which is what proves "rebuild from the checkpoint" rather
// than "replay from the beginning".
func startBlock(dsn, chainID string) uint64 {
	if os.Getenv("CHAINPULSE_GRACEFUL_START_FROM_CHECKPOINT") != "1" {
		return 0
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = db.Close() }()

	var block uint64
	err = db.QueryRow(
		"SELECT last_indexed_block FROM indexing_state WHERE chain_id = $1", chainID,
	).Scan(&block)
	if err == sql.ErrNoRows {
		fmt.Println("no checkpoint yet, starting at block 0")
		return 0
	}
	if err != nil {
		fatal(err)
	}
	fmt.Printf("resuming after checkpoint, starting at block %d\n", block+1)
	return block + 1
}

func main() {
	dsn := os.Getenv("CHAINPULSE_TEST_DATABASE_URL")
	chainID := os.Getenv("CHAINPULSE_GRACEFUL_CHAIN_ID")
	if dsn == "" || chainID == "" {
		fatal(fmt.Errorf("CHAINPULSE_TEST_DATABASE_URL and CHAINPULSE_GRACEFUL_CHAIN_ID are required"))
	}

	batchSize := defaultBatchSize
	if v := os.Getenv("CHAINPULSE_GRACEFUL_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batchSize = n
		}
	}

	// An optional pause between "the events are stored" and "the transaction
	// commits". It exists so a crash test can land a signal inside the
	// transaction deterministically instead of hoping a jittered timer happens
	// to hit the window; the events are already written at that point, so a
	// half-commit is exactly the failure mode being probed.
	txDelay := time.Duration(0)
	if v := os.Getenv("CHAINPULSE_GRACEFUL_TX_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			txDelay = time.Duration(n) * time.Millisecond
		}
	}

	first := startBlock(dsn, chainID)

	ctx := context.Background()
	adapter := database.NewPostgreSQLPluginAdapter(
		testhelpers.NewTestLogger(),
		testhelpers.NewTestMetricsCollector(),
	)
	if err := adapter.Initialize(ctx, core.Config{DatabaseURL: dsn}); err != nil {
		fatal(err)
	}
	if err := adapter.Start(ctx); err != nil {
		fatal(err)
	}

	drain := bootstrap.NewDrainController()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("draining")
		if !drain.Drain(15 * time.Second) {
			fmt.Fprintln(os.Stderr, "graceful: drain timed out")
			os.Exit(2)
		}
	}()

	for block := first; ; block++ {
		if !drain.Begin() {
			break // draining: no new batches
		}
		fmt.Printf("begin tx %d\n", block)
		err := commitBatch(ctx, adapter, chainID, block, batchSize, txDelay)
		drain.End()
		if err != nil {
			fatal(err)
		}
		fmt.Printf("committed block %d\n", block)
		time.Sleep(5 * time.Millisecond)
	}

	if err := adapter.Stop(ctx); err != nil {
		fatal(err)
	}
	fmt.Println("stopped cleanly")
	os.Exit(0)
}

// commitBatch writes batchSize events at `block` and advances the checkpoint to
// that block, all inside one transaction.
func commitBatch(ctx context.Context, a *database.PostgreSQLPluginAdapter, chainID string, block uint64, size int, txDelay time.Duration) error {
	tx, err := a.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	blockHash := common.HexToHash(fmt.Sprintf("0x%064x", block+1))
	for i := 0; i < size; i++ {
		e := &blockchain.BlockchainEvent{
			ChainID:         chainID,
			BlockNumber:     block,
			BlockHash:       blockHash,
			TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", block*100+uint64(i)+1)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000dd"),
			EventName:       "Transfer",
			BlockTimestamp:  1700000400,
		}
		e.EventHash = core.ComputeEventHash(e)
		e.ID = e.EventHash
		if err := tx.StoreEvent(ctx, e); err != nil {
			return err
		}
	}
	if err := tx.SaveCheckpoint(ctx, chainID, block, blockHash.Hex()); err != nil {
		return err
	}
	if txDelay > 0 {
		time.Sleep(txDelay)
	}
	return tx.Commit(ctx)
}
