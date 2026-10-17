package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

type progressCheckpointReader interface {
	ChainCheckpoint(context.Context, string) (core.Checkpoint, error)
}

type progressSource interface {
	GetLatestBlock(context.Context) (uint64, error)
	GetLastBlockNumber() uint64
	FinalizedHead(context.Context) (uint64, error)
}

// ConfigureProgress wires measurements to the same durable checkpoint as the
// writer. RPC and DB probes are bounded; unknown finality stays explicit.
func (m *monolithicPullerRuntime) ConfigureProgress(reader progressCheckpointReader, metrics core.MetricsCollector, alerts *observability.AlertManager) error {
	recorder, err := observability.NewLagRecorder(metrics)
	if err != nil {
		return err
	}
	if reader == nil {
		return fmt.Errorf("checkpoint reader is required")
	}
	m.progressOK = make(map[string]bool)
	chainMetrics := make(map[string]*observability.IndexerMetrics)
	healthChecks := make(map[string]*observability.IndexerHealth)
	for _, p := range m.pullers {
		chain := p.GetConfig().ServiceName
		m.progressOK[chain] = false
		chainMetrics[chain] = observability.NewIndexerMetrics()
		healthChecks[chain] = observability.NewIndexerHealth(chainMetrics[chain], 100, 0.05, 0, 5*time.Second)
	}
	m.progress = func(ctx context.Context) {
		for _, p := range m.pullers {
			chain := p.GetConfig().ServiceName
			source, ok := p.(progressSource)
			if !ok {
				continue
			}
			probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			head, headErr := source.GetLatestBlock(probeCtx)
			checkpoint, checkpointErr := reader.ChainCheckpoint(probeCtx, chain)
			observed := source.GetLastBlockNumber()
			finalized, finalityErr := source.FinalizedHead(probeCtx)
			cancel()
			available := headErr == nil && checkpointErr == nil && observed <= head && checkpoint.BlockNumber <= observed
			if runtime := m.reorgChains[chain]; runtime != nil && runtime.snapshot().LastError != "" {
				available = false
			}
			if finalized > checkpoint.BlockNumber {
				finalized = checkpoint.BlockNumber
			}
			if available {
				if err := recorder.RecordHeights(chain, head, observed, checkpoint.BlockNumber, finalized); err != nil {
					available = false
				}
				chainMetrics[chain].RecordIndexingProgress(checkpoint.BlockNumber, head)
				available = available && healthChecks[chain].CheckHealth(ctx).Status != observability.HealthStatusUnhealthy
				if alerts != nil {
					alerts.Evaluate(observability.LagMetricName(observability.LagCommit), float64(observed-checkpoint.BlockNumber))
				}
			}
			value := 0.0
			if available {
				value = 1
			}
			metrics.RecordGauge("chainpulse_indexer_progress_available", value, map[string]string{"chain_id": chain})
			known := 0.0
			if finalityErr == nil {
				known = 1
			}
			metrics.RecordGauge("chainpulse_indexer_finality_known", known, map[string]string{"chain_id": chain})
			m.stateMu.Lock()
			m.progressOK[chain] = available
			m.progressAt = time.Now()
			m.stateMu.Unlock()
		}
	}
	return nil
}

func (m *monolithicPullerRuntime) progressReady() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	if len(m.progressOK) == 0 || time.Since(m.progressAt) > 15*time.Second {
		return false
	}
	for _, ok := range m.progressOK {
		if !ok {
			return false
		}
	}
	return true
}

type runtimeProbeHandler struct {
	ready func() bool
}

func (*runtimeProbeHandler) Initialize(*core.Config) error { return nil }
func (*runtimeProbeHandler) Stop() error                   { return nil }
func (h *runtimeProbeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	if !h.ready() {
		status = http.StatusServiceUnavailable
	}
	api.WriteEnvelope(w, status, map[string]bool{"ready": status == http.StatusOK})
}
