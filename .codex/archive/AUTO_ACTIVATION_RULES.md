# Codex Auto-Activation Rules

**Purpose**: Automatically activate relevant skills based on file paths and change patterns.

## Auto-Activation Rules

### Rule 1: File Path Triggers

```yaml
# pkg/domain/** changes
triggers:
  - path: "pkg/domain/**/*.go"
    skills:
      - web3-go-architecture-guardrails
      - deterministic-testing
    reason: "Domain layer must stay pure, tests must be deterministic"

# pkg/adapters/** changes
triggers:
  - path: "pkg/adapters/**/*.go"
    skills:
      - adapter-contract-testing
      - web3-go-architecture-guardrails
    reason: "Adapters must implement domain interfaces with contract tests"

# pkg/services/indexing/** changes
triggers:
  - path: "pkg/services/indexing/**/*.go"
    skills:
      - web3-reorg-idempotency
      - indexer-state-consistency
      - event-ordering-finality
    reason: "Indexer must handle reorgs and maintain state consistency"

# pkg/plugins/rpc/** changes
triggers:
  - path: "pkg/plugins/rpc/**/*.go"
    skills:
      - chaos-resilience-testing
      - rate-limiting-backpressure
      - gas-cost-optimization
    reason: "RPC clients must handle failures and optimize costs"

# pkg/plugins/database/** changes
triggers:
  - path: "pkg/plugins/database/**/*.go"
    skills:
      - schema-migration-safety
      - adapter-contract-testing
    reason: "Database changes need migration safety and contract tests"

# pkg/observability/** changes
triggers:
  - path: "pkg/observability/**/*.go"
    skills:
      - observability-slo-gates
    reason: "Observability changes must maintain SLO alignment"

# Contract interaction code
triggers:
  - path: "pkg/adapters/contracts/**/*.go"
    skills:
      - smart-contract-integration-safety
      - web3-security-patterns
    reason: "Contract calls need safety checks and security validation"

# Concurrency code
triggers:
  - pattern: "go func|make(chan|sync\\."
    skills:
      - go-concurrency-patterns
      - concurrency-safety
    reason: "Goroutines and channels need lifecycle management"

# Security-sensitive code
triggers:
  - pattern: "crypto\\.|keystore|private.*key|signature"
    skills:
      - web3-security-patterns
      - security-compliance-baseline
    reason: "Crypto operations need security review"

# Test files
triggers:
  - path: "**/*_test.go"
    skills:
      - deterministic-testing
    reason: "Tests must be reproducible"

# API changes
triggers:
  - path: "pkg/adapters/api/**/*.go"
    skills:
      - api-contract-compatibility
      - observability-slo-gates
    reason: "API changes need compatibility and observability"

# Performance-critical paths
triggers:
  - path: "pkg/services/indexing/**/*.go"
  - path: "pkg/services/query/**/*.go"
    skills:
      - performance-capacity-guardrails
    reason: "Hot paths need performance validation"
```

### Rule 2: Change Pattern Triggers

```yaml
# New files
triggers:
  - change_type: "added"
    skills:
      - code-organization-placement
    reason: "New files must follow directory structure"

# go.mod changes
triggers:
  - path: "go.mod"
    skills:
      - dependency-upgrade-governance
    reason: "Dependency changes need approval"

# Schema changes
triggers:
  - path: "**/*migration*.sql"
  - path: "**/*schema*.sql"
    skills:
      - schema-migration-safety
    reason: "Schema changes need rollback plan"

# Multi-file changes (>10 files)
triggers:
  - change_count: ">10"
    skills:
      - release-rollback-readiness
    reason: "Large changes need rollback plan"
```

### Rule 3: Always Active

```yaml
mandatory:
  - design-review-gate
    reason: "Spec approval required for all changes"

  - micro-loop-delivery
    reason: "All changes follow micro-loop process"
```

## Auto-Activation Script

```bash
#!/bin/bash
# scripts/auto-activate-skills.sh

set -e

echo "🎯 Auto-Activating Skills"
echo "========================="

CHANGED_FILES=$(git diff --cached --name-only --diff-filter=AM)
ACTIVE_SKILLS=()

# Always active
ACTIVE_SKILLS+=("design-review-gate")
ACTIVE_SKILLS+=("micro-loop-delivery")

# Check file paths
while IFS= read -r file; do
  case "$file" in
    pkg/domain/*.go)
      ACTIVE_SKILLS+=("web3-go-architecture-guardrails")
      ACTIVE_SKILLS+=("deterministic-testing")
      ;;
    pkg/adapters/*.go)
      ACTIVE_SKILLS+=("adapter-contract-testing")
      ;;
    pkg/services/indexing/*.go)
      ACTIVE_SKILLS+=("web3-reorg-idempotency")
      ACTIVE_SKILLS+=("indexer-state-consistency")
      ;;
    pkg/plugins/rpc/*.go)
      ACTIVE_SKILLS+=("chaos-resilience-testing")
      ACTIVE_SKILLS+=("rate-limiting-backpressure")
      ;;
    pkg/adapters/contracts/*.go)
      ACTIVE_SKILLS+=("smart-contract-integration-safety")
      ;;
    *_test.go)
      ACTIVE_SKILLS+=("deterministic-testing")
      ;;
    go.mod)
      ACTIVE_SKILLS+=("dependency-upgrade-governance")
      ;;
  esac
done <<< "$CHANGED_FILES"

# Check patterns in changed files
if git diff --cached | grep -qE "go func|make\(chan|sync\."; then
  ACTIVE_SKILLS+=("go-concurrency-patterns")
  ACTIVE_SKILLS+=("concurrency-safety")
fi

if git diff --cached | grep -qE "crypto\.|keystore|private.*key"; then
  ACTIVE_SKILLS+=("web3-security-patterns")
  ACTIVE_SKILLS+=("security-compliance-baseline")
fi

# Remove duplicates
UNIQUE_SKILLS=($(echo "${ACTIVE_SKILLS[@]}" | tr ' ' '\n' | sort -u))

# Generate active-skills.md
cat > .codex/active-skills.md <<EOF
# Auto-Activated Skills

**Generated**: $(date +%Y-%m-%d)
**Changed Files**: $(echo "$CHANGED_FILES" | wc -l)

## Active Skills

EOF

for skill in "${UNIQUE_SKILLS[@]}"; do
  echo "- $skill" >> .codex/active-skills.md
done

echo ""
echo "✅ Activated ${#UNIQUE_SKILLS[@]} skills:"
printf '   - %s\n' "${UNIQUE_SKILLS[@]}"
echo ""
echo "📝 See .codex/active-skills.md for details"
```

## Integration with Git Hooks

```bash
# .git/hooks/pre-commit
#!/bin/bash

# Auto-activate skills
scripts/auto-activate-skills.sh

# Run pre-coding checklist
scripts/pre-coding-checklist.sh

# If checks pass, proceed with commit
```

## Usage

### Manual Activation
```bash
# Generate active skills based on staged changes
./scripts/auto-activate-skills.sh
```

### Automatic Activation
```bash
# Install git hook
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# Now auto-activates on every commit
git add pkg/domain/indexer.go
git commit -m "fix: indexer bug"
# → Auto-activates: design-review-gate, web3-go-architecture-guardrails, deterministic-testing
```

## Override Mechanism

If auto-activation is wrong, manually edit `.codex/active-skills.md`:

```markdown
# Active Skills (Manual Override)

## Active Skills
- design-review-gate
- web3-reorg-idempotency
- deterministic-testing

## Removed (with justification)
- ~~performance-capacity-guardrails~~ - Not a hot path, trivial change
```
