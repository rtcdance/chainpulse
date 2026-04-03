package api

import "testing"

func TestBuildRouteOwnershipParityHint(t *testing.T) {
	tests := []struct {
		name                  string
		service               string
		runtimeSignalsPresent bool
		want                  string
	}{
		{
			name:                  "absent",
			service:               "api-service",
			runtimeSignalsPresent: false,
			want:                  "api-service ownership runtime parity with monolith is not yet wired",
		},
		{
			name:                  "present",
			service:               "api-gateway",
			runtimeSignalsPresent: true,
			want:                  "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildRouteOwnershipParityHint(tt.service, tt.runtimeSignalsPresent); got != tt.want {
				t.Fatalf("expected hint %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildOwnershipParityReviewFields(t *testing.T) {
	got := BuildOwnershipParityReviewFields("runtime_routes_enabled", "event_query_enabled")
	want := "runtime_routes_enabled,event_query_enabled,ownership_runtime_parity"
	if got != want {
		t.Fatalf("expected review fields %q, got %q", want, got)
	}
}

func TestBuildOwnershipParityReviewFieldsDeduplicates(t *testing.T) {
	got := BuildOwnershipParityReviewFields("runtime_routes_enabled", OwnershipRuntimeParityReviewField, "runtime_routes_enabled")
	want := "runtime_routes_enabled,ownership_runtime_parity"
	if got != want {
		t.Fatalf("expected review fields %q, got %q", want, got)
	}
}

func TestBuildRouteOwnershipParityState(t *testing.T) {
	got := BuildRouteOwnershipParityState("api-service", true, "runtime_routes_enabled", "event_query_enabled")
	if got.Service != "api-service" {
		t.Fatalf("expected service api-service, got %q", got.Service)
	}
	if !got.RuntimeSignalsPresent {
		t.Fatal("expected runtime signals present")
	}
	if got.Hint != "api-service runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("unexpected hint %q", got.Hint)
	}
	if got.ReviewFields != "runtime_routes_enabled,event_query_enabled,ownership_runtime_parity" {
		t.Fatalf("unexpected review fields %q", got.ReviewFields)
	}
}

func TestBuildRouteOwnershipParityStateFromSource(t *testing.T) {
	got := BuildRouteOwnershipParityStateFromSource("api-gateway", RouteOwnershipParitySourceFunc(func() RouteOwnershipParitySourceSnapshot {
		return RouteOwnershipParitySourceSnapshot{RuntimeSignalsPresent: true}
	}), "runtime_routes_enabled")

	if got.Hint != "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("unexpected hint %q", got.Hint)
	}
	if got.ReviewFields != "runtime_routes_enabled,ownership_runtime_parity" {
		t.Fatalf("unexpected review fields %q", got.ReviewFields)
	}
}

func TestBuildRouteOwnershipParitySourceSnapshotFromReadinessDetailsShadow(t *testing.T) {
	got := BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(map[string]interface{}{
		"ownership_mode":                  "shadow",
		"rollout_ready_for_runtime_owned": false,
		"rollout_status":                  "shadow-observe",
		"rollout_reason":                  "shared runtime still coexists with legacy writes",
	})

	if !got.RuntimeSignalsPresent {
		t.Fatal("expected runtime signals present for shadow ownership mode")
	}
	if got.MonolithOwnershipMode != "shadow" {
		t.Fatalf("expected ownership mode shadow, got %q", got.MonolithOwnershipMode)
	}
	if got.MonolithRolloutReady {
		t.Fatal("expected rollout ready false")
	}
	if got.MonolithRolloutStatus != "shadow-observe" {
		t.Fatalf("expected rollout status shadow-observe, got %q", got.MonolithRolloutStatus)
	}
	if got.MonolithRolloutReason != "shared runtime still coexists with legacy writes" {
		t.Fatalf("unexpected rollout reason %q", got.MonolithRolloutReason)
	}
	if got.MonolithParityPosture != "monolith-shadow-observe" {
		t.Fatalf("expected parity posture monolith-shadow-observe, got %q", got.MonolithParityPosture)
	}
	if got.MonolithParityHint != "monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet" {
		t.Fatalf("unexpected parity hint %q", got.MonolithParityHint)
	}
	if got.MonolithTargetDecision != "target-shadow" {
		t.Fatalf("expected target decision target-shadow, got %q", got.MonolithTargetDecision)
	}
	if got.MonolithActionGuidance != "keep route parity in observe mode until the monolith exits shadow posture" {
		t.Fatalf("unexpected action guidance %q", got.MonolithActionGuidance)
	}
	if got.MonolithTargetReady {
		t.Fatal("expected target ready false")
	}
}

func TestBuildRouteOwnershipParitySourceSnapshotFromReadinessDetailsRuntimeOwned(t *testing.T) {
	got := BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(map[string]interface{}{
		"ownership_mode":                  "runtime-owned",
		"rollout_ready_for_runtime_owned": true,
		"rollout_status":                  "ready",
		"rollout_reason":                  "shared runtime owns observed writes",
	})

	if !got.RuntimeSignalsPresent {
		t.Fatal("expected runtime signals present for runtime-owned mode")
	}
	if !got.MonolithRolloutReady {
		t.Fatal("expected rollout ready true")
	}
	if got.MonolithRolloutStatus != "ready" {
		t.Fatalf("expected rollout status ready, got %q", got.MonolithRolloutStatus)
	}
	if got.MonolithParityPosture != "monolith-runtime-owned-ready" {
		t.Fatalf("expected parity posture monolith-runtime-owned-ready, got %q", got.MonolithParityPosture)
	}
	if got.MonolithParityHint != "monolith ownership rollout is runtime-owned and ready; use this as the target parity posture" {
		t.Fatalf("unexpected parity hint %q", got.MonolithParityHint)
	}
	if got.MonolithTargetDecision != "target-ready" {
		t.Fatalf("expected target decision target-ready, got %q", got.MonolithTargetDecision)
	}
	if got.MonolithActionGuidance != "use the monolith runtime-owned rollout as the current route parity target" {
		t.Fatalf("unexpected action guidance %q", got.MonolithActionGuidance)
	}
	if !got.MonolithTargetReady {
		t.Fatal("expected target ready true")
	}
}

func TestBuildRouteOwnershipParitySourceSnapshotFromReadinessDetailsIdle(t *testing.T) {
	got := BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(map[string]interface{}{
		"ownership_mode":                  "idle",
		"rollout_ready_for_runtime_owned": false,
		"rollout_status":                  "idle",
	})

	if got.RuntimeSignalsPresent {
		t.Fatal("expected runtime signals absent for idle ownership mode")
	}
	if got.MonolithParityPosture != "monolith-idle" {
		t.Fatalf("expected parity posture monolith-idle, got %q", got.MonolithParityPosture)
	}
	if got.MonolithParityHint != "monolith ownership rollout is idle; there is no active runtime-owned parity target yet" {
		t.Fatalf("unexpected parity hint %q", got.MonolithParityHint)
	}
	if got.MonolithTargetDecision != "target-unavailable" {
		t.Fatalf("expected target decision target-unavailable, got %q", got.MonolithTargetDecision)
	}
	if got.MonolithActionGuidance != "wait for an active monolith runtime-owned target before advancing route parity" {
		t.Fatalf("unexpected action guidance %q", got.MonolithActionGuidance)
	}
}

func TestClassifyMonolithOwnershipParityPosture(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		status       string
		ready        bool
		wantPosture  string
	}{
		{name: "runtime-owned-ready", mode: "runtime-owned", status: "ready", ready: true, wantPosture: "monolith-runtime-owned-ready"},
		{name: "shadow", mode: "shadow", status: "shadow-observe", ready: false, wantPosture: "monolith-shadow-observe"},
		{name: "legacy-only", mode: "legacy-only", status: "legacy-observe", ready: false, wantPosture: "monolith-legacy-only"},
		{name: "idle", mode: "idle", status: "idle", ready: false, wantPosture: "monolith-idle"},
		{name: "unknown", mode: "weird", status: "unknown", ready: false, wantPosture: "monolith-investigate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyMonolithOwnershipParityPosture(tt.mode, tt.status, tt.ready); got != tt.wantPosture {
				t.Fatalf("expected posture %q, got %q", tt.wantPosture, got)
			}
		})
	}
}

