# Microservices Implementation Guide
**Date**: January 12, 2026  
**Target**: Enterprise Web3 Project  
**Scope**: Step-by-step implementation of microservices architecture  

---

## Overview

This guide provides step-by-step instructions for implementing the microservices architecture for ChainPulse. It covers:
- Creating microservice entry points
- Extracting service logic
- Setting up service discovery
- Implementing horizontal scaling
- Deploying to Kubernetes

---

## Step 1: Create Microservice Entry Points

### 1.1 API Gateway Entry Point

**File**: `cmd/chainpulse-api-gateway/main.go`

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "chainpulse/pkg/core"
    "chainpulse/pkg/infrastructure/deployment"
)

func main() {
    fmt.Println("╔════════════════════════════════════════════════════════════╗")
    fmt.Println("║         ChainPulse - API Gateway Service                   ║")
    fmt.Println("║              Enterprise Web3 Event Indexing                ║")
    fmt.Println("╚════════════════════════════════════════════════════════════╝")
    fmt.Println()

    // Load configuration
    config := loadGatewayConfig()
    
    // Initialize core services
    logger := core.NewDefaultLogger(core.LogLevelInfo)
    metrics := core.NewDefaultMetricsCollector()
    registry := core.NewPluginRegistry(logger)

    fmt.Println("✓ Core services initialized")

    // Create API Gateway deployment
    gatewayDeployment := deployment.NewAPIGatewayDeployment(
        config,
        registry,
        logger,
        metrics,
    )

    // Register gateway service
    if err := gatewayDeployment.RegisterService(
        initializeGateway,
        startGateway,
        stopGateway,
    ); err != nil {
        logger.Error("Failed to register gateway", "error", err.Error())
        os.Exit(1)
    }

    fmt.Println("✓ API Gateway registered")

    // Initialize
    if err := gatewayDeployment.Initialize(context.Background()); err != nil {
        logger.Error("Failed to initialize", "error", err.Error())
        os.Exit(1)
    }

    fmt.Println("✓ API Gateway initialized")
    fmt.Printf("Listening on port %d\n", config.Port)

    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start service
    go func() {
        if err := gatewayDeployment.Start(context.Background()); err != nil {
            logger.Error("Gateway error", "error", err.Error())
        }
    }()

    // Wait for shutdown
    sig := <-sigChan
    fmt.Printf("\nReceived signal: %v\n", sig)

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := gatewayDeployment.Stop(ctx); err != nil {
        logger.Error("Error stopping gateway", "error", err.Error())
    }

    fmt.Println("✓ API Gateway stopped")
}

func initializeGateway() error {
    // Initialize gateway components
    return nil
}

func startGateway() error {
    // Start gateway server
    return nil
}

func stopGateway() error {
    // Stop gateway server
    return nil
}

type GatewayConfig struct {
    Port                int
    TLSEnabled          bool
    TLSCertPath         string
    TLSKeyPath          string
    UpstreamServices    []string
    RateLimitPerSecond  int
    AuthenticationToken string
}

func loadGatewayConfig() GatewayConfig {
    return GatewayConfig{
        Port:               getEnvInt("GATEWAY_PORT", 8080),
        TLSEnabled:         getEnvBool("GATEWAY_TLS_ENABLED", false),
        TLSCertPath:        getEnv("GATEWAY_TLS_CERT", ""),
        TLSKeyPath:         getEnv("GATEWAY_TLS_KEY", ""),
        RateLimitPerSecond: getEnvInt("GATEWAY_RATE_LIMIT", 1000),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        return value == "true" || value == "1"
    }
    return defaultValue
}
```

### 1.2 API Service Entry Point

**File**: `cmd/chainpulse-api-service/main.go`

Similar structure to API Gateway but for API service:
- Registers with Consul
- Connects to database
- Connects to cache
- Starts GraphQL/REST/gRPC endpoints
- Publishes health checks

### 1.3 Event Processor Entry Point

**File**: `cmd/chainpulse-event-processor/main.go`

Similar structure but for event processing:
- Connects to Kafka
- Registers consumer group
- Processes events
- Publishes to downstream topics

### 1.4 Puller Entry Point

**File**: `cmd/chainpulse-puller/main.go`

Similar structure but for data pulling:
- Connects to blockchain nodes
- Registers with Consul
- Fetches blocks/transactions
- Publishes to Kafka

---

## Step 2: Create Deployment Abstractions

### 2.1 API Gateway Deployment

**File**: `pkg/infrastructure/deployment/api_gateway_deployment.go`

```go
package deployment

