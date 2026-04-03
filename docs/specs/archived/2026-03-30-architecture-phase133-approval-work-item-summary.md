# Phase 133 Approval Work Item Summary

## Title
Phase 133 - Emit approval work item summary for monolithic operator handoff

## Type
- architecture
- indexing
- observability

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
platform-team

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase132-operator-handoff-summary.md`

## Context
Phase 132 tells operators whether a handoff is needed, but it still does not
package that handoff as an approval work item with a clear owner and the first
fields to inspect.

## Problem Statement
Without an approval work item summary, the system signals "operator review" but
does not clearly say who should act or which rollout fields should be reviewed
first.

## Scope
- Add a non-blocking approval work item classifier derived from operator
  handoff.
- Emit work item summary through:
  - readiness details
  - runtime metric gauge
  - structured startup/shutdown logs
  - console summary lines
- Include owner and primary review fields in the summary.
- Add focused tests for classification and metric/readiness exposure.

## Non-Goals
- No cutover enforcement.
- No ticket creation or external integrations.
- No microservice integration.

## Selected Approach
- Reuse the operator handoff as the source of truth.
- Normalize to a compact summary with:
  - `status`
  - `owner`
  - `review_fields`
  - `reason`
- Keep the logic local to monolithic rollout aggregation.

## Data / Contract Impact
- Readiness details expand with approval work item metadata.
- Runtime metrics expand with an approval work item status code gauge.
- Monolithic logs and console summary gain additive approval work item fields.

## Observability
- Operators can tell:
  - whether there is an actionable approval work item
  - who should act
  - which rollout fields to inspect first

## Risks
- Low risk; additive signaling only.

## Rollback Plan
- Remove the approval work item classifier and its related metric/log/summary
  outputs.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase133-approval-work-item-summary.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase133-approval-work-item-summary.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as an audit and operator-usability improvement above operator
  handoff, while preserving non-blocking rollout behavior.
- Implemented with readiness, metrics, structured logs, console summary, and
  focused monolithic tests.
