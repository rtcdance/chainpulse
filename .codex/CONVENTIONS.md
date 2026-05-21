# ChainPulse AI Coding Conventions

This directory contains project-local skills that enforce consistent architecture, quality gates, and Web3/Go delivery patterns.

## How It Works

1. **Skills** (`.codex/skills/`) — 27 domain-specific engineering guides for reorg handling, contract testing, chaos resilience, etc.
2. **Behavioral Constraints** — Hard rules on AI coding behavior (minimal implementation, no unsolicited improvements, test discipline).
3. **Auto-Activation** — Skills are automatically selected based on file paths and change patterns.

## Quick Reference

### Behavioral Constraints
- **Minimal Implementation Only** — Solve ONLY the stated requirement. Add abstraction when 3rd usage appears.
- **No Unsolicited Improvements** — Change ONLY what's needed for the task at hand.
- **Test Determinism** — All tests must be deterministic. No sleep-based timing assumptions.
- **Dependency Hygiene** — New dependencies require justification in `DEPENDENCY_APPROVAL.md`.
- **Comment Discipline** — Comments explain WHY, not WHAT. No stale comments.

### Mandatory Gates Before Coding
1. **Spec** — Follow approved spec in `docs/specs/` (or create one for non-trivial changes)
2. **Skills** — Declare which skills apply from `.codex/skills/INDEX.md`
3. **Blast Radius** — Assess files changed, layers affected, breaking changes
4. **Tests** — Unit tests for logic, integration for adapters, contract for interfaces
5. **Rollback** — Document rollback plan for production-impacting changes

### Skill Auto-Activation by Path
| Path Pattern | Activated Skills |
|---|---|
| `pkg/domain/**` | web3-go-architecture, deterministic-testing |
| `pkg/adapters/**` | adapter-contract-testing, web3-go-architecture |
| `pkg/services/indexing/**` | web3-reorg-idempotency, indexer-state-consistency |
| `pkg/services/reorg/**` | web3-reorg-idempotency, event-ordering-finality |
| `**/*_test.go` | deterministic-testing, chaos-resilience |
| `**/go.mod` | dependency-upgrade-governance |

### Active Skills Catalog
See [skills/INDEX.md](skills/INDEX.md) for the full catalog of 27 skills.

### Templates
- Active skills declaration: [ACTIVE_SKILLS_TEMPLATE.md](ACTIVE_SKILLS_TEMPLATE.md)

## History

This system replaces the previous multi-file governance setup. The following files have been consolidated into this document:
- `BEHAVIORAL_CONSTRAINTS.md`
- `PRE_CODING_CHECKLIST.md`
- `READINESS_CHECK.md`
- `AI_AUTO_TRIGGER_GUIDE.md`
- `AI_DOCUMENTATION_POLICY.md`
- `AI_SESSION_INIT.md`
- `AUTO_ACTIVATION_RULES.md`
- `AUTO_ACTIVATION_USAGE.md`

Previous governance files are archived in `.codex/archive/` for reference.
