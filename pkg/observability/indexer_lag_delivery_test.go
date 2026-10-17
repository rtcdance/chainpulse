package observability

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestLagAllowsIngestCaughtUpWhileCommitIsBehind(t *testing.T) {
	t.Parallel()
	r, err := NewLagRecorder(core.NewDefaultMetricsCollector())
	require.NoError(t, err)
	// head=100, observed=100, committed=80, finalized=70.
	require.NoError(t, r.RecordLagSnapshot("ethereum", 0, 20, 30))
}
