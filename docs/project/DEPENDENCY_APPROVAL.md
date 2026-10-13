# Dependency Approval Log

Approved by: mingo (2026-06-07) — in-flight refactor (massive dead code removal across pkg/)

**Purpose**: Track and approve all external dependencies before adding to go.mod

## Approval Process

1. AI proposes dependency with justification
2. Architect reviews: license, maintenance, size, alternatives
3. Architect approves or rejects
4. Only approved dependencies can be added

## Pending Approvals

- [ ] github.com/testcontainers/testcontainers-go v0.35.0
  - **Purpose**: Programmatic Docker container management for integration tests
  - **Why needed**: Current integration tests (test/integration/) hardcode external DB/MQ URLs; without testcontainers they require pre-running docker-compose infrastructure which isn't CI-friendly
  - **Why not stdlib**: Go stdlib has no Docker API bindings
  - **Alternatives**: os/exec + docker CLI (brittle), docker-compose (already exists but requires manual orchestration)
  - **License**: MIT
  - **Size**: ~5MB
  - **Scope**: test-only dependency (not in production binary)
  - **Approved by**: [pending]

<!-- Add new dependencies here -->

## Approved Dependencies

### Standard Library Preference
- **Rule**: Use Go stdlib when sufficient
- **Examples**:
  - ✅ Use `net/http` not `gin` for simple HTTP
  - ✅ Use `encoding/json` not `jsoniter` unless proven bottleneck
  - ✅ Use `database/sql` with driver, not ORM

### Current Approved

### Go Toolchain

#### Go 1.26.9 (bumped from 1.26.6, originally 1.26.0)
- **Purpose**: Build toolchain and standard library
- **Why**: govulncheck reports advisories against the standard library itself, and they can only be fixed by moving the toolchain. 1.26.0 carried 35; moving to 1.26.6 cleared those. A later scan then reported 6 *newer* stdlib advisories (GO-2026-6600, 6603, 6607, 6610, 6612, 6613 — `net/http`, `net/textproto`, `crypto/tls`, `html/template`) whose fix lands in **1.26.9**, so the target moved again. The lesson is that the target version has to be re-derived from the database, not fixed once.
- **Why not stdlib**: not applicable — this *is* the standard library.
- **Alternatives**: stay on an older patch and suppress the advisories — rejected, because it would ship known-vulnerable stdlib code in the binary.
- **License**: BSD-3-Clause (Go)
- **Size**: not applicable (build-time only; no runtime footprint)
- **Scope**: build toolchain. Declared in `go.mod`, `go.work` and the six `pkg/*` submodules (8 declarations total). CI uses `go-version-file: go.mod` or `go-version: '1.26'`; the Dockerfiles use `golang:1.26-alpine` — all of which track the 1.26 line.
- **Security**: clears 35 stdlib advisories (via 1.26.6) and a further 6 (via 1.26.9); stdlib findings now 0
- **Approved by**: mingo
- **Date**: 2026-10-13 (1.26.6), 2026-10-14 (1.26.9)

#### github.com/ethereum/go-ethereum v1.13.0
- **Purpose**: Ethereum client library
- **Why**: Core Web3 functionality (RPC, ABI, crypto)
- **Alternatives**: None (standard Ethereum library)
- **License**: LGPL-3.0
- **Size**: ~50MB
- **Approved by**: [architect-name]
- **Date**: 2026-03-30

#### github.com/prometheus/client_golang v1.18.0
- **Purpose**: Metrics collection
- **Why**: Industry standard for observability
- **Alternatives**: Custom metrics (too much work)
- **License**: Apache-2.0
- **Size**: 5MB
- **Approved by**: [architect-name]
- **Date**: 2026-03-30

#### go.uber.org/zap v1.26.0
- **Purpose**: Structured logging
- **Why**: High performance, structured logs
- **Alternatives**: log/slog (Go 1.21+, consider migration)
- **License**: MIT
- **Size**: 500KB
- **Approved by**: [architect-name]
- **Date**: 2026-03-30

## Rejected Dependencies

### Example: github.com/gin-gonic/gin
- **Reason**: net/http sufficient for our use case
- **Alternative**: Use stdlib net/http with chi router if needed
- **Rejected by**: [architect-name]
- **Date**: 2026-03-30

## Dependency Review Criteria

### Must Answer
1. **Why needed?** What problem does it solve?
2. **Why not stdlib?** What's missing from standard library?
3. **Alternatives?** What other options exist?
4. **Maintenance?** Last commit, open issues, stars
5. **License?** Compatible with project license
6. **Size?** Impact on binary size
7. **Security?** Known vulnerabilities

### Red Flags
- ❌ Last commit >1 year ago
- ❌ Many open critical issues
- ❌ Viral license (GPL without LGPL exception)
- ❌ >10MB for simple functionality
- ❌ Known CVEs without fixes

### Approval Threshold
- **Low risk**: <1MB, MIT/Apache, active maintenance → Auto-approve
- **Medium risk**: 1-10MB, LGPL, moderate activity → Architect review
- **High risk**: >10MB, GPL, inactive, CVEs → Reject or deep review
