# Skills Auto-Activation System

**Status**: Active | **Last Updated**: 2026-03-30

## Quick Start

### 1. Install Git Hooks
```bash
./scripts/install-hooks.sh
```

### 2. Make Changes
```bash
git add pkg/services/indexing/indexer.go
git commit -m "fix: indexer reorg handling"
```

### 3. Auto-Activation Happens
```
🎯 Auto-Activating Skills Based on Changes
===========================================
✅ Activated 4 skills:
   - design-review-gate
   - event-ordering-finality
   - indexer-state-consistency
   - web3-reorg-idempotency

📝 Details: .codex/active-skills.md
```

## How It Works

### Trigger Rules

| File Path | Auto-Activated Skills |
|-----------|----------------------|
| `pkg/domain/**/*.go` | web3-go-architecture-guardrails, deterministic-testing |
| `pkg/adapters/**/*.go` | adapter-contract-testing, web3-go-architecture-guardrails |
| `pkg/services/indexing/**/*.go` | web3-reorg-idempotency, indexer-state-consistency |
| `pkg/plugins/rpc/**/*.go` | chaos-resilience-testing, rate-limiting-backpressure |
| `pkg/adapters/contracts/**/*.go` | smart-contract-integration-safety, web3-security-patterns |
| `**/*_test.go` | deterministic-testing |
| `go.mod` | dependency-upgrade-governance |

### Pattern Triggers

| Code Pattern | Auto-Activated Skills |
|--------------|----------------------|
| `go func`, `make(chan`, `sync.` | go-concurrency-patterns, concurrency-safety |
| `crypto.`, `keystore`, `private.*key` | web3-security-patterns, security-compliance-baseline |

### Always Active

- `design-review-gate` - Mandatory for all changes

## Manual Usage

### Check What Would Activate
```bash
git add <files>
./scripts/auto-activate-skills.sh
cat .codex/active-skills.md
```

### Override Auto-Activation
Edit `.codex/active-skills.md`:
```markdown
## Removed Skills
- ~~performance-capacity-guardrails~~ - Not a hot path, trivial getter
```

## Integration with Workflow

```
1. Create Spec
   ↓
2. Stage Changes (git add)
   ↓
3. Auto-Activate Skills ← Automatic
   ↓
4. Review .codex/active-skills.md
   ↓
5. Run Pre-Coding Checklist ← Automatic
   ↓
6. Commit (if checks pass)
```

## Troubleshooting

### Skills Not Activating
```bash
# Check if hooks installed
ls -la .git/hooks/pre-commit

# Reinstall hooks
./scripts/install-hooks.sh
```

### Wrong Skills Activated
```bash
# Manually edit
vim .codex/active-skills.md

# Or disable auto-activation for this commit
git commit --no-verify
```

### Skip Checks Temporarily
```bash
# Skip all pre-commit checks (use sparingly)
git commit --no-verify -m "message"
```

## Configuration

See `.codex/AUTO_ACTIVATION_RULES.md` for full trigger rules.

To modify triggers, edit `scripts/auto-activate-skills.sh`.

## Benefits

- ✅ No manual skill selection needed
- ✅ Consistent skill application
- ✅ Catches missing skills automatically
- ✅ Reduces human error
- ✅ Enforces quality gates
