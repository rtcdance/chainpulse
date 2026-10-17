package pullers

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func newFinalityTestPuller(chainID string) *HTTPSJSONRPCPuller {
	return NewHTTPSJSONRPCPuller(
		core.Config{ChainID: chainID},
		testhelpers.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)
}

func TestFinalityTargetNilRegistryPassesHead(t *testing.T) {
	t.Parallel()
	puller := newFinalityTestPuller("1")
	if got := puller.finalityTarget(context.Background(), 100); got != 100 {
		t.Fatalf("finalityTarget with nil registry = %d, want 100", got)
	}
}

func TestFinalityTargetConfirmationsCap(t *testing.T) {
	t.Parallel()
	registry, err := finality.NewRegistry(
		finality.Strategy{Mode: finality.StrategyConfirmations, Confirmations: 12},
		map[string]finality.Strategy{
			"polygon": {Mode: finality.StrategyConfirmations, Confirmations: 128},
		},
	)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	cases := []struct {
		name    string
		chainID string
		latest  uint64
		want    uint64
	}{
		{name: "default 12 confirmations", chainID: "1", latest: 100, want: 88},
		{name: "per-chain 128 confirmations", chainID: "polygon", latest: 200, want: 72},
		{name: "saturates at zero", chainID: "1", latest: 5, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			puller := newFinalityTestPuller(tc.chainID)
			puller.SetFinalityRegistry(registry)
			if got := puller.finalityTarget(context.Background(), tc.latest); got != tc.want {
				t.Fatalf("finalityTarget(%d) = %d, want %d", tc.latest, got, tc.want)
			}
		})
	}
}

func TestFinalityTargetLatestModePassesHead(t *testing.T) {
	t.Parallel()
	registry, err := finality.NewRegistry(finality.Strategy{Mode: finality.StrategyLatest}, nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	puller := newFinalityTestPuller("anvil")
	puller.SetFinalityRegistry(registry)
	if got := puller.finalityTarget(context.Background(), 100); got != 100 {
		t.Fatalf("finalityTarget latest mode = %d, want 100", got)
	}
}

func TestFinalityTargetTagFallbackWithoutClient(t *testing.T) {
	t.Parallel()
	registry, err := finality.NewRegistry(finality.Strategy{Mode: finality.StrategyTag, Tag: "safe"}, nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	puller := newFinalityTestPuller("base")
	puller.SetFinalityRegistry(registry)
	if got := puller.finalityTarget(context.Background(), 100); got != 100 {
		t.Fatalf("finalityTarget tag fallback = %d, want head 100", got)
	}
}

func TestSetFinalityRegistryNilDisablesCap(t *testing.T) {
	t.Parallel()
	registry, err := finality.NewRegistry(finality.Strategy{Mode: finality.StrategyConfirmations, Confirmations: 12}, nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	puller := newFinalityTestPuller("1")
	puller.SetFinalityRegistry(registry)
	puller.SetFinalityRegistry(nil)
	if got := puller.finalityTarget(context.Background(), 100); got != 100 {
		t.Fatalf("finalityTarget after nil registry = %d, want 100", got)
	}
}
