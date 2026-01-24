# Implementation Plan: Infrastructure Config Compilation Fixes

## Overview

Fix 10+ typecheck errors in `pkg/infrastructure/config/` package by correcting Kafka API calls, Consul API usage, and encryption type conversions.

## Tasks

- [ ] 1. Fix Kafka CreateTopics API Call
  - Remove variadic operator from CreateTopics call
  - Pass topicConfigs slice directly instead of unpacking
  - Verify method signature matches Kafka admin library
  - _Requirements: 1.1, 1.2, 1.3_

- [ ] 2. Fix Kafka TopicMetadata Type Reference
  - Investigate Kafka library version and available types
  - Either import correct TopicMetadata type or define locally
  - Update all references to use correct type
  - _Requirements: 1.2, 1.3_

- [ ] 3. Fix Kafka CreateTopic Method Calls
  - Update all CreateTopic calls to use correct signature
  - Remove extra parameters (partitions, replication) if not supported
  - Verify calls match actual Kafka cluster API
  - _Requirements: 1.3, 1.4_

- [ ] 4. Fix Kafka Brokers Field Access
  - Replace direct Brokers field access with correct method/property
  - Investigate KafkaCluster type for correct API
  - Update all three locations (lines 99, 193, 240)
  - _Requirements: 1.4_

- [ ] 5. Fix Consul WatchPlan API
  - Investigate Consul library version and available APIs
  - Replace api.NewWatchPlan with correct function
  - Verify watch plan creation works correctly
  - _Requirements: 2.1_

- [ ] 6. Fix Encryption Type Conversions
  - Convert string to []byte before GCM.Seal operation
  - Convert []byte to string after GCM.Open operation
  - Verify encryption/decryption operations work correctly
  - _Requirements: 3.1, 3.2_

- [ ] 7. Verify Clean Compilation
  - Run `go build ./pkg/infrastructure/config/...`
  - Run `golangci-lint run ./pkg/infrastructure/config/...`
  - Verify 0 typecheck errors
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 8. Verify Full Project Build
  - Run `go build ./...`
  - Verify full project compiles without errors
  - Check for any new errors introduced
  - _Requirements: 4.3, 4.4_

## Task Dependencies

```
Task 1 → Task 2 → Task 3 → Task 4 → Task 7
Task 5 → Task 7
Task 6 → Task 7
Task 7 → Task 8
```

## Estimated Effort

- Task 1: 15 minutes
- Task 2: 20 minutes
- Task 3: 20 minutes
- Task 4: 20 minutes
- Task 5: 15 minutes
- Task 6: 15 minutes
- Task 7: 10 minutes
- Task 8: 10 minutes

**Total**: ~2 hours

## Success Criteria

- [ ] All Kafka API calls use correct signatures
- [ ] All Consul API calls use correct functions
- [ ] All encryption operations use correct types
- [ ] `golangci-lint run ./pkg/infrastructure/config/...` returns 0 typecheck errors
- [ ] `go build ./...` succeeds
- [ ] No new warnings introduced
- [ ] All existing tests pass
