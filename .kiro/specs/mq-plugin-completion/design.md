# MQ Plugin Completion Design

## Overview

This design document specifies the complete implementation of the Message Queue (MQ) plugin system in ChainPulse. The MQ plugin provides a unified interface for message queue operations with support for multiple backends (Kafka, Redis, ZeroMQ).

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  (Event Indexer, Data Processor, etc.)                      │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                  MQ Plugin Interface                         │
│  (Publish, Consume, Acknowledge, Retry, DLQ)               │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
    ┌────────┐  ┌────────┐  ┌──────────┐
    │ Kafka  │  │ Redis  │  │ ZeroMQ  │
    │ Plugin │  │ Plugin │  │ Plugin   │
    └────────┘  └────────┘  └──────────┘
        │            │            │
        └────────────┼────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │  Message Queue Broker  │
        │  (Kafka/Redis/ZeroMQ)  │
        └────────────────────────┘
```

### Core Components

#### 1. MQ Plugin Interface

```go
type MQPlugin interface {
    // Lifecycle
    Initialize() error
    Start() error
    Stop() error
    Health() *HealthStatus
    
    // Publishing
    PublishMessage(ctx context.Context, message MessageQueueMessage) error
    
    // Consumption
    ConsumeMessages(ctx context.Context, topic string, handler func(MessageQueueMessage) error) error
    
    // Acknowledgment
    AcknowledgeMessage(ctx context.Context, message MessageQueueMessage) error
    
    // Retry
    RetryMessage(ctx context.Context, message MessageQueueMessage) error
    
    // Dead Letter Queue
    SendToDeadLetterQueue(ctx context.Context, message MessageQueueMessage, reason string) error
    GetDeadLetterQueueMessages(ctx context.Context, limit int) ([]MessageQueueMessage, error)
    
    // Statistics
    GetStats() MessageQueueStats
    
    // Configuration
    SetBatchSize(size int)
    SetMaxRetries(maxRetries int)
    SetRetryDelay(delay time.Duration)
}
```

#### 2. Message Structure

```go
type MessageQueueMessage struct {
    ID              string            // Unique message identifier
    Topic           string            // Topic/channel name
    Payload         []byte            // Message data
    Timestamp       time.Time         // Creation time
    Offset          int64             // Position in topic
    PartitionKey    string            // Partition routing key
    RetryCount      int               // Number of retry attempts
    DeadLetterReason string           // Reason for DLQ move
    Headers         map[string]string // Custom headers
}
```

#### 3. Statistics Structure

```go
type MessageQueueStats struct {
    MessageCount        int64         // Total messages published
    ErrorCount          int64         // Total errors
    DeadLetterQueueSize int64         // DLQ message count
    AverageProcessTime  int64         // Average processing time (ns)
    LastError           error         // Last error encountered
    LastErrorTime       time.Time     // Time of last error
    IsRunning           bool          // Plugin running status
}
```

## Components and Interfaces

### BaseMQPlugin

Provides common functionality for all MQ implementations:

- **Lifecycle Management**: Initialize, Start, Stop
- **Message Publishing**: PublishMessage with metrics recording
- **Message Consumption**: ConsumeMessages with handler invocation
- **Acknowledgment**: AcknowledgeMessage with offset tracking
- **Retry Logic**: RetryMessage with exponential backoff
- **Dead Letter Queue**: SendToDeadLetterQueue with reason preservation
- **Statistics**: GetStats with comprehensive metrics
- **Configuration**: SetBatchSize, SetMaxRetries, SetRetryDelay

### KafkaMQPlugin

Kafka-specific implementation extending BaseMQPlugin:

- **Producer**: kafka-go Writer for publishing
- **Consumer**: kafka-go Reader for consuming
- **Offset Tracking**: Maintains per-topic offsets
- **Consumer Group**: Supports consumer group coordination
- **Broker Management**: Handles broker connections and failover

### RedisMQPlugin (Future)

Redis-specific implementation:

- **Pub/Sub**: Redis PUBLISH/SUBSCRIBE for messaging
- **Streams**: Redis Streams for persistent queues
- **Consumer Groups**: Redis consumer group support

### ZeroMQPlugin (Future)

ZeroMQ-specific implementation:

- **Push/Pull**: ZMQ_PUSH/ZMQ_PULL for request-reply
- **Pub/Sub**: ZMQ_PUB/ZMQ_SUB for publish-subscribe
- **Router/Dealer**: ZMQ_ROUTER/ZMQ_DEALER for async messaging

## Data Models

### Message Flow

```
1. Publish Phase
   ├─ Create MessageQueueMessage
   ├─ Assign unique ID and timestamp
   ├─ Route to topic based on partition key
   ├─ Send to broker
   └─ Record metrics

2. Consumption Phase
   ├─ Retrieve message from broker
   ├─ Invoke handler function
   ├─ Handle success/failure
   └─ Record metrics

3. Acknowledgment Phase
   ├─ Mark message as processed
   ├─ Update consumer offset
   ├─ Commit offset to broker
   └─ Record metrics

4. Retry Phase (on failure)
   ├─ Increment retry count
   ├─ Check if max retries exceeded
   ├─ If not exceeded: requeue with delay
   ├─ If exceeded: send to DLQ
   └─ Record metrics

5. Dead Letter Queue Phase
   ├─ Create DLQ message with reason
   ├─ Send to DLQ topic
   ├─ Preserve original payload
   └─ Record metrics
```

### State Transitions

```
┌─────────────┐
│ Uninitialized
└──────┬──────┘
       │ Initialize()
       ▼
┌─────────────┐
│ Initialized
└──────┬──────┘
       │ Start()
       ▼
