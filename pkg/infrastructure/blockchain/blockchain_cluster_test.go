package blockchain

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/assert"
)

func skipBlockchainClusterConcurrencyTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping blockchain cluster concurrency test in short mode")
	}
}

func TestNewBlockchainCluster(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 2, 5)

	assert.NotNil(t, cluster)
	assert.Equal(t, "test-cluster", cluster.id)
	assert.Equal(t, "EVM", cluster.blockchainType)
	assert.Equal(t, 2, cluster.minInstances)
	assert.Equal(t, 5, cluster.maxInstances)
	assert.Equal(t, 0, len(cluster.instances))
}

func TestDeployCluster(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 2, 5)

	err := cluster.Deploy(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 2, len(cluster.instances))
	assert.Equal(t, 2, cluster.GetInstanceCount())
}

func TestClusterProcessEvent(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	event := &core.BlockchainEvent{
		ID:              "test-event",
		ChainID:         "EVM",
		ContractAddress: common.Address{0x1},
		EventName:       "Transfer",
		CreatedAt:       time.Now(),
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.NoError(t, err)
}

func TestProcessEventNoInstances(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 0, 5)

	event := &core.BlockchainEvent{
		ID:      "test-event",
		ChainID: "EVM",
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no instances available")
}

func TestProcessEventNoRunningInstances(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	event := &core.BlockchainEvent{
		ID:      "test-event",
		ChainID: "EVM",
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running instances available")
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 2, 5)
	_ = cluster.Deploy(context.Background())

	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	err := cluster.HealthCheck(context.Background())

	assert.NoError(t, err)
	metrics := cluster.GetMetrics()
	assert.Equal(t, 2, metrics.InstancesHealthy)
}

func TestClusterGetMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	metrics := cluster.GetMetrics()

	assert.Equal(t, 1, metrics.InstancesDeployed)
	assert.Greater(t, metrics.InstancesDeployed, 0)
}

func TestGetBlockchainType(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "Cosmos", "Cosmos", 1, 5)

	blockchainType := cluster.GetBlockchainType()

	assert.Equal(t, "Cosmos", blockchainType)
}

func TestGetInstanceCount(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 3, 5)
	_ = cluster.Deploy(context.Background())

	count := cluster.GetInstanceCount()

	assert.Equal(t, 3, count)
}

func TestNewMultiBlockchainClusterManager(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	assert.NotNil(t, manager)
	assert.Equal(t, 0, len(manager.clusters))
}

func TestRegisterCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)

	err := manager.RegisterCluster(cluster)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.clusters))
}

func TestRegisterClusterDuplicate(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster1 := NewBlockchainCluster("test-cluster-1", "EVM", "EVM", 1, 5)
	cluster2 := NewBlockchainCluster("test-cluster-2", "EVM", "EVM", 1, 5)

	_ = manager.RegisterCluster(cluster1)
	err := manager.RegisterCluster(cluster2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestManagerProcessEventWithCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	_ = manager.RegisterCluster(cluster)

	event := &core.BlockchainEvent{
		ID:              "test-event",
		ChainID:         "EVM",
		ContractAddress: common.Address{0x1},
		EventName:       "Transfer",
		CreatedAt:       time.Now(),
	}

	err := manager.ProcessEvent(context.Background(), event)

	assert.NoError(t, err)
}

func TestManagerProcessEventNoCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	event := &core.BlockchainEvent{
		ID:      "test-event",
		ChainID: "EVM",
	}

	err := manager.ProcessEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no cluster")
}

func TestGetCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = manager.RegisterCluster(cluster)

	retrieved, err := manager.GetCluster("EVM")

	assert.NoError(t, err)
	assert.Equal(t, cluster, retrieved)
}

func TestGetClusterNotFound(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	_, err := manager.GetCluster("EVM")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetAllClusters(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster1 := NewBlockchainCluster("test-cluster-1", "EVM", "EVM", 1, 5)
	cluster2 := NewBlockchainCluster("test-cluster-2", "Cosmos", "Cosmos", 1, 5)

	_ = manager.RegisterCluster(cluster1)
	_ = manager.RegisterCluster(cluster2)

	clusters := manager.GetAllClusters()

	assert.Equal(t, 2, len(clusters))
}

func TestManagerGetMetrics(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 2, 5)
	_ = cluster.Deploy(context.Background())
	_ = manager.RegisterCluster(cluster)

	metrics := manager.GetMetrics()

	assert.Equal(t, 1, metrics["total_clusters"])
	assert.Equal(t, 2, metrics["total_instances"])
}

