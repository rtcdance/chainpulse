# ChainPulse Event Processor - Quick Start

Get the Event Processor running in 5 minutes.

## Prerequisites

- Go 1.21+
- Kafka 3+
- Docker (optional)

## Option 1: Local Development

### 1. Build the Service

```bash
cd cmd/chainpulse-event-processor
make build
```

### 2. Set Environment Variables

```bash
export PROCESSOR_PORT=8082
export INSTANCE_ID=event-processor-1
export KAFKA_BROKERS=localhost:9092
export KAFKA_CONSUMER_GROUP=event-processor-consumers
export KAFKA_INPUT_TOPICS=raw-events,blockchain-events
export KAFKA_OUTPUT_TOPICS=processed-events,indexed-events
export BATCH_SIZE=100
export LOG_LEVEL=info
```

### 3. Start the Service

```bash
./chainpulse-event-processor
```

You should see:
```
╔════════════════════════════════════════════════════════════╗
║      ChainPulse - Event Processor Service                  ║
║              Web3 Event Indexing System                    ║
╚════════════════════════════════════════════════════════════╝

Configuration Loaded:
  Service Port:       8082
  Instance ID:        event-processor-1
  Kafka Brokers:      [localhost:9092]
  ...

✓ All services started successfully

Status: Running
Event Processor available at: http://localhost:8082
Health Check available at: http://localhost:8082/health
```

### 4. Test the Service

```bash
# Check health
curl http://localhost:8082/health

# Get metrics
curl http://localhost:8082/metrics

# Get status
curl http://localhost:8082/status
```

## Option 2: Docker

### 1. Build Docker Image

```bash
docker build -f docker/event-processor.Dockerfile -t chainpulse-event-processor:latest .
```

### 2. Run Container

```bash
docker run -p 8082:8082 \
  -e PROCESSOR_PORT=8082 \
  -e KAFKA_BROKERS=kafka:9092 \
  -e KAFKA_CONSUMER_GROUP=event-processor-consumers \
  chainpulse-event-processor:latest
```

## Option 3: Docker Compose

### 1. Start All Services

```bash
docker-compose -f docker/docker-compose.yml up event-processor
```

This starts:
- Event Processor (8082)
- Kafka
- Zookeeper

### 2. Verify Services

```bash
# Check Event Processor
curl http://localhost:8082/health

# Check other services
docker-compose ps
```

## Option 4: Kubernetes

### 1. Deploy to Kubernetes

```bash
kubectl apply -f k8s/event-processor-deployment.yaml
```

### 2. Verify Deployment

```bash
# Check deployment
kubectl get deployment event-processor

# Check pods
kubectl get pods -l app=event-processor

# Check logs
kubectl logs -f deployment/event-processor
```

### 3. Port Forward (optional)

```bash
kubectl port-forward svc/event-processor 8082:8082
```

### 4. Test Service

```bash
curl http://localhost:8082/health
```

## Common Tasks

### View Logs

**Local:**
```bash
# Logs are printed to stdout
```

**Docker:**
```bash
docker logs -f <container-id>
```

**Kubernetes:**
```bash
kubectl logs -f deployment/event-processor
```

### Monitor Processing

```bash
# Check metrics
curl http://localhost:8082/metrics | grep processor

# Check status
curl http://localhost:8082/status
```

### Scale Service

**Docker Compose:**
```bash
docker-compose up -d --scale event-processor=3
```

**Kubernetes:**
```bash
kubectl scale deployment event-processor --replicas=3
```

### Stop Service

**Local:**
```bash
# Press Ctrl+C
```

**Docker:**
```bash
docker stop <container-id>
```

**Kubernetes:**
```bash
kubectl delete deployment event-processor
```

## Troubleshooting

### Service won't start

**Check Kafka connection:**
```bash
# Verify Kafka is running
kafka-broker-api-versions.sh --bootstrap-server localhost:9092

# Check topics exist
kafka-topics.sh --list --bootstrap-server localhost:9092
```

**Check environment variables:**
```bash
echo $KAFKA_BROKERS $KAFKA_CONSUMER_GROUP
```

### Events not being processed

**Check input topics:**
```bash
# List topics
kafka-topics.sh --list --bootstrap-server localhost:9092

# Check topic content
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic raw-events --from-beginning
```

