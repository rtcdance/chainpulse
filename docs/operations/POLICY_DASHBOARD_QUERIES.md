# Policy Dashboard Query Templates

**Status**: Active | **Last Updated**: 2026-03-30 | **Deadline**: 2026-06-15

## Owner

sre-team

## Delivery Status

Implemented

## Notes

- Metric names are source-of-truth from runtime:
  - `core_config_overrides_policy_evaluation_total`
  - `core_config_overrides_applied_total`
- Tags used in queries:
  - `profile`
  - `policy_enforcement`
  - `policy_violation`
  - `policy_blocked`
  - `policy_violation_code`
  - `policy_preset`
  - `policy_preset_source`

## PromQL Templates

1. Total policy evaluations (5m rate)
```promql
sum(rate(core_config_overrides_policy_evaluation_total[5m]))
```

2. Violation rate (%)
```promql
100 *
sum(rate(core_config_overrides_policy_evaluation_total{policy_violation="true"}[5m]))
/
clamp_min(sum(rate(core_config_overrides_policy_evaluation_total[5m])), 1)
```

3. Blocked rate (%) in enforce mode
```promql
100 *
sum(rate(core_config_overrides_policy_evaluation_total{policy_enforcement="enforce",policy_blocked="true"}[5m]))
/
clamp_min(sum(rate(core_config_overrides_policy_evaluation_total{policy_enforcement="enforce"}[5m])), 1)
```

4. Top violation codes (15m)
```promql
topk(
  5,
  sum by (policy_violation_code) (
    increase(core_config_overrides_policy_evaluation_total{policy_violation="true"}[15m])
  )
)
```

5. Violation rate by profile
```promql
100 *
sum by (profile) (rate(core_config_overrides_policy_evaluation_total{policy_violation="true"}[5m]))
/
clamp_min(sum by (profile) (rate(core_config_overrides_policy_evaluation_total[5m])), 1)
```

6. Enforce vs audit traffic split
```promql
sum by (policy_enforcement) (rate(core_config_overrides_policy_evaluation_total[5m]))
```

## Recommended Panels

1. `Policy Violation Rate (%)` (single stat + sparkline)
2. `Policy Blocked Rate (%)` (single stat + threshold coloring)
3. `Top Violation Codes` (bar chart)
4. `Violation Rate by Profile` (stacked area)
5. `Enforcement Mode Distribution` (pie or bar)

## Alert Query Seeds

1. Warning: violation rate > 1% for 30m
```promql
(
  100 *
  sum(rate(core_config_overrides_policy_evaluation_total{policy_violation="true"}[5m]))
  /
  clamp_min(sum(rate(core_config_overrides_policy_evaluation_total[5m])), 1)
) > 1
```

2. Critical: blocked rate > 2% for 15m in enforce mode
```promql
(
  100 *
  sum(rate(core_config_overrides_policy_evaluation_total{policy_enforcement="enforce",policy_blocked="true"}[5m]))
  /
  clamp_min(sum(rate(core_config_overrides_policy_evaluation_total{policy_enforcement="enforce"}[5m])), 1)
) > 2
```
