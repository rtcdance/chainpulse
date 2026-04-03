package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RuntimeControlCore struct {
	Paused      bool   `json:"paused"`
	State       string `json:"state"`
	Reason      string `json:"reason"`
	LastAction  string `json:"last_action"`
	UpdatedUnix int64  `json:"updated_unix"`
}

const (
	RuntimeControlTargetPollingLoop       = "polling-loop"
	RuntimeControlTargetConsumeLoopIntake = "consume-loop-intake"
)

type RuntimeControlEnvelope struct {
	Service   string             `json:"service"`
	Timestamp int64              `json:"timestamp"`
	Control   RuntimeControlCore `json:"control"`
}

type RuntimeControlEnvelopeWithTarget struct {
	Service   string             `json:"service"`
	Timestamp int64              `json:"timestamp"`
	Target    string             `json:"target"`
	Control   RuntimeControlCore `json:"control"`
}

func WriteRuntimeControlEnvelope(w http.ResponseWriter, service string, control RuntimeControlCore) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(RuntimeControlEnvelope{
		Service:   service,
		Timestamp: time.Now().Unix(),
		Control:   control,
	})
}

func WriteRuntimeControlEnvelopeWithTarget(w http.ResponseWriter, service, target string, control RuntimeControlCore) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(RuntimeControlEnvelopeWithTarget{
		Service:   service,
		Timestamp: time.Now().Unix(),
		Target:    target,
		Control:   control,
	})
}

func ValidateRuntimeControlCore(control RuntimeControlCore) error {
	switch control.State {
	case "running", "paused", "idle":
	default:
		return fmt.Errorf("unexpected control state %q", control.State)
	}
	if control.LastAction == "" {
		if control.UpdatedUnix != 0 {
			return fmt.Errorf("control updated_unix must be zero when last_action is empty")
		}
		return nil
	}
	if control.UpdatedUnix < 0 {
		return fmt.Errorf("control updated_unix must be non-negative")
	}
	return nil
}

func ValidateRuntimeControlEnvelope(envelope RuntimeControlEnvelope, service string) error {
	if envelope.Service != service {
		return fmt.Errorf("expected service %q, got %q", service, envelope.Service)
	}
	if envelope.Timestamp <= 0 {
		return fmt.Errorf("runtime control timestamp must be positive")
	}
	return ValidateRuntimeControlCore(envelope.Control)
}

func ValidateRuntimeControlEnvelopeWithTarget(envelope RuntimeControlEnvelopeWithTarget, service, target string) error {
	if envelope.Target != target {
		return fmt.Errorf("expected target %q, got %q", target, envelope.Target)
	}
	return ValidateRuntimeControlEnvelope(RuntimeControlEnvelope{
		Service:   envelope.Service,
		Timestamp: envelope.Timestamp,
		Control:   envelope.Control,
	}, service)
}
