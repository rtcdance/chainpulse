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

#### Security-driven module bumps (2026-10-14)
- **Modules**: `golang.org/x/net` v0.58.0 → v0.60.0; `go.opentelemetry.io/otel/sdk` v1.44.0 → v1.45.0 and `.../exporters/otlp/otlptrace` + `.../otlptracegrpc` v1.43.0 → v1.45.0; `go.mongodb.org/mongo-driver` v1.14.0 → v1.17.7; `github.com/jackc/pgx/v5` v5.8.0 → v5.9.2.
- **Why**: each carried a govulncheck advisory reachable from this codebase — GO-2026-6617/6612/6611/6603 (x/net), GO-2026-6505/5327/5004 (otel), GO-2026-5327 (mongo-driver), GO-2026-5004 (pgx). Together with the toolchain move to 1.26.9, these took the scan from **14 callable findings to 0**.
- **Note**: `go.mongodb.org/mongo-driver` is deprecated upstream in favour of `.../v2`. This bump deliberately stays on v1, because a major-version migration is a separate change; v1.17.7 is the fixed v1 release.
- **Note**: these modules were never listed in this file even before the bump — the log only covered go-ethereum, prometheus and zap. Recorded here so the security-relevant set is visible.
- **License**: BSD-3-Clause (x/net), Apache-2.0 (otel, mongo-driver), MIT (pgx)
- **Scope**: dependency version bumps only; no new modules introduced, no API changes required (build, vet, lint and the short suite all pass).
- **Approved by**: mingo
- **Date**: 2026-10-14

#### Frontend test dependencies (2026-10-14)
- **Packages**: `jsdom` ^30.1.2, `@testing-library/react` ^16.3.3, `@testing-library/jest-dom` ^7.0.1 — all **devDependencies** of `frontend/`, none shipped in the bundle.
- **Why**: `frontend/vitest.config.ts` sets `environment: 'jsdom'`, `frontend/src/test-setup.ts` imports `@testing-library/jest-dom`, and the component tests import `render`/`screen` from `@testing-library/react`. **All three were declared in neither `package.json` nor `package-lock.json`**, so `npm ci` installed none of them and `npm run test` could not run: vitest reported `MISSING DEPENDENCY  Cannot find dependency 'jsdom'` and collected **zero tests** while the three test files errored. Nothing ran the frontend tests, so nothing reported it.
- **Alternatives**: switch the environment to `happy-dom` — rejected, because it would be a second, different DOM implementation and the existing tests were written against jsdom's behaviour. Removing the component tests — rejected; they cover `ErrorBoundary` and `Skeleton`, and the non-DOM tests in the same suite pass.
- **License**: MIT (all three)
- **Size**: dev-only; 56 packages added to the install tree, 0 bytes in the built bundle.
- **Scope**: `frontend/package.json` and `frontend/package-lock.json` only.
- **Note**: this is a **gap being closed, not an upgrade**. The tests had never been runnable. After the change, `npm --prefix frontend run test` reports 21 passed / 21 across 3 files, and `npm --prefix frontend run build` still exits 0.
- **Approved by**: mingo
- **Date**: 2026-10-14

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
