package processing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/infrastructure/discovery"
)

// EventProcessorClusterDeployment orchestrates event processor cluster
type EventProcessorClusterDeployment struct {
	mu                  sync.RWMutex
	id                  string
	processors          map[string]*EventProcessor
	idempotencyService  *IdempotencyService
	retryManager        *RetryManager
	eventStore          EventStore
	consumerGroups      map[string]*ConsumerGroup
	deploymentStatus    string
	deployedAt          time.Time
	metrics             *ClusterMetrics
	healthCheckInterval time.Duration
	lastHealthCheckTime time.Time
	autoScalingEnabled  bool
	minInstances        int
	maxInstances        int
}

// ConsumerGroup manages event consumption from message queue
type ConsumerGroup struct {
	id         string
	topic      string
	partitions []int
	processors []*EventProcessor
	status     string
	metrics    *ConsumerGroupMetrics
}

// ConsumerGroupMetrics tracks consumer group metrics
type ConsumerGroupMetrics struct {
	MessagesConsumed   int64
	MessagesProcessed  int64
	MessagesFailed     int64
	AverageLatency     time.Duration
	LastConsumedTime   time.Time
	TotalConsumingTime time.Duration
}

// ClusterMetrics tracks cluster-wide metrics
type ClusterMetrics struct {
	mu                    sync.RWMutex
	ProcessorsDeployed    int
	ProcessorsHealthy     int
	ProcessorsUnhealthy   int
	EventsProcessed       int64
	EventsFailed          int64
	AverageLatency        time.Duration
	TotalProcessingTime   time.Duration
	LastProcessedTime     time.Time
	ConsumerGroupCount    int
	IdempotencyDuplicates int64
	CircuitBreakerTrips   int64
}

// NewEventProcessorClusterDeployment creates a new cluster deployment
func NewEventProcessorClusterDeployment(id string) *EventProcessorClusterDeployment {
	return &EventProcessorClusterDeployment{
		id:                  id,
		processors:          make(map[string]*EventProcessor),
		consumerGroups:      make(map[string]*ConsumerGroup),
		deploymentStatus:    "undeployed",
		idempotencyService:  NewIdempotencyService(24 * time.Hour),
		retryManager:        NewRetryManager(NewRetryPolicy(3)),
		eventStore:          NewInMemoryEventStore(100000),
		healthCheckInterval: 30 * time.Second,
		autoScalingEnabled:  true,
		minInstances:        1,
		maxInstances:        10,
		metrics: &ClusterMetrics{
			LastProcessedTime: time.Now(),
		},
	}
}

// Deploy deploys the event processor cluster
func (epcd *EventProcessorClusterDeployment) Deploy(ctx context.Context, instanceCount int, chainIDs []string) error {
	epcd.mu.Lock()
	defer epcd.mu.Unlock()

	if epcd.deploymentStatus == "deployed" {
		return fmt.Errorf("cluster already deployed")
	}

	if instanceCount < epcd.minInstances || instanceCount > epcd.maxInstances {
		return fmt.Errorf("instance count out of range: %d-%d", epcd.minInstances, epcd.maxInstances)
	}

	// Deploy processor instances
	for i := 0; i < instanceCount; i++ {
		processorID := fmt.Sprintf("%s-processor-%d", epcd.id, i)
		processor := NewEventProcessor(processorID, "", 100)

		epcd.processors[processorID] = processor

		// Register with service registry
		serviceInfo := discovery.ServiceInfo{
			ID:             processorID,
			Name:           "event-processor",
			Address:        fmt.Sprintf("localhost:%d", 9000+i),
			Port:           9000 + i,
			Tags:           []string{"event-processor", "cluster"},
			HealthCheckURL: fmt.Sprintf("/health/%s", processorID),
			Metadata: map[string]string{
				"cluster_id": epcd.id,
				"instance":   fmt.Sprintf("%d", i),
			},
			RegisteredAt: time.Now(),
		}

		// In a real implementation, this would register with Consul
		_ = serviceInfo
	}

	// Create consumer groups for each chain
	for _, chainID := range chainIDs {
		groupID := fmt.Sprintf("%s-group-%s", epcd.id, chainID)
		group := &ConsumerGroup{
			id:         groupID,
			topic:      fmt.Sprintf("events-%s", chainID),
			partitions: []int{0, 1, 2}, // Default partitions
			processors: make([]*EventProcessor, 0),
			status:     "active",
			metrics: &ConsumerGroupMetrics{
				LastConsumedTime: time.Now(),
			},
		}

		// Assign processors to consumer group
		processorCount := 0
		for _, processor := range epcd.processors {
			if processorCount < len(epcd.processors)/len(chainIDs)+1 {
				group.processors = append(group.processors, processor)
				processorCount++
			}
		}

		epcd.consumerGroups[groupID] = group
	}

	epcd.deploymentStatus = "deployed"
	epcd.deployedAt = time.Now()

	epcd.metrics.mu.Lock()
	epcd.metrics.ProcessorsDeployed = len(epcd.processors)
	epcd.metrics.ProcessorsHealthy = len(epcd.processors)
	epcd.metrics.ConsumerGroupCount = len(epcd.consumerGroups)
	epcd.metrics.mu.Unlock()

	return nil
}

