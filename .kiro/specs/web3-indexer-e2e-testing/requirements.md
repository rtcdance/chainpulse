# Requirements: Web3 Indexer E2E Testing Framework

## Introduction

This specification defines a comprehensive end-to-end testing framework for the ChainPulse blockchain event indexer. The framework integrates industry-standard Web3 testing tools (Anvil, Hardhat) with Go testing libraries (testify, gopter) to validate the complete indexing pipeline from blockchain event emission through API query responses.

## Glossary

- **Anvil**: Ethereum test node that simulates blockchain behavior with deterministic block production
- **Hardhat**: JavaScript development environment for deploying and interacting with smart contracts
- **E2E Test**: End-to-end test that validates complete workflows from data source to API response
- **Test Fixture**: Pre-configured test data including smart contracts, events, and blockchain state
- **Indexer**: The ChainPulse system that collects, processes, and stores blockchain events
- **Event Pipeline**: Complete flow from blockchain event emission → data puller → processor → storage → API query
- **Property-Based Test**: Test that validates universal properties across many generated inputs
- **Testcontainers**: Docker-based test infrastructure for running databases and services
- **Mock Blockchain**: Simulated blockchain environment for deterministic testing

## Requirements

### Requirement 1: Anvil-Based Test Environment Setup

**User Story:** As a test engineer, I want to set up a deterministic blockchain test environment, so that I can reliably test indexer behavior with controlled blockchain state.

#### Acceptance Criteria

1. WHEN the E2E test suite starts, THE test framework SHALL initialize an Anvil instance with deterministic block production
2. WHEN Anvil is initialized, THE test framework SHALL deploy standard ERC20 and ERC721 contracts for testing
3. WHEN test contracts are deployed, THE test framework SHALL capture contract addresses and ABIs for event monitoring
4. WHEN a test completes, THE test framework SHALL clean up Anvil resources and reset blockchain state
5. WHEN Anvil encounters errors, THE test framework SHALL log detailed error information and fail gracefully

### Requirement 2: Smart Contract Event Emission

**User Story:** As a test engineer, I want to emit blockchain events from test contracts, so that I can validate the indexer's event collection capabilities.

#### Acceptance Criteria

1. WHEN a test needs to emit events, THE test framework SHALL provide functions to trigger ERC20 Transfer events
2. WHEN a test needs to emit events, THE test framework SHALL provide functions to trigger ERC721 Mint/Burn events
3. WHEN events are emitted, THE test framework SHALL capture transaction hashes and log indices for verification
4. WHEN multiple events are emitted in a single transaction, THE test framework SHALL preserve event ordering
5. WHEN events are emitted across multiple blocks, THE test framework SHALL maintain block number accuracy

### Requirement 3: Data Puller Integration

**User Story:** As a test engineer, I want to validate that the data puller correctly collects events from the test blockchain, so that I can ensure event collection reliability.

#### Acceptance Criteria

1. WHEN the data puller connects to Anvil, THE indexer SHALL successfully retrieve blockchain events
2. WHEN events are emitted on the test blockchain, THE data puller SHALL detect and collect them within 5 seconds
3. WHEN the data puller encounters network errors, THE indexer SHALL implement retry logic with exponential backoff
4. WHEN multiple events are emitted, THE data puller SHALL collect all events without loss or duplication
5. WHEN blockchain reorganization occurs, THE data puller SHALL handle reorg detection and recovery

### Requirement 4: Event Processing Pipeline

**User Story:** As a test engineer, I want to validate the complete event processing pipeline, so that I can ensure events are correctly decoded and stored.

#### Acceptance Criteria

1. WHEN events are collected by the data puller, THE event processor SHALL decode event data according to contract ABI
2. WHEN events are decoded, THE processor SHALL extract indexed and non-indexed parameters correctly
3. WHEN events are processed, THE processor SHALL detect and prevent duplicate event storage
4. WHEN events are processed, THE processor SHALL maintain event ordering and block number accuracy
5. WHEN processing encounters errors, THE processor SHALL log errors and continue processing subsequent events

### Requirement 5: Database Persistence Validation

**User Story:** As a test engineer, I want to validate that processed events are correctly persisted, so that I can ensure data integrity.

#### Acceptance Criteria

1. WHEN events are processed, THE database plugin SHALL store events with all required fields (block number, tx hash, log index, event data)
2. WHEN events are stored, THE database SHALL maintain referential integrity and prevent duplicate storage
3. WHEN querying stored events, THE database SHALL return events in correct block order with accurate timestamps
4. WHEN database connections fail, THE indexer SHALL implement connection pooling and retry logic
5. WHEN database transactions fail, THE indexer SHALL implement rollback and recovery mechanisms

### Requirement 6: API Query Validation

**User Story:** As a test engineer, I want to validate that the API correctly returns indexed events, so that I can ensure query accuracy.

#### Acceptance Criteria

1. WHEN querying events via REST API, THE API SHALL return events matching the query filter (contract address, event type)
2. WHEN querying events, THE API SHALL support pagination with limit and offset parameters
3. WHEN querying events, THE API SHALL return events in correct chronological order
4. WHEN querying events, THE API SHALL include all event data fields in the response
5. WHEN API encounters errors, THE API SHALL return appropriate HTTP status codes and error messages

