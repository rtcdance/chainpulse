package main

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

func TestCapturePullerBlockProgress(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	progress := &pullerLoopRuntimeProgress{}
	checkpointSource := newPullerRuntimeCheckpointSource()
	multi := pullers.NewMultiChainDataPuller(logger)

	if err := multi.RegisterPuller("eth", &pullersTestPlugin{name: "eth", latestBlock: 120, lastBlock: 118}); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}
	if err := multi.RegisterPuller("polygon", &pullersTestPlugin{name: "polygon", latestBlock: 150, lastBlock: 144}); err != nil {
		t.Fatalf("register polygon puller: %v", err)
	}

	capturePullerBlockProgress(context.Background(), multi, checkpointSource, 100, logger, progress)

	snapshot := progress.snapshot()
	if snapshot.ObservedBlock != 150 {
		t.Fatalf("expected observed block 150, got %d", snapshot.ObservedBlock)
	}
	if snapshot.ProcessedBlock != 144 {
		t.Fatalf("expected processed block 144, got %d", snapshot.ProcessedBlock)
	}

	checkpointSnapshot := checkpointSource.Snapshot(context.Background())
	if checkpointSnapshot.HighestCheckpointBlock != 0 {
		t.Fatalf("expected no persisted checkpoint at non-boundary block, got %d", checkpointSnapshot.HighestCheckpointBlock)
	}
}

func TestCapturePullerBlockProgressPersistsCheckpointAtBoundary(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	progress := &pullerLoopRuntimeProgress{}
	checkpointSource := newPullerRuntimeCheckpointSource()
	multi := pullers.NewMultiChainDataPuller(logger)

	if err := multi.RegisterPuller("eth", &pullersTestPlugin{name: "eth", latestBlock: 120, lastBlock: 100}); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}
	if err := multi.RegisterPuller("polygon", &pullersTestPlugin{name: "polygon", latestBlock: 150, lastBlock: 144}); err != nil {
		t.Fatalf("register polygon puller: %v", err)
	}

	capturePullerBlockProgress(context.Background(), multi, checkpointSource, 100, logger, progress)

	checkpointSnapshot := checkpointSource.Snapshot(context.Background())
	if checkpointSnapshot.HighestCheckpointBlock != 100 {
		t.Fatalf("expected persisted checkpoint block 100, got %d", checkpointSnapshot.HighestCheckpointBlock)
	}
	if checkpointSnapshot.TrackedChains != 1 {
		t.Fatalf("expected one tracked checkpoint chain, got %d", checkpointSnapshot.TrackedChains)
	}
	if got := formatPullerCheckpointChainSummaryAt(time.Now(), 12, checkpointSnapshot); got != "eth=checkpoint-recorded:fresh@100" {
		t.Fatalf("expected checkpoint chain summary for eth, got %q", got)
	}
	if got := formatPullerCheckpointChainPostureSummaryAt(time.Now(), 12, checkpointSnapshot); got != "eth=recorded-healthy" {
		t.Fatalf("expected checkpoint chain posture summary for eth, got %q", got)
	}
	if got := formatPullerCheckpointCoverageSummary(checkpointSnapshot); got != "tracked=1,recorded=1,reorg_risk=0,reorg_reconciled=0" {
		t.Fatalf("expected checkpoint coverage summary for eth, got %q", got)
	}
	if got := classifyPullerCheckpointCoveragePosture(checkpointSnapshot); got != "coverage-healthy" {
		t.Fatalf("expected coverage posture coverage-healthy, got %q", got)
	}
}

