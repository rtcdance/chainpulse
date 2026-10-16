package main

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
	"github.com/stretchr/testify/require"
)

type explicitFinalityPuller struct {
	*pullerExecutionTestPlugin
	target, seenFallback uint64
}

func (p *explicitFinalityPuller) ResolveFinalityTarget(_ context.Context, latest, fallback uint64) uint64 {
	p.seenFallback = fallback
	return p.target
}

func TestNativeIntakeUsesExplicitFinalityInsteadOfLegacyCap(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ latest, last, target, fallback uint64 }{{120, 100, 110, 108}, {5, 0, 5, 0}} {
		plugin := &explicitFinalityPuller{pullerExecutionTestPlugin: &pullerExecutionTestPlugin{name: "eth", latestBlock: tc.latest, lastBlock: tc.last}, target: tc.target}
		logger := core.NewDefaultLogger(core.LogLevelInfo)
		metrics := core.NewDefaultMetricsCollector()
		multi := pullers.NewMultiChainDataPuller(logger)
		require.NoError(t, multi.RegisterPuller("eth", plugin))
		runtime := newPullerExecutionRuntime(logger, metrics, &pullerExecutionTestPublisher{}, nil)
		require.NoError(t, runtime.Poll(context.Background(), multi, PullerConfig{BlockConfirmation: 12}))
		require.Equal(t, tc.fallback, plugin.seenFallback)
		require.Equal(t, tc.target, plugin.lastBlock, "explicit latest/tag policy must not be silently overridden by legacy confirmation cap")
	}
}
