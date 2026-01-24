# GraphQL API Refactoring - Tasks

## Architecture Decision: Option A (GraphQL as True Plugin)

GraphQL API will be implemented as a proper plugin implementing the `APIPlugin` interface, with an HTTP Adapter layer to bridge HTTP and plugin interfaces.

---

## Phase 1: Query Service Layer ✅ COMPLETE

### Task 1.1: Create Query Service Interface ✅
**Status**: COMPLETE

### Task 1.2: Implement Query Service ✅
**Status**: COMPLETE

### Task 1.3: Unit Tests for Query Service ✅
**Status**: COMPLETE

### Task 1.4: Property Tests for Query Service ✅
**Status**: COMPLETE

---

## Phase 2: GraphQL Plugin Implementation

### Task 2.1: Create GraphQL Plugin
**File**: `pkg/plugins/api/graphql_plugin.go`

**Deliverables**:
- [ ] GraphQLPlugin struct extending BaseAPIPlugin
- [ ] Constructor function
- [ ] Initialize method
- [ ] Start method
- [ ] Stop method
- [ ] Health method
- [ ] HandleRequest method (converts APIRequest to GraphQL query)
- [ ] GetStats method

**Acceptance Criteria**:
- Implements APIPlugin interface completely
- All methods implemented
- Proper error handling
- Metrics recording
- Code compiles without errors

### Task 2.2: Create HTTP Adapter
**File**: `pkg/plugins/api/http_adapter.go`

**Deliverables**:
- [ ] HTTPAdapter struct
- [ ] Constructor function
- [ ] HandleHTTP method (converts HTTP to APIRequest)
- [ ] Request parsing
- [ ] Response formatting

**Acceptance Criteria**:
- Converts HTTP requests to APIRequest
- Converts APIResponse to HTTP responses
- Proper error handling
- Content-Type handling
- Code compiles without errors

### Task 2.3: Update GraphQL Resolver
**File**: `pkg/plugins/api/graphql_resolver.go`

**Deliverables**:
- [ ] Remove database dependency
- [ ] Remove cache dependency
- [ ] Remove indexer dependencies
- [ ] Add query service dependency
- [ ] Update all query methods to use query service
- [ ] Update resolver constructor
- [ ] Update type conversions

**Acceptance Criteria**:
- All methods use query service
- No direct database access
- No direct cache access
- No direct indexer access
- Type conversions work correctly
- Tests pass
- Code compiles without errors

### Task 2.4: Update GraphQL Server
**File**: `pkg/plugins/api/graphql_server.go`

**Deliverables**:
- [ ] Update constructor
- [ ] Update request handling
- [ ] Update error handling
- [ ] Ensure compatibility with plugin

**Acceptance Criteria**:
- Server works with plugin
- Request handling correct
- Error handling correct
- Tests pass
- Code compiles without errors

### Task 2.5: Update GraphQL Tests
**Files**:
- `pkg/plugins/api/graphql_plugin_test.go` (NEW)
- `pkg/plugins/api/graphql_server_test.go`
- `pkg/plugins/api/graphql_resolver_test.go`

**Deliverables**:
- [ ] Plugin lifecycle tests
- [ ] HandleRequest tests
- [ ] HTTP Adapter tests
- [ ] Resolver tests with QueryService
- [ ] Server tests
- [ ] Integration tests

**Acceptance Criteria**:
- >80% code coverage
- All tests pass
- Plugin lifecycle verified
- Error cases covered
- Edge cases tested

### Task 2.6: Property Tests for GraphQL Plugin
**File**: `pkg/plugins/api/graphql_plugin_property_test.go` (NEW)

**Deliverables**:
- [ ] Property tests for request handling
- [ ] Property tests for error handling
- [ ] Property tests for concurrent access
- [ ] Property tests for caching behavior

**Acceptance Criteria**:
- All properties verified
- Tests pass consistently
- No race conditions
- Performance acceptable

---

## Phase 3: Plugin Integration

### Task 3.1: Register GraphQL Plugin
**File**: `cmd/chainpulse/main.go`

**Deliverables**:
- [ ] Create GraphQL plugin instance
- [ ] Register with plugin registry
- [ ] Configure plugin lifecycle
- [ ] Add to deployment

**Acceptance Criteria**:
- Plugin registers successfully
- Plugin lifecycle works
- Plugin can be started/stopped
- No errors during registration

