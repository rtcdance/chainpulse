# Skill: concurrency-safety

## Trigger

Use this skill when changing goroutines, channels, locks, worker pools, retries, or shared state.

## Must Do

1. Define goroutine lifecycle and cancellation via `context.Context`.
2. Ensure bounded concurrency:
   - worker pool size
   - queue/backpressure limits
   - retry caps
3. Protect shared mutable state explicitly (mutex/atomic/channel ownership).
4. Verify behavior under load and shutdown:
   - graceful stop
   - no goroutine leaks
5. Run race-focused verification for changed scope.

## Must Not

- No fire-and-forget goroutines without ownership.
- No unbounded channel growth or retry loops.
- No lock patterns that can deadlock under normal failure paths.

## Exit Criteria

- Concurrency model is explicit and bounded.
- Race/leak/deadlock risks are addressed and tested.
