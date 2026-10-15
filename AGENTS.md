# AGENTS.md

**The single source of truth for agent rules in this repository.**
`CLAUDE.md`, `.github/copilot-instructions.md`, and `.claude/rules.md` are **pointers only** — never add rules there.

Format follows the cross-tool [AGENTS.md](https://agents.md) convention (read natively by Codex, Cursor, GitHub Copilot, Gemini CLI, Zed, Aider, Warp, and others). It **complements** `README.md`: quick-start and architecture prose for humans stay in the README, while agent-operational detail lives here.

**Two authoring rules for this file**
1. Write only rules that are **executable and falsifiable** — no unverifiable slogans.
2. **Never hand-copy information that goes stale.** Package layout: run `ls pkg`. Commands: run `make help`. The previous rule files rotted precisely because they hand-copied a package tree (they ended up wrong in 5 places).

---

## Project overview

ChainPulse is a **multi-chain event indexer** written in Go, supporting both `monolithic` and `microservices` deployment modes.
Stack: go-ethereum, PostgreSQL/MongoDB, Kafka/ZeroMQ, Redis, Consul, OpenTelemetry.
Scale reference: ~92.5k non-test lines in `pkg`+`cmd`; ~245k lines of Go overall (recompute with `wc -l` if it matters).

Entry points live in `cmd/`. Everything else is under `pkg/` — **ask the filesystem, not this document**, for the current package list.

---

## Dev environment setup (do this first, every session)

| # | Requirement | Command | Consequence if skipped (measured) |
| - | --- | --- | --- |
| 1 | **Set GOPROXY** | `export GOPROXY=https://goproxy.cn,direct` | `sum.golang.org` is unreachable here; with `GOPROXY=direct`, a **cold-cache / clean checkout** `go build ./...` fails (82 lines of `Bad Gateway`). ⚠️ With a **warm module cache it can still succeed** (verified: `GOPROXY=direct go build ./...` → exit 0), so this failure is easy to miss locally and only bites in CI / a fresh clone. Set the proxy **before** any `go` command. |
| 2 | **Go version** | `go version` → must be `go1.26.x` | `go.mod`/`go.work` declare `go 1.26.0`; older versions trigger a toolchain download |
| 3 | **Linter version** | `"$(go env GOBIN)/golangci-lint" --version` → must contain `built with go1.26` | Otherwise the linter **refuses to start**: `built with go1.x < targeted 1.26.0`. ⚠️ Homebrew's v2.8.0 (built with go1.25.5) does **not** qualify |
| 4 | **Establish a baseline** | `git status --short` + `go build ./...` | The working tree usually holds dozens of **other people's uncommitted changes**; without a baseline you cannot tell whether you broke something |

---

GitHub-hosted CI uses `GOPROXY=https://proxy.golang.org|https://goproxy.cn|direct`; the pipe-separated fallbacks apply on transport/HTTP failures. Local development keeps the proxy above. Do not disable checksum verification (`GOSUMDB=off`) to hide a network failure.

## Build and test commands

> For the complete target list run **`make help`** or read the `Makefile`. **This file deliberately does not duplicate it.**

| Purpose | Command | Requires |
| --- | --- | --- |
| Build | `make build` | — |
| Zero-dependency smoke run | `make playground` | — (in-memory; best way to eyeball behavior) |
| **Fast pre-push gate** | `make check` | — (equals `vet` + `fmt-check`) |
| Format / check formatting | `make fmt` / `make fmt-check` | ⚠️ Uses **gofumpt**, not gofmt. ⚠️ `make fmt` also runs `go mod tidy`, so it can modify `go.mod` / `go.sum` |
| **Lint** | `make lint` | Needs the linter from §setup. ⚠️ Runs with `--tests=false` **and** `--new-from-rev=HEAD~1`. |
| Unit tests | `make test-unit` | — |
| Short tests (unit + cmd) | `make test-short` | — |
| Integration / E2E | `make test-integration` / `make test-e2e` | Docker (`make docker-up`) |
| Anvil on-chain tests | `make test-anvil` | Foundry (installed at `~/.foundry/bin`) |
| Race / coverage / benchmarks | `make test-race` / `test-coverage` / `test-bench` | — |

---

## Definition of Done — **zero false passes**

Before you claim a change is complete, run these in order and **paste the raw output**. Verbal "looks fine to me" does not count.

```bash
export GOPROXY=https://goproxy.cn,direct
go build ./...            # must exit 0
make check                # vet + gofumpt formatting
make lint                 # must report 0 issues
make test-short           # must be all green
git diff --stat           # must touch only the files you intended
```

**Four anti-false-pass rules**

1. **Compile first, lint second.** A `typecheck` failure makes golangci-lint **abort every linter**, so the "1 issue" it reports is an illusion — it does not mean only one problem remains.
2. **Do not mix lint scopes.** `make lint` passes `--tests=false`; running the linter without that flag yields 40+ more/fewer issues (`--tests=false` reports symbols that are *only referenced by tests* as `unused`). **`make lint` is the authoritative gate** — but see rule 4.
3. **A skip is not a pass.** `go test` exiting 0 includes `t.Skip` — that gate is tautological. When "pre-existing failures" are involved, the criterion must be a concrete skip count.
4. **`make lint` is diff-scoped — "0 issues" does not mean the repo is clean.** It passes `--new-from-rev=HEAD~1`, so it only reports issues in code changed since the previous commit. To see the full backlog, run the unscoped command:
   `"$(go env GOBIN)/golangci-lint" run --tests=false --timeout=10m ./pkg/... ./cmd/...`
   (Note: with `--tests=false`, helpers that live in a non-`_test.go` file but are only *called* by tests are reported as `unused` — that is a scope artifact, not dead code.)

---

## Code style

- **DDD layering**: `pkg/domain` → `pkg/application` → `pkg/plugins` / `pkg/services` / `pkg/infrastructure`.
  ⚠️ **`pkg/adapters` does not exist.** Any mention of `adapters/` in older docs (historical README, some `.codex/skills/` files) is wrong and has been removed from the README.
- **Error handling**: always wrap with context — `fmt.Errorf("...: %w", err)`. Never swallow an error.
- **Context**: `context.Context` is the first parameter; honor cancellation.
- **Minimal code**: implement only what the current requirement needs; no speculative abstraction.
- No global variables. No deep nesting. *(Review-enforced only — `gochecknoglobals` is **not** enabled in `.golangci.yml`, so no tool will catch this.)*

---

## Testing instructions

- Table-driven tests with `t.Parallel()`. New features must ship with tests.
- **Focus a single test**: `go test ./pkg/<pkg>/... -run <TestName> -v` (add `-race` for concurrency code).
- `make test-integration` / `make test-e2e` **skip rather than fail** when Docker is unavailable (`test/integration/container_helper.go`). A green run with silently skipped integration tests proves nothing — see anti-false-pass rule 3.
- The repo contains ~**89 `t.Skip("... pre-existing ...")`** sites. Before touching them, classify them first — **42 are `undefined at HEAD`** (the production function was deleted; you cannot reconstruct intent from nothing). **Never** bulk-delete them just to turn the suite green.
- **Coverage has exactly one authoritative definition: the CI gate** (`.github/workflows/test.yml`, `MIN_COVERAGE = 40`). **Do not introduce "aim for X% coverage" targets** — unverifiable metrics have been explicitly retired from this project's planning docs.

---

## Security considerations

- Never commit secrets or credentials; never log sensitive values; SQL must be parameterized.
- **Fix the known defects when you touch the area** (tracked in `optimization-plan/MASTER_PLAN.md`, items A-4a…A-4f). In particular, **`pkg/siwe`'s SIWE verification does not validate `domain` / `URI` / `chainID`** — domain spoofing currently bypasses it.
- Value/precision: in `pkg/evm`, **signed types (int256/int24) must not be decoded with `new(big.Int).SetBytes(...)`** — that silently turns negative values into huge positives.

---

## Commit and PR guidelines

- **Conventional Commits**: `type(scope): subject`, e.g. `fix(api): reorder HandleVerify checks`, `refactor(core): ...`.
  Types actually used here: `fix` (most common) / `feat` / `chore` / `refactor` / `test` / `docs` / `ci` / `style` (plus a few non-standard `lint:`).
- **PRs** use `.github/PULL_REQUEST_TEMPLATE.md`; the `make build` / `make test-unit` / `make vet` boxes in it must all pass. Dependency changes also require updating `docs/project/DEPENDENCY_APPROVAL.md`.
- **Version/config changes go in their own commit**, never mixed with business code (the tree usually holds other people's in-flight work).

---

## Repo-specific gotchas

- **This is a Go workspace with 7 modules**: the root `go.mod` plus one under each of `pkg/*/` (enumerate with `ls go.mod pkg/*/go.mod`), tied together by `go.work`. Any shared change — the Go version, a common dependency — must be applied to **every affected module**, and `go.work`'s `go` directive must not fall below any module's. A previous Go version bump required editing 8 declarations across the workspace.
- `optimization-plan/` is **gitignored** (its contents are candid about the project's weaknesses). To commit it: `git add -f optimization-plan/MASTER_PLAN.md`.
- Expect **dozens of dirty files** from concurrent work. **Never** "clean up" with `git checkout -- .`, `git reset --hard`, or `git clean -fd` — that destroys uncommitted work that isn't yours.
- **Do not add a `toolchain` directive to `go.mod`** — it previously broke the CI setup cache.
- `.codex/skills/` holds 28 local skills plus `INDEX.md` (machine-local). For a matching domain task, **read `.codex/skills/INDEX.md` first**.
- This project follows a **spec-first gate** (see `design-review-gate` in that index): write `docs/specs/<yyyy-mm-dd>-<topic>.md` (template: `docs/specs/TEMPLATE.md`) before writing code.

---

## Maintaining this file

- Rules change **here and only here**. Adding another per-tool rule file creates drift — that is exactly how three files once stated three different Go versions (1.25 / 1.21 / 1.21).
- Any "state" fact (version numbers, directory layout, coverage targets) must be **verifiable by a single command — or it must not be written down**.
- If this file disagrees with the code: **the code wins.** Fix this file in the same change.
