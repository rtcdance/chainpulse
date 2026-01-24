# Design: Infrastructure Config Compilation Fixes

## Overview

This document outlines the design approach for fixing compilation errors in the `pkg/infrastructure/config/` package.

## Current Error Analysis

### Error 1: Kafka CreateTopics Variadic Arguments
**Location**: `kafka_config.go:56`
**Error**: `cannot use ... in call to non-variadic k.admin.CreateTopics`
**Root Cause**: The Kafka admin API expects a slice, not variadic arguments
**Solution**: Remove the `...` operator and pass the slice directly

### Error 2: Undefined kafka.TopicMetadata
**Location**: `kafka_config.go:109`
**Error**: `undefined: kafka.TopicMetadata`
**Root Cause**: Type doesn't exist in the Kafka library or needs to be imported differently
**Solution**: Check Kafka library version and use correct type name or define locally

### Error 3: Encryption Type Mismatch (String vs []byte)
**Location**: `config_manager.go:186-187`
**Error**: Type mismatch between string and []byte in encryption operations
**Root Cause**: GCM cipher expects []byte but code uses string
**Solution**: Convert string to []byte before encryption/decryption

### Error 4: Consul WatchPlan Undefined
**Location**: `consul_config.go:118`
**Error**: `undefined: api.NewWatchPlan`
**Root Cause**: Function doesn't exist in Consul API or needs different import
**Solution**: Check Consul library version and use correct API

### Error 5: Kafka CreateTopic Wrong Signature
**Location**: `kafka_advanced.go:43, 81`
**Error**: `too many arguments in call to kam.cluster.CreateTopic`
**Root Cause**: Method signature doesn't match actual API
**Solution**: Update calls to use correct parameters

### Error 6: Kafka Brokers Field Undefined
**Location**: `kafka_advanced.go:99, 193, 240`
**Error**: `kam.cluster.Brokers undefined`
**Root Cause**: Field doesn't exist on KafkaCluster type
**Solution**: Use correct method or property to access broker information

## Implementation Strategy

### Phase 1: Kafka API Fixes

#### Step 1.1: Fix CreateTopics Call
- Change from: `k.admin.CreateTopics(ctx, topicConfigs...)`
- Change to: `k.admin.CreateTopics(ctx, topicConfigs)`
- File: `kafka_config.go:56`

#### Step 1.2: Fix TopicMetadata Type
- Investigate Kafka library version
- Either import correct type or define locally
- File: `kafka_config.go:109`

#### Step 1.3: Fix CreateTopic Calls
- Update all calls to use correct signature
- Remove extra parameters (partitions, replication)
- Files: `kafka_advanced.go:43, 81`

#### Step 1.4: Fix Brokers Access
- Replace direct field access with method call
- Or use alternative API to get broker information
- Files: `kafka_advanced.go:99, 193, 240`

### Phase 2: Consul API Fixes

#### Step 2.1: Fix WatchPlan Creation
- Investigate Consul library version
- Use correct API for creating watch plans
- File: `consul_config.go:118`

### Phase 3: Encryption Fixes

#### Step 3.1: Fix Type Conversions
- Convert string to []byte before GCM operations
- Use `[]byte(data)` for string to bytes conversion
- Use `string(data)` for bytes to string conversion
- File: `config_manager.go:186-187`

### Phase 4: Verification

#### Step 4.1: Build Package
```bash
go build ./pkg/infrastructure/config/...
```

#### Step 4.2: Run golangci-lint
```bash
golangci-lint run ./pkg/infrastructure/config/...
```

#### Step 4.3: Build Full Project
```bash
go build ./...
```

## Files to Modify

1. **pkg/infrastructure/config/kafka_config.go**
   - Fix CreateTopics call (line 56)
   - Fix TopicMetadata reference (line 109)

2. **pkg/infrastructure/config/kafka_advanced.go**
   - Fix CreateTopic calls (lines 43, 81)
   - Fix Brokers field access (lines 99, 193, 240)

3. **pkg/infrastructure/config/consul_config.go**
   - Fix WatchPlan creation (line 118)

4. **pkg/infrastructure/config/config_manager.go**
   - Fix encryption type conversions (lines 186-187)

## Correctness Properties

1. **API Compatibility**: All method calls match the actual library API
2. **Type Safety**: All type conversions are explicit and correct
3. **Compilation**: Code compiles without errors
4. **Functionality**: Existing functionality is preserved

## Risk Assessment

- **Low Risk**: These are straightforward API fixes
- **No Breaking Changes**: Changes are internal to the config package
- **No Data Loss**: Type conversions preserve data integrity

## Testing Strategy

1. **Unit Tests**: Verify config creation works correctly
2. **Integration Tests**: Verify Kafka and Consul connections work
3. **Build Tests**: Verify full project builds successfully

## Success Metrics

- [ ] 0 typecheck errors in pkg/infrastructure/config
- [ ] `go build ./pkg/infrastructure/config/...` succeeds
- [ ] `go build ./...` succeeds
- [ ] All existing tests pass
- [ ] No new warnings introduced