import (
    "context"
    "chainpulse/pkg/core"
)

type APIGatewayDeployment struct {
    config           core.Config
    registry         core.PluginRegistry
    logger           core.Logger
    metricsCollector core.MetricsCollector
    // ... other fields
}

func NewAPIGatewayDeployment(
    config core.Config,
    registry core.PluginRegistry,
    logger core.Logger,
    metricsCollector core.MetricsCollector,
) *APIGatewayDeployment {
    return &APIGatewayDeployment{
        config:           config,
        registry:         registry,
        logger:           logger,
        metricsCollector: metricsCollector,
    }
}

func (agd *APIGatewayDeployment) RegisterService(
    initializer func() error,
    starter func() error,
    stopper func() error,
) error {
    // Register service lifecycle
    return nil
}

func (agd *APIGatewayDeployment) Initialize(ctx context.Context) error {
    // Initialize gateway
    return nil
}

func (agd *APIGatewayDeployment) Start(ctx context.Context) error {
    // Start gateway
    return nil
}

func (agd *APIGatewayDeployment) Stop(ctx context.Context) error {
    // Stop gateway
    return nil
}
```

---

## Step 3: Service Discovery Integration

### 3.1 Consul Registration

**File**: `pkg/infrastructure/service_discovery/consul_registry.go`

```go
package service_discovery

import (
    "fmt"
    "github.com/hashicorp/consul/api"
)

type ConsulRegistry struct {
    client *api.Client
}

func NewConsulRegistry(address string) (*ConsulRegistry, error) {
    config := api.DefaultConfig()
    config.Address = address
    
    client, err := api.NewClient(config)
    if err != nil {
        return nil, err
    }
    
    return &ConsulRegistry{client: client}, nil
}

func (cr *ConsulRegistry) Register(
    serviceID string,
    serviceName string,
    port int,
    address string,
    tags []string,
) error {
    registration := &api.AgentServiceRegistration{
        ID:      serviceID,
        Name:    serviceName,
        Port:    port,
        Address: address,
        Tags:    tags,
        Check: &api.AgentServiceCheck{
            HTTP:     fmt.Sprintf("http://%s:%d/health", address, port),
            Interval: "10s",
            Timeout:  "5s",
        },
    }
    
    return cr.client.Agent().ServiceRegister(registration)
}

func (cr *ConsulRegistry) Deregister(serviceID string) error {
    return cr.client.Agent().ServiceDeregister(serviceID)
}

func (cr *ConsulRegistry) Discover(serviceName string) ([]*api.ServiceEntry, error) {
    entries, _, err := cr.client.Health().Service(serviceName, "", true, nil)
    return entries, err
}
```

---

## Step 4: Kubernetes Manifests

### 4.1 API Gateway Deployment

**File**: `k8s/api-gateway-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chainpulse-api-gateway
  namespace: chainpulse-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: chainpulse-api-gateway
  template:
    metadata:
      labels:
        app: chainpulse-api-gateway
    spec:
      containers:
      - name: api-gateway
        image: chainpulse/api-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: GATEWAY_PORT
          value: "8080"
        - name: CONSUL_ADDRESS
          value: "consul:8500"
        - name: LOG_LEVEL
          value: "info"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: chainpulse-api-gateway
  namespace: chainpulse-api
spec:
  type: LoadBalancer
  selector:
    app: chainpulse-api-gateway
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
```

### 4.2 API Service Deployment

**File**: `k8s/api-service-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chainpulse-api-service
  namespace: chainpulse-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chainpulse-api-service
  template:
    metadata:
      labels:
        app: chainpulse-api-service
    spec:
      containers:
      - name: api-service
        image: chainpulse/api-service:latest
        ports:
        - containerPort: 8081
        env:
        - name: SERVICE_PORT
          value: "8081"
        - name: CONSUL_ADDRESS
          value: "consul:8500"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-credentials
              key: url
        - name: REDIS_CLUSTER
          value: "redis-1:6379,redis-2:6379,redis-3:6379"
        - name: KAFKA_BROKERS
          value: "kafka-1:9092,kafka-2:9092,kafka-3:9092"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 1000m
            memory: 1Gi
          limits:
            cpu: 2000m
            memory: 2Gi