func TestCapturePullerBlockProgressMarksReorgRiskWhenProcessedBlockRegresses(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	progress := &pullerLoopRuntimeProgress{}
	checkpointSource := newPullerRuntimeCheckpointSource()
	if err := checkpointSource.SaveCheckpoint(context.Background(), "eth", 100); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	multi := pullers.NewMultiChainDataPuller(logger)
	if err := multi.RegisterPuller("eth", &pullersTestPlugin{name: "eth", latestBlock: 120, lastBlock: 90}); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}

	capturePullerBlockProgress(context.Background(), multi, checkpointSource, 100, logger, progress)

	checkpointSnapshot := checkpointSource.Snapshot(context.Background())
	if checkpointSnapshot.LastReorgRiskBlock != 100 {
		t.Fatalf("expected reorg risk block 100, got %d", checkpointSnapshot.LastReorgRiskBlock)
	}
	if got := formatPullerCheckpointChainSummaryAt(time.Now(), 12, checkpointSnapshot); got != "eth=reorg-risk:fresh@100" {
		t.Fatalf("expected reorg-risk checkpoint chain summary, got %q", got)
	}
	if got := formatPullerCheckpointChainPostureSummaryAt(time.Now(), 12, checkpointSnapshot); got != "eth=risk" {
		t.Fatalf("expected checkpoint chain posture summary for risk, got %q", got)
	}
	if got := formatPullerCheckpointCoverageSummary(checkpointSnapshot); got != "tracked=1,recorded=0,reorg_risk=1,reorg_reconciled=0" {
		t.Fatalf("expected reorg-risk checkpoint coverage summary, got %q", got)
	}
	if got := classifyPullerCheckpointCoveragePosture(checkpointSnapshot); got != "coverage-risk" {
		t.Fatalf("expected coverage posture coverage-risk, got %q", got)
	}
}

func TestCapturePullerBlockProgressMarksReorgReconciledAfterNewCheckpoint(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	progress := &pullerLoopRuntimeProgress{}
	checkpointSource := newPullerRuntimeCheckpointSource()
	if err := checkpointSource.SaveCheckpoint(context.Background(), "eth", 100); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	if err := checkpointSource.ObserveChainProgress(context.Background(), "eth", 90); err != nil {
		t.Fatalf("observe chain progress: %v", err)
	}

	multi := pullers.NewMultiChainDataPuller(logger)
	if err := multi.RegisterPuller("eth", &pullersTestPlugin{name: "eth", latestBlock: 220, lastBlock: 200}); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}

	capturePullerBlockProgress(context.Background(), multi, checkpointSource, 100, logger, progress)

	checkpointSnapshot := checkpointSource.Snapshot(context.Background())
	if checkpointSnapshot.LastReconciledBlock != 200 {
		t.Fatalf("expected reconciled checkpoint block 200, got %d", checkpointSnapshot.LastReconciledBlock)
	}
	if got := formatPullerCheckpointChainSummaryAt(time.Now(), 12, checkpointSnapshot); got != "eth=reorg-reconciled:fresh@200" {
		t.Fatalf("expected reconciled checkpoint chain summary, got %q", got)
	}
	if got := formatPullerCheckpointChainPostureSummaryAt(time.Now(), 12, checkpointSnapshot); got != "eth=reconciled" {
		t.Fatalf("expected checkpoint chain posture summary for reconciled, got %q", got)
	}
	if got := formatPullerCheckpointCoverageSummary(checkpointSnapshot); got != "tracked=1,recorded=0,reorg_risk=0,reorg_reconciled=1" {
		t.Fatalf("expected reconciled checkpoint coverage summary, got %q", got)
	}
	if got := classifyPullerCheckpointCoveragePosture(checkpointSnapshot); got != "coverage-reconciled" {
		t.Fatalf("expected coverage posture coverage-reconciled, got %q", got)
	}
}

type pullersTestPlugin struct {
	name        string
	latestBlock uint64
	lastBlock   uint64
}

func (p *pullersTestPlugin) Name() string                        { return p.name }
func (p *pullersTestPlugin) Version() string                     { return "test" }
func (p *pullersTestPlugin) Initialize(config core.Config) error { return nil }
func (p *pullersTestPlugin) Start() error                        { return nil }
func (p *pullersTestPlugin) Stop() error                         { return nil }
func (p *pullersTestPlugin) Health() error                       { return nil }
func (p *pullersTestPlugin) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (p *pullersTestPlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return p.latestBlock, nil
}

func (p *pullersTestPlugin) SubscribeToEvents(ctx context.Context, handler func(blockchain.BlockchainEvent)) error {
	return nil
}
func (p *pullersTestPlugin) ChainID() string            { return p.name }
func (p *pullersTestPlugin) GetStats() map[string]any   { return map[string]any{} }
func (p *pullersTestPlugin) GetLastBlockNumber() uint64 { return p.lastBlock }
