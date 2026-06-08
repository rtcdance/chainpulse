package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/indexing"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"

	"github.com/ethereum/go-ethereum/common"

	appindexingadapter "github.com/rtcdance/chainpulse/pkg/application/bootstrap"
)

func TestParseNodeURLs(t *testing.T) {
	got, err := parseNodeURLs(" http://localhost:8545, http://localhost:8546 ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 node URLs, got %d", len(got))
	}
	if got[0] != "http://localhost:8545" || got[1] != "http://localhost:8546" {
		t.Fatalf("unexpected node URLs: %#v", got)
	}
}

func TestMonolithicPullerRuntimeRunLoopRestartsAfterFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	puller := &stubMonolithicPollingPuller{
		config: core.Config{ServiceName: "ethereum"},
		poll: func() func(context.Context) error {
			call := 0
			return func(ctx context.Context) error {
				call++
				if call == 1 {
					return errors.New("boom")
				}
				cancel()
				<-ctx.Done()
				return ctx.Err()
			}
		}(),
	}

	runtime := &monolithicPullerRuntime{
		logger:      testhelpers.NewTestLogger(),
		pullers:     []monolithicPollingPuller{puller},
		loopChains:  map[string]*monolithicPullLoopRuntime{"ethereum": {chainID: "ethereum", state: "primed"}},
		backoffBase: time.Millisecond,
		backoffMax:  2 * time.Millisecond,
		startedAt:   time.Now(),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go runtime.runPullerLoop(ctx, &wg, puller)
	wg.Wait()

	status := runtime.PullerStatus()
	if status.LoopRestartTotal != 1 {
		t.Fatalf("expected loop restart total 1, got %d", status.LoopRestartTotal)
	}
	if status.LoopFailureTotal != 1 {
		t.Fatalf("expected loop failure total 1, got %d", status.LoopFailureTotal)
	}
	if status.LastBackoffMS <= 0 {
		t.Fatalf("expected positive last backoff, got %d", status.LastBackoffMS)
	}
}

func TestParseNodeURLsRequiresAtLeastOneValue(t *testing.T) {
	if _, err := parseNodeURLs(" , "); err == nil {
		t.Fatalf("expected error for empty node URLs")
	}
}

func TestSubscribeMonolithicIndexerRoutesBlockchainEvents(t *testing.T) {
	logger := testhelpers.NewTestLogger()
	eventBus := core.NewEventBus(logger)
	indexer := indexing.NewMultiChainIndexer(logger, nil)
	chainIndexer := newRecordingChainIndexer("ethereum")
	if err := indexer.RegisterChainIndexer("ethereum", chainIndexer); err != nil {
		t.Fatalf("register chain indexer: %v", err)
	}

	if err := subscribeMonolithicIndexer(context.Background(), eventBus, indexer, logger); err != nil {
		t.Fatalf("subscribe monolithic indexer: %v", err)
	}

	event := blockchain.BlockchainEvent{
		ID:              "evt-1",
		ChainID:         "ethereum",
		BlockNumber:     12,
		BlockHash:       common.HexToHash("0xabc"),
		TransactionHash: common.HexToHash("0xdef"),
		LogIndex:        1,
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000010"),
		EventName:       "Transfer",
		CreatedAt:       time.Unix(1700000000, 0).UTC(),
	}

	if err := eventBus.PublishSync(context.Background(), monolithicEventTopic, event); err != nil {
		t.Fatalf("publish sync: %v", err)
	}

	if chainIndexer.count != 1 {
		t.Fatalf("expected indexed event count 1, got %d", chainIndexer.count)
	}
	if chainIndexer.lastChain != "ethereum" {
		t.Fatalf("expected last chain ethereum, got %s", chainIndexer.lastChain)
	}
	if chainIndexer.lastEvent == nil || chainIndexer.lastEvent.BlockHash != event.BlockHash {
		t.Fatalf("expected routed event with block hash %s", event.BlockHash.Hex())
	}
}

func TestNewMonolithicPullerRuntimeRequiresAlignedChainsAndNodeURLs(t *testing.T) {
	logger := testhelpers.NewTestLogger()
	metrics := core.NewDefaultMetricsCollector()
	indexer := indexing.NewMultiChainIndexer(logger, nil)
	db := appindexingadapter.NewMonolithicMemoryDatabase(logger)
	if err := db.Initialize(context.Background(), core.Config{}); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("start db: %v", err)
	}

	_, err := newMonolithicPullerRuntime(context.Background(), core.Config{}, "http://localhost:8545", []string{"ethereum", "polygon"}, logger, metrics, db, indexer)
	if err == nil {
		t.Fatalf("expected alignment error")
	}
}

