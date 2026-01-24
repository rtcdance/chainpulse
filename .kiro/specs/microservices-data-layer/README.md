# Microservices Data Layer Integration Specification

**Date:** January 12, 2026  
**Status:** ✅ Complete - Ready for Implementation  
**Feature Name:** microservices-data-layer

## Overview

This specification defines the integration of MongoDB (NoSQL) and PostgreSQL (relational) database layers into the API Service and Event Processor microservices. It addresses the gap where these services were missing proper data layer implementations.

## Problem Statement

The API Service and Event Processor microservices were initialized with only core services and message queues, but lacked:

- **API Service:**
  - MongoDB connection and queries
  - PostgreSQL connection and queries
  - Cache integration for data access
  - Query service implementation

- **Event Processor:**
  - MongoDB event storage
  - PostgreSQL event metadata storage
  - Batch insert operations
  - Event retrieval service

## Solution

This specification provides a comprehensive solution with:

1. **8 Core Requirements** - Clear acceptance criteria for all functionality
2. **Detailed Design** - Architecture, components, data models, correctness properties
3. **60 Implementation Tasks** - Organized across 6 phases with property-based tests
4. **Comprehensive Documentation** - Guides, troubleshooting, monitoring

## Architecture

### Cache-First Pattern

```
Read Flow:
  Redis Cache → MongoDB → PostgreSQL → Cache Result

Write Flow:
  MongoDB (fast) → PostgreSQL (consistency) → Invalidate Cache
```

### Components

- **DatabaseManager** - Connection pool management
- **Query Service** - Cache-first query execution
- **Event Store** - MongoDB event storage
- **Event Metadata Store** - PostgreSQL metadata storage
- **Error Handler** - Circuit breaker, retry logic, reconciliation

## Key Features

✅ **Connection Pooling** - Efficient resource management  
✅ **Health Checks** - Continuous database monitoring  
✅ **Error Handling** - Circuit breaker, retry logic, exponential backoff  
✅ **Data Consistency** - Transactional semantics across databases  
✅ **Batch Operations** - Efficient bulk inserts  
✅ **TTL Policies** - Automatic data expiration  
✅ **Configuration** - Environment-based setup  
✅ **Monitoring** - Comprehensive metrics and logging  

## Documents

### 1. Requirements Document

**File:** `requirements.md`

Defines 8 core requirements with acceptance criteria:

1. API Service - NoSQL Integration
2. API Service - PostgreSQL Integration
3. Event Processor - NoSQL Integration
4. Event Processor - PostgreSQL Integration
5. Shared Database Configuration
6. Database Connection Pooling
7. Error Handling and Resilience
8. Data Consistency

**Read:** `cat requirements.md`

### 2. Design Document

**File:** `design.md`

Provides detailed design with:

- Architecture overview
- Component design
- Data models (MongoDB and PostgreSQL)
- Correctness properties (8 properties)
- Error handling strategies
- Testing strategy
- Implementation phases

**Read:** `cat design.md`

### 3. Implementation Tasks

**File:** `tasks.md`

Contains 60 implementation tasks across 6 phases:

- Phase 1: Database Connection Management (9 tasks)
- Phase 2: API Service Data Layer (11 tasks)
- Phase 3: Event Processor Data Layer (12 tasks)
- Phase 4: Error Handling and Resilience (11 tasks)
- Phase 5: Configuration and Deployment (11 tasks)
- Phase 6: Documentation and Monitoring (6 tasks)

**Read:** `cat tasks.md`

## Quick Start

### 1. Review Specification

```bash
# Read all documents
cat requirements.md
cat design.md
cat tasks.md
```

### 2. Understand Architecture

```
API Service:
  - Query Service (cache-first pattern)
  - MongoDB Adapter (fast queries)
  - PostgreSQL Adapter (complex queries)
  - Cache Service (Redis integration)

Event Processor:
  - Event Store (MongoDB storage)
  - Event Metadata Store (PostgreSQL storage)
  - Event Retrieval Service (query both)
  - Batch Operations (efficient inserts)
```

### 3. Start Implementation