func TestConcurrentProcessing(t *testing.T) {
	t.Parallel()
	skipBlockchainClusterConcurrencyTestsInShortMode(t)

	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 2, 5)
	_ = cluster.Deploy(context.Background())

	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := &core.BlockchainEvent{
				ID:              fmt.Sprintf("event-%d", idx),
				ChainID:         "EVM",
				ContractAddress: common.Address{0x1},
				EventName:       "Transfer",
				CreatedAt:       time.Now(),
			}
			if err := cluster.ProcessEvent(context.Background(), event); err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&successCount))
}

func TestMultipleBlockchains(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	evmCluster := NewBlockchainCluster("evm-cluster", "EVM", "EVM", 1, 5)
	_ = evmCluster.Deploy(context.Background())
	for _, instance := range evmCluster.instances {
		instance.Status = "running"
	}

	cosmosCluster := NewBlockchainCluster("cosmos-cluster", "Cosmos", "Cosmos", 1, 5)
	_ = cosmosCluster.Deploy(context.Background())
	for _, instance := range cosmosCluster.instances {
		instance.Status = "running"
	}

	_ = manager.RegisterCluster(evmCluster)
	_ = manager.RegisterCluster(cosmosCluster)

	evmEvent := &core.BlockchainEvent{
		ID:              "evm-event",
		ChainID:         "EVM",
		ContractAddress: common.Address{0x1},
		EventName:       "Transfer",
		CreatedAt:       time.Now(),
	}

	cosmosEvent := &core.BlockchainEvent{
		ID:        "cosmos-event",
		ChainID:   "Cosmos",
		EventName: "Transfer",
		CreatedAt: time.Now(),
	}

	err1 := manager.ProcessEvent(context.Background(), evmEvent)
	err2 := manager.ProcessEvent(context.Background(), cosmosEvent)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestInstanceMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	for i := 0; i < 5; i++ {
		event := &core.BlockchainEvent{
			ID:              fmt.Sprintf("event-%d", i),
			ChainID:         "EVM",
			ContractAddress: common.Address{0x1},
			EventName:       "Transfer",
			CreatedAt:       time.Now(),
		}
		_ = cluster.ProcessEvent(context.Background(), event)
	}

	metrics := cluster.GetMetrics()

	assert.Equal(t, int64(5), metrics.EventsProcessed)
}

func TestClusterWithEventStore(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	event := &core.BlockchainEvent{
		ID:              "test-event",
		ChainID:         "EVM",
		ContractAddress: common.Address{0x1},
		EventName:       "Transfer",
		DecodedData:     map[string]any{"from": "0x123", "to": "0x456"},
		CreatedAt:       time.Now(),
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.NoError(t, err)
}

func TestBlockchainInstanceCreation(t *testing.T) {
	t.Parallel()
	instance := &BlockchainInstance{
		ID:             "instance-1",
		BlockchainType: "EVM",
		Status:         "running",
		CreatedAt:      time.Now(),
		Metrics: &InstanceMetrics{
			LastProcessedTime: time.Now(),
		},
	}

	assert.NotNil(t, instance)
	assert.Equal(t, "instance-1", instance.ID)
	assert.Equal(t, "EVM", instance.BlockchainType)
	assert.Equal(t, "running", instance.Status)
}

func TestClusterScaling(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	initialCount := cluster.GetInstanceCount()
	assert.Equal(t, 1, initialCount)

	cluster.mu.Lock()
	for i := 1; i < 3; i++ {
		instanceID := fmt.Sprintf("test-cluster-instance-%d", i)
		instance := &BlockchainInstance{
			ID:             instanceID,
			BlockchainType: "EVM",
			Status:         "syncing",
			CreatedAt:      time.Now(),
			Metrics: &InstanceMetrics{
				LastProcessedTime: time.Now(),
			},
		}
		cluster.instances[instanceID] = instance
	}
	cluster.mu.Unlock()

	finalCount := cluster.GetInstanceCount()
	assert.Equal(t, 3, finalCount)
}

func TestHealthCheckMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", "EVM", 3, 5)
	_ = cluster.Deploy(context.Background())

	instances := cluster.instances
	count := 0
	for _, instance := range instances {
		if count < 2 {
			instance.Status = "running"
		} else {
			instance.Status = "syncing"
		}
		count++
	}

	_ = cluster.HealthCheck(context.Background())

	metrics := cluster.GetMetrics()
	assert.Equal(t, 2, metrics.InstancesHealthy)
	assert.Equal(t, 1, metrics.InstancesUnhealthy)
}
