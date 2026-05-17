package main

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

func TestPullerLoopControllerPauseResume(t *testing.T) {
	controller := newPullerLoopController()

	if got := controller.Snapshot().State; got != "running" {
		t.Fatalf("expected initial running state, got %q", got)
	}

	paused := controller.Pause("operator-requested pause")
	if !paused.Paused || paused.State != "paused" {
		t.Fatalf("expected paused snapshot, got %#v", paused)
	}

	resumed := controller.Resume("operator-requested resume")
	if resumed.Paused || resumed.State != "running" {
		t.Fatalf("expected running snapshot after resume, got %#v", resumed)
	}
}

func TestExecutePullerPollTickSkipsWhenPaused(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	progress := &pullerLoopRuntimeProgress{}
	checkpointSource := newPullerRuntimeCheckpointSource()
	multi := pullers.NewMultiChainDataPuller(logger)
	controller := newPullerLoopController()
	controller.Pause("operator-requested pause")

	ok := executePullerPollTick(
		context.Background(),
		multi,
		PullerConfig{InstanceID: "puller-1", CheckpointInterval: 100},
		logger,
		metrics,
		checkpointSource,
		progress,
		controller,
		nil,
	)

	if ok {
		t.Fatal("expected paused control to skip poll tick")
	}
	if snapshot := progress.snapshot(); snapshot.PollCount != 0 {
		t.Fatalf("expected no poll count while paused, got %d", snapshot.PollCount)
	}
	if got := metrics.GetCounter("puller_poll_skips", map[string]string{"instance": "puller-1", "reason": "paused"}); got != 1 {
		t.Fatalf("expected puller_poll_skips counter 1, got %d", got)
	}
}
