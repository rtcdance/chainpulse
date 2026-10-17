package bootstrap

import (
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/observability"
	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
)

func provideLogger() portcontract.Logger {
	return core.NewSlogLogger(core.LogLevelInfo, "slog")
}

// provideMetrics returns the runtime metrics collector, wrapped in the label
// guard so an unbounded label (tx hash, address, block height) cannot create one
// time series per event. The guard keeps the measurement and bounds the series
// count; see observability.UnboundedLabelKeys.
func provideMetrics() core.MetricsCollector {
	return observability.Guarded(core.NewDefaultMetricsCollector())
}
