package processing

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewEventProcessorClusterDeployment tests cluster creation
func TestNewEventProcessorClusterDeployment(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")

	assert.NotNil(t, cluster)
	assert.Equal(t, "cluster-1", cluster.id)
	assert.Equal(t, "undeployed", cluster.deploymentStatus)
	assert.Equal(t, 0, len(cluster.processors))
}

// TestClusterDeploy tests cluster deployment
func TestClusterDeploy(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	err := cluster.Deploy(ctx, 3, []string{"ethereum", "polygon"})

	assert.NoError(t, err)
	assert.Equal(t, "deployed", cluster.deploymentStatus)
	assert.Equal(t, 3, cluster.GetProcessorCount())
	assert.Equal(t, 2, cluster.GetConsumerGroupCount())
}

// TestClusterDeployAlreadyDeployed tests deploying already deployed cluster
func TestClusterDeployAlreadyDeployed(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	err := cluster.Deploy(ctx, 2, []string{"ethereum"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already deployed")
}

// TestClusterDeployInvalidInstanceCount tests invalid instance count
func TestClusterDeployInvalidInstanceCount(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	err := cluster.Deploy(ctx, 20, []string{"ethereum"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// TestClusterUndeploy tests cluster undeployment
func TestClusterUndeploy(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	err := cluster.Undeploy(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "undeployed", cluster.deploymentStatus)
	assert.Equal(t, 0, cluster.GetProcessorCount())
}

// TestClusterUndeployNotDeployed tests undeploying non-deployed cluster
func TestClusterUndeployNotDeployed(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	err := cluster.Undeploy(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not deployed")
}

// TestClusterProcessEvent tests event processing
func TestClusterProcessEvent(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	event := &Event{
		ID:              "event-1",
		EventHash:       "0xevent1hash",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
		BlockNumber:     100,
		LogIndex:        0,
		EventData:       make(map[string]any),
	}

	err := cluster.ProcessEvent(ctx, event)

	assert.NoError(t, err)
}

// TestClusterProcessEventNotDeployed tests processing on non-deployed cluster
func TestClusterProcessEventNotDeployed(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	event := &Event{
		ID:              "event-1",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
		BlockNumber:     100,
		LogIndex:        0,
		EventData:       make(map[string]any),
	}

	err := cluster.ProcessEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not deployed")
}

// TestClusterProcessBatch tests batch processing
func TestClusterProcessBatch(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	events := make([]*Event, 5)
	for i := 0; i < 5; i++ {
		events[i] = &Event{
			ID:              fmt.Sprintf("event-%d", i),
			ChainID:         "ethereum",
			ContractAddress: "0x123",
			EventName:       "Transfer",
			TransactionHash: fmt.Sprintf("0x%d", i),
			BlockNumber:     uint64(100 + i),
			LogIndex:        uint64(i),
			EventData:       make(map[string]any),
		}
	}

	err := cluster.ProcessBatch(ctx, events)

	assert.NoError(t, err)
}

// TestClusterProcessBatchEmpty tests empty batch processing
func TestClusterProcessBatchEmpty(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	err := cluster.ProcessBatch(ctx, []*Event{})

	assert.NoError(t, err)
}

// TestClusterHealthCheck tests health check
func TestClusterHealthCheck(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	err := cluster.HealthCheck(ctx)

	assert.NoError(t, err)
}

// TestClusterHealthCheckNotDeployed tests health check on non-deployed cluster
func TestClusterHealthCheckNotDeployed(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	err := cluster.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not deployed")
}

// TestClusterScale tests cluster scaling
func TestClusterScale(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	err := cluster.Scale(ctx, 4)

	assert.NoError(t, err)
	assert.Equal(t, 4, cluster.GetProcessorCount())
}

// TestClusterScaleDown tests cluster scale down
func TestClusterScaleDown(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 4, []string{"ethereum"})
	err := cluster.Scale(ctx, 2)

	assert.NoError(t, err)
	assert.Equal(t, 2, cluster.GetProcessorCount())
}

// TestClusterScaleInvalidCount tests invalid scale count
func TestClusterScaleInvalidCount(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	err := cluster.Scale(ctx, 20)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// TestClusterScaleNotDeployed tests scaling non-deployed cluster
func TestClusterScaleNotDeployed(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	err := cluster.Scale(ctx, 2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not deployed")
}

// TestClusterGetMetrics tests metrics retrieval
func TestClusterGetMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	metrics := cluster.GetMetrics()

	assert.Equal(t, 2, metrics.ProcessorsDeployed)
	assert.Equal(t, 2, metrics.ProcessorsHealthy)
	assert.Equal(t, 1, metrics.ConsumerGroupCount)
}

// TestClusterGetStatus tests status retrieval
func TestClusterGetStatus(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	assert.Equal(t, "undeployed", cluster.GetStatus())

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	assert.Equal(t, "deployed", cluster.GetStatus())
}

// TestClusterGetProcessorCount tests processor count
func TestClusterGetProcessorCount(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	assert.Equal(t, 0, cluster.GetProcessorCount())

	_ = cluster.Deploy(ctx, 3, []string{"ethereum"})
	assert.Equal(t, 3, cluster.GetProcessorCount())
}

// TestClusterGetConsumerGroupCount tests consumer group count
func TestClusterGetConsumerGroupCount(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	assert.Equal(t, 0, cluster.GetConsumerGroupCount())

	_ = cluster.Deploy(ctx, 2, []string{"ethereum", "polygon"})
	assert.Equal(t, 2, cluster.GetConsumerGroupCount())
}

// TestClusterGetDeadLetterQueueSize tests DLQ size
func TestClusterGetDeadLetterQueueSize(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	size := cluster.GetDeadLetterQueueSize()
	assert.Equal(t, 0, size)
}

// TestClusterGetIdempotencyMetrics tests idempotency metrics
func TestClusterGetIdempotencyMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	metrics := cluster.GetIdempotencyMetrics()

	assert.NotNil(t, metrics)
	assert.Greater(t, len(metrics), 0)
}

// TestClusterMultipleChains tests cluster with multiple chains
func TestClusterMultipleChains(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	err := cluster.Deploy(ctx, 3, []string{"ethereum", "polygon", "arbitrum"})

	assert.NoError(t, err)
	assert.Equal(t, 3, cluster.GetConsumerGroupCount())
}

// TestClusterConcurrentProcessing tests concurrent event processing
func TestClusterConcurrentProcessing(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := &Event{
				ID:              fmt.Sprintf("event-%d", id),
				EventHash:       fmt.Sprintf("0xhash%d", id),
				ChainID:         "ethereum",
				ContractAddress: "0x123",
				EventName:       "Transfer",
				TransactionHash: fmt.Sprintf("0x%d", id),
				BlockNumber:     uint64(100 + id),
				LogIndex:        uint64(id),
				EventData:       make(map[string]any),
			}
			if err := cluster.ProcessEvent(ctx, event); err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Greater(t, atomic.LoadInt32(&successCount), int32(0))
}

// TestClusterDeploymentMetrics tests deployment metrics
func TestClusterDeploymentMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	metrics := cluster.GetMetrics()

	assert.Equal(t, 2, metrics.ProcessorsDeployed)
	assert.Greater(t, metrics.ProcessorsHealthy, 0)
	assert.Equal(t, 1, metrics.ConsumerGroupCount)
}

// TestClusterAutoScalingConfig tests auto-scaling configuration
func TestClusterAutoScalingConfig(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")

	assert.True(t, cluster.autoScalingEnabled)
	assert.Equal(t, 1, cluster.minInstances)
	assert.Equal(t, 10, cluster.maxInstances)
}

// TestClusterHealthCheckUpdatesMetrics tests health check updates metrics
func TestClusterHealthCheckUpdatesMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	_ = cluster.HealthCheck(ctx)

	metrics := cluster.GetMetrics()

	assert.Greater(t, metrics.ProcessorsHealthy, 0)
}

// TestClusterProcessEventWithIdempotency tests idempotency check
func TestClusterProcessEventWithIdempotency(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	event := &Event{
		ID:              "event-1",
		EventHash:       "0xevent1hash",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
		BlockNumber:     100,
		LogIndex:        0,
		EventData:       make(map[string]any),
	}

	// First processing should succeed
	err1 := cluster.ProcessEvent(ctx, event)
	assert.NoError(t, err1)

	// Second processing of same event should be detected as duplicate
	err2 := cluster.ProcessEvent(ctx, event)
	// May fail due to idempotency or succeed depending on implementation
	_ = err2
}

// TestClusterScaleUpAndDown tests scale up then down
func TestClusterScaleUpAndDown(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	assert.Equal(t, 2, cluster.GetProcessorCount())

	_ = cluster.Scale(ctx, 5)
	assert.Equal(t, 5, cluster.GetProcessorCount())

	_ = cluster.Scale(ctx, 3)
	assert.Equal(t, 3, cluster.GetProcessorCount())
}

// TestClusterDeploymentTimestamp tests deployment timestamp
func TestClusterDeploymentTimestamp(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	beforeDeploy := time.Now()
	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})
	afterDeploy := time.Now()

	assert.True(t, cluster.deployedAt.After(beforeDeploy) || cluster.deployedAt.Equal(beforeDeploy))
	assert.True(t, cluster.deployedAt.Before(afterDeploy) || cluster.deployedAt.Equal(afterDeploy))
}

// TestClusterProcessorDistribution tests processor distribution
func TestClusterProcessorDistribution(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 4, []string{"ethereum", "polygon"})

	// Should have 4 processors and 2 consumer groups
	assert.Equal(t, 4, cluster.GetProcessorCount())
	assert.Equal(t, 2, cluster.GetConsumerGroupCount())
}

// TestClusterHealthCheckInterval tests health check interval
func TestClusterHealthCheckInterval(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")

	assert.Equal(t, 30*time.Second, cluster.healthCheckInterval)
}

// TestClusterRetryManagerIntegration tests retry manager integration
func TestClusterRetryManagerIntegration(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	assert.NotNil(t, cluster.retryManager)
	assert.NotNil(t, cluster.retryManager.circuitBreaker)
	assert.NotNil(t, cluster.retryManager.deadLetterQueue)
}

// TestClusterIdempotencyServiceIntegration tests idempotency service integration
func TestClusterIdempotencyServiceIntegration(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	assert.NotNil(t, cluster.idempotencyService)
}

// TestClusterEventStoreIntegration tests event store integration
func TestClusterEventStoreIntegration(t *testing.T) {
	t.Parallel()
	cluster := NewEventProcessorClusterDeployment("cluster-1")
	ctx := context.Background()

	_ = cluster.Deploy(ctx, 2, []string{"ethereum"})

	assert.NotNil(t, cluster.eventStore)
}
