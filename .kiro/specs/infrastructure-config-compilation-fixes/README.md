# Infrastructure Config Compilation Fixes Specification

## Overview

This specification provides a comprehensive plan to fix all compilation errors in the `pkg/infrastructure/config/` package. The package currently has 10+ typecheck errors that prevent the full project from building.

## Status

**Phase**: Ready for Implementation  
**Errors**: 10+ typecheck errors  
**Files**: 4 files to modify  
**Estimated Time**: 1.5-2 hours

## Quick Links

- **[GETTING_STARTED.md](GETTING_STARTED.md)** - Start here for quick overview
- **[SUMMARY.md](SUMMARY.md)** - Executive summary of issues and approach
- **[requirements.md](requirements.md)** - Detailed requirements and acceptance criteria
- **[design.md](design.md)** - Implementation design and strategy
- **[tasks.md](tasks.md)** - Task list and implementation plan
- **[CHECKLIST.md](CHECKLIST.md)** - Verification checklist

## Problem Statement

The `pkg/infrastructure/config/` package has multiple compilation errors:

1. **Kafka API Issues** (5 errors)
   - Incorrect method signatures
   - Undefined types
   - Wrong parameter counts

2. **Consul API Issues** (1 error)
   - Undefined API functions

3. **Encryption Issues** (1 error)
   - Type mismatches in encryption operations

These errors prevent the full project from compiling with `go build ./...`

## Solution Overview

The specification provides a structured approach to fix all errors:

1. **Phase 1**: Fix Kafka API calls (Tasks 1-4)
2. **Phase 2**: Fix Consul API calls (Task 5)
3. **Phase 3**: Fix encryption type conversions (Task 6)
4. **Phase 4**: Verification (Tasks 7-8)

## Key Achievements

Upon completion, the project will have:

- ✅ All Kafka API calls using correct signatures
- ✅ All Consul API calls using correct functions
- ✅ All encryption operations using correct types
- ✅ Clean compilation: `go build ./...`
- ✅ Zero typecheck errors in pkg/infrastructure/config

## Files Involved

### Files to Modify

1. **pkg/infrastructure/config/kafka_config.go**
   - Fix CreateTopics variadic operator
   - Fix TopicMetadata type reference

2. **pkg/infrastructure/config/kafka_advanced.go**
   - Fix CreateTopic method calls
   - Fix Brokers field access

3. **pkg/infrastructure/config/consul_config.go**
   - Fix WatchPlan API call

4. **pkg/infrastructure/config/config_manager.go**
   - Fix encryption type conversions

## Implementation Tasks

| Task | Description | Effort | Status |
|------|-------------|--------|--------|
| 1 | Fix Kafka CreateTopics API Call | 15 min | ⏳ |
| 2 | Fix Kafka TopicMetadata Type Reference | 20 min | ⏳ |
| 3 | Fix Kafka CreateTopic Method Calls | 20 min | ⏳ |
| 4 | Fix Kafka Brokers Field Access | 20 min | ⏳ |
| 5 | Fix Consul WatchPlan API | 15 min | ⏳ |
| 6 | Fix Encryption Type Conversions | 15 min | ⏳ |
| 7 | Verify Clean Compilation | 10 min | ⏳ |
| 8 | Verify Full Project Build | 10 min | ⏳ |

**Total Estimated Time**: ~2 hours

## Success Criteria

- [ ] All 10+ compilation errors resolved
- [ ] `go build ./pkg/infrastructure/config/...` succeeds
- [ ] `go build ./...` succeeds
- [ ] `golangci-lint run ./pkg/infrastructure/config/...` returns 0 errors
- [ ] No new warnings introduced
- [ ] All existing tests pass

## How to Use This Spec

### For Quick Start
1. Read [GETTING_STARTED.md](GETTING_STARTED.md)
2. Follow the task checklist
3. Verify with build commands

### For Detailed Understanding
1. Read [SUMMARY.md](SUMMARY.md) for overview
2. Read [requirements.md](requirements.md) for detailed requirements
3. Read [design.md](design.md) for implementation approach
4. Follow [tasks.md](tasks.md) for implementation

### For Verification
1. Use [CHECKLIST.md](CHECKLIST.md) to track progress
2. Run verification commands after each task
3. Complete final verification

## Related Documentation

- **Previous Work**: [BUILD_COMPILATION_FIXES_COMPLETE.md](../../BUILD_COMPILATION_FIXES_COMPLETE.md)
- **Test Integration Fixes**: [TASK_2_COMPLETION_SUMMARY.md](../../TASK_2_COMPLETION_SUMMARY.md)
- **Processor Package Fixes**: [GOLANGCI_LINT_FIXES_COMPLETE.md](../../GOLANGCI_LINT_FIXES_COMPLETE.md)

## Next Steps

1. ✅ Review this specification
2. ⏭️ Read [GETTING_STARTED.md](GETTING_STARTED.md)
3. ⏭️ Start with Task 1
4. ⏭️ Follow tasks in order
5. ⏭️ Complete verification

## Questions?

Refer to the appropriate document:
- **Overview**: [SUMMARY.md](SUMMARY.md)
- **Getting Started**: [GETTING_STARTED.md](GETTING_STARTED.md)
- **Requirements**: [requirements.md](requirements.md)
- **Design**: [design.md](design.md)
- **Tasks**: [tasks.md](tasks.md)
- **Checklist**: [CHECKLIST.md](CHECKLIST.md)

---

**Specification Created**: January 17, 2026  
**Status**: Ready for Implementation  
**Estimated Duration**: 1.5-2 hours
