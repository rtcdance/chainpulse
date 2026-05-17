package api

import "strings"

const OwnershipRuntimeParityReviewField = "ownership_runtime_parity"

type RouteOwnershipParityState struct {
	Service               string
	RuntimeSignalsPresent bool
	Hint                  string
	ReviewFields          string
}

type RouteOwnershipParitySourceSnapshot struct {
	RuntimeSignalsPresent  bool
	MonolithOwnershipMode  string
	MonolithRolloutReady   bool
	MonolithRolloutStatus  string
	MonolithRolloutReason  string
	MonolithParityPosture  string
	MonolithParityHint     string
	MonolithTargetReady    bool
	MonolithTargetDecision string
	MonolithActionGuidance string
}

type MonolithOwnershipParityRecommendationBundle struct {
	Posture        string
	Hint           string
	TargetDecision string
	ActionGuidance string
}

type RouteOwnershipParitySource interface {
	SnapshotRouteOwnershipParity() RouteOwnershipParitySourceSnapshot
}

type RouteOwnershipParitySourceFunc func() RouteOwnershipParitySourceSnapshot

func (fn RouteOwnershipParitySourceFunc) SnapshotRouteOwnershipParity() RouteOwnershipParitySourceSnapshot {
	if fn == nil {
		return RouteOwnershipParitySourceSnapshot{}
	}
	return fn()
}

type OwnershipParityApprovalWorkItemInput struct {
	State  RouteOwnershipParityState
	Status string
	Owner  string
}

func BuildRouteOwnershipParityHint(service string, runtimeSignalsPresent bool) string {
	if runtimeSignalsPresent {
		return service + " runtime wiring is present, but ownership runtime parity with monolith is still pending"
	}
	return service + " ownership runtime parity with monolith is not yet wired"
}

func BuildOwnershipParityReviewFields(fields ...string) string {
	parts := make([]string, 0, len(fields)+1)
	seen := make(map[string]struct{}, len(fields)+1)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		parts = append(parts, field)
	}
	if _, ok := seen[OwnershipRuntimeParityReviewField]; !ok {
		parts = append(parts, OwnershipRuntimeParityReviewField)
	}
	return strings.Join(parts, ",")
}

func BuildRouteOwnershipParityState(service string, runtimeSignalsPresent bool, reviewFields ...string) RouteOwnershipParityState {
	return RouteOwnershipParityState{
		Service:               service,
		RuntimeSignalsPresent: runtimeSignalsPresent,
		Hint:                  BuildRouteOwnershipParityHint(service, runtimeSignalsPresent),
		ReviewFields:          BuildOwnershipParityReviewFields(reviewFields...),
	}
}

func BuildRouteOwnershipParityStateFromSource(service string, source RouteOwnershipParitySource, reviewFields ...string) RouteOwnershipParityState {
	snapshot := RouteOwnershipParitySourceSnapshot{}
	if source != nil {
		snapshot = source.SnapshotRouteOwnershipParity()
	}
	return BuildRouteOwnershipParityState(service, snapshot.RuntimeSignalsPresent, reviewFields...)
}

func BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(details map[string]any) RouteOwnershipParitySourceSnapshot {
	mode := ownershipParityStringValue(details["ownership_mode"])
	status := ownershipParityStringValue(details["rollout_status"])
	ready := ownershipParityBoolValue(details["rollout_ready_for_runtime_owned"])
	reason := ownershipParityStringValue(details["rollout_reason"])
	posture := ClassifyMonolithOwnershipParityPosture(mode, status, ready)
	decision := BuildMonolithOwnershipParityTargetDecision(posture)

	return RouteOwnershipParitySourceSnapshot{
		RuntimeSignalsPresent:  mode != "" && mode != "idle",
		MonolithOwnershipMode:  mode,
		MonolithRolloutReady:   ready,
		MonolithRolloutStatus:  status,
		MonolithRolloutReason:  reason,
		MonolithParityPosture:  posture,
		MonolithParityHint:     BuildMonolithOwnershipParityHint(posture),
		MonolithTargetReady:    decision == "target-ready",
		MonolithTargetDecision: decision,
		MonolithActionGuidance: BuildMonolithOwnershipParityActionGuidance(decision),
	}
}

