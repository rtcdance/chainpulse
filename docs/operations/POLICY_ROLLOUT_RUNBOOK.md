# Policy Rollout Runbook

**Status**: Active | **Last Updated**: 2026-03-30

## Purpose

Provide a safe, repeatable procedure to roll policy changes from `audit` to `enforce` with explicit rollback triggers.

## Preconditions

1. Runtime emits:
- `core_config_overrides_policy_evaluation_total`
- `core_config_overrides_applied_total`

2. Dashboard panels are configured from:
- `docs/operations/POLICY_DASHBOARD_QUERIES.md`

3. SLO targets are accepted:
- `docs/operations/POLICY_ROLLOUT_SLO.md`

## Rollout Procedure

1. Enable audit mode
```bash
export CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT=audit
```

2. Observe 3-7 days
- Track violation rate and top violation codes.
- Verify no production high-risk codes.

3. Remediate violations
- Fix startup flags/env/CLI overrides generating violations.
- Re-deploy and confirm trend is downward.

4. Enforce canary
- Switch one canary instance:
```bash
export CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT=enforce
```
- Watch blocked-rate and startup failures for 24h.

5. Progressive enforce rollout
- 10% -> 50% -> 100% instances.
- Hold each stage at least 1h with stable metrics.

## Rollback Triggers

Rollback to `audit` immediately if any condition is met:

1. `policy_blocked_rate > 2%` for 15m
2. Repeated startup failures tied to `policy_code`
3. Production high-risk violation appears in enforced deployment

## Rollback Action

1. Set:
```bash
export CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT=audit
```

2. Restart affected workload set.

3. Confirm:
- `policy_blocked=false` dominates within 10m.
- violation codes are visible for triage.

## Incident Triage Checklist

1. Identify top `policy_violation_code`.
2. Confirm `profile` and `policy_preset`.
3. Validate startup args:
- `--core-api-type`
- `--core-api-port`
- `--core-feature-flags`
4. Validate env:
- `CHAINPULSE_CORE_API_TYPE`
- `CHAINPULSE_CORE_API_PORT`
- `CHAINPULSE_CORE_FEATURE_FLAGS`
5. Decide:
- config fix
- preset adjustment
- temporary allowlist (non-production only)

## Post-Incident

1. Record incident summary and root cause.
2. Add missing guardrail test if applicable.
3. Update SLO/query/runbook docs if new failure mode is discovered.
