---
name: "release-rollback-readiness"
description: "Require explicit rollout and rollback steps. Require release-window verification and alert watchpoints. Invoke for changes that can affect runtime stability, data correctness, or service contracts."
---

# Skill: release-rollback-readiness

## Trigger

Use this skill for changes that can affect runtime stability, data correctness, or service contracts.

## Must Do

1. Define release plan:
   - rollout sequence
   - canary/partial rollout if available
2. Define rollback plan with triggers.
3. Provide pre-release and post-release verification checklist.
4. Identify critical alerts and on-call signals for the release window.
5. Document operational runbook links in change notes.

## Must Not

- No production-critical change without rollback path.
- No release with undefined verification gates.

## Exit Criteria

- Release and rollback steps are explicit and testable.
- Operational verification is complete.
