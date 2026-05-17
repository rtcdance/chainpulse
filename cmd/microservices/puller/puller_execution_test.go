package main

import (
	"context"
	"encoding/json"
	"testing"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/pullers"
	"github.com/ethereum/go-ethereum/common"
)

type pullerExecutionTestPublisher struct {
	messages []pullerExecutionPublishedMessage
}

type pullerExecutionPublishedMessage struct {
	topic   string
	payload []byte
}

func (p *pullerExecutionTestPublisher) Publish(ctx context.Context, topic string, message []byte) error {
	p.messages = append(p.messages, pullerExecutionPublishedMessage{
		topic:   topic,
		payload: append([]byte(nil), message...),
	})
	return nil
}

type pullerExecutionTestPlugin struct {
	name        string
	latestBlock uint64
	lastBlock   uint64
	events      []core.BlockchainEvent
}

func (p *pullerExecutionTestPlugin) Name() string                        { return p.name }
func (p *pullerExecutionTestPlugin) Version() string                     { return "test" }
func (p *pullerExecutionTestPlugin) Initialize(config core.Config) error { return nil }
func (p *pullerExecutionTestPlugin) Start() error                        { return nil }
func (p *pullerExecutionTestPlugin) Stop() error                         { return nil }
func (p *pullerExecutionTestPlugin) Health() error                       { return nil }
func (p *pullerExecutionTestPlugin) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	return append([]core.BlockchainEvent(nil), p.events...), nil
}

func (p *pullerExecutionTestPlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return p.latestBlock, nil
}

func (p *pullerExecutionTestPlugin) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	return nil
}

func (p *pullerExecutionTestPlugin) ChainID() string { return p.name }

func (p *pullerExecutionTestPlugin) GetStats() map[string]any {
	return map[string]any{}
}
func (p *pullerExecutionTestPlugin) GetLastBlockNumber() uint64      { return p.lastBlock }
func (p *pullerExecutionTestPlugin) SetLastBlockNumber(block uint64) { p.lastBlock = block }

type pullerExecutionTestStatusProvider struct {
	snapshot pullerExecutionRuntimeSnapshot
}

func (p pullerExecutionTestStatusProvider) ExecutionSnapshot() pullerExecutionRuntimeSnapshot {
	return p.snapshot
}

func TestRegisterConfiguredPullersParsesOverridesAndInfersChains(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	multi := pullers.NewMultiChainDataPuller(logger)

	count, err := registerConfiguredPullers(multi, PullerConfig{
		BlockchainRPCs: []string{
			"eth=http://localhost:8545",
			"http://polygon-rpc:8545",
		},
		MaxRetries: 3,
		LogLevel:   "info",
	}, logger, metrics)
	if err != nil {
		t.Fatalf("register configured pullers: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected 2 configured pullers, got %d", count)
	}
	chains := multi.RegisteredChains()
	if len(chains) != 2 || chains[0] != "eth" || chains[1] != "polygon" {
		t.Fatalf("expected [eth polygon], got %#v", chains)
	}
}

