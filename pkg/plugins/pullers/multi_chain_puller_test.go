package pullers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

type multiChainPullerTestPlugin struct {
	name        string
	latestBlock uint64
	lastBlock   uint64
}

func (p *multiChainPullerTestPlugin) Name() string    { return p.name }
func (p *multiChainPullerTestPlugin) Version() string { return "test" }
func (p *multiChainPullerTestPlugin) Initialize(ctx context.Context, config core.Config) error {
	_ = ctx
	return nil
}
func (p *multiChainPullerTestPlugin) Start(ctx context.Context) error  { _ = ctx; return nil }
func (p *multiChainPullerTestPlugin) Stop(ctx context.Context) error   { _ = ctx; return nil }
func (p *multiChainPullerTestPlugin) Health(ctx context.Context) error { _ = ctx; return nil }
func (p *multiChainPullerTestPlugin) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (p *multiChainPullerTestPlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return p.latestBlock, nil
}

func (p *multiChainPullerTestPlugin) SubscribeToEvents(ctx context.Context, handler func(blockchain.BlockchainEvent)) error {
	return nil
}

func (p *multiChainPullerTestPlugin) GetStats() map[string]any {
	return map[string]any{}
}
func (p *multiChainPullerTestPlugin) ChainID() string            { return p.name }
func (p *multiChainPullerTestPlugin) GetLastBlockNumber() uint64 { return p.lastBlock }
func (p *multiChainPullerTestPlugin) SetLastBlockNumber(block uint64) {
	p.lastBlock = block
}

func TestMultiChainDataPullerGetHighestLatestBlock(t *testing.T) {
	t.Parallel()
	puller := NewMultiChainDataPuller(nil)
	if err := puller.RegisterPuller("eth", &multiChainPullerTestPlugin{name: "eth", latestBlock: 120}); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}
	if err := puller.RegisterPuller("polygon", &multiChainPullerTestPlugin{name: "polygon", latestBlock: 150}); err != nil {
		t.Fatalf("register polygon puller: %v", err)
	}

	highest, err := puller.GetHighestLatestBlock(context.Background())
	if err != nil {
		t.Fatalf("get highest latest block: %v", err)
	}
	if highest != 150 {
		t.Fatalf("expected highest latest block 150, got %d", highest)
	}
}

func TestMultiChainDataPullerGetHighestProcessedBlock(t *testing.T) {
	t.Parallel()
	puller := NewMultiChainDataPuller(nil)
	if err := puller.RegisterPuller("eth", &multiChainPullerTestPlugin{name: "eth", lastBlock: 118}); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}
	if err := puller.RegisterPuller("polygon", &multiChainPullerTestPlugin{name: "polygon", lastBlock: 144}); err != nil {
		t.Fatalf("register polygon puller: %v", err)
	}

	highest := puller.GetHighestProcessedBlock()
	if highest != 144 {
		t.Fatalf("expected highest processed block 144, got %d", highest)
	}
}

func TestMultiChainDataPullerRegisteredChainsAndSetLastProcessedBlock(t *testing.T) {
	t.Parallel()
	puller := NewMultiChainDataPuller(nil)
	eth := &multiChainPullerTestPlugin{name: "eth", lastBlock: 118}
	polygon := &multiChainPullerTestPlugin{name: "polygon", lastBlock: 144}

	if err := puller.RegisterPuller("polygon", polygon); err != nil {
		t.Fatalf("register polygon puller: %v", err)
	}
	if err := puller.RegisterPuller("eth", eth); err != nil {
		t.Fatalf("register eth puller: %v", err)
	}

	chains := puller.RegisteredChains()
	if len(chains) != 2 || chains[0] != "eth" || chains[1] != "polygon" {
		t.Fatalf("expected sorted chain IDs [eth polygon], got %#v", chains)
	}

	if err := puller.SetLastProcessedBlock("eth", 125); err != nil {
		t.Fatalf("set last processed block: %v", err)
	}
	if got := eth.lastBlock; got != 125 {
		t.Fatalf("expected eth last block 125, got %d", got)
	}
}

// cancelProbePlugin blocks inside PullEvents until the context is cancelled and
// records concurrency, so the test can prove the fan-out runs a fixed-size pool
// and that every started worker converges after cancel.
type cancelProbePlugin struct {
	multiChainPullerTestPlugin
	started *atomic.Int64
	exited  *atomic.Int64
	active  *atomic.Int64
	peak    *atomic.Int64
}

func (p *cancelProbePlugin) PullEvents(ctx context.Context, _, _ uint64) ([]blockchain.BlockchainEvent, error) {
	p.started.Add(1)
	cur := p.active.Add(1)
	for {
		high := p.peak.Load()
		if cur <= high || p.peak.CompareAndSwap(high, cur) {
			break
		}
	}

	<-ctx.Done()

	p.active.Add(-1)
	p.exited.Add(1)
	return nil, ctx.Err()
}

// TestMultiChainDataPullerCancelConvergesBoundedWorkers covers the T10
// acceptance criterion "取消后 worker 可收敛；队列不超配置": the per-chain fan-out
// must never exceed the configured pool size, and cancelling mid-flight must
// make every started worker exit within a short (t.Deadline-friendly) budget.
func TestMultiChainDataPullerCancelConvergesBoundedWorkers(t *testing.T) {
	t.Parallel()

	const (
		chainCount = 40
		poolSize   = 4
	)

	var started, exited, active, peak atomic.Int64

	puller := NewMultiChainDataPuller(nil)
	puller.SetMaxConcurrency(poolSize)
	for i := 0; i < chainCount; i++ {
		name := fmt.Sprintf("chain-%02d", i)
		probe := &cancelProbePlugin{
			multiChainPullerTestPlugin: multiChainPullerTestPlugin{name: name},
			started:                    &started,
			exited:                     &exited,
			active:                     &active,
			peak:                       &peak,
		}
		if err := puller.RegisterPuller(name, probe); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type outcome struct {
		results map[string][]blockchain.BlockchainEvent
		err     error
	}
	done := make(chan outcome, 1)
	go func() {
		results, err := puller.PullEventsFromAllChains(ctx, 1, 10)
		done <- outcome{results, err}
	}()

	budget := 3 * time.Second
	if dl, ok := t.Deadline(); ok {
		if rem := time.Until(dl) - time.Second; rem > 0 && rem < budget {
			budget = rem
		}
	}
	startWaitDeadline := time.Now().Add(budget)

	// The pool must fill up before cancelling, otherwise the run proves nothing.
	for started.Load() < poolSize {
		if time.Now().After(startWaitDeadline) {
			t.Fatalf("only %d/%d workers started before deadline", started.Load(), poolSize)
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	if got := peak.Load(); got > poolSize {
		t.Fatalf("worker pool exceeded its bound: peak %d > %d", got, poolSize)
	}

	cancel()

	select {
	case out := <-done:
		if out.err == nil {
			t.Fatal("expected an error after cancelling mid-flight")
		}
	case <-time.After(budget):
		t.Fatal("PullEventsFromAllChains did not return after cancel")
	}

	s, e := started.Load(), exited.Load()
	if s == 0 || s != e {
		t.Fatalf("workers did not converge after cancel: started=%d exited=%d", s, e)
	}
	if got := active.Load(); got != 0 {
		t.Fatalf("expected no active workers after return, got %d", got)
	}
}
