package performance

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
)

func TestStress_MemoryMQ_HighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	mq := mq.NewMemoryMQ()
	_ = mq.Initialize(core.Config{})
	_ = mq.Start()
	defer func() { _ = mq.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const numPublishers = 10
	const numSubscribers = 5
	const messagesPerPublisher = 1000

	received := make(chan int, numSubscribers)
	var wg sync.WaitGroup

	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			count := 0
			_ = mq.Subscribe(ctx, "stress-topic", func(msg []byte) {
				count++
			})
			received <- count
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerPublisher; j++ {
				_ = mq.Publish(ctx, "stress-topic", []byte("stress message"))
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalMessages := numPublishers * messagesPerPublisher
	throughput := float64(totalMessages) / duration.Seconds()

	t.Logf("Stress test completed:")
	t.Logf("  Publishers: %d", numPublishers)
	t.Logf("  Messages: %d", totalMessages)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f msg/sec", throughput)
}

func TestStress_MultiChain_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const numChains = 5
	const eventsPerChain = 500

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	for chain := 0; chain < numChains; chain++ {
		wg.Add(1)
		go func(chainID int) {
			defer wg.Done()

			mq := mq.NewMemoryMQ()
			_ = mq.Initialize(core.Config{})
			_ = mq.Start()
			defer func() { _ = mq.Stop() }()

			for event := 0; event < eventsPerChain; event++ {
				_ = mq.Publish(ctx, "chain-events", []byte("event"))
			}
		}(chain)
	}

	wg.Wait()
	duration := time.Since(start)

	totalEvents := numChains * eventsPerChain
	throughput := float64(totalEvents) / duration.Seconds()

	t.Logf("Multi-chain stress test:")
	t.Logf("  Chains: %d", numChains)
	t.Logf("  Events: %d", totalEvents)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f events/sec", throughput)

	if throughput < 100 {
		t.Errorf("Throughput too low: %.2f events/sec", throughput)
	}
}
