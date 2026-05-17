# ADR-005: Error Code Taxonomy

**Date**: 2026-05-13

### Status

Accepted

### Context

ChainPulse's error handling evolved organically: initial validation errors (`ErrInvalidBlockNumber`), then `SystemError` with classification (transient/permanent/critical), then API-level `APIError` with HTTP status mapping. The result was a working but inconsistent error surface:

1. Some errors are `errors.New` sentinels (validation package)
2. Some errors are `*SystemError` with codes like `NETWORK_ERROR`
3. API responses use `APIError` with separate code strings like `"SERVICE_UNAVAILABLE"`
4. Different layers use different error representations with no clear mapping

For enterprise consumers (API clients, dashboards, alerting), errors must have stable, machine-readable codes that are consistent across all layers.

### Decision

Establish a three-layer error taxonomy with consistent code propagation:

```
Layer 1 — Core (pkg/core/errors.go)
├── SystemError: {Type, Code, Message, Details}
├── 23 error codes: VALIDATION_ERROR, NOT_FOUND, BLOCK_NOT_FOUND, RPC_RATE_LIMITED, ...
├── 25+ sentinel errors with pre-assigned codes
└── ClassifyErrorCode(err) → stable string for metrics

Layer 2 — API (pkg/plugins/api/errors.go)
├── APIError: {Code, Message, Status, Details}
├── MapErrorToAPIError(err) → maps any error to APIError
├── mapSystemError(se) → SystemError.Code → HTTP status + API code
└── All Web3 codes (BLOCK_NOT_FOUND → 404, RPC_RATE_LIMITED → 429, ...)

Layer 3 — Observability (pkg/observability/red_metrics.go)
├── RED metrics tagged by error_code
├── ClassifyErrorCode() output used as metric tag
└── Enables alerting on specific error codes
```

Error code assignment rules:

| Prefix | Range | Examples |
|--------|-------|---------|
| VALIDATION_ | Client input errors | VALIDATION_ERROR, INVALID_PARAMETER |
| NOT_FOUND | Resource missing | NOT_FOUND, BLOCK_NOT_FOUND, EVENT_NOT_FOUND |
| RPC_ | Upstream RPC issues | RPC_ERROR, RPC_RATE_LIMITED |
| INTERNAL_ | Server errors | INTERNAL_ERROR, CONFIG_ERROR |
| SERVICE_ | Dependency failures | SERVICE_UNAVAILABLE |

### Consequences

- **Positive**: API clients can write stable error handling logic using codes
- **Positive**: Metrics and alerting can aggregate by error_code
- **Positive**: New error types follow a documented pattern
- **Negative**: 23 error codes requires developer awareness; undocumented codes may be misused
- **Negative**: Legacy `errors.New` sentinels remain in parallel (for `errors.Is` checks) — cannot remove without breaking callers

### Amendments

None.
