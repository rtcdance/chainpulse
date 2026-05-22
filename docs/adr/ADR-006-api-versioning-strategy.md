# ADR-006: API Versioning Strategy

**Date**: 2026-05-13

### Status

Accepted

### Context

ChainPulse exposes REST, gRPC, and GraphQL APIs. Historically all routes were unversioned (`/events`, `/health`). As the API surface grows and external consumers integrate, unversioned APIs create breaking change risk:

1. Adding a required field to a response breaks existing clients
2. Changing error codes breaks client error handling
3. No mechanism for deprecation or migration windows

Enterprise API consumers (dApps, block explorers, monitoring tools) require stability guarantees.

### Decision

Adopt URL-based versioning for REST, header-based for gRPC, with a unified response envelope (`APIEnvelope`):

#### REST Versioning

```
Current (final decision):   /events, /health, /runtime/summary
Target (post-migration):    /api/v1/events, /api/v1/health, /api/v1/runtime/summary
```

**Actual implementation note (2026-05):** The final decision was to keep unprefixed routes.
No `/api/v1/` migration was performed. All API routes use clean paths without version
prefixes. See `docs/api/API_REFERENCE.md` for the current API surface.

All responses use the standard envelope:

```json
{
  "data": { ... },
  "error": null,
  "meta": { "timestamp": 1778679315, "api_version": "v1" }
}
```

#### Response Compatibility Rules

1. **Never remove fields** from v1 responses — mark as `deprecated` in OpenAPI spec
2. **Add-only**: new fields can be added to v1 responses (clients must ignore unknown fields)
3. **Error codes are stable**: once documented, an error code string cannot change meaning
4. **Status codes**: adhere to HTTP semantics strictly (200=ok, 404=not found, 429=rate limited, 502=upstream error)

#### gRPC Versioning

gRPC uses package-based versioning in proto definitions:

```protobuf
package chainpulse.v1;
service EventService { ... }
```

When breaking changes are needed, create `package chainpulse.v2` and host both. The API gateway routes based on client capabilities.

#### GraphQL Versioning

GraphQL is inherently evolvable (field addition is non-breaking). Deprecate fields using the `@deprecated` directive. Avoid removing fields; mark them as `@deprecated(reason: "use fieldX instead")`.

### Consequences

- **Positive**: API consumers get stability guarantees
- **Positive**: Clear migration path for breaking changes
- **Positive**: Standard envelope simplifies client implementation
- **Negative**: URL changes require client updates (mitigated by keeping old routes during deprecation window)
- **Negative**: Version maintenance adds overhead (mitigated by minimal v1 surface; most changes are additive)

### Migration Plan

```
Phase 1 (current): unprefixed routes — /events, /health, /runtime/summary
Phase 2 (next release): remove unversioned routes, document v1 as stable
Phase 3 (future): v2 when needed, with 12-month deprecation window for v1
```

### Amendments

None.
