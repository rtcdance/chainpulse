# Infrastructure Config Compilation Fixes - Checklist

## Pre-Implementation

- [ ] Review SUMMARY.md
- [ ] Review requirements.md
- [ ] Review design.md
- [ ] Review GETTING_STARTED.md
- [ ] Understand all 10+ errors
- [ ] Identify all 4 files to modify

## Task 1: Fix Kafka CreateTopics API Call

**File**: `pkg/infrastructure/config/kafka_config.go`  
**Line**: 56

- [ ] Locate the CreateTopics call
- [ ] Remove the `...` operator
- [ ] Pass topicConfigs slice directly
- [ ] Verify syntax is correct
- [ ] Test: `go build ./pkg/infrastructure/config/...`

## Task 2: Fix Kafka TopicMetadata Type Reference

**File**: `pkg/infrastructure/config/kafka_config.go`  
**Line**: 109

- [ ] Investigate Kafka library version
- [ ] Check available types in Kafka library
- [ ] Either import correct type or define locally
- [ ] Update all references to use correct type
- [ ] Test: `go build ./pkg/infrastructure/config/...`

## Task 3: Fix Kafka CreateTopic Method Calls

**File**: `pkg/infrastructure/config/kafka_advanced.go`  
**Lines**: 43, 81

- [ ] Locate first CreateTopic call (line 43)
- [ ] Check method signature
- [ ] Remove extra parameters if not supported
- [ ] Update call to use correct signature
- [ ] Locate second CreateTopic call (line 81)
- [ ] Update call to use correct signature
- [ ] Test: `go build ./pkg/infrastructure/config/...`

## Task 4: Fix Kafka Brokers Field Access

**File**: `pkg/infrastructure/config/kafka_advanced.go`  
**Lines**: 99, 193, 240

- [ ] Locate first Brokers access (line 99)
- [ ] Investigate KafkaCluster type for correct API
- [ ] Replace with correct method/property
- [ ] Locate second Brokers access (line 193)
- [ ] Replace with correct method/property
- [ ] Locate third Brokers access (line 240)
- [ ] Replace with correct method/property
- [ ] Test: `go build ./pkg/infrastructure/config/...`

## Task 5: Fix Consul WatchPlan API

**File**: `pkg/infrastructure/config/consul_config.go`  
**Line**: 118

- [ ] Investigate Consul library version
- [ ] Check available APIs for watch plans
- [ ] Replace api.NewWatchPlan with correct function
- [ ] Verify watch plan creation works
- [ ] Test: `go build ./pkg/infrastructure/config/...`

## Task 6: Fix Encryption Type Conversions

**File**: `pkg/infrastructure/config/config_manager.go`  
**Lines**: 186-187

- [ ] Locate encryption operation (line 186)
- [ ] Convert string to []byte before GCM.Seal
- [ ] Locate decryption operation (line 187)
- [ ] Convert []byte to string after GCM.Open
- [ ] Verify encryption/decryption works
- [ ] Test: `go build ./pkg/infrastructure/config/...`

## Task 7: Verify Clean Compilation

- [ ] Run: `go build ./pkg/infrastructure/config/...`
- [ ] Verify: No compilation errors
- [ ] Run: `golangci-lint run ./pkg/infrastructure/config/...`
- [ ] Verify: 0 typecheck errors
- [ ] Check: No new warnings introduced

## Task 8: Verify Full Project Build

- [ ] Run: `go build ./...`
- [ ] Verify: Full project compiles
- [ ] Check: No new errors introduced
- [ ] Run: `go test ./...` (optional)
- [ ] Verify: All tests pass (if applicable)

## Post-Implementation

- [ ] All tasks completed
- [ ] All verifications passed
- [ ] No new errors introduced
- [ ] No new warnings introduced
- [ ] Documentation updated
- [ ] Ready for next phase

## Error Tracking

### Kafka CreateTopics (Task 1)
- [ ] Error fixed
- [ ] Verified with build

### Kafka TopicMetadata (Task 2)
- [ ] Error fixed
- [ ] Verified with build

### Kafka CreateTopic (Task 3)
- [ ] Error 1 fixed (line 43)
- [ ] Error 2 fixed (line 81)
- [ ] Verified with build

### Kafka Brokers (Task 4)
- [ ] Error 1 fixed (line 99)
- [ ] Error 2 fixed (line 193)
- [ ] Error 3 fixed (line 240)
- [ ] Verified with build

### Consul WatchPlan (Task 5)
- [ ] Error fixed
- [ ] Verified with build

### Encryption Types (Task 6)
- [ ] Error 1 fixed (line 186)
- [ ] Error 2 fixed (line 187)
- [ ] Verified with build

### Final Verification (Tasks 7-8)
- [ ] Package builds clean
- [ ] Full project builds clean
- [ ] 0 typecheck errors
- [ ] 0 new warnings

## Sign-Off

- **Completed By**: _______________
- **Date**: _______________
- **Verified By**: _______________
- **Date**: _______________

## Notes

_Use this space for any notes or issues encountered during implementation_

---

