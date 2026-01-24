# Getting Started: Infrastructure Config Compilation Fixes

## Quick Overview

This spec addresses 10+ compilation errors in the `pkg/infrastructure/config/` package. The errors are primarily related to:
- Kafka API method signatures
- Consul API function calls
- Encryption type conversions

## Current Status

**Errors**: 10+ typecheck errors  
**Package**: `pkg/infrastructure/config/`  
**Scope**: 4 files to modify  
**Estimated Time**: 1.5-2 hours

## Error Summary

```
pkg/infrastructure/config/kafka_config.go:56:12: cannot use ... in call to non-variadic k.admin.CreateTopics
pkg/infrastructure/config/kafka_config.go:109:84: undefined: kafka.TopicMetadata
pkg/infrastructure/config/config_manager.go:186:45: cannot use data[nonceSize:] (value of type []byte) as string value in assignment
pkg/infrastructure/config/config_manager.go:187:41: cannot use ciphertext (variable of type string) as []byte value in argument to gcm.Open
pkg/infrastructure/config/consul_config.go:118:19: undefined: api.NewWatchPlan
pkg/infrastructure/config/kafka_advanced.go:43:54: too many arguments in call to kam.cluster.CreateTopic
pkg/infrastructure/config/kafka_advanced.go:81:50: too many arguments in call to kam.cluster.CreateTopic
pkg/infrastructure/config/kafka_advanced.go:99:45: kam.cluster.Brokers undefined
pkg/infrastructure/config/kafka_advanced.go:193:37: kcm.cluster.Brokers undefined
pkg/infrastructure/config/kafka_advanced.go:240:35: kcm.cluster.Brokers undefined
```

## How to Use This Spec

### 1. Review the Specification
- Read `SUMMARY.md` for overview
- Read `requirements.md` for detailed requirements
- Read `design.md` for implementation approach

### 2. Follow the Task List
- Start with Task 1: Fix Kafka CreateTopics API Call
- Complete tasks in order (they have dependencies)
- Verify after each task

### 3. Verify Your Work
- After each task: `go build ./pkg/infrastructure/config/...`
- After all tasks: `golangci-lint run ./pkg/infrastructure/config/...`
- Final verification: `go build ./...`

## Task Checklist

- [ ] Task 1: Fix Kafka CreateTopics API Call
- [ ] Task 2: Fix Kafka TopicMetadata Type Reference
- [ ] Task 3: Fix Kafka CreateTopic Method Calls
- [ ] Task 4: Fix Kafka Brokers Field Access
- [ ] Task 5: Fix Consul WatchPlan API
- [ ] Task 6: Fix Encryption Type Conversions
- [ ] Task 7: Verify Clean Compilation
- [ ] Task 8: Verify Full Project Build

## Key Files to Modify

1. **pkg/infrastructure/config/kafka_config.go**
   - Line 56: Fix CreateTopics call
   - Line 109: Fix TopicMetadata reference

2. **pkg/infrastructure/config/kafka_advanced.go**
   - Lines 43, 81: Fix CreateTopic calls
   - Lines 99, 193, 240: Fix Brokers access

3. **pkg/infrastructure/config/consul_config.go**
   - Line 118: Fix WatchPlan creation

4. **pkg/infrastructure/config/config_manager.go**
   - Lines 186-187: Fix encryption type conversions

## Quick Start Commands

### Check Current Errors
```bash
golangci-lint run ./pkg/infrastructure/config/... --timeout=5m
```

### Build Package
```bash
go build ./pkg/infrastructure/config/...
```

### Build Full Project
```bash
go build ./...
```

## Common Issues and Solutions

### Issue: "cannot use ... in call to non-variadic"
**Solution**: Remove the `...` operator and pass the slice directly
```go
// Before
k.admin.CreateTopics(ctx, topicConfigs...)

// After
k.admin.CreateTopics(ctx, topicConfigs)
```

### Issue: "undefined: kafka.TopicMetadata"
**Solution**: Check Kafka library version and use correct type name
```go
// Check what types are available in the Kafka library
// May need to use different type or define locally
```

### Issue: "too many arguments in call to"
**Solution**: Check method signature and remove extra parameters
```go
// Before
kam.cluster.CreateTopic(ctx, topic, partitions, replication)

// After
kam.cluster.CreateTopic(ctx, topic)
```

### Issue: "Type mismatch string/[]byte"
**Solution**: Explicitly convert between types
```go
// Before
ciphertext := gcm.Seal(nil, nonce, data[nonceSize:], nil)

// After
ciphertext := gcm.Seal(nil, nonce, []byte(data[nonceSize:]), nil)
```

## Next Steps

1. ✅ Review this spec
2. ⏭️ Start with Task 1
3. ⏭️ Follow tasks in order
4. ⏭️ Verify after each task
5. ⏭️ Complete full project build verification

## Questions?

Refer to:
- `design.md` for implementation details
- `requirements.md` for acceptance criteria
- `tasks.md` for task descriptions
