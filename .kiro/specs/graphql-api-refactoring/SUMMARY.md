# GraphQL API Refactoring - Specification Summary

**Status**: Ready for Implementation  
**Date**: January 9, 2026  
**Estimated Duration**: 6-7 days

## Quick Overview

This specification outlines the refactoring of the GraphQL API from a tightly-coupled plugin architecture to a service-oriented architecture with clear separation of concerns.

## Problem

The current GraphQL implementation has architectural conflicts:
- Not a true plugin (doesn't implement `APIPlugin` interface)
- Tightly coupled to database, cache, and indexers
- Inconsistent package structure
- Difficult to test and maintain

## Solution

Introduce a **Query Service Layer** that abstracts data access, allowing both GraphQL and REST APIs to use the same interface.

```
HTTP Requests
    ↓
┌───────────────────┐
│ GraphQL │ REST    │
└────┬────────┬─────┘
     │        │
     └───┬────┘
         ↓
    Query Service
         ↓
    ┌────┴────┐
    ↓         ↓
Database    Cache
```

## Key Components

### 1. Query Service Layer
- **Location**: `pkg/services/query/`
- **Purpose**: Unified data access interface
- **Features**: Caching, error handling, metrics

### 2. GraphQL Service
- **Location**: `pkg/plugins/graphql/`
- **Purpose**: GraphQL query execution
- **Change**: Uses query service instead of direct database access

### 3. REST API Plugin
- **Location**: `pkg/plugins/api/rest_server.go`
- **Purpose**: REST API implementation
- **Feature**: Implements `APIPlugin` interface

### 4. HTTP Server
- **Location**: `pkg/plugins/api/http_server.go`
- **Purpose**: Unified HTTP endpoint
- **Feature**: Routes both GraphQL and REST requests

## Implementation Phases

### Phase 1: Query Service Layer (2 days)
- Create query service interface
- Implement default query service
- Write comprehensive tests

### Phase 2: GraphQL Refactoring (1-2 days)
- Move GraphQL files to new package
- Update resolver to use query service
- Update tests

### Phase 3: REST API Plugin (1 day)
- Create REST API plugin
- Implement endpoints
- Write tests

### Phase 4: HTTP Server Integration (1 day)
- Create unified HTTP server
- Implement routing
- Write tests

### Phase 5: Testing & Documentation (1-2 days)
- Integration tests
- Performance testing
- Documentation updates

## Success Criteria

✅ All existing tests pass  
✅ Code coverage >80%  
✅ No breaking changes  
✅ Performance maintained  
✅ Clear separation of concerns  
✅ Improved maintainability  

## Files to Create

```
pkg/services/query/
├── query_service.go              # Interface
├── query_service_impl.go         # Implementation
├── types.go                      # Data types
├── query_service_test.go         # Unit tests
└── query_service_property_test.go # Property tests

pkg/plugins/graphql/
├── server.go                     # Moved from api/
├── resolver.go                   # Moved from api/
├── schema.go                     # Moved from api/
├── server_test.go                # Moved from api/
└── resolver_test.go              # Moved from api/

pkg/plugins/api/
├── rest_server.go                # New REST plugin
├── http_server.go                # New HTTP server
├── rest_server_test.go           # New tests
└── http_server_test.go           # New tests
```

## Files to Modify

```
pkg/plugins/api/
├── api_plugin.go                 # No changes (interface stays same)
└── (remove graphql_*.go files)   # Move to graphql/

cmd/chainpulse/main.go            # Update initialization
```

## Key Design Decisions

1. **Query Service as Interface**: Allows easy mocking and testing
2. **Caching at Service Level**: Unified cache strategy
3. **GraphQL as Independent Service**: Not tied to plugin system
4. **REST as Plugin**: Maintains plugin architecture
5. **HTTP Server for Routing**: Single entry point for both APIs

## Backward Compatibility

- ✅ Existing GraphQL queries unchanged
- ✅ New REST API is additive
- ✅ No breaking changes to data structures
- ✅ Gradual migration path for clients

## Performance Considerations

- Cache TTL: 5 minutes for metadata, 1 minute for events
- Pagination: Default 20, max 100 items
- Concurrent access: Thread-safe with RWMutex
- Metrics: Track cache hits/misses, query times

## Testing Strategy

- **Unit Tests**: Query service with mocked dependencies
- **Integration Tests**: With real database/cache
- **Property Tests**: Caching, pagination, concurrency
- **End-to-End Tests**: Full HTTP request/response cycle

## Documentation Updates

- Architecture documentation
- API documentation
- Migration guide
- Troubleshooting guide
- Code examples

## Next Steps

1. Review and approve specification
2. Begin Phase 1: Query Service Layer
3. Follow implementation phases sequentially
4. Run tests at each phase
5. Update documentation as you go

## References

- `docs/progress/GRAPHQL_API_ARCHITECTURE_REFACTORING.md` - Detailed analysis
- `docs/guides/GRAPHQL_API_REFACTORING_IMPLEMENTATION.md` - Implementation guide
- `pkg/plugins/api/api_plugin.go` - API plugin interface
- `pkg/core/plugin.go` - Plugin system

## Questions?

Refer to the detailed design document for architecture details, or the tasks document for specific implementation steps.