┌─────────────┐
│ Running     │◄─────────────┐
└──────┬──────┘              │
       │ Stop()              │ Reconnect on error
       ▼                     │
┌─────────────┐              │
│ Stopped     │──────────────┘
└─────────────┘
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Message Delivery Guarantee

**For any** message published to a topic, the message SHALL either be successfully delivered to a consumer or moved to the dead letter queue after max retries.

**Validates: Requirements 2.1, 3.1, 5.1, 6.1**

### Property 2: Exactly-Once Semantics

**For any** message that is acknowledged, the message SHALL not be redelivered to other consumers in the same consumer group.

**Validates: Requirements 4.1, 4.2**

### Property 3: Retry Count Enforcement

**For any** message that fails processing, the retry count SHALL not exceed the configured max retries before moving to DLQ.

**Validates: Requirements 5.2, 5.3**

### Property 4: Dead Letter Queue Consistency

**For any** message moved to the dead letter queue, the failure reason SHALL be preserved and retrievable.

**Validates: Requirements 6.2, 6.3**

### Property 5: Message Ordering Per Partition

**For any** messages published with the same partition key, the messages SHALL be consumed in the same order they were published.

**Validates: Requirements 2.5**

### Property 6: Batch Processing Atomicity

**For any** batch of messages, either all messages are processed successfully or all are retried together.

**Validates: Requirements 7.2, 7.3**

### Property 7: Metrics Accuracy

**For any** MQ operation, the corresponding metric SHALL be recorded accurately and consistently.

**Validates: Requirements 8.1, 8.2, 8.3**

### Property 8: Thread-Safe Concurrent Access

**For any** concurrent operations on the MQ plugin, the plugin SHALL maintain data consistency and prevent race conditions.

**Validates: Requirements 9.1, 9.2, 9.3**

### Property 9: Configuration Validation

**For any** configuration provided to the MQ plugin, invalid configurations SHALL be rejected with clear error messages.

**Validates: Requirements 12.1, 12.5**

### Property 10: Graceful Shutdown

**For any** in-flight operations when the plugin is stopped, the plugin SHALL wait for completion before fully shutting down.

**Validates: Requirements 1.4, 9.5**

## Error Handling

### Error Categories

1. **Connection Errors**
   - Broker unreachable
   - Network timeout
   - Authentication failure
   - **Recovery**: Retry with exponential backoff

2. **Message Errors**
   - Invalid message format
   - Message too large
   - Serialization failure
   - **Recovery**: Send to DLQ with error reason

3. **Processing Errors**
   - Handler panic
   - Handler timeout
   - Handler returns error
   - **Recovery**: Retry with delay, then DLQ

4. **Configuration Errors**
   - Missing required parameters
   - Invalid parameter values
   - Incompatible settings
   - **Recovery**: Fail initialization with clear error

### Error Handling Strategy

```
┌─────────────────────┐
│ Operation Attempt   │
└──────────┬──────────┘
           │
           ▼
    ┌──────────────┐
    │ Success?     │
    └──┬───────┬───┘
       │ Yes   │ No
       │       ▼
       │   ┌──────────────┐
       │   │ Retryable?   │
       │   └──┬───────┬───┘
       │      │ Yes   │ No
       │      │       ▼
       │      │   ┌──────────────┐
       │      │   │ Send to DLQ  │
       │      │   └──────────────┘
       │      │
       │      ▼
       │   ┌──────────────┐
       │   │ Retry Count  │
       │   │ < Max?       │
       │   └──┬───────┬───┘
       │      │ Yes   │ No
       │      │       ▼
       │      │   ┌──────────────┐
       │      │   │ Send to DLQ  │
       │      │   └──────────────┘
       │      │
       │      ▼
       │   ┌──────────────┐
       │   │ Wait Delay   │
       │   └──────┬───────┘
       │          │
       │          ▼
       │   ┌──────────────┐
       │   │ Retry        │
       │   └──────┬───────┘
       │          │
       └──────────┴──────────┐
                             │
                             ▼
                        ┌─────────┐
                        │ Complete│
                        └─────────┘
```

## Testing Strategy

### Unit Tests

- **Plugin Lifecycle**: Initialize, Start, Stop, Health
- **Message Publishing**: Single message, batch, error cases
- **Message Consumption**: Single message, batch, handler errors
- **Acknowledgment**: Successful, failed, concurrent
- **Retry Logic**: Retry count, delay, max retries
- **Dead Letter Queue**: Move to DLQ, retrieve, consistency
- **Statistics**: Accuracy, concurrent updates
- **Configuration**: Validation, application

### Property-Based Tests

- **Property 1**: Message delivery guarantee across random messages
- **Property 2**: Exactly-once semantics with concurrent consumers
- **Property 3**: Retry count enforcement with random failures
- **Property 4**: DLQ consistency with random failure reasons
- **Property 5**: Message ordering with random partition keys
- **Property 6**: Batch atomicity with random batch sizes
- **Property 7**: Metrics accuracy with random operations
- **Property 8**: Thread-safe access with concurrent goroutines
- **Property 9**: Configuration validation with random configs
- **Property 10**: Graceful shutdown with random in-flight operations

### Integration Tests

- **Kafka Integration**: Connect to real Kafka broker
- **Multi-Topic**: Publish/consume across multiple topics
- **Consumer Groups**: Multiple consumers in same group
- **Offset Tracking**: Offset persistence and recovery
- **Failure Scenarios**: Broker failure, network partition, timeout

### Performance Tests

- **Throughput**: Messages per second
- **Latency**: End-to-end message latency
- **Memory**: Memory usage under load
- **CPU**: CPU usage under load
- **Scalability**: Performance with increasing message volume

