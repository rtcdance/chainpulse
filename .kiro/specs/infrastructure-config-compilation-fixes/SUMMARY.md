# Infrastructure Config Compilation Fixes - Spec Summary

## Status: Ready for Implementation

This spec provides a comprehensive plan to fix all compilation errors in the `pkg/infrastructure/config/` package.

## Key Issues Addressed

### 1. Kafka API Issues (5 errors)
- **CreateTopics variadic operator**: Method expects slice, not variadic args
- **TopicMetadata undefined**: Type not found in Kafka library
- **CreateTopic wrong signature**: Method called with incorrect parameters
- **Brokers field undefined**: Field doesn't exist on KafkaCluster type

### 2. Consul API Issues (1 error)
- **WatchPlan undefined**: Function not found in Consul API

### 3. Encryption Issues (1 error)
- **Type mismatch**: String/[]byte conversion in GCM operations

## Implementation Approach

1. **Phase 1**: Fix Kafka API calls (Tasks 1-4)
2. **Phase 2**: Fix Consul API calls (Task 5)
3. **Phase 3**: Fix encryption type conversions (Task 6)
4. **Phase 4**: Verification (Tasks 7-8)

## Expected Outcomes

- ✅ All Kafka API calls use correct signatures
- ✅ All Consul API calls use correct functions
- ✅ All encryption operations use correct types
- ✅ `golangci-lint run ./pkg/infrastructure/config/...` returns 0 errors
- ✅ Full project builds: `go build ./...`

## Files to Modify

1. `pkg/infrastructure/config/kafka_config.go` - 2 fixes
2. `pkg/infrastructure/config/kafka_advanced.go` - 3 fixes
3. `pkg/infrastructure/config/consul_config.go` - 1 fix
4. `pkg/infrastructure/config/config_manager.go` - 1 fix

## Error Summary

| File | Line | Error | Type |
|------|------|-------|------|
| kafka_config.go | 56 | Variadic operator on non-variadic method | Kafka API |
| kafka_config.go | 109 | Undefined TopicMetadata type | Kafka API |
| kafka_advanced.go | 43, 81 | Too many arguments to CreateTopic | Kafka API |
| kafka_advanced.go | 99, 193, 240 | Brokers field undefined | Kafka API |
| consul_config.go | 118 | Undefined NewWatchPlan | Consul API |
| config_manager.go | 186-187 | Type mismatch string/[]byte | Encryption |

## Next Steps

1. Review this spec for accuracy
2. Start with Task 1: Fix Kafka CreateTopics API Call
3. Follow the task list sequentially
4. Run verification after each phase
5. Complete full project build verification

## Estimated Effort

- **Core fixes**: 1.5-2 hours
- **Total tasks**: 8 (all required)

## Success Criteria

- [ ] All compilation errors resolved
- [ ] `go build ./pkg/infrastructure/config/...` succeeds
- [ ] `go build ./...` succeeds
- [ ] `golangci-lint run ./pkg/infrastructure/config/...` returns 0 errors
- [ ] No new warnings introduced
- [ ] All existing tests pass

## Related Documentation

- Previous fixes: `BUILD_COMPILATION_FIXES_COMPLETE.md`
- Test integration fixes: `TASK_2_COMPLETION_SUMMARY.md`
- Processor package fixes: `GOLANGCI_LINT_FIXES_COMPLETE.md`