```bash
# Phase 1: Database Connection Management
# - Create DatabaseManager interface
# - Implement MongoDB connection pool
# - Implement PostgreSQL connection pool
# - Add health checks

# Phase 2: API Service Data Layer
# - Implement Query Service
# - Implement MongoDB Adapter
# - Implement PostgreSQL Adapter
# - Integrate cache

# ... continue with remaining phases
```

### 4. Run Tests

```bash
# Unit tests
go test ./pkg/infrastructure/database/...

# Integration tests
go test -tags=integration ./test/integration/...

# Property-based tests
go test -run Property ./pkg/infrastructure/database/...
```

## Configuration

### Environment Variables

```bash
# MongoDB
MONGODB_URI=mongodb://mongo-1:27017,mongo-2:27017,mongo-3:27017
MONGODB_DATABASE=chainpulse
MONGODB_TIMEOUT_MS=5000

# PostgreSQL
DATABASE_URL=postgres://user:password@postgres:5432/chainpulse
DATABASE_POOL_SIZE=20
DATABASE_TIMEOUT_MS=5000

# Connection Management
DB_POOL_SIZE=10
DB_TIMEOUT_MS=5000
DB_RETRY_ATTEMPTS=3
DB_RETRY_DELAY_MS=100

# Cache
REDIS_CLUSTER=redis-1:6379,redis-2:6379,redis-3:6379
CACHE_TTL_SECONDS=3600

# Event Processor
EVENT_BATCH_SIZE=100
EVENT_TTL_DAYS=30
```

## Correctness Properties

The specification includes 8 correctness properties that must be validated:

1. **Cache-First Consistency** - Cached data returned without DB query
2. **Fallback Chain Correctness** - MongoDB miss → PostgreSQL query
3. **Write Atomicity** - Both writes succeed or both fail
4. **Connection Pool Reuse** - Connections reused from pool
5. **Health Check Accuracy** - Health check reflects actual status
6. **Batch Insert Completeness** - All or nothing batch operations
7. **TTL Expiration** - Documents auto-deleted after TTL
8. **Error Logging Completeness** - Errors logged with full context

Each property is validated through property-based tests.

## Implementation Phases

### Phase 1: Database Connection Management
- DatabaseManager interface
- MongoDB connection pool
- PostgreSQL connection pool
- Health checks
- **Duration:** 1-2 days

### Phase 2: API Service Data Layer
- Query Service
- MongoDB Adapter
- PostgreSQL Adapter
- Cache integration
- **Duration:** 2-3 days

### Phase 3: Event Processor Data Layer
- Event Store
- Event Metadata Store
- Event Retrieval Service
- Batch operations
- **Duration:** 2-3 days

### Phase 4: Error Handling and Resilience
- Error handling
- Circuit breaker
- Retry logic
- Data consistency
- **Duration:** 2-3 days

### Phase 5: Configuration and Deployment
- Environment variables
- Configuration validation
- Database initialization
- Deployment updates
- **Duration:** 1-2 days

### Phase 6: Documentation and Monitoring
- API documentation
- Troubleshooting guide
- Monitoring setup
- Operational runbook
- **Duration:** 1 day

**Total Estimated Duration:** 9-14 days

## Related Documentation

- **Quick Reference:** `docs/guides/MICROSERVICES_DATA_LAYER_QUICK_REFERENCE.md`
- **Analysis Complete:** `docs/progress/MICROSERVICES_DATA_LAYER_ANALYSIS_COMPLETE.md`
- **Spec Created:** `docs/progress/MICROSERVICES_DATA_LAYER_SPEC_CREATED.md`

## Next Steps

1. **Review** - Read all specification documents
2. **Approve** - Confirm requirements and design
3. **Implement** - Follow the task list in order
4. **Test** - Run unit, integration, and property-based tests
5. **Deploy** - Update configurations and deploy

## Support

For questions or clarifications:

1. Review the relevant specification document
2. Check the quick reference guide
3. Refer to the troubleshooting section
4. Review existing implementation examples

---

**Status:** ✅ Specification complete and ready for implementation.

**Next Action:** Review the specification and start Phase 1 implementation.
