# Skill: deterministic-testing

## Trigger

Use this skill for any new/changed test, flaky test, property test, or CI instability issue.

## Must Do

1. Make tests replayable:
   - fixed random seed for randomized cases
   - stable clock/time injection where time matters
   - deterministic fixture inputs
2. Keep tests isolated:
   - no hidden dependency on local machine state
   - no implicit ordering dependency between tests
3. For failure reproduction:
   - record seed/input and replay path
4. For CI reliability:
   - keep tests deterministic under `-race` and repeated runs
5. Document deterministic assumptions in test setup when non-obvious.

## Must Not

- No unbounded randomness without seed capture.
- No tests depending on wall-clock timing only.
- No flaky tests left unresolved.

## Exit Criteria

- New/updated tests can be reproduced reliably.
- Flaky behavior is eliminated or explicitly blocked from merge.
