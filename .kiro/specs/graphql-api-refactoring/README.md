# GraphQL API Refactoring Specification

## Overview

This specification defines the complete refactoring of the ChainPulse GraphQL API from a tightly-coupled plugin architecture to a service-oriented architecture with clear separation of concerns.

## Documents

### 1. **SUMMARY.md** - Start Here
Quick overview of the problem, solution, and implementation plan. Read this first to understand the big picture.

### 2. **requirements.md** - User Stories & Acceptance Criteria
Detailed user stories, acceptance criteria, and success metrics. Use this to understand what needs to be built.

### 3. **design.md** - Architecture & Technical Details
Complete architecture design, package structure, interfaces, data types, and implementation details. Use this as a reference during implementation.

### 4. **tasks.md** - Step-by-Step Implementation
Detailed task breakdown organized by phase. Use this as a checklist during implementation.

## Quick Start

1. **Understand the Problem**: Read `SUMMARY.md`
2. **Review Requirements**: Read `requirements.md`
3. **Study the Design**: Read `design.md`
4. **Follow the Tasks**: Use `tasks.md` as your implementation guide

## Key Concepts

### Current Architecture (Problems)
```
GraphQL Plugin
    ↓
Direct Database Access
    ↓
Tight Coupling
```

### New Architecture (Solution)
```
GraphQL Service + REST Plugin
    ↓
Query Service Layer
    ↓
Database + Cache
    ↓
Loose Coupling
```

## Implementation Phases

| Phase | Duration | Focus |
|-------|----------|-------|
| 1 | 2 days | Query Service Layer |
| 2 | 1-2 days | GraphQL Refactoring |
| 3 | 1 day | REST API Plugin |
| 4 | 1 day | HTTP Server Integration |
| 5 | 1-2 days | Testing & Documentation |

**Total**: 6-7 days

## Success Criteria

- ✅ All tests pass
- ✅ Code coverage >80%
- ✅ No breaking changes
- ✅ Performance maintained
- ✅ Clear separation of concerns
- ✅ Improved maintainability

## Key Files to Create

```
pkg/services/query/
├── query_service.go
├── query_service_impl.go
├── types.go
├── query_service_test.go
└── query_service_property_test.go

pkg/plugins/graphql/
├── server.go
├── resolver.go
├── schema.go
├── server_test.go
└── resolver_test.go

pkg/plugins/api/
├── rest_server.go
├── http_server.go
├── rest_server_test.go
└── http_server_test.go
```

## Key Files to Move

```
pkg/plugins/api/graphql_server.go → pkg/plugins/graphql/server.go
pkg/plugins/api/graphql_resolver.go → pkg/plugins/graphql/resolver.go
pkg/plugins/api/graphql_schema.go → pkg/plugins/graphql/schema.go
```

## Implementation Checklist

### Phase 1: Query Service
- [ ] Create query service interface
- [ ] Implement default query service
- [ ] Write unit tests
- [ ] Write property tests
- [ ] All tests pass

### Phase 2: GraphQL Refactoring
- [ ] Move GraphQL files
- [ ] Update resolver to use query service
- [ ] Update tests
- [ ] All tests pass

### Phase 3: REST API Plugin
- [ ] Create REST plugin
- [ ] Implement endpoints
- [ ] Write tests
- [ ] All tests pass

### Phase 4: HTTP Server
- [ ] Create HTTP server
- [ ] Implement routing
- [ ] Write tests
- [ ] All tests pass

### Phase 5: Testing & Documentation
- [ ] Integration tests
- [ ] Performance tests
- [ ] Update documentation
- [ ] All tests pass

## Important Notes

1. **Backward Compatibility**: Maintain all existing GraphQL queries
2. **No Breaking Changes**: REST API is additive
3. **Incremental**: Complete each phase before moving to the next
4. **Testing**: Write tests as you go
5. **Documentation**: Update docs at each phase

## References

- `docs/progress/GRAPHQL_API_ARCHITECTURE_REFACTORING.md` - Detailed analysis
- `docs/guides/GRAPHQL_API_REFACTORING_IMPLEMENTATION.md` - Implementation guide
- `pkg/plugins/api/api_plugin.go` - API plugin interface
- `pkg/core/plugin.go` - Plugin system

## Questions?

- For architecture questions: See `design.md`
- For implementation questions: See `tasks.md`
- For requirements questions: See `requirements.md`
- For overview: See `SUMMARY.md`

## Status

**Current**: Specification Complete  
**Next**: Begin Phase 1 Implementation

---

**Created**: January 9, 2026  
**Last Updated**: January 9, 2026  
**Status**: Ready for Implementation
