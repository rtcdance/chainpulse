package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestWriteRuntimeControlEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	err := WriteRuntimeControlEnvelope(rec, "puller", RuntimeControlCore{
		Paused:      true,
		State:       "paused",
		Reason:      "operator-requested pause",
		LastAction:  "pause",
		UpdatedUnix: 1712345678,
	})
	if err != nil {
		t.Fatalf("write runtime control envelope: %v", err)
	}

	var payload RuntimeControlEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Service != "puller" {
		t.Fatalf("expected service puller, got %q", payload.Service)
	}
	if !payload.Control.Paused {
		t.Fatal("expected paused control")
	}
	if err := ValidateRuntimeControlEnvelope(payload, "puller"); err != nil {
		t.Fatalf("expected runtime control envelope validation: %v", err)
	}
}

func TestWriteRuntimeControlEnvelopeWithTarget(t *testing.T) {
	rec := httptest.NewRecorder()
	err := WriteRuntimeControlEnvelopeWithTarget(rec, "event-processor", RuntimeControlTargetConsumeLoopIntake, RuntimeControlCore{
		Paused:      false,
		State:       "running",
		Reason:      "operator-requested intake resume",
		LastAction:  "resume-intake",
		UpdatedUnix: 1712345678,
	})
	if err != nil {
		t.Fatalf("write runtime control envelope with target: %v", err)
	}

	var payload RuntimeControlEnvelopeWithTarget
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Target != RuntimeControlTargetConsumeLoopIntake {
		t.Fatalf("expected target %s, got %q", RuntimeControlTargetConsumeLoopIntake, payload.Target)
	}
	if payload.Control.LastAction != "resume-intake" {
		t.Fatalf("expected last action resume-intake, got %q", payload.Control.LastAction)
	}
	if err := ValidateRuntimeControlEnvelopeWithTarget(payload, "event-processor", RuntimeControlTargetConsumeLoopIntake); err != nil {
		t.Fatalf("expected runtime control envelope-with-target validation: %v", err)
	}
}

func TestWriteRuntimeControlEnvelopeWithPollingTarget(t *testing.T) {
	rec := httptest.NewRecorder()
	err := WriteRuntimeControlEnvelopeWithTarget(rec, "puller", RuntimeControlTargetPollingLoop, RuntimeControlCore{
		Paused:      true,
		State:       "paused",
		Reason:      "operator-requested pause",
		LastAction:  "pause",
		UpdatedUnix: 1712345678,
	})
	if err != nil {
		t.Fatalf("write runtime control envelope with polling target: %v", err)
	}

	var payload RuntimeControlEnvelopeWithTarget
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if err := ValidateRuntimeControlEnvelopeWithTarget(payload, "puller", RuntimeControlTargetPollingLoop); err != nil {
		t.Fatalf("expected runtime control envelope-with-target validation: %v", err)
	}
}

func TestValidateRuntimeControlCoreRejectsUnexpectedState(t *testing.T) {
	err := ValidateRuntimeControlCore(RuntimeControlCore{
		State:       "unknown",
		LastAction:  "pause",
		UpdatedUnix: 1712345678,
	})
	if err == nil {
		t.Fatal("expected validation error for unexpected state")
	}
}
