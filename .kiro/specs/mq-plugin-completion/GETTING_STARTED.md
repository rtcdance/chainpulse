# MQ Plugin Completion - Getting Started

## Quick Overview

This guide helps you understand and execute the MQ plugin completion specification.

## What is Being Built?

A complete Message Queue (MQ) plugin system for ChainPulse that:

- Provides a unified interface for multiple MQ backends (Kafka, Redis, ZeroMQ)
- Handles message publishing, consumption, acknowledgment, and retry logic
- Manages dead letter queues for failed messages
- Collects comprehensive metrics and monitoring data
- Ensures thread-safe concurrent operations
- Provides graceful error handling and recovery

## Current State

### What Exists
- ✅ BaseMQPlugin with core functionality
- ✅ KafkaMQPlugin skeleton implementation
- ✅ Unit tests for BaseMQPlugin
- ✅ Property tests for correctness properties

### What Needs Completion
- ❌ Fix Kafka plugin import issues
- ❌ Complete Kafka producer/consumer implementation
- ❌ Implement batch processing
- ❌ Implement comprehensive metrics collection
- ❌ Implement Redis and ZeroMQ plugins
- ❌ Integration tests with real brokers

## Key Files

### Source Code
- `pkg/core/mq_plugin.go` - BaseMQPlugin implementation
- `pkg/core/mq_plugin_test.go` - Unit tests
- `pkg/core/mq_plugin_property_test.go` - Property tests
- `pkg/plugins/mq/kafka_mq.go` - Kafka implementation
- `pkg/plugins/mq/kafka_mq_test.go` - Kafka tests
- `pkg/plugins/mq/kafka_mq_property_test.go` - Kafka property tests

### Specification
- `requirements.md` - Detailed requirements
- `design.md` - Architecture and design
- `tasks.md` - Implementation tasks
- `SUMMARY.md` - Specification summary

## How to Execute

### Step 1: Review the Specification

1. Read `requirements.md` to understand what needs to be built
2. Review `design.md` to understand the architecture
3. Check `tasks.md` for the implementation plan

### Step 2: Execute Tasks in Order

Start with Task 1 and work through sequentially:

```bash
# Task 1: Fix Kafka Plugin and Core Implementation
# - Fix import paths
# - Complete Kafka producer/consumer
# - Add error handling

# Task 2: Implement Message Publishing
# - Add metrics recording
# - Implement partition key routing
# - Add message ID generation

# Task 3: Implement Message Consumption
# - Add handler support
# - Implement offset tracking
# - Add graceful shutdown

# ... continue with remaining tasks
```

### Step 3: Run Tests

After each task, run the relevant tests:

```bash
# Unit tests
go test ./pkg/core -v -run TestMQPlugin

# Property tests
go test ./pkg/core -v -run TestProperty

# Kafka tests
go test ./pkg/plugins/mq -v -run TestKafka
```

### Step 4: Verify Correctness Properties

Ensure all property-based tests pass:

```bash
# Run all property tests
go test ./pkg/core -v -run Property
go test ./pkg/plugins/mq -v -run Property
```

## Key Concepts

### Message Flow

```
1. Publish
   └─ Message → Topic → Broker

2. Consume
   └─ Broker → Topic → Handler

3. Acknowledge
   └─ Handler Success → Offset Update

4. Retry (on failure)
   └─ Increment Retry Count → Requeue or DLQ

5. Dead Letter Queue
   └─ Max Retries Exceeded → DLQ Topic
```

### Retry Logic

```
Attempt 1 → Fail → Retry (delay 1s)
Attempt 2 → Fail → Retry (delay 2s)
Attempt 3 → Fail → Retry (delay 4s)
Attempt 4 → Fail → Send to DLQ
```

### Correctness Properties

The specification includes 10 properties that must be verified:

1. **Message Delivery** - Messages reach consumer or DLQ
2. **Exactly-Once** - No duplicate processing
3. **Retry Enforcement** - Max retries respected
4. **DLQ Consistency** - Failure reasons preserved
5. **Message Ordering** - Order maintained per partition
6. **Batch Atomicity** - Batches processed together
7. **Metrics Accuracy** - Metrics recorded correctly
8. **Thread Safety** - No race conditions
9. **Config Validation** - Invalid configs rejected
10. **Graceful Shutdown** - Clean shutdown

## Common Tasks

### Fix Kafka Plugin Imports

The Kafka plugin has import issues that need fixing:

```go
// Current (broken)
message core.core.MessageQueueMessage

// Should be
message core.MessageQueueMessage
```

### Add Batch Processing

Implement batch collection and processing:

```go
type BatchProcessor struct {
    messages []MessageQueueMessage
    size     int
    timeout  time.Duration
}

func (bp *BatchProcessor) Add(msg MessageQueueMessage) {
    bp.messages = append(bp.messages, msg)
    if len(bp.messages) >= bp.size {
        bp.Process()
    }
}
```

### Implement Metrics Collection

Record metrics for all operations:

```go
// Publish metrics
metricsCollector.RecordMetric("mq_messages_published", 1, 
    map[string]string{"topic": topic})

// Error metrics
metricsCollector.RecordMetric("mq_errors", 1, nil)

// DLQ metrics
metricsCollector.RecordMetric("mq_dead_letter_queue_size", dlqSize, nil)
```

## Testing Strategy

### Unit Tests
- Test individual functions
- Test error cases
- Test edge cases

### Property Tests
- Test universal properties
- Generate random inputs
- Verify properties hold

### Integration Tests
- Test with real Kafka broker
- Test multi-topic operations
- Test consumer groups

## Performance Considerations

### Throughput
- Target: 10,000+ messages/second
- Batch size: 100 messages
- Parallel consumers: 4+

### Latency
- Target: <100ms end-to-end
- Publish latency: <10ms
- Consume latency: <50ms

### Memory
- Per message: ~1KB
- Per consumer: ~10MB
- Per broker connection: ~5MB

## Troubleshooting

### Kafka Connection Issues

```bash
# Check Kafka broker is running
docker ps | grep kafka

# Check broker connectivity
telnet localhost 9092

# Check logs
docker logs kafka
```

### Test Failures

```bash
# Run with verbose output
go test -v -run TestName

# Run with race detector
go test -race -run TestName

# Run with coverage
go test -cover -run TestName
```

### Import Errors

```bash
# Update imports
go mod tidy

# Verify imports
go mod verify

# Check for circular dependencies
go mod graph
```

## Next Steps

1. **Start with Task 1** - Fix Kafka plugin imports
2. **Run unit tests** - Verify basic functionality
3. **Run property tests** - Verify correctness properties
4. **Move to Task 2** - Implement message publishing
5. **Continue sequentially** - Follow task list

## Resources

- [Kafka Go Client](https://github.com/segmentio/kafka-go)
- [Redis Go Client](https://github.com/redis/go-redis)
- [ZeroMQ Go Bindings](https://github.com/pebbe/zmq4)
- [Property-Based Testing](https://hypothesis.works/articles/what-is-property-based-testing/)

## Questions?

Refer to:
- `requirements.md` - For what needs to be built
- `design.md` - For how to build it
- `tasks.md` - For specific implementation steps