### Task 3.2: Update Deployment Configurations
**Files**:
- `k8s/chainpulse-monolithic-deployment.yaml`
- `k8s/chainpulse-microservice-deployment.yaml`
- `docker/docker-compose.yml`

**Deliverables**:
- [ ] Update monolithic deployment
- [ ] Update microservice deployment
- [ ] Update Docker Compose
- [ ] Add GraphQL plugin configuration

**Acceptance Criteria**:
- Deployments include GraphQL plugin
- Configuration correct
- Plugin starts successfully
- No deployment errors

### Task 3.3: Integration Tests
**File**: `test/integration/graphql_plugin_integration_test.go` (NEW)

**Deliverables**:
- [ ] Plugin lifecycle tests
- [ ] End-to-end GraphQL tests
- [ ] HTTP Adapter tests
- [ ] Error scenario tests
- [ ] Concurrent request tests

**Acceptance Criteria**:
- All tests pass
- No race conditions
- Performance acceptable
- Error handling verified

---

## Phase 4: Testing & Documentation

### Task 4.1: Comprehensive Testing
**Deliverables**:
- [ ] Run all unit tests
- [ ] Run all integration tests
- [ ] Run all property tests
- [ ] Check code coverage
- [ ] Fix any failures

**Acceptance Criteria**:
- All tests pass
- Coverage >80%
- No warnings
- No errors

### Task 4.2: Performance Testing
**Deliverables**:
- [ ] Benchmark GraphQL plugin
- [ ] Benchmark HTTP adapter
- [ ] Benchmark query service
- [ ] Compare with original implementation
- [ ] Document results

**Acceptance Criteria**:
- Performance acceptable
- No significant degradation
- Caching effective
- Results documented

### Task 4.3: Update Documentation
**Files**:
- `docs/guides/GRAPHQL_API_PLUGIN_GUIDE.md` (NEW)
- `docs/progress/GRAPHQL_API_PLUGIN_COMPLETE.md` (NEW)

**Deliverables**:
- [ ] Update architecture documentation
- [ ] Update API documentation
- [ ] Create plugin integration guide
- [ ] Update examples
- [ ] Create troubleshooting guide

**Acceptance Criteria**:
- Documentation is clear
- Examples work
- Plugin integration documented
- All changes documented

### Task 4.4: Verify All Tests Pass
**Deliverables**:
- [ ] Run all unit tests
- [ ] Run all integration tests
- [ ] Run all property tests
- [ ] Check code coverage
- [ ] Fix any failures

**Acceptance Criteria**:
- All tests pass
- Coverage >80%
- No warnings
- No errors

---

## Checklist

### Code Quality
- [ ] All code follows project conventions
- [ ] No linting errors
- [ ] No type errors
- [ ] Proper error handling
- [ ] Comprehensive logging

### Testing
- [ ] Unit tests written
- [ ] Integration tests written
- [ ] Property tests written
- [ ] All tests pass
- [ ] Coverage >80%

### Documentation
- [ ] Code documented
- [ ] API documented
- [ ] Architecture documented
- [ ] Plugin integration guide created
- [ ] Examples provided

### Plugin Integration
- [ ] Plugin implements APIPlugin
- [ ] Plugin registered with registry
- [ ] Plugin lifecycle works
- [ ] Deployment configurations updated
- [ ] No breaking changes

### Performance
- [ ] Caching works
- [ ] No performance degradation
- [ ] Concurrent access safe
- [ ] Metrics recorded

---

## Dependencies

- Query Service must be completed before GraphQL Plugin (Phase 1 ✅)
- GraphQL Plugin must be completed before Plugin Integration (Phase 2 → Phase 3)
- Plugin Integration must be completed before Testing (Phase 3 → Phase 4)

## Timeline

| Phase | Tasks | Duration | Status |
|-------|-------|----------|--------|
| 1 | Query Service | 2 days | ✅ COMPLETE |
| 2 | GraphQL Plugin | 2-3 days | ⏳ READY |
| 3 | Plugin Integration | 1 day | 📋 PLANNED |
| 4 | Testing & Docs | 1-2 days | 📋 PLANNED |

**Total**: 6-8 days

## Success Metrics

- ✅ GraphQL implements APIPlugin interface
- ✅ All tests pass
- ✅ Code coverage >80%
- ✅ No breaking changes
- ✅ Performance maintained
- ✅ Documentation complete
- ✅ Plugin lifecycle works
- ✅ Architecture improved