func TestPullerExecutionRuntimePollPublishesAndShadows(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	multi := pullers.NewMultiChainDataPuller(logger)
	plugin := &pullerExecutionTestPlugin{
		name:        "eth",
		latestBlock: 120,
		lastBlock:   100,
		events: []core.BlockchainEvent{
			{
				ID:              "evt-1",
				BlockNumber:     120,
				TransactionHash: common.HexToHash("0x1"),
				LogIndex:        2,
				ChainID:         "eth",
			},
		},
	}
	if err := multi.RegisterPuller("eth", plugin); err != nil {
		t.Fatalf("register puller: %v", err)
	}

	publisher := &pullerExecutionTestPublisher{}
	runtime := newPullerExecutionRuntime(logger, metrics, publisher, []string{"raw-events", "blockchain-events"})
	runtime.SetConfiguredPullers(1)

	if err := runtime.Poll(context.Background(), multi, PullerConfig{
		InstanceID:        "puller-1",
		OutputTopics:      []string{"raw-events", "blockchain-events"},
		BlockConfirmation: 0,
	}); err != nil {
		t.Fatalf("poll execution runtime: %v", err)
	}

	if got := plugin.lastBlock; got != 120 {
		t.Fatalf("expected last processed block 120, got %d", got)
	}
	if len(publisher.messages) != 2 {
		t.Fatalf("expected 2 published messages, got %d", len(publisher.messages))
	}

	var event core.BlockchainEvent
	if err := json.Unmarshal(publisher.messages[0].payload, &event); err != nil {
		t.Fatalf("unmarshal published payload: %v", err)
	}
	if event.ID != "evt-1" {
		t.Fatalf("expected published event evt-1, got %q", event.ID)
	}

	snapshot := runtime.ExecutionSnapshot()
	if !snapshot.Enabled {
		t.Fatal("expected execution runtime snapshot to be enabled")
	}
	if snapshot.ConfiguredPullers != 1 {
		t.Fatalf("expected configured pullers 1, got %d", snapshot.ConfiguredPullers)
	}
	if snapshot.RuntimeCount != 1 {
		t.Fatalf("expected one shared runtime, got %d", snapshot.RuntimeCount)
	}
	if snapshot.ProcessedEvents != 1 {
		t.Fatalf("expected one processed shadow event, got %d", snapshot.ProcessedEvents)
	}
	if snapshot.PublishedEvents != 1 {
		t.Fatalf("expected one published event, got %d", snapshot.PublishedEvents)
	}
	if snapshot.PublishedMessages != 2 {
		t.Fatalf("expected two published messages, got %d", snapshot.PublishedMessages)
	}
	if snapshot.LastCheckpointChain != "eth" {
		t.Fatalf("expected last checkpoint chain eth, got %q", snapshot.LastCheckpointChain)
	}
}

func TestBuildPullerRuntimeRolloutStateIncludesExecutionRuntime(t *testing.T) {
	state := buildPullerRuntimeRolloutState(
		context.Background(),
		&pullerTestDatabaseManager{postgresHealthy: true},
		&pullerTestKafkaHealth{status: "healthy"},
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     []string{"http://ethereum-rpc:8545"},
			PollInterval:       12,
			CheckpointInterval: 100,
		},
		nil,
		nil,
		pullerExecutionTestStatusProvider{
			snapshot: pullerExecutionRuntimeSnapshot{
				Enabled:              true,
				ConfiguredPullers:    2,
				AttachedPullers:      1,
				PublishedEvents:      3,
				PublishedMessages:    6,
				RuntimeCount:         1,
				ProcessedEvents:      3,
				SkippedDuplicates:    1,
				RoutedFailures:       2,
				LastCheckpointChain:  "eth",
				LastCheckpointCursor: "eth:120:2",
				LastCheckpointBlock:  120,
				LastError:            "shadow warning",
			},
		},
	)

	if !state.ExecutionRuntimeEnabled {
		t.Fatal("expected execution runtime to be enabled")
	}
	if state.ConfiguredPullerCount != 2 {
		t.Fatalf("expected configured puller count 2, got %d", state.ConfiguredPullerCount)
	}
	if state.AttachedPullerCount != 1 {
		t.Fatalf("expected attached puller count 1, got %d", state.AttachedPullerCount)
	}

	details := buildPullerRuntimeReadinessDetails(state)
	if got := details["shared_runtime_processed_events"]; got != int64(3) {
		t.Fatalf("expected shared runtime processed events 3, got %#v", got)
	}
	if got := details["published_messages"]; got != int64(6) {
		t.Fatalf("expected published messages 6, got %#v", got)
	}
	if got := details["shared_runtime_last_checkpoint_chain"]; got != "eth" {
		t.Fatalf("expected shared runtime last checkpoint chain eth, got %#v", got)
	}
}
