# Requirements: Infrastructure Config Compilation Fixes

## Introduction

The `pkg/infrastructure/config/` package has 10+ typecheck errors that prevent the project from compiling. These errors stem from:

1. **Kafka API Issues**: Incorrect method signatures and undefined types
2. **Consul API Issues**: Deprecated or undefined API calls
3. **Encryption Issues**: Type mismatches in encryption operations
4. **Missing Type Definitions**: Undefined types referenced in code

## Glossary

- **Kafka Admin API**: Kafka cluster administration interface
- **Consul API**: HashiCorp Consul service discovery API
- **TopicMetadata**: Kafka topic metadata structure
- **WatchPlan**: Consul watch plan for monitoring key changes
- **GCM**: Galois/Counter Mode encryption cipher

## Requirements

### Requirement 1: Fix Kafka CreateTopics API Usage

**User Story:** As a developer, I want Kafka topic creation to use the correct API, so that the code compiles and topics are created properly.

#### Acceptance Criteria

1. WHEN CreateTopics is called on Kafka admin, THEN it SHALL accept a slice of TopicConfig, not variadic arguments
2. WHEN topic configuration is created, THEN it SHALL use the correct TopicConfig structure
3. THE method call SHALL match the actual Kafka admin library API
4. WHEN topics are created, THEN the operation SHALL succeed without type errors

### Requirement 2: Fix Kafka TopicMetadata Type Reference

**User Story:** As a developer, I want TopicMetadata to be properly defined or imported, so that code referencing it compiles.

#### Acceptance Criteria

1. WHEN TopicMetadata is referenced in code, THEN it SHALL be defined or properly imported
2. THE type SHALL be available from the Kafka library or defined locally
3. WHEN metadata is accessed, THEN all fields SHALL be properly typed
4. THE type definition SHALL match the Kafka library version being used

### Requirement 3: Fix Kafka CreateTopic Method Signature

**User Story:** As a developer, I want CreateTopic method calls to match the actual API, so that the code compiles.

#### Acceptance Criteria

1. WHEN CreateTopic is called with parameters, THEN the method signature SHALL match the actual Kafka cluster API
2. THE method SHALL accept only the required parameters (context and topic name)
3. WHEN called with extra parameters, THEN the code SHALL be updated to use the correct signature
4. THE method call SHALL succeed without type errors

### Requirement 4: Fix Kafka Brokers Field Access

**User Story:** As a developer, I want to access broker information from KafkaCluster, so that the code compiles.

#### Acceptance Criteria

1. WHEN accessing Brokers field on KafkaCluster, THEN the field SHALL exist or be replaced with correct API
2. THE broker information SHALL be retrievable through the correct method or property
3. WHEN broker data is accessed, THEN it SHALL be properly typed
4. THE access pattern SHALL match the Kafka library API

### Requirement 5: Fix Consul WatchPlan API

**User Story:** As a developer, I want to use the correct Consul API for watching key changes, so that the code compiles.

#### Acceptance Criteria

1. WHEN creating a watch plan in Consul, THEN the API call SHALL use the correct function name
2. THE function SHALL be available from the Consul library or defined locally
3. WHEN watch plan is created, THEN it SHALL properly monitor the specified key
4. THE API call SHALL match the Consul library version being used

### Requirement 6: Fix Encryption Type Mismatches

**User Story:** As a developer, I want encryption operations to use correct types, so that the code compiles.

#### Acceptance Criteria

1. WHEN encrypting data, THEN the ciphertext type SHALL match what GCM.Seal returns ([]byte)
2. WHEN decrypting data, THEN the input type SHALL match what GCM.Open expects ([]byte)
3. THE type conversions between string and []byte SHALL be explicit and correct
4. WHEN encryption/decryption operations complete, THEN the result types SHALL be correct

### Requirement 7: Verify Clean Compilation

**User Story:** As a developer, I want the infrastructure/config package to compile without errors, so that the full project builds.

#### Acceptance Criteria

1. WHEN golangci-lint runs on pkg/infrastructure/config, THEN it SHALL report 0 typecheck errors
2. WHEN the package is built, THEN it SHALL compile successfully
3. WHEN the full project builds, THEN all dependencies SHALL resolve correctly
4. WHEN tests are executed, THEN they SHALL run without compilation errors

## Implementation Approach

1. **Phase 1**: Analyze Kafka API and fix method signatures
2. **Phase 2**: Fix Consul API calls
3. **Phase 3**: Fix encryption type mismatches
4. **Phase 4**: Verify clean compilation

## Success Criteria

- [ ] All Kafka API calls use correct signatures
- [ ] All Consul API calls use correct functions
- [ ] All encryption operations use correct types
- [ ] `golangci-lint run ./pkg/infrastructure/config/...` returns 0 typecheck errors
- [ ] `go build ./pkg/infrastructure/config/...` succeeds
- [ ] Full project builds: `go build ./...`
