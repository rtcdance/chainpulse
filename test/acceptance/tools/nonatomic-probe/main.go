// Command nonatomic-probe is a NEGATIVE CONTROL, not a workload.
//
// It is byte-for-byte the same workload as tools/graceful except for one thing:
// it commits the events and the checkpoint advance in TWO separate transactions
// instead of one. That is the pre-T05 shape (`CheckAndMark` followed by a
// separate write and a separate checkpoint update), and splitting it is the only
// difference.
//
// It exists so the crash tests can demonstrate that their assertions actually
// discriminate: TestCrashInsideCommitLeavesNothingBehind asserts a kill inside
// the commit leaves nothing behind, and that assertion is only meaningful if a
// non-atomic implementation would FAIL it. The control test kills this process
// in the same window and asserts the durable state is inconsistent, i.e. that
// events survived without their checkpoint — the half-commit the plan forbids.
//
// Never use this in a real run.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

const defaultBatchSize = 3

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "nonatomic-probe: %v\n", err)
	os.Exit(1)
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
	txDelay := time.Duration(0)
	if v := os.Getenv("CHAINPULSE_GRACEFUL_TX_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			txDelay = time.Duration(n) * time.Millisecond
		}
	}

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
		if !drain.Drain(15 * time.Second) {
			os.Exit(2)
		}
	}()

	for block := uint64(0); ; block++ {
		if !drain.Begin() {
			break
		}
		fmt.Printf("begin tx %d\n", block)
		if err := commitEventsThenCheckpoint(ctx, adapter, chainID, block, batchSize, txDelay); err != nil {
			fatal(err)
		}
		drain.End()
		fmt.Printf("committed block %d\n", block)
		time.Sleep(5 * time.Millisecond)
	}
}

// commitEventsThenCheckpoint writes the events in one transaction and advances
// the checkpoint in a SECOND one, pausing in between. A crash during that pause
// leaves events durable with no checkpoint advance — the state the atomic
// committer exists to make impossible.
func commitEventsThenCheckpoint(
	ctx context.Context, a *database.PostgreSQLPluginAdapter,
	chainID string, block uint64, size int, betweenTxDelay time.Duration,
) error {
	blockHash := common.HexToHash(fmt.Sprintf("0x%064x", block+1))

	eventsTx, err := a.BeginTx(ctx)
	if err != nil {
		return err
	}
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
		if err := eventsTx.StoreEvent(ctx, e); err != nil {
			_ = eventsTx.Rollback(ctx)
			return err
		}
	}
	if err := eventsTx.Commit(ctx); err != nil {
		return err
	}
	// The marker the control test waits for: events are durable, the checkpoint
	// is not. A kill after this line is a half-commit in the making.
	fmt.Printf("events committed %d\n", block)

	if betweenTxDelay > 0 {
		time.Sleep(betweenTxDelay)
	}

	checkpointTx, err := a.BeginTx(ctx)
	if err != nil {
		return err
	}
	if err := checkpointTx.SaveCheckpoint(ctx, chainID, block, blockHash.Hex()); err != nil {
		_ = checkpointTx.Rollback(ctx)
		return err
	}
	return checkpointTx.Commit(ctx)
}
