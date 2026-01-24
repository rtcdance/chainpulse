# Design: GitHub CI golangci-lint 404 Error Fix

## Overview

The GitHub Actions CI/CD pipeline is experiencing 404 errors when attempting to download golangci-lint binaries. This design addresses the root causes and implements a robust solution with fallback mechanisms, version pinning, and improved error handling.

## Architecture

### Current Problem Analysis

The 404 error occurs in the golangci-lint-action v4 when:
1. The action attempts to download the pre-compiled binary for the specified version
2. The GitHub release server is temporarily unavailable or the binary URL is incorrect
3. Network connectivity issues prevent the download
4. The specified version (v1.60.0) may have compatibility issues with the action

### Solution Architecture

```
GitHub Actions Workflow
    ↓
[Lint Job Triggered]
    ↓
[Validate Go Environment]
    ↓
[Configure golangci-lint]
    ├─ Primary: Use action with stable version
    ├─ Fallback: Install via go install
    └─ Fallback: Use pre-cached binary
    ↓
[Run Linting]
    ├─ Execute with timeout
    ├─ Capture output
    └─ Report results
    ↓
[Handle Errors]
    ├─ Retry on transient failures
    ├─ Log diagnostics
    └─ Continue other jobs
```

## Components and Interfaces

### 1. Workflow Configuration

**File**: `.github/workflows/test.yml` (lint job section)

**Responsibilities**:
- Define the lint job with proper Go environment setup
- Configure golangci-lint action with stable version
- Implement retry logic for transient failures
- Set appropriate timeouts and resource limits

**Key Changes**:
- Update golangci-lint-action to use a stable, well-tested version
- Add explicit cache configuration for Go modules
- Implement step-level error handling with continue-on-error
- Add diagnostic logging for troubleshooting

### 2. Fallback Installation Method

**Approach**: If the action fails, fall back to installing golangci-lint via `go install`

**Implementation**:
```bash
# Primary: Use action (v4 with stable version)
# Fallback: Install via go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1

# Run linting
golangci-lint run --timeout=5m
```

**Advantages**:
- Direct control over installation
- No dependency on action binary downloads
- Faster fallback if action fails
- Clear error messages if installation fails

### 3. Error Handling Strategy

**Transient Error Handling**:
- Detect 404 and network errors
- Implement exponential backoff retry (max 3 attempts)
- Log each attempt with timestamp
- Fail gracefully with diagnostic information

**Permanent Error Handling**:
- Continue other jobs (build, test) even if lint fails
- Provide clear error messages in workflow summary
- Generate diagnostic report for troubleshooting

### 4. Version Management

**Stable Version Selection**:
- Use v1.59.1 (latest stable, well-tested)
- Pin version explicitly in workflow
- Document version selection rationale
- Plan quarterly updates to newer stable versions

**Version Compatibility**:
- Ensure Go version (1.24) is compatible with golangci-lint
- Verify action version (v4) supports the linter version
- Test locally before updating in CI

## Data Models

### Workflow Configuration

```yaml
lint:
  name: Lint Code
  runs-on: ubuntu-latest
  
  steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.24'
        cache: true
    
    - name: Run golangci-lint (Primary)
      uses: golangci/golangci-lint-action@v4
      with:
        version: v1.59.1
        args: --timeout=5m --out-format=colored-line-number
      continue-on-error: true
      id: lint_action
    
    - name: Fallback: Install golangci-lint
      if: steps.lint_action.outcome == 'failure'
      run: |
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
    
    - name: Fallback: Run golangci-lint
      if: steps.lint_action.outcome == 'failure'
      run: |
        golangci-lint run --timeout=5m --out-format=colored-line-number
```

### Error Diagnostic Information

