package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPostgresReplicationConfigStructure tests PostgresReplicationConfig structure
func TestPostgresReplicationConfigStructure(t *testing.T) {
	config := PostgresReplicationConfig{
		PrimaryAddress:      "localhost:5432",
		ReplicaAddresses:    []string{"replica1:5432", "replica2:5432"},
		SyncInterval:        10 * time.Second,
		MaxSyncRetries:      3,
		WALLevel:            "replica",
		MaxWALSenders:       10,
		MaxReplicationSlots: 10,
	}

	assert.Equal(t, "localhost:5432", config.PrimaryAddress)
	assert.Equal(t, 2, len(config.ReplicaAddresses))
	assert.Equal(t, 10*time.Second, config.SyncInterval)
	assert.Equal(t, 3, config.MaxSyncRetries)
	assert.Equal(t, "replica", config.WALLevel)
}

// TestNewPostgresAdvancedManager tests creating a new advanced PostgreSQL manager
func TestNewPostgresAdvancedManager(t *testing.T) {
	cluster := &PostgresCluster{}
	manager := NewPostgresAdvancedManager(cluster)

	assert.NotNil(t, manager)
	assert.Equal(t, cluster, manager.cluster)
}

// TestPostgresReplicationConfigWALLevels tests various WAL levels
func TestPostgresReplicationConfigWALLevels(t *testing.T) {
	walLevels := []string{"minimal", "replica", "logical"}

	for _, level := range walLevels {
		config := PostgresReplicationConfig{
			WALLevel: level,
		}
		assert.Equal(t, level, config.WALLevel)
	}
}

// TestPostgresReplicationConfigMaxWALSenders tests max WAL senders
func TestPostgresReplicationConfigMaxWALSenders(t *testing.T) {
	tests := []struct {
		name          string
		maxWALSenders int
	}{
		{"small", 5},
		{"medium", 10},
		{"large", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := PostgresReplicationConfig{
				MaxWALSenders: tt.maxWALSenders,
			}
			assert.Equal(t, tt.maxWALSenders, config.MaxWALSenders)
		})
	}
}

// TestPostgresReplicationConfigMaxReplicationSlots tests max replication slots
func TestPostgresReplicationConfigMaxReplicationSlots(t *testing.T) {
	config := PostgresReplicationConfig{
		MaxReplicationSlots: 10,
	}

	assert.Equal(t, 10, config.MaxReplicationSlots)
}

// TestPostgresReplicationConfigSyncInterval tests sync interval
func TestPostgresReplicationConfigSyncInterval(t *testing.T) {
	tests := []struct {
		name         string
		syncInterval time.Duration
	}{
		{"1 second", 1 * time.Second},
		{"5 seconds", 5 * time.Second},
		{"10 seconds", 10 * time.Second},
		{"1 minute", 1 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := PostgresReplicationConfig{
				SyncInterval: tt.syncInterval,
			}
			assert.Equal(t, tt.syncInterval, config.SyncInterval)
		})
	}
}

// TestPostgresReplicationConfigMaxSyncRetries tests max sync retries
func TestPostgresReplicationConfigMaxSyncRetries(t *testing.T) {
	tests := []struct {
		name           string
		maxSyncRetries int
	}{
		{"no retries", 0},
		{"few retries", 3},
		{"many retries", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := PostgresReplicationConfig{
				MaxSyncRetries: tt.maxSyncRetries,
			}
			assert.Equal(t, tt.maxSyncRetries, config.MaxSyncRetries)
		})
	}
}

// TestPostgresReplicationConfigMultipleReplicas tests multiple replicas
func TestPostgresReplicationConfigMultipleReplicas(t *testing.T) {
	config := PostgresReplicationConfig{
		PrimaryAddress: "primary:5432",
		ReplicaAddresses: []string{
			"replica1:5432",
			"replica2:5432",
			"replica3:5432",
		},
	}

	assert.Equal(t, 3, len(config.ReplicaAddresses))
}

