# Phase 214 - Puller Progress Extractor

## Status
Status: Approved

## Why
- `puller` already exposed a stronger `poll_activity_state`, but the progress
  extraction still lived inline inside runtime support.
- After Phase 212 extracted a dedicated consumer-progress helper for
  `event-processor`, the next useful slice is to give `puller` the same kind of
  stable landing point for future progress evolution.

## Scope
- Keep rollout semantics unchanged.
- Extract puller poll progress assembly into a dedicated helper.
- Preserve existing focused coverage and parity behavior.

## Implementation
- Add a dedicated puller poll progress snapshot builder that owns:
  - poll count extraction
  - last poll timestamp extraction
  - poll activity classification
- Make runtime support consume that snapshot instead of assembling the fields
  inline.
- Add focused helper coverage for the new progress snapshot builder.

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `puller` poll progress assembly has a dedicated extractor/helper.
- Existing rollout payload shape and semantics remain unchanged.