func AppendOwnershipParityHintReason(parts []string, service string, runtimeSignalsPresent bool) ([]string, string) {
	hint := BuildRouteOwnershipParityHint(service, runtimeSignalsPresent)
	if hint == "" {
		return parts, ""
	}
	return append(parts, "ownership_parity_hint: "+hint), hint
}

func AppendRouteOwnershipParityStateReason(parts []string, state RouteOwnershipParityState) ([]string, string) {
	if state.Hint == "" {
		return parts, ""
	}
	return append(parts, "ownership_parity_hint: "+state.Hint), state.Hint
}

func BuildOwnershipParityApprovalWorkItem(input OwnershipParityApprovalWorkItemInput) RolloutReportApprovalWorkItemInput {
	return RolloutReportApprovalWorkItemInput{
		WorkItem: RolloutReportApprovalItem{
			Status:       input.Status,
			Owner:        input.Owner,
			ReviewFields: input.State.ReviewFields,
			Reason:       input.State.Hint,
		},
	}
}

func ClassifyMonolithOwnershipParityPosture(mode, rolloutStatus string, rolloutReady bool) string {
	switch {
	case mode == "runtime-owned" && rolloutReady:
		return "monolith-runtime-owned-ready"
	case mode == "shadow" || rolloutStatus == "shadow-observe":
		return "monolith-shadow-observe"
	case mode == "legacy-only":
		return "monolith-legacy-only"
	case mode == "idle" || mode == "":
		return "monolith-idle"
	default:
		return "monolith-investigate"
	}
}

func BuildMonolithOwnershipParityHint(posture string) string {
	switch posture {
	case "monolith-runtime-owned-ready":
		return "monolith ownership rollout is runtime-owned and ready; use this as the target parity posture"
	case "monolith-shadow-observe":
		return "monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet"
	case "monolith-legacy-only":
		return "monolith ownership rollout remains legacy-only; investigate before expanding route-oriented ownership parity"
	case "monolith-idle":
		return "monolith ownership rollout is idle; there is no active runtime-owned parity target yet"
	default:
		return "investigate monolith ownership rollout posture before advancing route-oriented ownership parity"
	}
}

func BuildMonolithOwnershipParityTargetDecision(posture string) string {
	switch posture {
	case "monolith-runtime-owned-ready":
		return "target-ready"
	case "monolith-shadow-observe":
		return "target-shadow"
	case "monolith-legacy-only":
		return "target-blocked"
	case "monolith-idle":
		return "target-unavailable"
	default:
		return "target-investigate"
	}
}

func BuildMonolithOwnershipParityActionGuidance(decision string) string {
	switch decision {
	case "target-ready":
		return "use the monolith runtime-owned rollout as the current route parity target"
	case "target-shadow":
		return "keep route parity in observe mode until the monolith exits shadow posture"
	case "target-blocked":
		return "hold route-oriented parity expansion until the monolith leaves legacy-only posture"
	case "target-unavailable":
		return "wait for an active monolith runtime-owned target before advancing route parity"
	default:
		return "investigate monolith ownership parity target state before advancing route parity"
	}
}

func BuildMonolithOwnershipParityRecommendationBundle(snapshot RouteOwnershipParitySourceSnapshot) MonolithOwnershipParityRecommendationBundle {
	return MonolithOwnershipParityRecommendationBundle{
		Posture:        snapshot.MonolithParityPosture,
		Hint:           snapshot.MonolithParityHint,
		TargetDecision: snapshot.MonolithTargetDecision,
		ActionGuidance: snapshot.MonolithActionGuidance,
	}
}

func AppendMonolithOwnershipParityReason(parts []string, snapshot RouteOwnershipParitySourceSnapshot) []string {
	bundle := BuildMonolithOwnershipParityRecommendationBundle(snapshot)
	if bundle.Posture != "" {
		parts = append(parts, "monolith_parity_posture: "+bundle.Posture)
	}
	if bundle.Hint != "" {
		parts = append(parts, "monolith_parity_hint: "+bundle.Hint)
	}
	if bundle.TargetDecision != "" {
		parts = append(parts, "monolith_parity_target_decision: "+bundle.TargetDecision)
	}
	if bundle.ActionGuidance != "" {
		parts = append(parts, "monolith_parity_action_guidance: "+bundle.ActionGuidance)
	}
	return parts
}

func ownershipParityStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func ownershipParityBoolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}