```
Lint Job Execution Report:
- Timestamp: [ISO 8601]
- Go Version: [version]
- golangci-lint Version: [version]
- Action Status: [success/failure]
- Fallback Used: [yes/no]
- Error Type: [404/timeout/other]
- Retry Attempts: [count]
- Final Status: [passed/failed]
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Lint Job Completion

**For any** workflow run, the lint job SHALL complete with a defined status (success or failure) without hanging or timing out indefinitely.

**Validates: Requirements 1.1, 1.3**

### Property 2: Fallback Mechanism Activation

**For any** failed primary golangci-lint action execution, the fallback installation method SHALL be automatically triggered and attempted.

**Validates: Requirements 3.1, 3.2**

### Property 3: Version Consistency

**For any** workflow execution, the golangci-lint version used SHALL match the pinned version specified in the workflow configuration.

**Validates: Requirements 2.1, 2.3**

### Property 4: Error Message Clarity

**For any** lint job failure, the workflow logs SHALL contain a clear, actionable error message that identifies the root cause (404, timeout, configuration error, etc.).

**Validates: Requirements 3.3, 5.2**

### Property 5: Non-Blocking Lint Failures

**For any** lint job failure, the build and test jobs SHALL continue to execute independently without being blocked by the lint job failure.

**Validates: Requirements 3.4**

### Property 6: Diagnostic Information Capture

**For any** workflow execution, the workflow logs SHALL capture and report: Go version, golangci-lint version, action status, fallback usage, and final status.

**Validates: Requirements 5.1, 5.3**

## Error Handling

### Transient Errors (Retryable)

**404 Binary Download Failure**:
- Cause: Temporary GitHub release server unavailability
- Detection: HTTP 404 response from action
- Handling: Trigger fallback installation via `go install`
- Logging: Log attempt number and timestamp

**Network Timeout**:
- Cause: Slow network or temporary connectivity issue
- Detection: Connection timeout during download
- Handling: Retry with exponential backoff (1s, 2s, 4s)
- Logging: Log timeout duration and retry count

### Permanent Errors (Non-Retryable)

**Incompatible Go Version**:
- Cause: Go version doesn't support golangci-lint version
- Detection: Build error during `go install`
- Handling: Fail with clear error message
- Logging: Log Go version and required version

**Configuration Error**:
- Cause: Invalid workflow configuration or arguments
- Detection: YAML parsing error or invalid flag
- Handling: Fail with configuration error message
- Logging: Log configuration details and error

### Error Recovery Strategy

1. **Primary Attempt**: Use golangci-lint-action v4 with v1.59.1
2. **First Fallback**: Install via `go install` and run directly
3. **Final Fallback**: Skip linting and continue other jobs
4. **Logging**: Capture all errors and diagnostic information
5. **Notification**: Report status in workflow summary

## Testing Strategy

### Unit Testing (Not Applicable)

Workflow files cannot be unit tested directly. Testing occurs through workflow execution.

### Integration Testing

**Test 1: Successful Lint Execution**
- Trigger workflow on valid code
- Verify lint job completes successfully
- Verify no fallback is triggered
- Verify exit code is 0

**Test 2: Fallback Mechanism Activation**
- Simulate action failure by using invalid version
- Verify fallback installation is triggered
- Verify linting completes via fallback
- Verify diagnostic logs show fallback usage

**Test 3: Error Handling**
- Introduce linting errors in code
- Verify lint job fails with appropriate exit code
- Verify error messages are clear and actionable
- Verify other jobs continue to execute

**Test 4: Version Consistency**
- Verify pinned version is used in all executions
- Verify version is logged in workflow output
- Verify version matches configuration

### Property-Based Testing

**Property 1 Test**: Lint Job Completion
- Run workflow multiple times
- Verify all executions complete with defined status
- Verify no timeouts or hangs occur

**Property 2 Test**: Fallback Mechanism
- Simulate primary action failure
- Verify fallback is triggered automatically
- Verify fallback completes successfully

**Property 3 Test**: Version Consistency
- Extract version from workflow logs
- Verify version matches pinned version
- Verify consistency across multiple runs

**Property 4 Test**: Error Message Clarity
- Trigger various error conditions
- Verify error messages are present and clear
- Verify messages identify root cause

**Property 5 Test**: Non-Blocking Failures
- Fail lint job intentionally
- Verify build and test jobs continue
- Verify workflow doesn't block on lint failure

**Property 6 Test**: Diagnostic Information
- Execute workflow
- Extract diagnostic information from logs
- Verify all required fields are present

## Implementation Notes

### Version Selection Rationale

- **v1.59.1**: Latest stable version with proven compatibility
- **Go 1.24**: Latest Go version with good linter support
- **Action v4**: Latest action version with improved reliability

### Fallback Strategy Rationale

- **go install**: Direct installation bypasses action binary download
- **Automatic triggering**: No manual intervention required
- **Graceful degradation**: Lint failures don't block other jobs

### Monitoring and Observability

- Log all workflow steps with timestamps
- Capture Go and golangci-lint versions
- Report fallback usage in workflow summary
- Generate diagnostic report on failure

