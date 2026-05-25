package pullers

import "github.com/rtcdance/chainpulse/pkg/core"

type noopLogger struct{}

func (l *noopLogger) Debug(msg string, fields ...any) {}
func (l *noopLogger) Info(msg string, fields ...any)  {}
func (l *noopLogger) Warn(msg string, fields ...any)  {}
func (l *noopLogger) Error(msg string, fields ...any) {}
func (l *noopLogger) Fatal(msg string, fields ...any) {}
func (l *noopLogger) WithCorrelationID(id string) core.Logger {
	return l
}

type noopMetrics struct{}

func (m *noopMetrics) RecordCounter(name string, value int64, tags map[string]string)     {}
func (m *noopMetrics) RecordGauge(name string, value float64, tags map[string]string)     {}
func (m *noopMetrics) RecordHistogram(name string, value float64, tags map[string]string) {}
func (m *noopMetrics) GetMetrics() map[string]any {
	return map[string]any{}
}
