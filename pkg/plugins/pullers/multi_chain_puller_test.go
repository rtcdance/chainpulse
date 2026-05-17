package pullers

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type multiChainPullerTestPlugin struct {
	name        string
	latestBlock uint64
	lastBlock   uint64
}

func (p *multiChainPullerTestPlugin) Name() string                        { return p.name }
func (p *multiChainPullerTestPlugin) Version() string                     { return "test" }
func (p *multiChainPullerTestPlugin) Initialize(config core.Config) error { return nil }
func (p *multiChainPullerTestPlugin) Start() error                        { return nil }
func (p *multiChainPullerTestPlugin) Stop() error                         { return nil }
func (p *multiChainPullerTestPlugin) Health() error                       { return nil }
func (p *multiChainPullerTestPlugin) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	return nil, nil
}

func (p *multiChainPullerTestPlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return p.latestBlock, nil
}

func (p *multiChainPullerTestPlugin) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
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
