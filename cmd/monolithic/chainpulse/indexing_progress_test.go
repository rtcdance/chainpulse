package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type deliveryProgressSource struct{ fail bool }

func (*deliveryProgressSource) Start(context.Context) error { return nil }
func (*deliveryProgressSource) Stop(context.Context) error  { return nil }
func (*deliveryProgressSource) Poll(context.Context) error  { return nil }
func (*deliveryProgressSource) GetConfig() core.Config      { return core.Config{ServiceName: "ethereum"} }
func (*deliveryProgressSource) GetStats() map[string]any    { return nil }
func (*deliveryProgressSource) GetLastBlockNumber() uint64  { return 95 }
func (*deliveryProgressSource) FinalizedHead(context.Context) (uint64, error) {
	return 80, nil
}

func (s *deliveryProgressSource) GetLatestBlock(context.Context) (uint64, error) {
	if s.fail {
		return 0, errors.New("RPC unavailable")
	}
	return 100, nil
}

type deliveryCheckpointReader struct{}

func (deliveryCheckpointReader) ChainCheckpoint(context.Context, string) (core.Checkpoint, error) {
	return core.Checkpoint{BlockNumber: 90}, nil
}

func TestProductionProgressPublishesLagAndFailsReadinessOnRPCFault(t *testing.T) {
	t.Parallel()
	source := &deliveryProgressSource{}
	metrics := core.NewDefaultMetricsCollector()
	r := &monolithicPullerRuntime{pullers: []monolithicPollingPuller{source}}
	require.NoError(t, r.ConfigureProgress(deliveryCheckpointReader{}, metrics, nil))
	r.progress(context.Background())
	require.True(t, r.progressReady())
	require.Contains(t, metrics.ExportPrometheus(), "chainpulse_indexer_lag_blocks_commit")
	require.Contains(t, metrics.ExportPrometheus(), "chainpulse_indexer_lag_blocks_finalized")
	source.fail = true
	r.progress(context.Background())
	require.False(t, r.progressReady(), "an RPC outage must not leave readiness green")
}