// TestPostgresReplicationConfigSingleReplica tests single replica
func TestPostgresReplicationConfigSingleReplica(t *testing.T) {
	config := PostgresReplicationConfig{
		PrimaryAddress:   "primary:5432",
		ReplicaAddresses: []string{"replica:5432"},
	}

	assert.Equal(t, 1, len(config.ReplicaAddresses))
}

// TestPostgresReplicationConfigNoReplicas tests no replicas
func TestPostgresReplicationConfigNoReplicas(t *testing.T) {
	config := PostgresReplicationConfig{
		PrimaryAddress:   "primary:5432",
		ReplicaAddresses: []string{},
	}

	assert.Equal(t, 0, len(config.ReplicaAddresses))
}

// TestPostgresAdvancedManagerMutex tests mutex protection
func TestPostgresAdvancedManagerMutex(t *testing.T) {
	cluster := &PostgresCluster{}
	manager := NewPostgresAdvancedManager(cluster)

	manager.mutex.Lock()
	assert.NotNil(t, manager.cluster)
	manager.mutex.Unlock()
}

// TestPostgresAdvancedManagerClusterReference tests cluster reference
func TestPostgresAdvancedManagerClusterReference(t *testing.T) {
	cluster := &PostgresCluster{}
	manager := NewPostgresAdvancedManager(cluster)

	manager.mutex.RLock()
	assert.Equal(t, cluster, manager.cluster)
	manager.mutex.RUnlock()
}

// TestPostgresReplicationConfigDefaults tests default values
func TestPostgresReplicationConfigDefaults(t *testing.T) {
	config := PostgresReplicationConfig{}

	assert.Equal(t, "", config.PrimaryAddress)
	assert.Equal(t, 0, len(config.ReplicaAddresses))
	assert.Equal(t, time.Duration(0), config.SyncInterval)
}

// TestPostgresReplicationConfigAddressFormats tests various address formats
func TestPostgresReplicationConfigAddressFormats(t *testing.T) {
	addresses := []string{
		"localhost:5432",
		"127.0.0.1:5432",
		"db.example.com:5432",
		"192.168.1.1:5432",
	}

	for _, addr := range addresses {
		config := PostgresReplicationConfig{
			PrimaryAddress: addr,
		}
		assert.Equal(t, addr, config.PrimaryAddress)
	}
}

// TestPostgresReplicationConfigHighMaxWALSenders tests high max WAL senders
func TestPostgresReplicationConfigHighMaxWALSenders(t *testing.T) {
	config := PostgresReplicationConfig{
		MaxWALSenders: 100,
	}

	assert.Equal(t, 100, config.MaxWALSenders)
}

// TestPostgresReplicationConfigHighMaxReplicationSlots tests high max replication slots
func TestPostgresReplicationConfigHighMaxReplicationSlots(t *testing.T) {
	config := PostgresReplicationConfig{
		MaxReplicationSlots: 100,
	}

	assert.Equal(t, 100, config.MaxReplicationSlots)
}

// TestPostgresAdvancedManagerConcurrentAccess tests concurrent access
func TestPostgresAdvancedManagerConcurrentAccess(t *testing.T) {
	cluster := &PostgresCluster{}
	manager := NewPostgresAdvancedManager(cluster)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			manager.mutex.RLock()
			_ = manager.cluster
			manager.mutex.RUnlock()
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestPostgresReplicationConfigZeroSyncInterval tests zero sync interval
func TestPostgresReplicationConfigZeroSyncInterval(t *testing.T) {
	config := PostgresReplicationConfig{
		SyncInterval: 0,
	}

	assert.Equal(t, time.Duration(0), config.SyncInterval)
}

// TestPostgresReplicationConfigLargeSyncInterval tests large sync interval
func TestPostgresReplicationConfigLargeSyncInterval(t *testing.T) {
	config := PostgresReplicationConfig{
		SyncInterval: 1 * time.Hour,
	}

	assert.Equal(t, 1*time.Hour, config.SyncInterval)
}