**Check consumer group:**
```bash
# List consumer groups
kafka-consumer-groups.sh --list --bootstrap-server localhost:9092

# Check group status
kafka-consumer-groups.sh --describe --group event-processor-consumers \
  --bootstrap-server localhost:9092
```

### High latency

**Check batch size:**
```bash
# Increase batch size for throughput
export BATCH_SIZE=200
```

**Check worker threads:**
```bash
# Increase threads for parallel processing
export WORKER_THREADS=8
```

### Memory issues

**Check memory usage:**
```bash
# Local
top

# Docker
docker stats

# Kubernetes
kubectl top pod
```

**Reduce batch size:**
```bash
export BATCH_SIZE=50
```

### Dead-letter queue growing

**Check failed events:**
```bash
# Read dead-letter topic
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic dead-letters --from-beginning
```

**Review logs for errors:**
```bash
# Check logs
kubectl logs -f deployment/event-processor | grep ERROR
```

## Testing Event Processing

### 1. Produce Test Events

```bash
# Create test event
echo '{"event": "test", "data": "value"}' | \
  kafka-console-producer.sh --broker-list localhost:9092 \
  --topic raw-events
```

### 2. Monitor Processing

```bash
# Watch output topic
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic processed-events --from-beginning
```

### 3. Check Metrics

```bash
# Get processing metrics
curl http://localhost:8082/metrics | grep -E "processed|consumed|latency"
```

## Next Steps

1. **Understand Pipeline**: Learn the processing pipeline
   - See `docs/guides/PLUGIN_ECOSYSTEM_QUICK_REFERENCE.md`

2. **Monitor Service**: Set up monitoring
   - See `docs/guides/DISTRIBUTED_ARCHITECTURE_MONITORING_ALERTING.md`

3. **Scale Service**: Deploy multiple replicas
   - See `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md`

4. **Integrate**: Connect to other services
   - See `cmd/chainpulse-puller/README.md`

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| PROCESSOR_PORT | 8082 | Port to listen on |
| INSTANCE_ID | event-processor-1 | Unique instance identifier |
| KAFKA_BROKERS | kafka-1:9092 | Kafka brokers |
| KAFKA_CONSUMER_GROUP | event-processor-consumers | Consumer group |
| KAFKA_INPUT_TOPICS | raw-events | Input topics |
| KAFKA_OUTPUT_TOPICS | processed-events | Output topics |
| BATCH_SIZE | 100 | Events per batch |
| BATCH_TIMEOUT | 5000 | Batch timeout (ms) |
| MAX_RETRIES | 3 | Retry attempts |
| WORKER_THREADS | 4 | Processing threads |
| LOG_LEVEL | info | Log level |

## Performance Tips

1. **Batch processing**: Increase `BATCH_SIZE` for throughput
2. **Parallel processing**: Increase `WORKER_THREADS`
3. **Consumer groups**: Use multiple instances with same group
4. **Monitoring**: Check `/metrics` endpoint regularly
5. **Scaling**: Add more replicas as needed

## Optional Security Surface

Enable the event-processor security surface only when you need to gate the
runtime control and read endpoints:

- `EVENT_PROCESSOR_AUTH_ENABLED=true`
- `EVENT_PROCESSOR_AUTH_JWT_SECRET=...`
- `EVENT_PROCESSOR_AUTH_API_KEYS=svc-key=client-1,svc-key-2=client-2`
- `EVENT_PROCESSOR_RATE_LIMIT_ENABLED=true`
- `EVENT_PROCESSOR_RATE_LIMIT=100`

## Monitoring Metrics

Key metrics to monitor:

```
# Events consumed per second
processor_events_consumed_total

# Events processed per second
processor_events_processed_total

# Processing latency
processor_processing_latency_seconds

# Error rate
processor_errors_total

# Dead-letter queue size
processor_dead_letter_queue_size
```

## Support

For more information:
- Full documentation: `cmd/chainpulse-event-processor/README.md`
- Architecture guide: `MICROSERVICES_ARCHITECTURE_START_HERE.md`
- Deployment guide: `DISTRIBUTED_DEPLOYMENT_START_HERE.md`