### Requirement 7: Multi-Chain Indexing

**User Story:** As a test engineer, I want to validate multi-chain indexing capabilities, so that I can ensure the indexer handles multiple blockchains correctly.

#### Acceptance Criteria

1. WHEN multiple Anvil instances are running, THE indexer SHALL maintain separate event collections per chain
2. WHEN events are emitted on different chains, THE indexer SHALL correctly tag events with chain identifiers
3. WHEN querying events, THE API SHALL support filtering by chain identifier
4. WHEN one chain experiences issues, THE indexer SHALL continue indexing other chains without interruption
5. WHEN chains have different block times, THE indexer SHALL handle timing differences correctly

### Requirement 8: Concurrent Event Processing

**User Story:** As a test engineer, I want to validate concurrent event processing, so that I can ensure the indexer handles high-throughput scenarios.

#### Acceptance Criteria

1. WHEN multiple events are emitted concurrently, THE indexer SHALL process all events without loss or corruption
2. WHEN processing concurrent events, THE indexer SHALL maintain event ordering within each block
3. WHEN processing concurrent events, THE indexer SHALL prevent race conditions in database writes
4. WHEN processing concurrent events, THE indexer SHALL maintain accurate metrics and counters
5. WHEN concurrent processing encounters errors, THE indexer SHALL implement proper error handling and recovery

### Requirement 9: Performance and Latency Validation

**User Story:** As a test engineer, I want to validate indexer performance, so that I can ensure it meets production requirements.

#### Acceptance Criteria

1. WHEN events are emitted, THE indexer SHALL index them within 2 seconds (end-to-end latency)
2. WHEN processing events, THE indexer SHALL maintain throughput of at least 1000 events per second
3. WHEN querying indexed events, THE API SHALL respond within 500ms for queries returning up to 1000 events
4. WHEN under load, THE indexer SHALL maintain memory usage below 500MB for 100,000 indexed events
5. WHEN processing events, THE indexer SHALL maintain CPU usage below 80% under normal load

### Requirement 10: Error Handling and Recovery

**User Story:** As a test engineer, I want to validate error handling and recovery mechanisms, so that I can ensure the indexer is resilient.

#### Acceptance Criteria

1. WHEN blockchain connection fails, THE indexer SHALL implement exponential backoff retry with maximum 5 retries
2. WHEN database connection fails, THE indexer SHALL queue events and retry after connection recovery
3. WHEN event processing fails, THE indexer SHALL log the error and continue processing subsequent events
4. WHEN API encounters errors, THE API SHALL return appropriate error responses without crashing
5. WHEN the indexer crashes, THE indexer SHALL recover from the last checkpoint and resume indexing

### Requirement 11: Test Fixture Management

**User Story:** As a test engineer, I want to manage test fixtures efficiently, so that I can reduce test setup time and improve test reliability.

#### Acceptance Criteria

1. WHEN a test starts, THE test framework SHALL provide pre-configured smart contracts and ABIs
2. WHEN a test needs specific blockchain state, THE test framework SHALL support state snapshots and restoration
3. WHEN tests complete, THE test framework SHALL clean up resources and reset state for subsequent tests
4. WHEN multiple tests run in parallel, THE test framework SHALL isolate test environments to prevent interference
5. WHEN test fixtures are reused, THE test framework SHALL validate fixture integrity before use

### Requirement 12: Monitoring and Observability

**User Story:** As a test engineer, I want to monitor test execution and collect metrics, so that I can validate system behavior and performance.

#### Acceptance Criteria

1. WHEN tests execute, THE test framework SHALL collect metrics on event processing latency
2. WHEN tests execute, THE test framework SHALL collect metrics on throughput (events per second)
3. WHEN tests execute, THE test framework SHALL collect metrics on error rates and recovery times
4. WHEN tests complete, THE test framework SHALL generate reports with performance metrics and test results
5. WHEN tests fail, THE test framework SHALL capture detailed logs and state information for debugging

## Testing Strategy

### Test Levels

1. **Unit Tests**: Validate individual components (decoders, validators, processors)
2. **Integration Tests**: Validate component interactions (puller + processor, processor + database)
3. **E2E Tests**: Validate complete workflows (blockchain → indexer → API)
4. **Property-Based Tests**: Validate universal properties across many generated scenarios

### Test Environment

- **Anvil**: Deterministic blockchain for event emission
- **Testcontainers**: PostgreSQL, MongoDB, Redis for persistence and caching
- **Mock Services**: Simulated blockchain nodes for failure scenarios
- **Test Fixtures**: Pre-configured contracts and test data

### Test Data

- **ERC20 Contracts**: Standard token contracts for Transfer events
- **ERC721 Contracts**: NFT contracts for Mint/Burn events
- **Custom Contracts**: Application-specific contracts for domain events
- **Event Scenarios**: Normal operation, edge cases, error conditions

### Success Criteria

- All E2E tests pass consistently
- Event collection latency < 2 seconds
- Event processing throughput > 1000 events/second
- API query latency < 500ms
- Zero event loss or duplication
- Proper error handling and recovery
- Memory usage < 500MB for 100,000 events