---
apiVersion: v1
kind: Service
metadata:
  name: chainpulse-api-service
  namespace: chainpulse-api
spec:
  clusterIP: None
  selector:
    app: chainpulse-api-service
  ports:
  - protocol: TCP
    port: 8081
    targetPort: 8081
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: chainpulse-api-service-hpa
  namespace: chainpulse-api
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: chainpulse-api-service
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## Step 5: Docker Images

### 5.1 API Gateway Dockerfile

**File**: `docker/api-gateway.Dockerfile`

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o api-gateway ./cmd/chainpulse-api-gateway

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]
```

---

## Step 6: Build Automation

### 6.1 Root Makefile

**File**: `Makefile`

```makefile
.PHONY: build-all build-api-gateway build-api-service build-event-processor build-puller

build-all: build-api-gateway build-api-service build-event-processor build-puller
	@echo "✓ All microservices built"

build-api-gateway:
	@cd cmd/chainpulse-api-gateway && make build

build-api-service:
	@cd cmd/chainpulse-api-service && make build

build-event-processor:
	@cd cmd/chainpulse-event-processor && make build

build-puller:
	@cd cmd/chainpulse-puller && make build

docker-build-all:
	@docker build -f docker/api-gateway.Dockerfile -t chainpulse/api-gateway:latest .
	@docker build -f docker/api-service.Dockerfile -t chainpulse/api-service:latest .
	@docker build -f docker/event-processor.Dockerfile -t chainpulse/event-processor:latest .
	@docker build -f docker/puller.Dockerfile -t chainpulse/puller:latest .

k8s-deploy:
	@kubectl apply -f k8s/api-gateway-deployment.yaml
	@kubectl apply -f k8s/api-service-deployment.yaml
	@kubectl apply -f k8s/event-processor-deployment.yaml
	@kubectl apply -f k8s/puller-deployment.yaml

k8s-scale-api:
	@kubectl scale deployment chainpulse-api-service --replicas=10 -n chainpulse-api

k8s-scale-processor:
	@kubectl scale deployment chainpulse-event-processor --replicas=20 -n chainpulse-processing

k8s-scale-puller:
	@kubectl scale deployment chainpulse-puller --replicas=15 -n chainpulse-data
```

---

## Step 7: Configuration Management

### 7.1 Service Configuration

**File**: `config/api-service.yaml`

```yaml
service:
  name: chainpulse-api-service
  port: 8081
  instance_id: ${HOSTNAME}

service_discovery:
  consul:
    address: ${CONSUL_ADDRESS:consul:8500}
    service_name: chainpulse-api-service
    health_check_interval: 5s

database:
  primary:
    host: ${DB_PRIMARY_HOST:postgres-primary}
    port: ${DB_PRIMARY_PORT:5432}
    user: ${DB_USER}
    password: ${DB_PASSWORD}
    database: chainpulse
    pool_size: 20
  replicas:
    - host: ${DB_REPLICA_1_HOST:postgres-replica-1}
      port: 5432
    - host: ${DB_REPLICA_2_HOST:postgres-replica-2}
      port: 5432

cache:
  redis:
    cluster:
      - ${REDIS_1:redis-1:6379}
      - ${REDIS_2:redis-2:6379}
      - ${REDIS_3:redis-3:6379}
    ttl: 3600s

message_queue:
  kafka:
    brokers:
      - ${KAFKA_1:kafka-1:9092}
      - ${KAFKA_2:kafka-2:9092}
      - ${KAFKA_3:kafka-3:9092}
    consumer_group: api-service-consumers

logging:
  level: ${LOG_LEVEL:info}
  format: json
```

---

## Deployment Checklist

- [ ] Create microservice entry points
- [ ] Create deployment abstractions
- [ ] Implement service discovery
- [ ] Create Kubernetes manifests
- [ ] Create Docker images
- [ ] Set up build automation
- [ ] Configure services
- [ ] Deploy to Kubernetes
- [ ] Verify service discovery
- [ ] Test horizontal scaling
- [ ] Set up monitoring
- [ ] Set up logging
- [ ] Document runbooks
- [ ] Train team

---

## Next Steps

1. Start with API Gateway implementation
2. Extract API service logic
3. Implement event processor
4. Implement puller services
5. Set up infrastructure
6. Load test and optimize
7. Deploy to production

---

**Status**: Ready for implementation  
**Estimated Time**: 16 weeks  
**Team Size**: 4-6 engineers  
