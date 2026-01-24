# Requirements: Database Manager Compilation Fixes

## Introduction

The database manager tests have compilation errors due to method signature mismatches. The `GetMongoClient()` and `GetPostgresDB()` methods now require a `context.Context` parameter, but the tests are calling them without context. Additionally, the property-based tests are using `*testing.T` directly with gopter, which requires a proper `gopter.Reporter` implementation.

## Glossary

- **DatabaseManager**: Interface for managing MongoDB and PostgreSQL connections
- **DefaultDatabaseManager**: Default implementation of DatabaseManager
- **gopter**: Property-based testing library for Go
- **gopter.Reporter**: Interface for reporting test results in property-based tests
- **context.Context**: Go context for managing timeouts and cancellation

## Requirements

### Requirement 1: Fix GetMongoClient Method Calls

**User Story:** As a developer, I want the database manager tests to compile correctly, so that I can run the test suite without errors.

#### Acceptance Criteria

1. WHEN a test calls `manager.GetMongoClient()` THEN the call SHALL include a `context.Context` parameter
2. WHEN a test receives the result from `GetMongoClient(ctx)` THEN it SHALL handle both the client and error return values
3. WHEN a test calls `GetMongoClient(ctx)` THEN the error handling SHALL check for initialization errors

### Requirement 2: Fix GetPostgresDB Method Calls

**User Story:** As a developer, I want the database manager tests to compile correctly, so that I can run the test suite without errors.

#### Acceptance Criteria

1. WHEN a test calls `manager.GetPostgresDB()` THEN the call SHALL include a `context.Context` parameter
2. WHEN a test receives the result from `GetPostgresDB(ctx)` THEN it SHALL handle both the database and error return values
3. WHEN a test calls `GetPostgresDB(ctx)` THEN the error handling SHALL check for initialization errors

### Requirement 3: Fix Property-Based Test Reporter

**User Story:** As a developer, I want property-based tests to work correctly with gopter, so that I can validate database manager properties.

#### Acceptance Criteria

1. WHEN a property test runs THEN it SHALL use a proper `gopter.Reporter` implementation instead of `*testing.T`
2. WHEN properties are tested THEN the test SHALL properly report results using the gopter framework
3. WHEN a property test fails THEN the failure SHALL be properly reported to the test framework

### Requirement 4: Update All Test Method Calls

**User Story:** As a developer, I want all database manager tests to compile and run, so that I can verify the implementation.

#### Acceptance Criteria

1. WHEN `manager_test.go` is compiled THEN all method calls SHALL use correct signatures
2. WHEN `manager_property_test.go` is compiled THEN all property tests SHALL use correct gopter patterns
3. WHEN tests are executed THEN all compilation errors SHALL be resolved