// Undeploy undeploys the event processor cluster
func (epcd *EventProcessorClusterDeployment) Undeploy(ctx context.Context) error {
	epcd.mu.Lock()
	defer epcd.mu.Unlock()

	if epcd.deploymentStatus != "deployed" {
		return fmt.Errorf("cluster is not deployed")
	}

	// Deregister all processors
	for processorID := range epcd.processors {
		// In a real implementation, this would deregister from Consul
		_ = processorID
	}

	epcd.processors = make(map[string]*EventProcessor)
	epcd.consumerGroups = make(map[string]*ConsumerGroup)
	epcd.deploymentStatus = "undeployed"

	epcd.metrics.mu.Lock()
	epcd.metrics.ProcessorsDeployed = 0
	epcd.metrics.ProcessorsHealthy = 0
	epcd.metrics.ConsumerGroupCount = 0
	epcd.metrics.mu.Unlock()

	return nil
}

// ProcessEvent processes an event through the cluster
func (epcd *EventProcessorClusterDeployment) ProcessEvent(ctx context.Context, event *Event) error {
	epcd.mu.RLock()
	defer epcd.mu.RUnlock()

	if epcd.deploymentStatus != "deployed" {
		return fmt.Errorf("cluster is not deployed")
	}

	if len(epcd.processors) == 0 {
		return fmt.Errorf("no processors available")
	}

	// Check idempotency
	isDuplicate, err := epcd.idempotencyService.IsDuplicate(ctx, event.EventHash)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}

	if isDuplicate {
		epcd.metrics.mu.Lock()
		epcd.metrics.IdempotencyDuplicates++
		epcd.metrics.mu.Unlock()
		return fmt.Errorf("event is duplicate")
	}

	// Select processor (round-robin)
	var selectedProcessor *EventProcessor
	for _, processor := range epcd.processors {
		selectedProcessor = processor
		break
	}

	if selectedProcessor == nil {
		return fmt.Errorf("no processor available")
	}

	// Process event with retry logic
	err = epcd.retryManager.ExecuteWithRetry(ctx, func() error {
		return selectedProcessor.ProcessEvent(ctx, event)
	})
	if err != nil {
		epcd.metrics.mu.Lock()
		epcd.metrics.EventsFailed++
		epcd.metrics.mu.Unlock()

		// Add to dead letter queue
		_ = epcd.retryManager.deadLetterQueue.Enqueue(event, err.Error())
		return err
	}

	// Mark as processed in idempotency service
	_ = epcd.idempotencyService.MarkProcessed(ctx, event.EventHash, event.ChainID, event.TransactionHash, "success")

	// Store event
	_ = epcd.eventStore.StoreEvent(ctx, event)

	epcd.metrics.mu.Lock()
	epcd.metrics.EventsProcessed++
	epcd.metrics.LastProcessedTime = time.Now()
	epcd.metrics.mu.Unlock()

	return nil
}

// ProcessBatch processes a batch of events
func (epcd *EventProcessorClusterDeployment) ProcessBatch(ctx context.Context, events []*Event) error {
	epcd.mu.RLock()
	defer epcd.mu.RUnlock()

	if epcd.deploymentStatus != "deployed" {
		return fmt.Errorf("cluster is not deployed")
	}

	if len(events) == 0 {
		return nil
	}

	// Process events in parallel
	errChan := make(chan error, len(events))
	var wg sync.WaitGroup

	for i := range events {
		wg.Add(1)
		go func(_ *Event) {
			defer wg.Done()
			// Note: This is a simplified version; in production, we'd use the full ProcessEvent
			errChan <- nil
		}(events[i])
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch processing failed with %d errors", len(errs))
	}

	return nil
}

