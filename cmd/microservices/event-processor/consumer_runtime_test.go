package main

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

type eventProcessorTestMessageConsumer struct {
	messagesByTopic map[string][]core.MessageQueueMessage
}

func (c *eventProcessorTestMessageConsumer) ConsumeMessages(ctx context.Context, topic string, handler func(core.MessageQueueMessage) error) error {
	for _, message := range c.messagesByTopic[topic] {
		if err := handler(message); err != nil {
			return err
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

type eventProcessorBlockingMessageConsumer struct {
	started chan struct{}
}

func (c *eventProcessorBlockingMessageConsumer) ConsumeMessages(ctx context.Context, topic string, handler func(core.MessageQueueMessage) error) error {
	select {
	case c.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}

type eventProcessorTestMessageProcessor struct {
	processed atomic.Int64
}

func (p *eventProcessorTestMessageProcessor) ProcessEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	p.processed.Add(1)
	return nil
}

func (p *eventProcessorTestMessageProcessor) Health() *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Message: "processor runtime healthy"}
}

func (p *eventProcessorTestMessageProcessor) GetProcessedCount() int64 { return p.processed.Load() }
func (p *eventProcessorTestMessageProcessor) GetFailedCount() int64    { return 0 }
func (p *eventProcessorTestMessageProcessor) GetDuplicateCount() int64 { return 0 }

func TestDecodeEventProcessorQueueMessage(t *testing.T) {
	event := blockchain.BlockchainEvent{
		ID:              "evt-1",
		BlockNumber:     123,
		TransactionHash: common.HexToHash("0x1"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		EventName:       "Transfer",
		Network:         "ethereum",
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	decoded, err := decodeEventProcessorQueueMessage(core.MessageQueueMessage{Payload: payload})
	if err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if decoded.ID != "evt-1" {
		t.Fatalf("expected event id evt-1, got %q", decoded.ID)
	}
}

func TestEventProcessorConsumeRuntimeProcessesMessages(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	processorRuntime := &eventProcessorTestMessageProcessor{}

	event := blockchain.BlockchainEvent{
		ID:              "evt-1",
		BlockNumber:     123,
		TransactionHash: common.HexToHash("0x1"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		EventName:       "Transfer",
		Network:         "ethereum",
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	consumer := &eventProcessorTestMessageConsumer{
		messagesByTopic: map[string][]core.MessageQueueMessage{
			"raw-events": {{
				ID:      "msg-1",
				Topic:   "raw-events",
				Payload: payload,
			}},
		},
	}

	consumeRuntime := newEventProcessorConsumeRuntime(
		logger,
		metrics,
		consumer,
		processorRuntime,
		nil, // publisher
		[]string{"raw-events"},
		[]string{"processed-events"},
		nil, // dlqDB
	)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	consumeRuntime.Start(ctx, &wg)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if processorRuntime.GetProcessedCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	wg.Wait()

	if got := processorRuntime.GetProcessedCount(); got != 1 {
		t.Fatalf("expected processed count 1, got %d", got)
	}
	snapshot := consumeRuntime.Snapshot()
	if got := snapshot.ConfiguredTopics; got != 1 {
		t.Fatalf("expected configured topics 1, got %d", got)
	}
}

func TestEventProcessorConsumeRuntimePauseResumeIntake(t *testing.T) {
	consumeRuntime := newEventProcessorConsumeRuntime(
		core.NewDefaultLogger(core.LogLevelInfo),
		core.NewDefaultMetricsCollector(),
		&eventProcessorBlockingMessageConsumer{started: make(chan struct{}, 4)},
		&eventProcessorTestMessageProcessor{},
		nil, // publisher
		[]string{"raw-events"},
		[]string{"processed-events"},
		nil, // dlqDB
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	consumeRuntime.Start(ctx, &wg)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := consumeRuntime.Snapshot(); snap.ActiveTopics == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	paused := consumeRuntime.PauseIntake("test pause")
	if !paused.Paused {
		t.Fatal("expected paused snapshot")
	}
	if paused.State != "paused" {
		t.Fatalf("expected paused state, got %q", paused.State)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := consumeRuntime.Snapshot(); snap.ActiveTopics == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	resumed := consumeRuntime.ResumeIntake("test resume")
	if resumed.Paused {
		t.Fatal("expected resumed snapshot")
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := consumeRuntime.Snapshot(); snap.ActiveTopics == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	wg.Wait()
}
