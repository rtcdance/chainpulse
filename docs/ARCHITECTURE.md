# chainpulse Web3 Indexer Architecture

## Overview

chainpulse is an enterprise-level Web3 indexer microservice architecture designed to support both current low-concurrency scenarios and future high-concurrency requirements. The system follows a plugin-based, event-driven architecture that enables easy scaling and microservice decomposition.

## Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │            chainpulse System            │
                    └─────────────────────────────────────────┘
                                      │
                    ┌─────────────────▼───────────────────────┐
                    │         Data Puller Layer               │
                    │  ┌──────────────┐  ┌──────────────┐    │
                    │  │ HTTPS-JSON   │  │ WebSocket-   │    │
                    │  │ RPC Plugin   │  │ JSONRPC      │    │
                    │  └──────────────┘  │ Plugin       │    │
                    │                    └──────────────┘    │
                    │  ┌──────────────┐                     │
                    │  │ gRPC Plugin  │                     │
                    │  └──────────────┘                     │
                    └─────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │         Message Queue Layer             │
                    │  ┌─────────┐  ┌─────────┐  ┌─────────┐ │
                    │  │  Kafka  │  │  Redis  │  │ ZeroMQ  │ │
                    │  │  Plugin │  │ Plugin  │  │ Plugin  │ │
                    │  └─────────┘  └─────────┘  └─────────┘ │
                    └─────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │        Data Processing Layer            │
                    │  ┌───────────────────────────────────┐  │
                    │  │     Event Processing Service      │  │
                    │  │  - Consume from MQ                │  │
                    │  │  - Process events                 │  │
                    │  │  - Store in PostgreSQL          │  │
                    │  │  - Update Redis cache             │  │
                    │  └───────────────────────────────────┘  │
                    └─────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │           API Gateway Layer             │
                    │  ┌──────────────┐  ┌──────────────┐    │
                    │  │   REST API   │  │   gRPC API   │    │
                    │  │   Plugin     │  │   Plugin     │    │
                    │  └──────────────┘  └──────────────┘    │
                    └─────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │              Data Layer                 │
                    │  ┌─────────────────┐  ┌───────────────┐ │
                    │  │   Database      │  │   Redis       │ │
                    │  │   (PostgreSQL)  │  │   Cache       │ │
                    │  └─────────────────┘  └───────────────┘ │
                    └─────────────────────────────────────────┘
                                      │
                    ┌─────────────────▼───────────────────────┐
                    │      Plugin Registry & Metrics          │
                    │  ┌──────────────┐  ┌──────────────┐    │
                    │  │ Data Puller  │  │ MQ Registry  │    │
                    │  │ Registry     │  │              │    │
                    │  └──────────────┘  └──────────────┘    │
                    │  ┌──────────────┐  ┌──────────────┐    │
                    │  │ API Registry │  │ Global       │    │
                    │  │              │  │ Metrics      │    │
                    │  └──────────────┘  │ (Prometheus) │    │
                    │                    └──────────────┘    │
                    └─────────────────────────────────────────┘
                                      │
                    ┌─────────────────▼───────────────────────┐
                    │            External Sources             │
                    │  ┌──────────────┐  ┌──────────────┐    │
                    │  │  Ethereum    │  │   Polygon    │    │
                    │  │  Network     │  │   Network    │    │
                    │  └──────────────┘  └──────────────┘    │
                    │  ┌──────────────┐                     │
                    │  │    BSC       │                     │
                    │  │  Network     │                     │
                    │  └──────────────┘                     │
                    └─────────────────────────────────────────┘