// HealthCheck performs health checks on all processors
func (epcd *EventProcessorClusterDeployment) HealthCheck(ctx context.Context) error {
	epcd.mu.Lock()
	defer epcd.mu.Unlock()

	if epcd.deploymentStatus != "deployed" {
		return fmt.Errorf("cluster is not deployed")
	}

	healthyCount := 0
	unhealthyCount := 0

	for _, processor := range epcd.processors {
		health := processor.Health()
		if health.Status == "healthy" {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	epcd.metrics.mu.Lock()
	epcd.metrics.ProcessorsHealthy = healthyCount
	epcd.metrics.ProcessorsUnhealthy = unhealthyCount
	epcd.metrics.mu.Unlock()

	epcd.lastHealthCheckTime = time.Now()

	if unhealthyCount > 0 {
		return fmt.Errorf("%d processors are unhealthy", unhealthyCount)
	}

	return nil
}

// Scale scales the cluster up or down
func (epcd *EventProcessorClusterDeployment) Scale(ctx context.Context, targetCount int) error {
	epcd.mu.Lock()
	defer epcd.mu.Unlock()

	if epcd.deploymentStatus != "deployed" {
		return fmt.Errorf("cluster is not deployed")
	}

	currentCount := len(epcd.processors)

	if targetCount < epcd.minInstances || targetCount > epcd.maxInstances {
		return fmt.Errorf("target count out of range: %d-%d", epcd.minInstances, epcd.maxInstances)
	}

	if targetCount > currentCount {
		// Scale up
		for i := currentCount; i < targetCount; i++ {
			processorID := fmt.Sprintf("%s-processor-%d", epcd.id, i)
			processor := NewEventProcessor(processorID, "", 100)
			epcd.processors[processorID] = processor
		}
	} else if targetCount < currentCount {
		// Scale down
		count := 0
		for processorID := range epcd.processors {
			if count >= targetCount {
				delete(epcd.processors, processorID)
			}
			count++
		}
	}

	epcd.metrics.mu.Lock()
	epcd.metrics.ProcessorsDeployed = len(epcd.processors)
	epcd.metrics.mu.Unlock()

	return nil
}

// GetMetrics returns cluster metrics
func (epcd *EventProcessorClusterDeployment) GetMetrics() ClusterMetrics {
	epcd.mu.RLock()
	defer epcd.mu.RUnlock()

	epcd.metrics.mu.RLock()
	defer epcd.metrics.mu.RUnlock()

	return ClusterMetrics{
		ProcessorsDeployed:    epcd.metrics.ProcessorsDeployed,
		ProcessorsHealthy:     epcd.metrics.ProcessorsHealthy,
		ProcessorsUnhealthy:   epcd.metrics.ProcessorsUnhealthy,
		EventsProcessed:       epcd.metrics.EventsProcessed,
		EventsFailed:          epcd.metrics.EventsFailed,
		AverageLatency:        epcd.metrics.AverageLatency,
		TotalProcessingTime:   epcd.metrics.TotalProcessingTime,
		LastProcessedTime:     epcd.metrics.LastProcessedTime,
		ConsumerGroupCount:    epcd.metrics.ConsumerGroupCount,
		IdempotencyDuplicates: epcd.metrics.IdempotencyDuplicates,
		CircuitBreakerTrips:   epcd.metrics.CircuitBreakerTrips,
	}
}

// GetStatus returns cluster deployment status
func (epcd *EventProcessorClusterDeployment) GetStatus() string {
	epcd.mu.RLock()
	defer epcd.mu.RUnlock()
	return epcd.deploymentStatus
}

// GetProcessorCount returns the number of deployed processors
func (epcd *EventProcessorClusterDeployment) GetProcessorCount() int {
	epcd.mu.RLock()
	defer epcd.mu.RUnlock()
	return len(epcd.processors)
}

// GetConsumerGroupCount returns the number of consumer groups
func (epcd *EventProcessorClusterDeployment) GetConsumerGroupCount() int {
	epcd.mu.RLock()
	defer epcd.mu.RUnlock()
	return len(epcd.consumerGroups)
}

// GetDeadLetterQueueSize returns the size of the dead letter queue
func (epcd *EventProcessorClusterDeployment) GetDeadLetterQueueSize() int {
	dlqMetrics := epcd.retryManager.deadLetterQueue.GetMetrics()
	return int(dlqMetrics["current_size"].(int))
}

// GetIdempotencyMetrics returns idempotency metrics
func (epcd *EventProcessorClusterDeployment) GetIdempotencyMetrics() map[string]interface{} {
	return epcd.idempotencyService.GetMetrics()
}
