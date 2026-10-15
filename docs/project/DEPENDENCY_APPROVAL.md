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

### Targeted security patches from first remote candidate scan (2026-10-15)

- Classification: security remediation under the Approved delivery plan T15 and the user's execute-to-completion instruction; this records execution authorization, not a newly invented human sign-off.
- Source: GitHub security run 38722855284, Trivy 0.70.0 filesystem report for candidate ad236be706aa8fe611b75c532b8118e438884711. Full report and raw job log are retained in the task evidence.
- Frontend only: pin axios 1.20.0 (from lockfile 1.16.1); its form-data dependency must resolve to >=4.0.6. Addresses the scanner's actionable HIGH advisories for these two packages. Existing frontend request/normalization/finality tests and production build must pass; no audit-fix or unrelated framework upgrades.
- Nested Go module only: align pkg/blockchain's go-ethereum 1.16.8 and x/crypto 0.49.0 with the already-used root workspace versions 1.17.0 and 0.57.0. This removes vulnerable standalone-module pins without introducing a new workspace runtime version. Other modules do not declare these dependencies. x/sys moves to the root-aligned 0.48.0; tidy removes obsolete geth transitive entries. Retain the explicit x/crypto 0.57.0 graph floor: without it geth's standalone module graph falls back to 0.44.0 even though its packages are not called by the current blockchain model. Verify both GOWORK=off nested tests and the actual workspace's signature, decoding, storage and entry-point paths. No toolchain directive.
- Runtime risks: HTTP proxy/adapter behavior and cryptographic/transaction types; targeted tests plus repeat remote govulncheck/Trivy are required before marking verified.
- Rollback: revert this dedicated dependency commit and redeploy the prior immutable digest only as an emergency recovery measure; that restores the old vulnerable dependencies and is not an accepted release state. Keep dependency changes separate from the CI routing fix.


### Delivery verification tool (2026-10-14)

- Tool: `honnef.co/go/tools/cmd/staticcheck@v0.7.0`, built with Go 1.26.9.
- Classification: verification compatibility fix. The installed v0.6.1 binary was built with Go 1.24.13 and rejected the workspace's Go 1.26 directives before analysis.
- Scope: isolated `/tmp/chainpulse-delivery-tools/staticcheck`; no go.mod/go.sum or production binary dependency changes. `STATICCHECK_BIN` selects it for the micro-loop gate.
- Authorization: approved delivery plan T01/T15 and the user's instruction to execute it. This records agent execution authorization, not a separate human dependency review.
- Verification: raw staticcheck/full-gate output is retained under `build/acceptance/20261014-working/`; a gate failure remains a failure.
- Rollback: unset `STATICCHECK_BIN`; the previous installed tool is preserved.

### CI tool and action locks (2026-10-14)

- GitHub actions retain their existing major versions and are locked to full commit SHAs resolved through each upstream repository's GitHub API. The 15 resolutions are retained in `build/acceptance/20261014-working/action-pins.json`.
- Go lint: `github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2`, matching the authoritative Makefile and local Go 1.26-compatible binary. The obsolete v1 installers and v1-only flags were replaced.
- Workflow verifier: `github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`, installed in the isolated verification-tools directory; its upstream release tag was checked via GitHub API. It caught invalid service-container contexts and a duplicate workflow env block before the corrected workflows passed.
- Coverage merge tool: `github.com/wadey/gocovmerge@b5bfa59ec0adc420475f97f89b58045c721d761c`, resolved through upstream API rather than `@latest`.
- Foundry EVM lane: v1.8.5, matching the locally executed Anvil version. Runtime Go versions are derived from go.mod.
- Scope: CI and verification tools; no additional production Go dependencies. Authorization derives from the approved delivery plan T01/T15. Remote execution and same-SHA verification remain required.

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