func TestBuildMonolithOwnershipParityHint(t *testing.T) {
	tests := []struct {
		name    string
		posture string
		want    string
	}{
		{
			name:    "runtime-owned-ready",
			posture: "monolith-runtime-owned-ready",
			want:    "monolith ownership rollout is runtime-owned and ready; use this as the target parity posture",
		},
		{
			name:    "shadow",
			posture: "monolith-shadow-observe",
			want:    "monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet",
		},
		{
			name:    "legacy-only",
			posture: "monolith-legacy-only",
			want:    "monolith ownership rollout remains legacy-only; investigate before expanding route-oriented ownership parity",
		},
		{
			name:    "idle",
			posture: "monolith-idle",
			want:    "monolith ownership rollout is idle; there is no active runtime-owned parity target yet",
		},
		{
			name:    "fallback",
			posture: "monolith-investigate",
			want:    "investigate monolith ownership rollout posture before advancing route-oriented ownership parity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildMonolithOwnershipParityHint(tt.posture); got != tt.want {
				t.Fatalf("expected hint %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildMonolithOwnershipParityTargetDecision(t *testing.T) {
	tests := []struct {
		name    string
		posture string
		want    string
	}{
		{name: "runtime-owned-ready", posture: "monolith-runtime-owned-ready", want: "target-ready"},
		{name: "shadow", posture: "monolith-shadow-observe", want: "target-shadow"},
		{name: "legacy", posture: "monolith-legacy-only", want: "target-blocked"},
		{name: "idle", posture: "monolith-idle", want: "target-unavailable"},
		{name: "fallback", posture: "monolith-investigate", want: "target-investigate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildMonolithOwnershipParityTargetDecision(tt.posture); got != tt.want {
				t.Fatalf("expected decision %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildMonolithOwnershipParityActionGuidance(t *testing.T) {
	tests := []struct {
		name     string
		decision string
		want     string
	}{
		{name: "ready", decision: "target-ready", want: "use the monolith runtime-owned rollout as the current route parity target"},
		{name: "shadow", decision: "target-shadow", want: "keep route parity in observe mode until the monolith exits shadow posture"},
		{name: "blocked", decision: "target-blocked", want: "hold route-oriented parity expansion until the monolith leaves legacy-only posture"},
		{name: "unavailable", decision: "target-unavailable", want: "wait for an active monolith runtime-owned target before advancing route parity"},
		{name: "fallback", decision: "target-investigate", want: "investigate monolith ownership parity target state before advancing route parity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildMonolithOwnershipParityActionGuidance(tt.decision); got != tt.want {
				t.Fatalf("expected guidance %q, got %q", tt.want, got)
			}
		})
	}
}

func TestAppendMonolithOwnershipParityReason(t *testing.T) {
	parts := AppendMonolithOwnershipParityReason([]string{"enabled: runtime_routes_enabled"}, RouteOwnershipParitySourceSnapshot{
		MonolithParityPosture:  "monolith-shadow-observe",
		MonolithParityHint:     "monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet",
		MonolithTargetDecision: "target-shadow",
		MonolithActionGuidance: "keep route parity in observe mode until the monolith exits shadow posture",
	})

	if len(parts) != 5 {
		t.Fatalf("expected 5 reason parts, got %d", len(parts))
	}
	if parts[1] != "monolith_parity_posture: monolith-shadow-observe" {
		t.Fatalf("unexpected posture part %q", parts[1])
	}
	if parts[2] != "monolith_parity_hint: monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet" {
		t.Fatalf("unexpected hint part %q", parts[2])
	}
	if parts[3] != "monolith_parity_target_decision: target-shadow" {
		t.Fatalf("unexpected target decision part %q", parts[3])
	}
	if parts[4] != "monolith_parity_action_guidance: keep route parity in observe mode until the monolith exits shadow posture" {
		t.Fatalf("unexpected action guidance part %q", parts[4])
	}
}

func TestBuildMonolithOwnershipParityRecommendationBundle(t *testing.T) {
	bundle := BuildMonolithOwnershipParityRecommendationBundle(RouteOwnershipParitySourceSnapshot{
		MonolithParityPosture:  "monolith-runtime-owned-ready",
		MonolithParityHint:     "monolith ownership rollout is runtime-owned and ready; use this as the target parity posture",
		MonolithTargetDecision: "target-ready",
		MonolithActionGuidance: "use the monolith runtime-owned rollout as the current route parity target",
	})

	if bundle.Posture != "monolith-runtime-owned-ready" {
		t.Fatalf("unexpected posture %q", bundle.Posture)
	}
	if bundle.Hint != "monolith ownership rollout is runtime-owned and ready; use this as the target parity posture" {
		t.Fatalf("unexpected hint %q", bundle.Hint)
	}
	if bundle.TargetDecision != "target-ready" {
		t.Fatalf("unexpected target decision %q", bundle.TargetDecision)
	}
	if bundle.ActionGuidance != "use the monolith runtime-owned rollout as the current route parity target" {
		t.Fatalf("unexpected action guidance %q", bundle.ActionGuidance)
	}
}

func TestAppendOwnershipParityHintReason(t *testing.T) {
	parts, hint := AppendOwnershipParityHintReason([]string{"enabled: runtime_routes_enabled"}, "api-service", true)
	if hint != "api-service runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("unexpected hint %q", hint)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 reason parts, got %d", len(parts))
	}
	if parts[1] != "ownership_parity_hint: "+hint {
		t.Fatalf("expected ownership parity reason part, got %q", parts[1])
	}
}

func TestAppendRouteOwnershipParityStateReason(t *testing.T) {
	state := BuildRouteOwnershipParityState("api-gateway", false, "runtime_routes_enabled")
	parts, hint := AppendRouteOwnershipParityStateReason([]string{"missing: domain_bridge_enabled"}, state)
	if hint != state.Hint {
		t.Fatalf("expected hint %q, got %q", state.Hint, hint)
	}
	if parts[1] != "ownership_parity_hint: "+state.Hint {
		t.Fatalf("expected ownership parity reason part, got %q", parts[1])
	}
}

func TestBuildOwnershipParityApprovalWorkItem(t *testing.T) {
	got := BuildOwnershipParityApprovalWorkItem(OwnershipParityApprovalWorkItemInput{
		State:  BuildRouteOwnershipParityState("api-gateway", true, "runtime_routes_enabled", "event_query_enabled"),
		Status: "none",
		Owner:  "none",
	})

	if got.WorkItem.Status != "none" {
		t.Fatalf("expected status none, got %q", got.WorkItem.Status)
	}
	if got.WorkItem.Owner != "none" {
		t.Fatalf("expected owner none, got %q", got.WorkItem.Owner)
	}
	if got.WorkItem.ReviewFields != "runtime_routes_enabled,event_query_enabled,ownership_runtime_parity" {
		t.Fatalf("unexpected review fields %q", got.WorkItem.ReviewFields)
	}
	if got.WorkItem.Reason != "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("unexpected reason %q", got.WorkItem.Reason)
	}
}
