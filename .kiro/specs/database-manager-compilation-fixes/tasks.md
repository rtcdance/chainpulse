# Implementation Plan: Database Manager Compilation Fixes

## Overview

Fix all compilation errors in the database manager tests by updating method calls to use correct signatures and implementing proper gopter reporter patterns.

## Tasks

- [x] 1. Fix manager_test.go GetMongoClient calls
  - Update TestGetMongoClientBeforeInitialize to create context and handle error return
  - Update TestGetMongoDatabaseBeforeInitialize to create context
  - Add proper error assertions for all GetMongoClient calls
  - _Requirements: 1.1, 1.2, 1.3_

- [ ] 2. Fix manager_test.go GetPostgresDB calls
  - Update TestGetPostgresDBBeforeInitialize to create context and handle error return
  - Add proper error assertions for all GetPostgresDB calls
  - Ensure all calls include context.Context parameter
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 3. Create gopter reporter wrapper
  - Create testReporter struct that implements gopter.Reporter
  - Implement ReportTestResult method to report to *testing.T
  - Add helper function to create reporter from *testing.T
  - _Requirements: 3.1, 3.2_

- [ ] 4. Fix manager_property_test.go reporter usage
  - Update TestConnectionPoolReuse to use proper reporter
  - Update TestHealthCheckAccuracy to use proper reporter
  - Update TestDatabaseManagerConcurrency to use proper reporter
  - Update TestDatabaseManagerStateTransitions to use proper reporter
  - Update TestDatabaseManagerConfigurationVariations to use proper reporter
  - _Requirements: 3.2, 3.3_

- [ ] 5. Fix manager_property_test.go method calls
  - Update all GetMongoClient calls to include context.Context
  - Update all GetPostgresDB calls to include context.Context
  - Handle error returns from these methods
  - Add proper error assertions in property tests
  - _Requirements: 1.1, 2.1, 4.1_

- [ ] 6. Verify compilation
  - Run `go build ./pkg/infrastructure/database/...` to verify no compilation errors
  - Run `golangci-lint run ./pkg/infrastructure/database/... --timeout=5m` to verify linting passes
  - Ensure all tests compile without errors
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 7. Run tests
  - Execute `go test ./pkg/infrastructure/database/... -v` to run all tests
  - Verify all unit tests pass
  - Verify all property tests pass with at least 10 iterations
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 8. Checkpoint - Verify all fixes complete
  - Ensure no compilation errors remain
  - Ensure all tests pass
  - Ensure golangci-lint reports no issues
  - Ask the user if questions arise