func TestMonolithicPullerRuntimeObserveEventDetectsAndHandlesReorg(t *testing.T) {
	logger := testhelpers.NewTestLogger()
	metrics := core.NewDefaultMetricsCollector()
	indexer := indexing.NewMultiChainIndexer(logger, nil)
	db := appindexingadapter.NewMonolithicMemoryDatabase(logger)
	if err := db.Initialize(context.Background(), core.Config{}); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("start db: %v", err)
	}

	runtime, err := newMonolithicPullerRuntime(context.Background(), core.Config{}, "http://localhost:8545", []string{"ethereum"}, logger, metrics, db, indexer)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	oldEvent := blockchain.BlockchainEvent{
		ID:              "evt-old",
		ChainID:         "ethereum",
		BlockNumber:     12,
		BlockHash:       common.HexToHash("0xabc"),
		TransactionHash: common.HexToHash("0xdef"),
		LogIndex:        1,
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000010"),
		EventName:       "Transfer",
		CreatedAt:       time.Unix(1700000000, 0).UTC(),
	}
	if err := db.StoreEvent(context.Background(), &oldEvent); err != nil {
		t.Fatalf("store old event: %v", err)
	}

	runtime.observeEvent(oldEvent)

	block, err := db.GetBlock(context.Background(), 12)
	if err != nil {
		t.Fatalf("get block after first observation: %v", err)
	}
	if block == nil || block.Hash != oldEvent.BlockHash {
		t.Fatalf("expected stored block hash %s, got %#v", oldEvent.BlockHash.Hex(), block)
	}

	newEvent := oldEvent
	newEvent.ID = "evt-new"
	newEvent.BlockHash = common.HexToHash("0x1234")

	runtime.observeEvent(newEvent)

	removed, err := db.GetEvent(context.Background(), oldEvent.ID)
	if err != nil {
		t.Fatalf("get old event after reorg: %v", err)
	}
	if removed == nil {
		t.Fatalf("expected old event to be marked as reorged, got nil")
	}
	if removed.Status != blockchain.EventStatusReorged {
		t.Fatalf("expected old event status %q, got %q", blockchain.EventStatusReorged, removed.Status)
	}

	block, err = db.GetBlock(context.Background(), 12)
	if err != nil {
		t.Fatalf("get block after reorg observation: %v", err)
	}
	if block == nil || block.Hash != newEvent.BlockHash {
		t.Fatalf("expected updated block hash %s, got %#v", newEvent.BlockHash.Hex(), block)
	}

	status := runtime.ReorgStatus()
	if !status.Wired {
		t.Fatal("expected reorg wiring enabled")
	}
	if status.DetectedTotal != 1 {
		t.Fatalf("expected detected total 1, got %d", status.DetectedTotal)
	}
	if status.HandledTotal != 1 {
		t.Fatalf("expected handled total 1, got %d", status.HandledTotal)
	}
	if status.LastHandledBlock != 12 {
		t.Fatalf("expected last handled block 12, got %d", status.LastHandledBlock)
	}
	if status.Posture != "monolithic-reorg-active" {
		t.Fatalf("expected posture monolithic-reorg-active, got %s", status.Posture)
	}
}

type recordingChainIndexer struct {
	chainID   string
	count     int
	lastChain string
	lastEvent *blockchain.BlockchainEvent
}

type stubMonolithicPollingPuller struct {
	config core.Config
	poll   func(context.Context) error
}

func (s *stubMonolithicPollingPuller) Start(_ context.Context) error { return nil }
func (s *stubMonolithicPollingPuller) Stop(_ context.Context) error  { return nil }
func (s *stubMonolithicPollingPuller) Poll(ctx context.Context) error {
	return s.poll(ctx)
}
func (s *stubMonolithicPollingPuller) GetConfig() core.Config { return s.config }
func (s *stubMonolithicPollingPuller) GetStats() map[string]any {
	return map[string]any{
		"is_running":      true,
		"request_count":   int64(0),
		"error_count":     int64(0),
		"last_error":      nil,
		"last_error_time": time.Time{},
	}
}

func newRecordingChainIndexer(chainID string) *recordingChainIndexer {
	return &recordingChainIndexer{chainID: chainID}
}

func (r *recordingChainIndexer) IndexEvents(_ context.Context, events []*blockchain.BlockchainEvent) error {
	r.count += len(events)
	if len(events) > 0 {
		r.lastChain = events[0].ChainID
		r.lastEvent = events[0]
	}
	return nil
}

func (r *recordingChainIndexer) GetChainID() string {
	return r.chainID
}

func (r *recordingChainIndexer) GetStatus() map[string]any {
	return map[string]any{"chain_id": r.chainID}
}

func (r *recordingChainIndexer) Close() error {
	return nil
}
