# GraphQL API Refactoring - Requirements

**Status**: In Progress  
**Date**: January 9, 2026  
**Phase**: Design & Implementation

## Problem Statement

The current GraphQL API implementation has architectural conflicts:
- GraphQL is treated as a plugin but doesn't implement the `APIPlugin` interface
- Tight coupling between GraphQL, database, cache, and indexers
- Package structure is inconsistent (`api_plugin.go` in core, `graphql_server.go` in api)
- Difficult to test and maintain
- No clear separation of concerns

## Solution Overview

Refactor the GraphQL API from a tightly-coupled plugin to a service-oriented architecture with:
1. **Query Service Layer** - Unified data access interface
2. **GraphQL Service** - Independent service (not a plugin)
3. **REST API Plugin** - Implements `APIPlugin` interface
4. **HTTP Server** - Routes REST and GraphQL requests

## User Stories

### US1: Create Query Service Layer
**As a** developer  
**I want** a unified query service that abstracts database and cache access  
**So that** both GraphQL and REST APIs can use the same data access layer

**Acceptance Criteria:**
- Query service interface defined with all query methods
- Default implementation with caching support
- Support for events, tokens, pools, and contracts
- Proper error handling and logging

### US2: Refactor GraphQL Service
**As a** developer  
**I want** GraphQL to use the query service instead of direct database access  
**So that** GraphQL is decoupled from the database layer

**Acceptance Criteria:**
- GraphQL resolver uses query service
- GraphQL server is independent of plugin system
- All existing GraphQL queries still work
- Tests pass for all GraphQL operations

### US3: Create REST API Plugin
**As a** developer  
**I want** a REST API plugin that implements the `APIPlugin` interface  
**So that** REST API is properly integrated with the plugin system

**Acceptance Criteria:**
- REST plugin implements `APIPlugin` interface
- REST plugin uses query service
- Basic REST endpoints for events, tokens, pools
- Proper HTTP status codes and error responses

### US4: Integrate HTTP Server
**As a** developer  
**I want** a unified HTTP server that handles both REST and GraphQL  
**So that** both APIs can run on the same port

**Acceptance Criteria:**
- HTTP server routes `/graphql` to GraphQL handler
- HTTP server routes `/api/*` to REST handler
- Both endpoints work simultaneously
- Proper request/response handling

## Implementation Plan

### Phase 1: Query Service Layer (Day 1-2)
1. Create `pkg/services/query/query_service.go` with interface
2. Create `pkg/services/query/query_service_impl.go` with implementation
3. Create `pkg/services/query/query_service_test.go` with unit tests
4. Create `pkg/services/query/query_service_property_test.go` with property tests

### Phase 2: GraphQL Refactoring (Day 2-3)
1. Move GraphQL files to `pkg/plugins/graphql/`
2. Update GraphQL resolver to use query service
3. Update GraphQL server to be independent
4. Update all imports and tests

### Phase 3: REST API Plugin (Day 3-4)
1. Create `pkg/plugins/api/rest_server.go`
2. Implement `APIPlugin` interface
3. Create REST endpoints
4. Add tests

### Phase 4: HTTP Server Integration (Day 4-5)
1. Create `pkg/plugins/api/http_server.go`
2. Implement request routing
3. Add middleware support
4. Add tests

### Phase 5: Testing & Documentation (Day 5-6)
1. Run all tests
2. Update documentation
3. Create migration guide
4. Performance testing

## Success Criteria

✅ All existing tests pass  
✅ New tests for query service (>80% coverage)  
✅ GraphQL and REST APIs work simultaneously  
✅ Clear separation of concerns  
✅ Improved code maintainability  
✅ No breaking changes to API contracts  

## Dependencies

- `pkg/core/plugin.go` - Plugin interface
- `pkg/plugins/database/database_plugin.go` - Database interface
- `pkg/plugins/cache/cache_plugin.go` - Cache interface
- `pkg/core/logger.go` - Logger interface
- `pkg/core/metrics.go` - Metrics interface

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Breaking existing GraphQL queries | Maintain backward compatibility in resolver |
| Performance degradation | Add caching at query service level |
| Complex refactoring | Incremental changes with tests at each step |
| Database access issues | Comprehensive integration tests |

## Timeline

- **Day 1-2**: Query Service Layer
- **Day 2-3**: GraphQL Refactoring
- **Day 3-4**: REST API Plugin
- **Day 4-5**: HTTP Server Integration
- **Day 5-6**: Testing & Documentation

**Total**: 6 days (estimated)

## Notes

- All code must follow existing project conventions
- Comprehensive test coverage required
- Documentation must be updated
- No breaking changes to public APIs
