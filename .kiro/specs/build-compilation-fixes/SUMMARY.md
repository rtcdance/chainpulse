# Build Compilation Fixes - Spec Summary

## Status: Ready for Implementation

This spec provides a comprehensive plan to fix all compilation errors in the codebase.

## Key Issues Addressed

### 1. Missing Type Definitions
- **ConsulConfig**: Configuration for Consul service discovery
- **KafkaConfig**: Configuration for Kafka message queue
- **HealthStatus**: Health status representation
- **ServiceInfo**: Service metadata and information

### 2. BlockchainEvent Field Mismatches
- Incorrect field references (EventID → ID, Timestamp → BlockTimestamp)
- Wrong BSON field names in MongoDB documents
- Type mismatches in conversions

### 3. MongoDB Driver API Issues
- Deprecated SetExpireAfter usage
- Incorrect parameter types for TTL index creation

### 4. Type Conversion Errors
- chainId: int32 → string
- blockNumber: int64 → uint64
- transactionHash: string → common.Hash
- contractAddress: string → common.Address

### 5. Import Chain Issues
- Query package dependencies not available
- Health status and service info types undefined

## Implementation Approach

1. **Phase 1**: Define missing infrastructure types (Task 1)
2. **Phase 2**: Fix MongoDB event store (Tasks 2-4)
3. **Phase 3**: Fix infrastructure and health references (Tasks 6-8)
4. **Phase 4**: Fix API service imports (Task 9)
5. **Phase 5**: Verification and testing (Tasks 10-13)

## Expected Outcomes

- ✅ All compilation errors resolved
- ✅ MongoDB event store properly typed
- ✅ Infrastructure types properly defined
- ✅ Import chains resolve correctly
- ✅ Full build succeeds: `go build ./...`

## Files to Create/Modify

### New Files
- `pkg/infrastructure/deployment/consul_config.go`
- `pkg/infrastructure/deployment/kafka_config.go`
- `pkg/infrastructure/deployment/types.go`

### Modified Files
- `pkg/services/query/mongodb_event_store.go`
- `pkg/infrastructure/deployment/initializer.go`
- `pkg/infrastructure/data/data_puller.go`
- `pkg/infrastructure/processing/event_processor.go`
- `pkg/infrastructure/processing/event_processor_cluster.go`
- `cmd/microservices/api-service/main.go`

## Correctness Properties

1. **Type Conversion Consistency**: BSON ↔ Go conversions preserve data
2. **Field Name Mapping**: BSON fields match struct tags
3. **MongoDB Index Creation**: Indexes created successfully
4. **Infrastructure Type Availability**: All types available and instantiated
5. **Import Chain Resolution**: All dependencies resolve correctly

## Next Steps

1. Start with Task 1: Define missing infrastructure types
2. Follow the task list sequentially
3. Run checkpoints after each phase
4. Complete optional tests if desired
5. Final verification with full build and test suite

## Estimated Effort

- **Core fixes**: 2-3 hours
- **With optional tests**: 4-5 hours
- **Total tasks**: 13 (10 required, 3 optional)

## Success Criteria

- [ ] All compilation errors resolved
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes (if tests included)
- [ ] No type errors or warnings
- [ ] All microservices compile successfully
