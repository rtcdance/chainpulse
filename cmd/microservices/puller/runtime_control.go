package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"chainpulse/pkg/core"
	pluginapi "chainpulse/pkg/plugins/api"
	"chainpulse/pkg/plugins/pullers"
)

type pullerLoopControlSnapshot struct {
	Paused      bool   `json:"paused"`
	State       string `json:"state"`
	Reason      string `json:"reason"`
	LastAction  string `json:"last_action"`
	UpdatedUnix int64  `json:"updated_unix"`
}

type pullerLoopController struct {
	mu         sync.RWMutex
	paused     bool
	reason     string
	lastAction string
	updatedAt  time.Time
}

func newPullerLoopController() *pullerLoopController {
	return &pullerLoopController{}
}

func (c *pullerLoopController) Pause(reason string) pullerLoopControlSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.paused = true
	c.reason = reason
	c.lastAction = "pause"
	c.updatedAt = time.Now()
	return c.snapshotLocked()
}

func (c *pullerLoopController) Resume(reason string) pullerLoopControlSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.paused = false
	c.reason = reason
	c.lastAction = "resume"
	c.updatedAt = time.Now()
	return c.snapshotLocked()
}

func (c *pullerLoopController) Snapshot() pullerLoopControlSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshotLocked()
}

func (c *pullerLoopController) snapshotLocked() pullerLoopControlSnapshot {
	state := "running"
	if c.paused {
		state = "paused"
	}
	updatedUnix := int64(0)
	if !c.updatedAt.IsZero() {
		updatedUnix = c.updatedAt.Unix()
	}
	return pullerLoopControlSnapshot{
		Paused:      c.paused,
		State:       state,
		Reason:      c.reason,
		LastAction:  c.lastAction,
		UpdatedUnix: updatedUnix,
	}
}

type pullerRuntimeControlResponse struct {
	Service   string                    `json:"service"`
	Timestamp int64                     `json:"timestamp"`
	Target    string                    `json:"target"`
	Control   pullerLoopControlSnapshot `json:"control"`
}

func (s pullerLoopControlSnapshot) runtimeControlCore() pluginapi.RuntimeControlCore {
	return pluginapi.RuntimeControlCore{
		Paused:      s.Paused,
		State:       s.State,
		Reason:      s.Reason,
		LastAction:  s.LastAction,
		UpdatedUnix: s.UpdatedUnix,
	}
}

func buildPullerRuntimeControlGetHandler(controller *pullerLoopController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondPullerRuntimeControl(w, controller.Snapshot())
	}
}

func buildPullerRuntimeControlPauseHandler(controller *pullerLoopController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondPullerRuntimeControl(w, controller.Pause("operator-requested pause"))
	}
}

func buildPullerRuntimeControlResumeHandler(controller *pullerLoopController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondPullerRuntimeControl(w, controller.Resume("operator-requested resume"))
	}
}

func respondPullerRuntimeControl(w http.ResponseWriter, snapshot pullerLoopControlSnapshot) {
	_ = pluginapi.WriteRuntimeControlEnvelopeWithTarget(
		w,
		"puller",
		pluginapi.RuntimeControlTargetPollingLoop,
		snapshot.runtimeControlCore(),
	)
}

func executePullerPollTick(
	ctx context.Context,
	puller *pullers.MultiChainDataPuller,
	config PullerConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
	checkpointSource pullerCheckpointSource,
	progress *pullerLoopRuntimeProgress,
	controller *pullerLoopController,
	execution *pullerExecutionRuntime,
) bool {
	if controller != nil && controller.Snapshot().Paused {
		if metrics != nil {
			metrics.RecordCounter("puller_poll_skips", 1, map[string]string{"instance": config.InstanceID, "reason": "paused"})
		}
		logger.Debug("Polling paused", "instance_id", config.InstanceID)
		return false
	}

	if progress != nil {
		progress.recordPoll(time.Now())
	}
	if execution != nil {
		if err := execution.Poll(ctx, puller, config); err != nil {
			logger.Warn("puller execution poll failed", "instance_id", config.InstanceID, "error", err.Error())
			if metrics != nil {
				metrics.RecordCounter("puller_poll_errors", 1, map[string]string{"instance": config.InstanceID})
			}
		}
	}
	capturePullerBlockProgress(ctx, puller, checkpointSource, config.CheckpointInterval, logger, progress)
	logger.Debug("Polling for new blocks", "instance_id", config.InstanceID)
	if metrics != nil {
		metrics.RecordCounter("puller_polls", 1, map[string]string{"instance": config.InstanceID})
	}
	return true
}
