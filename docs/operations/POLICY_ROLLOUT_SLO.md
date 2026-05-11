# Policy Rollout SLO

**Status**: Active | **Last Updated**: 2026-03-30 | **Deadline**: 2026-08-01

## Scope

This SLO set governs runtime override policy rollout using:
- `core_config_overrides_policy_evaluation_total`
- `core_config_overrides_applied_total`

These metrics are emitted by both startup modes:
- `cmd/monolithic/chainpulse`
- `cmd/microservices/api-service`

## Service Level Indicators (SLI)

1. `policy_blocked_rate`
- Definition: blocked policy evaluations / total policy evaluations.
- Source tags:
  - `policy_blocked=true|false`
  - `policy_enforcement=enforce|audit`

2. `policy_violation_rate`
- Definition: violation evaluations / total policy evaluations.
- Source tags:
  - `policy_violation=true|false`
  - `policy_violation_code`

3. `high_risk_violation_rate`
- Definition: share of violations with high-risk codes:
  - `POLICY_API_TYPE_DENIED`
  - `POLICY_FEATURE_FLAG_DENIED`

## SLO Targets

1. Enforce mode stability
- Target: `policy_blocked_rate <= 0.5%` over 24h rolling window.
- Error budget: `0.5%` daily.

2. Audit mode readiness
- Target: `policy_violation_rate <= 1.0%` for 7 consecutive days before switching to enforce.
- Error budget: `1.0%` per day during rollout.

3. High-risk guardrail
- Target: `high_risk_violation_rate == 0` in production profile.
- Error budget: `0` for production.

## Rollout Gates

1. Stage 1 (`audit`)
- Enable `CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT=audit`.
- Collect 3-7 days of violation distribution.

2. Stage 2 (stabilization)
- Reduce top violation codes to below target.
- Confirm runbook actions are validated.

3. Stage 3 (`enforce`)
- Switch to `CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT=enforce`.
- Monitor blocked-rate and rollback triggers for 24h.

## Alert Thresholds

1. Warning
- `policy_violation_rate > 1%` for 30m (any non-prod profile).

2. Critical
- `policy_blocked_rate > 2%` for 15m (enforce mode).
- Any production high-risk violation observed.

## Owner

indexer-team

## Delivery Status

Planned
