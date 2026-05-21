# Pre-Coding Checklist

**Purpose**: Mandatory gates before AI writes any code.

## 1. Specification Gate (BLOCKING)

```yaml
status: MANDATORY
enforcement: Pre-commit hook

rules:
  - No code without approved spec in docs/specs/
  - Spec must follow TEMPLATE.md format
  - Spec must have approval signature
  - Spec path: docs/specs/YYYY-MM-DD-<topic>.md
```

**Enforcement**:
```bash
#!/bin/bash
# scripts/check-spec-approval.sh
if ! grep -q "Status: Approved" docs/specs/*.md; then
  echo "❌ No approved spec found"
  exit 1
fi
```

## 2. Skill Selection (BLOCKING)

```yaml
status: MANDATORY
enforcement: AI must declare upfront

rules:
  - List applicable skills BEFORE coding
  - Minimum 2 skills per feature (design-review-gate + domain skill)
  - Document why each skill applies
```

**Template**:
```markdown
## Active Skills for This Task

1. design-review-gate - Spec: docs/specs/2026-03-30-indexer-fix.md
2. web3-reorg-idempotency - Modifying block rollback logic
3. deterministic-testing - Adding reorg test cases

## Exit Criteria Checklist
- [ ] Spec approved
- [ ] Reorg scenarios tested
- [ ] Tests deterministic
```

## 3. Blast Radius Assessment (BLOCKING)

```yaml
status: MANDATORY
enforcement: AI must answer before coding

questions:
  - How many files will change?
  - Which layers affected? (domain/application/adapters)
  - Any breaking changes?
  - Rollback plan?
```

**Risk Matrix**:
```
Files Changed | Risk Level | Required Approval
1-3          | Low        | Self-review
4-10         | Medium     | Peer review
>10          | High       | Architect review + phased rollout
```

## 4. Dependency Approval (BLOCKING)

```yaml
status: MANDATORY
enforcement: DEPENDENCY_APPROVAL.md must be updated

rules:
  - New dependency requires justification
  - Must document: why stdlib insufficient
  - Must check: license, maintenance, size
```

**Template**:
```markdown
# DEPENDENCY_APPROVAL.md

## Pending
- [ ] github.com/lib/pq v1.10.9
  - Why: PostgreSQL driver
  - Alternatives: database/sql (no Postgres support)
  - License: MIT
  - Size: 200KB
  - Approved by: [architect-name]
```

## 5. Test Strategy Declaration (BLOCKING)

```yaml
status: MANDATORY
enforcement: AI must declare before coding

rules:
  - Unit tests for business logic
  - Integration tests for adapters
  - Contract tests for interfaces
  - No tests for trivial getters/setters
```

**Template**:
```markdown
## Test Strategy

### Unit Tests
- pkg/domain/indexer/rollback_test.go
  - Test: rollback to safe block
  - Test: handle missing checkpoint

### Integration Tests
- pkg/adapters/rpc/client_test.go
  - Test: RPC timeout handling
  - Test: circuit breaker activation

### Contract Tests
- pkg/plugins/database/postgres_test.go
  - Test: implements DatabasePlugin interface
  - Test: behavior parity with MockDB
```

## 6. Rollback Plan (BLOCKING for production)

```yaml
status: MANDATORY for production changes
enforcement: Must document before deploy

rules:
  - How to detect failure?
  - How to rollback?
  - Data migration reversible?
  - Feature flag available?
```

**Template**:
```markdown
## Rollback Plan

### Detection
- Alert: indexer_lag > 100 blocks for 10m
- Metric: error_rate > 5%

### Rollback Steps
1. Revert commit: git revert abc123
2. Redeploy: kubectl rollout undo deployment/indexer
3. Verify: check indexer_lag returns to <10

### Data Safety
- Schema change: reversible (added column, not dropped)
- No data deletion
```

## 7. Performance Budget (BLOCKING for hot paths)

```yaml
status: MANDATORY for indexer/query/cache changes
enforcement: Benchmark required

rules:
  - Baseline benchmark before change
  - Target: no >10% regression
  - Document: expected throughput
```

**Template**:
```markdown
## Performance Budget

### Baseline
- Blocks/sec: 50
- Memory: 200MB
- RPC calls/block: 2

### Target
- Blocks/sec: ≥45 (no >10% regression)
- Memory: ≤250MB
- RPC calls/block: ≤2

### Benchmark
```bash
go test -bench=BenchmarkIndexer -benchmem
```
```

## 8. Security Review Trigger (BLOCKING)

```yaml
status: MANDATORY for sensitive changes
enforcement: Security checklist required

triggers:
  - Auth/authz changes
  - Crypto operations
  - External API exposure
  - Secret handling
  - Database access patterns
```

**Checklist**:
```markdown
## Security Review

- [ ] No secrets in code/logs
- [ ] Input validation at boundaries
- [ ] SQL injection prevention (parameterized queries)
- [ ] Rate limiting on public endpoints
- [ ] Private keys in keystore only
- [ ] Signature verification includes replay protection
```

## 9. Observability Requirements (BLOCKING)

```yaml
status: MANDATORY for new features
enforcement: Metrics + logs required

rules:
  - Add metrics for success/failure
  - Add structured logs for debugging
  - Define SLO for new operation
  - Add health check if new service
```

**Template**:
```markdown
## Observability

### Metrics
- indexer_blocks_processed_total (counter)
- indexer_processing_duration_seconds (histogram)
- indexer_errors_total (counter, label: error_type)

### Logs
- INFO: "block processed" (block_number, duration_ms)
- ERROR: "block processing failed" (block_number, error, retry_count)

### SLO
- 99% of blocks processed within 5s
- <1% error rate
```

## 10. Documentation Update (BLOCKING)

```yaml
status: MANDATORY for API/config changes
enforcement: Docs updated in same PR

rules:
  - API changes → update docs/api/
  - Config changes → update README.md
  - New feature → update docs/guides/
```

## Enforcement Script

```bash
#!/bin/bash
# scripts/pre-coding-checklist.sh

echo "Pre-Coding Checklist"
echo "===================="

# 1. Check spec approval
if ! scripts/check-spec-approval.sh; then
  exit 1
fi

# 2. Check skill declaration
if [ ! -f ".codex/active-skills.md" ]; then
  echo "❌ No active skills declared"
  exit 1
fi

# 3. Check dependency approval
if git diff --name-only | grep -q "go.mod"; then
  if ! grep -q "Approved by:" DEPENDENCY_APPROVAL.md; then
    echo "❌ New dependency without approval"
    exit 1
  fi
fi

echo "✅ Pre-coding checklist passed"
```

## Summary

**10 Blocking Gates**:
1. Spec approval
2. Skill selection
3. Blast radius assessment
4. Dependency approval
5. Test strategy
6. Rollback plan
7. Performance budget
8. Security review
9. Observability requirements
10. Documentation update

**Enforcement**: Pre-commit hooks + PR template + CI checks

**Bypass**: Only architect can override (with written justification)
