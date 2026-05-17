# Active Skills Declaration

**Task**: [Brief description of what you're implementing]

**Spec**: `docs/specs/YYYY-MM-DD-<topic>.md`

## Selected Skills

### 1. design-review-gate (MANDATORY)
- **Why**: Spec approval required before coding
- **Spec Path**: docs/specs/YYYY-MM-DD-<topic>.md
- **Status**: [ ] Approved

### 2. [skill-name]
- **Why**: [Explain why this skill applies]
- **Exit Criteria**:
  - [ ] [Criterion 1]
  - [ ] [Criterion 2]

### 3. [skill-name]
- **Why**: [Explain why this skill applies]
- **Exit Criteria**:
  - [ ] [Criterion 1]
  - [ ] [Criterion 2]

## Blast Radius

- **Files Changed**: [Estimate: 1-3 / 4-10 / >10]
- **Layers Affected**: [domain / application / adapters / infrastructure]
- **Breaking Changes**: [Yes/No]
- **Risk Level**: [Low / Medium / High]

## Test Strategy

### Unit Tests
- [ ] File: `pkg/.../xxx_test.go`
  - Test: [description]

### Integration Tests
- [ ] File: `pkg/.../xxx_integration_test.go`
  - Test: [description]

### Contract Tests
- [ ] File: `pkg/.../xxx_contract_test.go`
  - Test: [description]

## Rollback Plan

### Detection
- Alert: [metric/condition]
- Threshold: [value]

### Rollback Steps
1. [Step 1]
2. [Step 2]

### Data Safety
- [ ] Schema changes reversible
- [ ] No data deletion

## Observability

### Metrics
- `metric_name_total` (counter) - [description]
- `metric_name_duration_seconds` (histogram) - [description]

### Logs
- INFO: "[message]" (field1, field2)
- ERROR: "[message]" (field1, error)

### SLO
- [Target: e.g., 99% success rate, p95 <5s]

## Checklist

- [ ] Spec approved
- [ ] Skills selected and justified
- [ ] Test strategy defined
- [ ] Rollback plan documented
- [ ] Observability requirements met
- [ ] All skill exit criteria will be verified
