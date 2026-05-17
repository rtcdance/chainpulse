package main

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestNewEventProcessorProcessingRuntimeStartsProcessorLifecycle(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	runtime, err := newEventProcessorProcessingRuntime(EventProcessorConfig{
		Port:      8082,
		BatchSize: 100,
		LogLevel:  "info",
	}, logger, metrics)
	if err != nil {
		t.Fatalf("create processor runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Stop(); err != nil {
			t.Fatalf("stop processor runtime: %v", err)
		}
	})

	processorRuntime := runtime.Processor()
	if processorRuntime == nil {
		t.Fatal("expected processor runtime")
	}
	health := processorRuntime.Health()
	if health == nil {
		t.Fatal("expected processor health")
	}
	if health.Status != "healthy" {
		t.Fatalf("expected healthy processor runtime, got %q", health.Status)
	}
	if got := processorRuntime.GetProcessedCount(); got != 0 {
		t.Fatalf("expected processed count 0, got %d", got)
	}

	shadowProvider, ok := runtime.MessageProcessor().(eventProcessorSharedRuntimeShadowProvider)
	if !ok {
		t.Fatal("expected shared runtime shadow provider")
	}
	if got := shadowProvider.SharedRuntimeShadowSnapshot().Enabled; !got {
		t.Fatal("expected shared runtime shadow enabled")
	}
}

func TestEventProcessorProcessingRuntimeStopStopsLifecycle(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	runtime, err := newEventProcessorProcessingRuntime(EventProcessorConfig{
		Port:      8082,
		BatchSize: 50,
		LogLevel:  "info",
	}, logger, metrics)
	if err != nil {
		t.Fatalf("create processor runtime: %v", err)
	}

	if err := runtime.Stop(); err != nil {
		t.Fatalf("stop processor runtime: %v", err)
	}

	health := runtime.Processor().Health()
	if health == nil {
		t.Fatal("expected processor health after stop")
	}
	if health.Status != "unhealthy" {
		t.Fatalf("expected unhealthy processor health after stop, got %q", health.Status)
	}
}
