package observability

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Label values substituted by the guard.
const (
	// RedactedLabelValue replaces a label whose key is unbounded by nature.
	RedactedLabelValue = "__redacted__"
	// OverflowLabelValue replaces a value that exceeds its key's series budget.
	OverflowLabelValue = "__overflow__"
)

// UnboundedLabelKeys are label keys whose values differ per event or per block.
//
// A tx hash, a contract address or a block height as a metric label creates one
// time series per event, which is how a metrics backend is taken down. The plan
// requires that these never reach the collector, so the guard redacts them
// rather than trusting every call site to remember.
var UnboundedLabelKeys = map[string]struct{}{
	"tx_hash":          {},
	"transaction_hash": {},
	"txhash":           {},
	"address":          {},
	"contract_address": {},
	"from_address":     {},
	"to_address":       {},
	"block_hash":       {},
	"block_number":     {},
	"block_height":     {},
	"event_id":         {},
	"correlation_id":   {},
	"request_id":       {},
	"session_id":       {},
	"client_id":        {},
	"search_query":     {},
	"query":            {},
}

// LabelGuardOptions configures the guard.
type LabelGuardOptions struct {
	// MaxSeriesPerKey bounds how many distinct values one label key may carry.
	// Zero means DefaultMaxSeriesPerKey.
	MaxSeriesPerKey int
	// BannedKeys overrides UnboundedLabelKeys when non-nil (useful for tests and
	// for services that legitimately label by client).
	BannedKeys map[string]struct{}
	// OnRedact is called for each redacted or overflowed label, so the reason is
	// observable instead of silent.
	OnRedact func(key, original, replaced string)
}

// DefaultMaxSeriesPerKey is the per-key series budget.
const DefaultMaxSeriesPerKey = 64

// LabelGuardStats reports what the guard did.
type LabelGuardStats struct {
	Redacted int64
	Overflow int64
}

// MetricLabelGuard wraps a MetricsCollector and makes every label set bounded.
//
// It exists because MetricsCollector accepts an arbitrary map[string]string, so
// nothing stops a call site from labelling a counter with a tx hash. Redacting
// keeps the measurement (the counter still increments) while keeping the series
// count finite.
type MetricLabelGuard struct {
	inner core.MetricsCollector
	opts  LabelGuardOptions

	mu    sync.Mutex
	seen  map[string]map[string]struct{}
	stats LabelGuardStats
}

// MetricLabelGuard implements the collector interface.
var _ core.MetricsCollector = (*MetricLabelGuard)(nil)

// NewMetricLabelGuard wraps inner.
func NewMetricLabelGuard(inner core.MetricsCollector, opts LabelGuardOptions) (*MetricLabelGuard, error) {
	if inner == nil {
		return nil, fmt.Errorf("metrics collector is required")
	}
	if opts.MaxSeriesPerKey <= 0 {
		opts.MaxSeriesPerKey = DefaultMaxSeriesPerKey
	}
	if opts.BannedKeys == nil {
		opts.BannedKeys = UnboundedLabelKeys
	}
	return &MetricLabelGuard{
		inner: inner,
		opts:  opts,
		seen:  make(map[string]map[string]struct{}),
	}, nil
}

// RecordCounter implements core.MetricsCollector.
func (g *MetricLabelGuard) RecordCounter(name string, value int64, tags map[string]string) {
	g.inner.RecordCounter(name, value, g.Sanitize(tags))
}

// RecordGauge implements core.MetricsCollector.
func (g *MetricLabelGuard) RecordGauge(name string, value float64, tags map[string]string) {
	g.inner.RecordGauge(name, value, g.Sanitize(tags))
}

// RecordHistogram implements core.MetricsCollector.
func (g *MetricLabelGuard) RecordHistogram(name string, value float64, tags map[string]string) {
	g.inner.RecordHistogram(name, value, g.Sanitize(tags))
}

// GetMetrics implements core.MetricsCollector.
func (g *MetricLabelGuard) GetMetrics() map[string]any {
	return g.inner.GetMetrics()
}

// Stats reports the guard's redaction counters.
func (g *MetricLabelGuard) Stats() LabelGuardStats {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stats
}

// Sanitize returns a copy of tags in which no label is unbounded.
//
// Two rules, applied per key:
//
//  1. A key listed in BannedKeys has its VALUE replaced with RedactedLabelValue.
//     The key is kept so the series stays distinguishable from one that never
//     carried the label at all.
//  2. Any other key may carry at most MaxSeriesPerKey distinct values across the
//     process lifetime; beyond that the value becomes OverflowLabelValue.
func (g *MetricLabelGuard) Sanitize(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return tags
	}

	out := make(map[string]string, len(tags))
	for key, value := range tags {
		normalized := strings.ToLower(strings.TrimSpace(key))

		if _, banned := g.opts.BannedKeys[normalized]; banned {
			g.record(key, value, RedactedLabelValue, true)
			out[key] = RedactedLabelValue
			continue
		}

		if g.admit(normalized, value) {
			out[key] = value
			continue
		}

		g.record(key, value, OverflowLabelValue, false)
		out[key] = OverflowLabelValue
	}
	return out
}

// admit records a value for a key and reports whether it is within budget.
func (g *MetricLabelGuard) admit(key, value string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	values, ok := g.seen[key]
	if !ok {
		values = make(map[string]struct{})
		g.seen[key] = values
	}
	if _, known := values[value]; known {
		return true
	}
	if len(values) >= g.opts.MaxSeriesPerKey {
		return false
	}
	values[value] = struct{}{}
	return true
}

func (g *MetricLabelGuard) record(key, original, replaced string, redacted bool) {
	g.mu.Lock()
	if redacted {
		g.stats.Redacted++
	} else {
		g.stats.Overflow++
	}
	g.mu.Unlock()

	if g.opts.OnRedact != nil {
		g.opts.OnRedact(key, original, replaced)
	}
}
