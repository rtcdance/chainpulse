package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func newGuard(t *testing.T, opts LabelGuardOptions) *MetricLabelGuard {
	t.Helper()
	g, err := NewMetricLabelGuard(core.NewDefaultMetricsCollector(), opts)
	require.NoError(t, err)
	return g
}

// TestLabelGuardRedactsUnboundedKeys is the T13 acceptance test for label
// cardinality: a tx hash, an address or a block height must never reach the
// collector as a label value.
func TestLabelGuardRedactsUnboundedKeys(t *testing.T) {
	t.Parallel()

	g := newGuard(t, LabelGuardOptions{})

	cases := []map[string]string{
		{"tx_hash": "0xdeadbeef"},
		{"transaction_hash": "0xdeadbeef"},
		{"address": "0xabc"},
		{"contract_address": "0xabc"},
		{"block_hash": "0x1234"},
		{"block_number": "19000000"},
		{"block_height": "19000000"},
		{"event_id": "evt-1"},
		{"correlation_id": "corr-1"},
	}

	for _, tags := range cases {
		out := g.Sanitize(tags)
		for key, value := range out {
			assert.Equal(t, RedactedLabelValue, value,
				"label %q must be redacted, got %q", key, value)
		}
	}

	stats := g.Stats()
	assert.Equal(t, int64(len(cases)), stats.Redacted, "every redaction must be counted")
	assert.Zero(t, stats.Overflow)
}

// TestLabelGuardKeepsBoundedLabels: the guard must not damage the labels that
// are legitimately bounded.
func TestLabelGuardKeepsBoundedLabels(t *testing.T) {
	t.Parallel()

	g := newGuard(t, LabelGuardOptions{})

	tags := map[string]string{
		"chain_id": "ethereum",
		"phase":    "commit",
		"status":   "ok",
	}
	out := g.Sanitize(tags)

	assert.Equal(t, tags, out, "bounded labels must pass through unchanged")
	assert.Zero(t, g.Stats().Redacted)
}

// TestLabelGuardCapsSeriesPerKey covers the second rule: a key that is not
// banned by name still cannot carry unlimited distinct values.
func TestLabelGuardCapsSeriesPerKey(t *testing.T) {
	t.Parallel()

	g := newGuard(t, LabelGuardOptions{MaxSeriesPerKey: 3})

	for _, value := range []string{"a", "b", "c"} {
		out := g.Sanitize(map[string]string{"shard": value})
		assert.Equal(t, value, out["shard"], "the first values are within budget")
	}

	out := g.Sanitize(map[string]string{"shard": "d"})
	assert.Equal(t, OverflowLabelValue, out["shard"], "a value beyond the budget must collapse")

	// A value already seen is still admitted, so the budget does not leak.
	out = g.Sanitize(map[string]string{"shard": "a"})
	assert.Equal(t, "a", out["shard"])

	assert.Equal(t, int64(1), g.Stats().Overflow)
}

func TestLabelGuardReportsRedactions(t *testing.T) {
	t.Parallel()

	var seen []string
	g := newGuard(t, LabelGuardOptions{
		OnRedact: func(key, original, replaced string) {
			seen = append(seen, key+"="+original+"->"+replaced)
		},
	})

	_ = g.Sanitize(map[string]string{"tx_hash": "0xdead"})

	require.Len(t, seen, 1)
	assert.Equal(t, "tx_hash=0xdead->"+RedactedLabelValue, seen[0])
}

func TestLabelGuardRecordsThroughToInner(t *testing.T) {
	t.Parallel()

	inner := core.NewDefaultMetricsCollector()
	g, err := NewMetricLabelGuard(inner, LabelGuardOptions{})
	require.NoError(t, err)

	// The measurement must survive even when its label is redacted.
	g.RecordCounter("events_indexed_total", 1, map[string]string{
		"chain_id": "ethereum",
		"tx_hash":  "0xdeadbeef",
	})

	assert.NotEmpty(t, inner.GetMetrics(), "the counter must still be recorded")
}

func TestLabelGuardRequiresCollector(t *testing.T) {
	t.Parallel()

	_, err := NewMetricLabelGuard(nil, LabelGuardOptions{})
	require.Error(t, err)
}

// --------------------------------------------------------------------------- //
// Lag phases
// --------------------------------------------------------------------------- //

func TestLagPhasesAreDistinctSeries(t *testing.T) {
	t.Parallel()

	names := map[string]bool{}
	for _, phase := range AllLagPhases {
		name := LagMetricName(phase)
		assert.False(t, names[name], "phase %q must have its own series", phase)
		names[name] = true
	}
	assert.Len(t, names, 3, "ingest, commit and finalized must not share a series")

	assert.Equal(t, "chainpulse_indexer_lag_blocks_commit", LagMetricName(LagCommit))
}

func TestLagRecorderRecordsEveryPhase(t *testing.T) {
	t.Parallel()

	inner := core.NewDefaultMetricsCollector()
	rec, err := NewLagRecorder(inner)
	require.NoError(t, err)

	require.NoError(t, rec.RecordLagSnapshot("ethereum", 100, 90, 95))

	// ExportPrometheus renders the series, so the assertion is on what a scraper
	// would actually see rather than on an internal map shape.
	rendered := inner.ExportPrometheus()
	require.NotEmpty(t, rendered)
	for _, phase := range AllLagPhases {
		assert.Contains(t, rendered, LagMetricName(phase), "phase %q must be recorded", phase)
	}
}

// TestLagRecorderRejectsImpossibleOrderings: commit lag cannot exceed ingest
// lag, and finalized lag cannot be below commit lag. Publishing an impossible
// reading would make the dashboard lie.
func TestLagRecorderRejectsImpossibleOrderings(t *testing.T) {
	t.Parallel()

	rec, err := NewLagRecorder(core.NewDefaultMetricsCollector())
	require.NoError(t, err)

	require.Error(t, rec.RecordLagSnapshot("ethereum", 10, 20, 30), "commit > ingest is impossible")
	require.Error(t, rec.RecordLagSnapshot("ethereum", 100, 90, 5), "finalized < commit is impossible")
	require.Error(t, rec.RecordLagSnapshot("", 100, 90, 12), "chain id is required")
}

func TestLagRecorderRequiresCollector(t *testing.T) {
	t.Parallel()

	_, err := NewLagRecorder(nil)
	require.Error(t, err)
}

// TestLagRecorderAndGuardCompose: the two pieces must work together, since the
// lag recorder is meant to be wired behind the guard.
func TestLagRecorderAndGuardCompose(t *testing.T) {
	t.Parallel()

	guard := newGuard(t, LabelGuardOptions{})
	rec, err := NewLagRecorder(guard)
	require.NoError(t, err)

	require.NoError(t, rec.RecordLagSnapshot("ethereum", 100, 90, 95))
	assert.Zero(t, guard.Stats().Redacted, "chain_id is a bounded label and must not be redacted")
}
