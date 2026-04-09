package main

import (
	"net/http"

	pluginapi "chainpulse/pkg/plugins/api"
)

//nolint:unused
type eventProcessorRuntimeControlResponse struct {
	Service   string                            `json:"service"`
	Timestamp int64                             `json:"timestamp"`
	Target    string                            `json:"target"`
	Control   eventProcessorConsumeLoopSnapshot `json:"control"`
}

func (s eventProcessorConsumeLoopSnapshot) runtimeControlCore() pluginapi.RuntimeControlCore {
	return pluginapi.RuntimeControlCore{
		Paused:      s.Paused,
		State:       s.State,
		Reason:      s.Reason,
		LastAction:  s.LastAction,
		UpdatedUnix: s.UpdatedUnix,
	}
}

func buildEventProcessorRuntimeControlGetHandler(controller *eventProcessorConsumeRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondEventProcessorRuntimeControl(w, controller.Snapshot())
	}
}

func buildEventProcessorRuntimeControlPauseHandler(controller *eventProcessorConsumeRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondEventProcessorRuntimeControl(w, controller.PauseIntake("operator-requested intake pause"))
	}
}

func buildEventProcessorRuntimeControlResumeHandler(controller *eventProcessorConsumeRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		respondEventProcessorRuntimeControl(w, controller.ResumeIntake("operator-requested intake resume"))
	}
}

func respondEventProcessorRuntimeControl(w http.ResponseWriter, snapshot eventProcessorConsumeLoopSnapshot) {
	_ = pluginapi.WriteRuntimeControlEnvelopeWithTarget(
		w,
		"event-processor",
		pluginapi.RuntimeControlTargetConsumeLoopIntake,
		snapshot.runtimeControlCore(),
	)
}
