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
	"github.com/rtcdance/chainpulse/pkg/infrastructure/processing"
	"github.com/stretchr/testify/assert"
)

func skipBlockchainClusterConcurrencyTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping blockchain cluster concurrency test in short mode")
	}
}

// MockEventStore for testing
type MockEventStore struct {
	storedEvents []*processing.Event
	mu           sync.Mutex
}

func (m *MockEventStore) StoreEvent(ctx context.Context, event *processing.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storedEvents = append(m.storedEvents, event)
	return nil
}

func (m *MockEventStore) StoreBatch(ctx context.Context, events []*processing.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storedEvents = append(m.storedEvents, events...)
	return nil
}

func (m *MockEventStore) GetEvent(ctx context.Context, id string) (*processing.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, event := range m.storedEvents {
		if event.ID == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("event not found")
}

func (m *MockEventStore) GetAllEvents(ctx context.Context) ([]*processing.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storedEvents, nil
}

func (m *MockEventStore) DeleteEvent(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, event := range m.storedEvents {
		if event.ID == id {
			m.storedEvents = append(m.storedEvents[:i], m.storedEvents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("event not found")
}

func (m *MockEventStore) QueryEvents(ctx context.Context, filter *processing.EventFilter) ([]*processing.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storedEvents, nil
}

func (m *MockEventStore) BatchStoreEvents(ctx context.Context, events []*processing.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storedEvents = append(m.storedEvents, events...)
	return nil
}

func (m *MockEventStore) GetMetrics() processing.StorageMetrics {
	return processing.StorageMetrics{}
}

// TestNewBlockchainCluster tests cluster creation
func TestNewBlockchainCluster(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 2, 5)

	assert.NotNil(t, cluster)
	assert.Equal(t, "test-cluster", cluster.id)
	assert.Equal(t, "EVM", cluster.blockchainType)
	assert.Equal(t, 2, cluster.minInstances)
	assert.Equal(t, 5, cluster.maxInstances)
	assert.Equal(t, 0, len(cluster.instances))
}

// TestDeployCluster tests cluster deployment
func TestDeployCluster(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 2, 5)

	err := cluster.Deploy(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 2, len(cluster.instances))
	assert.Equal(t, 2, cluster.GetInstanceCount())
}

// TestClusterProcessEvent tests event processing
func TestClusterProcessEvent(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	// Set instance to running
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

// TestProcessEventNoInstances tests processing with no instances
func TestProcessEventNoInstances(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 0, 5)

	event := &core.BlockchainEvent{
		ID:      "test-event",
		ChainID: "EVM",
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no instances available")
}

// TestProcessEventBlockchainMismatch tests processing with blockchain type mismatch
func TestProcessEventBlockchainMismatch(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	event := &core.BlockchainEvent{
		ID:      "test-event",
		ChainID: "Solana",
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

// TestProcessEventNoRunningInstances tests processing with no running instances
func TestProcessEventNoRunningInstances(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	event := &core.BlockchainEvent{
		ID:      "test-event",
		ChainID: "EVM",
	}

	err := cluster.ProcessEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running instances available")
}

// TestHealthCheck tests health check
func TestHealthCheck(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 2, 5)
	_ = cluster.Deploy(context.Background())

	// Set instances to running
	for _, instance := range cluster.instances {
		instance.Status = "running"
	}

	err := cluster.HealthCheck(context.Background())

	assert.NoError(t, err)
	metrics := cluster.GetMetrics()
	assert.Equal(t, 2, metrics.InstancesHealthy)
}

// TestClusterGetMetrics tests metrics retrieval
func TestClusterGetMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	metrics := cluster.GetMetrics()

	assert.Equal(t, 1, metrics.InstancesDeployed)
	assert.Greater(t, metrics.InstancesDeployed, 0)
}

// TestGetBlockchainType tests blockchain type retrieval
func TestGetBlockchainType(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "Cosmos", 1, 5)

	blockchainType := cluster.GetBlockchainType()

	assert.Equal(t, "Cosmos", blockchainType)
}

// TestGetInstanceCount tests instance count retrieval
func TestGetInstanceCount(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 3, 5)
	_ = cluster.Deploy(context.Background())

	count := cluster.GetInstanceCount()

	assert.Equal(t, 3, count)
}

// TestNewMultiBlockchainClusterManager tests manager creation
func TestNewMultiBlockchainClusterManager(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	assert.NotNil(t, manager)
	assert.Equal(t, 0, len(manager.clusters))
}

// TestRegisterCluster tests cluster registration
func TestRegisterCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)

	err := manager.RegisterCluster(cluster)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.clusters))
}