```

## Component Details

### 1. Data Puller Layer (Data Pulling)
- **Responsibility**: Pull data from blockchain networks only
- Supports multiple protocols (HTTPS-JSONRPC, WebSocket-JSONRPC, gRPC)
- Publishes raw events directly to message queues
- Implements reorg detection and handles blockchain reorganizations
- Provides real-time and historical data pulling

### 2. Message Queue Layer (Write to MQ)
- **Responsibility**: Store events temporarily for processing
- Supports multiple MQ implementations (Kafka, Redis, ZeroMQ)
- Provides reliable message delivery
- Implements dead letter queues for failed messages
- Supports message batching for efficiency

### 3. Data Processing Layer (Read MQ & Write to Relational DB)
- **Responsibility**: Consume events from MQ and store in PostgreSQL
- Implements idempotency checks to prevent duplicate processing
- Handles rollback scenarios for data consistency
- Updates Redis cache after successful database writes
- Processes events asynchronously without blocking data pullers

### 4. API Gateway Layer (API Read)
- **Responsibility**: Handle API requests and return data
- Implements caching strategy: First check Redis, then PostgreSQL
- Provides REST and gRPC APIs for data access
- Implements rate limiting and authentication

### 5. Data Layer
- **PostgreSQL**: Persistent storage for structured blockchain data
- **Redis**: Cache for frequently accessed data with configurable TTL

## Data Flow

1. **Data Pulling**: Data Puller services pull blockchain events and publish them to MQ
2. **MQ Write**: Events are stored in message queues temporarily
3. **MQ Read**: Event Processing Service consumes events from MQ
4. **Relational DB Write**: Processed events are stored in PostgreSQL
5. **NoSQL Cache Update**: Redis cache is updated with new data
6. **API Read**: API services read from Redis first, then PostgreSQL if cache miss

## Plugin Architecture

### Plugin Registries
- **Data Puller Registry**: Manages data source plugins
- **MQ Registry**: Manages message queue plugins
- **API Registry**: Manages API service plugins

### Plugin Features
- **Extensible**: Easy to add new protocols and services
- **Configurable**: Runtime configuration via environment variables
- **Monitored**: Built-in metrics collection for all plugins
- **Swappable**: Can switch implementations without code changes

## Async Decoupling

### Event-Driven Architecture
- All services communicate through message queues
- Loose coupling between components
- Independent scaling of services
- Fault isolation and resilience

### Asynchronous Processing
- Data pulling happens independently of API requests
- Event processing is decoupled from data consumption
- Caching layer reduces database load
- Non-blocking I/O operations throughout the system

## Microservice Readiness

### Stateless Design
- No persistent local state in services
- All state stored in external systems
- Configuration-driven deployment
- Independent scaling capabilities

### Service Decomposition
- **Current**: Single binary with all services
- **Future**: Independent deployable services
- **Flexible**: Partial decomposition based on needs
- **Compatible**: Same interfaces for all deployment modes

## Scalability Features

### High Concurrency Support
- Worker pools for concurrent message processing
- Connection pooling for database and Redis
- Redis caching for frequently accessed data
- Batch processing for database operations

### Load Distribution
- Multiple instances of services can run simultaneously
- Message queues handle load distribution
- Database read replicas for query scaling
- CDN integration for static assets

## Monitoring and Observability

### Metrics Collection
- Prometheus-compatible metrics endpoint
- Per-plugin metrics collection
- Performance and error rate monitoring
- Resource utilization tracking

### Logging
- Structured logging with correlation IDs
- Distributed tracing support
- Error and performance logging
- Audit trail for data processing

## Deployment Modes

### Monolithic Mode
- Single binary deployment for low-concurrency scenarios
- All services run in a single process
- Simplified deployment and management
- Cost-effective for small-scale operations

### Microservice Mode
- Independent service deployments for high-concurrency scenarios
- Each service can be scaled independently
- Better fault isolation
- More complex deployment and management

## Configuration

### Environment Variables
- Centralized configuration via environment variables
- Service-specific configuration options
- Plugin-specific settings
- Connection string management

### Feature Flags
- Runtime feature toggling
- A/B testing capabilities
- Gradual rollout of new features
- Safe rollback mechanisms

## Security Considerations

### Authentication and Authorization
- API key-based authentication
- Role-based access control
- Rate limiting to prevent abuse
- Input validation and sanitization

### Data Protection
- Encryption at rest for sensitive data
- TLS for data in transit
- Secure credential management
- Regular security audits

## Performance Optimization

### Caching Strategy
- Multi-layer caching (Redis + in-memory)
- Cache invalidation strategies
- Cache warming mechanisms
- Performance monitoring for cache hit rates

### Database Optimization
- Connection pooling
- Query optimization
- Indexing strategies
- Read/write splitting

This architecture now more accurately reflects your requirements:
- Data pulling services only pull data and write to MQ
- Separate processing service reads from MQ and writes to PostgreSQL
- API services read from Redis first, then PostgreSQL if needed
- All components are properly decoupled and can be scaled independently