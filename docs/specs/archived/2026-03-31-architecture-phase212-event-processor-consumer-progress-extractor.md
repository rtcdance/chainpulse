# Phase 212 - Event Processor Consumer Progress Extractor

## Status
Status: Approved

## Why
- Phase 211 proved that `event-processor` can surface a conservative
  consumer-side progress posture, but the extraction, normalization, and
  classification logic still lived inline inside runtime support.
- The next useful slice is structural: give future offset/lag sources one
  stable landing point without changing the current rollout contract.

## Scope
- Keep rollout semantics unchanged.
- Extract consumer progress snapshot building into a dedicated helper.
- Preserve existing focused coverage and parity behavior.

## Implementation
- Add a dedicated consumer progress snapshot builder that owns:
  - consumer-group status extraction
  - consumer-group metric extraction
  - progress-state classification
- Make runtime support consume that snapshot instead of assembling the fields
  inline.
- Add focused helper coverage for the new snapshot builder.

## Validation
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `event-processor` consumer progress assembly has a dedicated extractor/helper.
- Existing rollout payload shape and semantics remain unchanged.