// TestRegisterClusterDuplicate tests registering duplicate cluster
func TestRegisterClusterDuplicate(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster1 := NewBlockchainCluster("test-cluster-1", "EVM", 1, 5)
	cluster2 := NewBlockchainCluster("test-cluster-2", "EVM", 1, 5)

	_ = manager.RegisterCluster(cluster1)
	err := manager.RegisterCluster(cluster2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestManagerProcessEventWithCluster tests processing event through manager
func TestManagerProcessEventWithCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	// Set instance to running
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

// TestManagerProcessEventNoCluster tests processing with no cluster
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

// TestGetCluster tests cluster retrieval
func TestGetCluster(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = manager.RegisterCluster(cluster)

	retrieved, err := manager.GetCluster("EVM")

	assert.NoError(t, err)
	assert.Equal(t, cluster, retrieved)
}

// TestGetClusterNotFound tests retrieving non-existent cluster
func TestGetClusterNotFound(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	_, err := manager.GetCluster("EVM")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetAllClusters tests retrieving all clusters
func TestGetAllClusters(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster1 := NewBlockchainCluster("test-cluster-1", "EVM", 1, 5)
	cluster2 := NewBlockchainCluster("test-cluster-2", "Cosmos", 1, 5)

	_ = manager.RegisterCluster(cluster1)
	_ = manager.RegisterCluster(cluster2)

	clusters := manager.GetAllClusters()

	assert.Equal(t, 2, len(clusters))
}

// TestManagerGetMetrics tests metrics retrieval
func TestManagerGetMetrics(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 2, 5)
	_ = cluster.Deploy(context.Background())
	_ = manager.RegisterCluster(cluster)

	metrics := manager.GetMetrics()

	assert.Equal(t, 1, metrics["total_clusters"])
	assert.Equal(t, 2, metrics["total_instances"])
}

// TestConcurrentProcessing tests concurrent event processing
func TestConcurrentProcessing(t *testing.T) {
	t.Parallel()
	skipBlockchainClusterConcurrencyTestsInShortMode(t)

	cluster := NewBlockchainCluster("test-cluster", "EVM", 2, 5)
	_ = cluster.Deploy(context.Background())

	// Set instances to running
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

// TestMultipleBlockchains tests processing multiple blockchains
func TestMultipleBlockchains(t *testing.T) {
	t.Parallel()
	manager := NewMultiBlockchainClusterManager()

	evmCluster := NewBlockchainCluster("evm-cluster", "EVM", 1, 5)
	_ = evmCluster.Deploy(context.Background())
	for _, instance := range evmCluster.instances {
		instance.Status = "running"
	}

	cosmosCluster := NewBlockchainCluster("cosmos-cluster", "Cosmos", 1, 5)
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

// TestInstanceMetrics tests instance metrics tracking
func TestInstanceMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
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

// TestClusterWithEventStore tests cluster with event store
func TestClusterWithEventStore(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	cluster.dataStore = &MockEventStore{}
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

// TestBlockchainInstanceCreation tests instance creation
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

// TestClusterScaling tests cluster scaling
func TestClusterScaling(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 1, 5)
	_ = cluster.Deploy(context.Background())

	initialCount := cluster.GetInstanceCount()
	assert.Equal(t, 1, initialCount)

	// Simulate scaling up
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

// TestHealthCheckMetrics tests health check metrics
func TestHealthCheckMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewBlockchainCluster("test-cluster", "EVM", 3, 5)
	_ = cluster.Deploy(context.Background())

	// Set some instances to running, others to syncing
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
