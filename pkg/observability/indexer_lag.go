package observability

import (
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// LagPhase names WHICH lag a measurement is.
//
// "Lag" alone is ambiguous, and the plan calls that out: a chain can be caught
// up on ingestion while its commits or its finality trail far behind, and one
// number cannot tell those apart. Each phase is a separate series so an alert
// can say which stage is behind.
type LagPhase string

const (
	// LagIngest: chain head minus the highest block the puller has observed.
	LagIngest LagPhase = "ingest"
	// LagCommit: highest block observed minus the highest block durably
	// committed (the checkpoint). This is the lag that loses data if the
	// process dies.
	LagCommit LagPhase = "commit"
	// LagFinalized: chain head minus the highest FINALIZED block indexed. It is
	// always at least the commit lag, and equals it only when the chain is fully
	// final — which is why reporting one number for both hides reorg exposure.
	LagFinalized LagPhase = "finalized"
)

// AllLagPhases is every phase, in the order they are usually observed.
var AllLagPhases = []LagPhase{LagIngest, LagCommit, LagFinalized}

// LagMetricName is the metric name for one phase.
func LagMetricName(phase LagPhase) string {
	return "chainpulse_indexer_lag_blocks_" + string(phase)
}

// LagRecorder records per-chain, per-phase lag.
type LagRecorder struct {
	metrics core.MetricsCollector
}

// NewLagRecorder creates a recorder.
func NewLagRecorder(metrics core.MetricsCollector) (*LagRecorder, error) {
	if metrics == nil {
		return nil, fmt.Errorf("metrics collector is required")
	}
	return &LagRecorder{metrics: metrics}, nil
}

// RecordLag records one phase's lag for one chain.
//
// chain_id is the only label: it is bounded by the deployment's chain list,
// unlike a block height or a tx hash (see MetricLabelGuard).
func (r *LagRecorder) RecordLag(chainID string, phase LagPhase, blocks uint64) {
	r.metrics.RecordGauge(LagMetricName(phase), float64(blocks), map[string]string{
		"chain_id": chainID,
	})
}

// RecordLagSnapshot records all three phases at once.
//
// It rejects orderings that cannot happen (commit lag above ingest lag, or
// finalized lag below commit lag) rather than recording an impossible reading:
// a violated phase relationship means the counters disagree, and silently
// publishing it would make the dashboard lie.
func (r *LagRecorder) RecordLagSnapshot(chainID string, ingest, commit, finalized uint64) error {
	if chainID == "" {
		return fmt.Errorf("chainID is required")
	}
	if commit > ingest {
		return fmt.Errorf("chain %s: commit lag %d cannot exceed ingest lag %d", chainID, commit, ingest)
	}
	if finalized < commit {
		return fmt.Errorf("chain %s: finalized lag %d cannot be below commit lag %d", chainID, finalized, commit)
	}

	for phase, blocks := range map[LagPhase]uint64{
		LagIngest:    ingest,
		LagCommit:    commit,
		LagFinalized: finalized,
	} {
		r.RecordLag(chainID, phase, blocks)
	}
	return nil
}
