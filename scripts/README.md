# Scripts

Automation and verification scripts for ChainPulse. The repository README links
here for the full list.

> This file exists because the README linked to it while it did not exist
> (finding F20: "README 无过时路径"). The index below is generated from the real
> layout — if a script moves, update this file.

## `scripts/acceptance/` — the acceptance toolchain (F21)

Four dependency-free (Python 3 stdlib) gates, each with a built-in `--self-test`.
They implement the tooling the delivery plan requires, so T18 consumes evidence
rather than assembling it by hand.

| Tool | Job | Run |
| --- | --- | --- |
| `verify-tests.py` | Parses `go test -json` JSONL and fails on the ways a suite can "pass" without testing anything: zero matched tests, no test files, all-skipped, a fixture that silently skipped. | `verify-tests.py --jsonl out.jsonl --min-tests N` |
| `collect-evidence.py` | Assembles the §10 evidence pack (15 required artifacts) and **fails when one is missing**; emits `manifest.json` + `checksums.txt`. Never writes a placeholder. | `collect-evidence.py --input build/raw --out build/evidence` |
| `bench-gate.py` | Micro-benchmark regression gate: blocks on >15% in `ns/op` or `B/op`, requires ≥5 samples, and treats "no match / missing baseline / no valid samples" as failures. | `bench-gate.py --bench-output raw.txt --baseline test/benchmarks/micro_baseline.json` |
| `dep-approval-gate.py` | Fails when a dependency manifest changed but `docs/project/DEPENDENCY_APPROVAL.md` did not. | `dep-approval-gate.py --base origin/main --head HEAD` |

Each supports `--self-test`; run it after changing the tool.

## `scripts/` — top level (25 scripts)

Grouped by what they are for:

- **Quality gates**: `check-file-organization.sh`, `check-policy-metric-contract.sh`, `spec-approval-check.sh`, `pre-coding-checklist.sh`, `alert-delivery-check.sh`
- **Deployment**: `deploy-staging.sh`, `deploy-event-emitter.sh`, `run-deploy-event-acceptance.sh`, `run-k8s-deploy.sh`, `production-rollout-check.sh`
- **Verification / rehearsal**: `production-verification-suite.sh`, `run-deployed-real-event-acceptance.sh`, `run-production-readiness-rehearsal.sh`, `deployment-readiness-check.sh`, `rollback-drill.sh`, `smoke-tests.sh`, `soak-check.sh`
- **Local development**: `dev-micro-loop.sh`, `run-local-runnable-app.sh`, `local-unit-tests.sh`, `run-docker-acceptance.sh`, `multi-chain-e2e.sh`, `run-task.sh`
- **Setup**: `install-hooks.sh`

## Subdirectories

| Path | Contents |
| --- | --- |
| `scripts/acceptance/` | The four gates above. |
| `scripts/bench/` | Load-generation helpers (`ingest.sh`, `query_events.js`). |
| `scripts/dev/` | `dev.sh` local development helper. |
| `scripts/lib/` | Shared shell libraries sourced by the scripts above. |
| `scripts/archive/` | Historical scripts kept for reference; not part of any lane. |
