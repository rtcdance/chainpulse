# ChainPulse Event Processor

The Event Processor is a critical microservice in the ChainPulse distributed architecture. It consumes raw blockchain events from Kafka, applies transformations, enrichment, and validation, then publishes processed events for storage and querying.

## Overview

The Event Processor handles:
- Consuming raw events from Kafka topics
- Event validation and schema verification
- Event transformation and normalization
- Event enrichment with metadata
- Batch processing for efficiency
- Publishing processed events to output topics
- Error handling and dead-letter queues

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Kafka Cluster                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ Input Topics: raw-events, blockchain-events            │ │
│  └────────────────────┬────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
   ┌────▼────┐     ┌────▼────┐     ┌────▼────┐
   │Processor│     │Processor│     │Processor│
   │   #1    │     │   #2    │     │   #3    │
   │ (8082)  │     │ (8082)  │     │ (8082)  │
   └────┬────┘     └────┬────┘     └────┬────┘
        │               │               │
        └───────────────┼───────────────┘
                        │
┌──────────────────────────────────────────────────────────────┐
│                    Kafka Cluster                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ Output Topics: processed-events, indexed-events        │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

## Configuration

Configure the Event Processor using environment variables:

```bash
# Service Configuration
PROCESSOR_PORT=8082            # Port to listen on (default: 8082)
INSTANCE_ID=event-processor-1  # Unique instance identifier
LOG_LEVEL=info                 # Log level: debug, info, warn, error

# Kafka Configuration
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
KAFKA_CONSUMER_GROUP=event-processor-consumers
KAFKA_INPUT_TOPICS=raw-events,blockchain-events
KAFKA_OUTPUT_TOPICS=processed-events,indexed-events

# Processing Configuration
BATCH_SIZE=100                 # Events per batch
BATCH_TIMEOUT=5000             # Batch timeout in milliseconds
MAX_RETRIES=3                  # Retry attempts for failed events
DEAD_LETTER_TOPIC=dead-letters # Topic for failed events

# Performance Configuration
WORKER_THREADS=4               # Number of processing threads
QUEUE_SIZE=1000                # Internal queue size
```

## Building

Build the Event Processor for your platform:

```bash
# Build for current platform
make build

# Build for specific platform
make build-linux
make build-macos
make build-windows

# Build for all platforms
make build-all

# Build with optimizations
make build-release

# Build with debug symbols
make build-debug
```

## Running

Start the Event Processor:

```bash
# Run with default configuration
make run

# Run with custom configuration
PROCESSOR_PORT=8082 INSTANCE_ID=event-processor-1 ./chainpulse-event-processor

# Run in debug mode
LOG_LEVEL=debug ./chainpulse-event-processor
```

## Endpoints

### Health Check
```
GET /health
```
Returns service health status and readiness.

### Metrics
```
GET /metrics
```
Prometheus-compatible metrics endpoint.

### Status
```
GET /status
```
Returns current processing status and statistics.

## Event Processing Pipeline

### 1. Consumption
- Consumes events from input Kafka topics
- Maintains consumer group offset
- Handles rebalancing

### 2. Validation
- Validates event schema
- Checks required fields
- Verifies data types

### 3. Transformation
- Normalizes event format
- Converts data types
- Applies field mappings

### 4. Enrichment
- Adds metadata
- Resolves contract information
- Adds timestamp information

### 5. Publishing
- Publishes to output topics
- Maintains ordering guarantees
- Handles failures

## Deployment

### Docker
```bash
docker build -f docker/event-processor.Dockerfile -t chainpulse-event-processor:latest .
docker run -p 8082:8082 \
  -e PROCESSOR_PORT=8082 \
  -e KAFKA_BROKERS=kafka:9092 \
  chainpulse-event-processor:latest
```

### Kubernetes
```bash
kubectl apply -f k8s/event-processor-deployment.yaml
kubectl scale deployment event-processor --replicas=3
```

### Docker Compose
```bash
docker-compose -f docker/docker-compose.yml up event-processor
```

## Monitoring

Monitor the Event Processor using:

- **Health Endpoint**: `http://localhost:8082/health`
- **Metrics Endpoint**: `http://localhost:8082/metrics`
- **Status Endpoint**: `http://localhost:8082/status`
- **Logs**: Check service logs for errors and warnings
- **Kubernetes**: `kubectl logs -f deployment/event-processor`

Key metrics to monitor:
- Events consumed per second
- Events processed per second
- Processing latency (p50, p95, p99)
- Error rate
- Dead-letter queue size

## Scaling

The Event Processor is horizontally scalable:

```bash
# Scale to 5 replicas
kubectl scale deployment event-processor --replicas=5

# Auto-scale based on CPU
kubectl autoscale deployment event-processor --min=2 --max=10 --cpu-percent=80
```

## Troubleshooting

### Events not being processed
- Check Kafka broker connectivity: `KAFKA_BROKERS`
- Verify consumer group: `KAFKA_CONSUMER_GROUP`
- Check input topics exist: `KAFKA_INPUT_TOPICS`
- Review logs for errors

### High latency
- Increase `BATCH_SIZE` for better throughput
- Increase `WORKER_THREADS` for parallel processing
- Check Kafka broker performance
- Monitor CPU and memory usage

### Memory issues
- Reduce `BATCH_SIZE`
- Reduce `QUEUE_SIZE`
- Check for memory leaks in logs
- Monitor with `kubectl top pod`

### Dead-letter queue growing
- Review error logs for failure reasons
- Check event schema compatibility
- Verify transformation logic
- Consider manual intervention

## Testing

Run tests for the Event Processor:

```bash
# Run all tests
make test

# Run specific test
go test -v ./... -run TestName

# Run with coverage
go test -cover ./...
```

## Development

### Code Quality
```bash
# Format code
make fmt

# Run linter
make lint

# Run static analysis
make vet
```

### Debugging
```bash
# Build with debug symbols
make build-debug

# Run with debug logging
LOG_LEVEL=debug ./chainpulse-event-processor-debug
```

## Performance Tuning

- **Batch Size**: Increase for higher throughput, decrease for lower latency
- **Worker Threads**: Increase for parallel processing
- **Queue Size**: Increase to handle spikes
- **Batch Timeout**: Adjust based on latency requirements

## Error Handling

- **Validation Errors**: Events sent to dead-letter queue
- **Processing Errors**: Retried up to `MAX_RETRIES` times
- **Kafka Errors**: Automatic reconnection with exponential backoff
- **Timeout Errors**: Events requeued for retry

## Related Services

- **Data Puller** (8083): Provides raw events
- **API Service** (8081): Consumes processed events
- **Database**: Stores processed events
- **Message Queue**: Kafka for event streaming

## Optional Security Surface

The event-processor runtime/control endpoints can be protected with the same
optional auth and rate-limit surface used by the other services:

- `EVENT_PROCESSOR_AUTH_ENABLED`
- `EVENT_PROCESSOR_AUTH_JWT_SECRET`
- `EVENT_PROCESSOR_AUTH_API_KEYS`
- `EVENT_PROCESSOR_RATE_LIMIT_ENABLED`
- `EVENT_PROCESSOR_RATE_LIMIT`

## Support

For issues or questions:
1. Check the logs: `kubectl logs -f deployment/event-processor`
2. Review configuration: Verify all environment variables
3. Check Kafka connectivity: Test broker connections
4. Review metrics: Check performance metrics at `/metrics`
5. Check dead-letter queue: Review failed events
