# Skill: state-checkpoint-recovery

## Trigger

Use this skill when modifying offsets, checkpoints, sync state, replay windows, or startup recovery paths.

## Must Do

1. Define checkpoint ownership and persistence strategy.
2. Ensure restart behavior is deterministic:
   - where to resume
   - what to replay
   - how to deduplicate
3. Define recovery objectives:
   - target RPO (data loss window)
   - target RTO (recovery time)
4. Add failure-path tests:
   - crash during write
   - partial commit
   - stale checkpoint
5. Document operational recovery runbook steps.

## Must Not

- No opaque checkpoint updates without durability rules.
- No restart path that can skip or double-process data silently.
- No recovery logic without verification signals.

## Exit Criteria

- Recovery path is testable and documented.
- Checkpoint behavior supports predictable restart and replay.
