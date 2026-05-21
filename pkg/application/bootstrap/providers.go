package bootstrap

import "github.com/rtcdance/chainpulse/pkg/core"

func provideLogger() core.Logger {
	return core.NewSlogLogger(core.LogLevelInfo, "slog")
}

func provideMetrics() core.MetricsCollector {
	return core.NewDefaultMetricsCollector()
}