package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestKafkaTopicConfigStructure tests KafkaTopicConfig structure
func TestKafkaTopicConfigStructure(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:              "test-topic",
		Partitions:        3,
		ReplicationFactor: 2,
		RetentionMs:       86400000,
		SegmentMs:         3600000,
		CompressionType:   "snappy",
		MinInSyncReplicas: 2,
		CleanupPolicy:     "delete",
	}

	assert.Equal(t, "test-topic", config.Name)
	assert.Equal(t, 3, config.Partitions)
	assert.Equal(t, 2, config.ReplicationFactor)
	assert.Equal(t, int64(86400000), config.RetentionMs)
	assert.Equal(t, "snappy", config.CompressionType)
	assert.Equal(t, "delete", config.CleanupPolicy)
}

// TestNewKafkaAdvancedManager tests creating a new advanced Kafka manager
func TestNewKafkaAdvancedManager(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewKafkaAdvancedManager(cluster)

	assert.NotNil(t, manager)
	assert.Equal(t, cluster, manager.cluster)
}

// TestKafkaTopicConfigWithCompaction tests topic config with compaction
func TestKafkaTopicConfigWithCompaction(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:          "compacted-topic",
		Partitions:    1,
		CleanupPolicy: "compact",
	}

	assert.Equal(t, "compact", config.CleanupPolicy)
}

// TestKafkaTopicConfigWithDeletion tests topic config with deletion
func TestKafkaTopicConfigWithDeletion(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:          "deletion-topic",
		Partitions:    3,
		CleanupPolicy: "delete",
	}

	assert.Equal(t, "delete", config.CleanupPolicy)
}

// TestKafkaTopicConfigCompressionTypes tests various compression types
func TestKafkaTopicConfigCompressionTypes(t *testing.T) {
	t.Parallel()
	compressionTypes := []string{"none", "gzip", "snappy", "lz4", "zstd"}

	for _, ct := range compressionTypes {
		config := KafkaTopicConfig{
			Name:            "test-topic",
			CompressionType: ct,
		}
		assert.Equal(t, ct, config.CompressionType)
	}
}

// TestKafkaTopicConfigPartitions tests various partition counts
func TestKafkaTopicConfigPartitions(t *testing.T) {
	t.Parallel()
	partitionCounts := []int{1, 3, 5, 10, 20}

	for _, count := range partitionCounts {
		config := KafkaTopicConfig{
			Name:       "test-topic",
			Partitions: count,
		}
		assert.Equal(t, count, config.Partitions)
	}
}

// TestKafkaTopicConfigReplicationFactor tests various replication factors
func TestKafkaTopicConfigReplicationFactor(t *testing.T) {
	t.Parallel()
	replicationFactors := []int{1, 2, 3}

	for _, rf := range replicationFactors {
		config := KafkaTopicConfig{
			Name:              "test-topic",
			ReplicationFactor: rf,
		}
		assert.Equal(t, rf, config.ReplicationFactor)
	}
}

// TestKafkaTopicConfigRetention tests retention configuration
func TestKafkaTopicConfigRetention(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		retentionMs int64
	}{
		{"1 hour", 3600000},
		{"1 day", 86400000},
		{"7 days", 604800000},
		{"30 days", 2592000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := KafkaTopicConfig{
				Name:        "test-topic",
				RetentionMs: tt.retentionMs,
			}
			assert.Equal(t, tt.retentionMs, config.RetentionMs)
		})
	}
}

// TestKafkaTopicConfigSegment tests segment configuration
func TestKafkaTopicConfigSegment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		segmentMs int64
	}{
		{"1 hour", 3600000},
		{"6 hours", 21600000},
		{"1 day", 86400000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := KafkaTopicConfig{
				Name:      "test-topic",
				SegmentMs: tt.segmentMs,
			}
			assert.Equal(t, tt.segmentMs, config.SegmentMs)
		})
	}
}

// TestKafkaTopicConfigMinInSyncReplicas tests min in-sync replicas
func TestKafkaTopicConfigMinInSyncReplicas(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:              "test-topic",
		ReplicationFactor: 3,
		MinInSyncReplicas: 2,
	}

	assert.Equal(t, 2, config.MinInSyncReplicas)
	assert.LessOrEqual(t, config.MinInSyncReplicas, config.ReplicationFactor)
}

// TestKafkaAdvancedManagerMutex tests mutex protection
func TestKafkaAdvancedManagerMutex(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewKafkaAdvancedManager(cluster)

	manager.mutex.Lock()
	assert.NotNil(t, manager.cluster)
	manager.mutex.Unlock()
}

// TestKafkaTopicConfigMultipleTopics tests multiple topic configurations
func TestKafkaTopicConfigMultipleTopics(t *testing.T) {
	t.Parallel()
	configs := []KafkaTopicConfig{
		{Name: "topic-1", Partitions: 3},
		{Name: "topic-2", Partitions: 5},
		{Name: "topic-3", Partitions: 10},
	}

	assert.Equal(t, 3, len(configs))
	assert.Equal(t, "topic-1", configs[0].Name)
	assert.Equal(t, "topic-2", configs[1].Name)
	assert.Equal(t, "topic-3", configs[2].Name)
}

// TestKafkaTopicConfigDefaults tests default values
func TestKafkaTopicConfigDefaults(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name: "test-topic",
	}

	assert.Equal(t, "test-topic", config.Name)
	assert.Equal(t, 0, config.Partitions)
	assert.Equal(t, 0, config.ReplicationFactor)
}

// TestKafkaAdvancedManagerClusterReference tests cluster reference
func TestKafkaAdvancedManagerClusterReference(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewKafkaAdvancedManager(cluster)

	manager.mutex.RLock()
	assert.Equal(t, cluster, manager.cluster)
	manager.mutex.RUnlock()
}

// TestKafkaTopicConfigNameValidation tests topic name
func TestKafkaTopicConfigNameValidation(t *testing.T) {
	t.Parallel()
	names := []string{"topic-1", "my_topic", "test.topic", "TOPIC"}

	for _, name := range names {
		config := KafkaTopicConfig{Name: name}
		assert.Equal(t, name, config.Name)
	}
}

// TestKafkaTopicConfigHighPartitionCount tests high partition count
func TestKafkaTopicConfigHighPartitionCount(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:       "high-partition-topic",
		Partitions: 100,
	}

	assert.Equal(t, 100, config.Partitions)
}

// TestKafkaTopicConfigHighReplicationFactor tests high replication factor
func TestKafkaTopicConfigHighReplicationFactor(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:              "replicated-topic",
		ReplicationFactor: 5,
	}

	assert.Equal(t, 5, config.ReplicationFactor)
}

// TestKafkaTopicConfigLongRetention tests long retention period
func TestKafkaTopicConfigLongRetention(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:        "long-retention-topic",
		RetentionMs: 31536000000, // 1 year
	}

	assert.Equal(t, int64(31536000000), config.RetentionMs)
}

// TestKafkaAdvancedManagerConcurrentAccess tests concurrent access
func TestKafkaAdvancedManagerConcurrentAccess(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewKafkaAdvancedManager(cluster)

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

// TestKafkaTopicConfigZeroRetention tests zero retention
func TestKafkaTopicConfigZeroRetention(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:        "no-retention-topic",
		RetentionMs: 0,
	}

	assert.Equal(t, int64(0), config.RetentionMs)
}

// TestKafkaTopicConfigNegativeRetention tests negative retention (unlimited)
func TestKafkaTopicConfigNegativeRetention(t *testing.T) {
	t.Parallel()
	config := KafkaTopicConfig{
		Name:        "unlimited-retention-topic",
		RetentionMs: -1,
	}

	assert.Equal(t, int64(-1), config.RetentionMs)
}

func TestNewConsumerGroupManager(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewConsumerGroupManager(cluster)

	assert.NotNil(t, manager)
	assert.Equal(t, cluster, manager.cluster)
}

func TestConsumerGroupManager_CreateConsumerGroup(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewConsumerGroupManager(cluster)

	err := manager.CreateConsumerGroup(context.Background(), "test-group", "test-topic", 3)
	assert.NoError(t, err)
}

func TestConsumerGroupManager_GetConsumerGroupStatus(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewConsumerGroupManager(cluster)

	status, err := manager.GetConsumerGroupStatus(context.Background(), "test-group")
	assert.NoError(t, err)
	assert.Equal(t, "test-group", status.GroupID)
	assert.False(t, status.Timestamp.IsZero())
}

func TestNewKafkaClusterMonitor(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	monitor := NewKafkaClusterMonitor(cluster)

	assert.NotNil(t, monitor)
	assert.Equal(t, cluster, monitor.cluster)
}

func TestKafkaClusterMonitor_GetClusterStatus(t *testing.T) {
	t.Parallel()
	monitor := NewKafkaClusterMonitor(&KafkaCluster{brokers: []string{"localhost:9092"}})

	status, err := monitor.GetClusterStatus(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.False(t, status.Timestamp.IsZero())
}

func TestConsumerGroupManager_Mutex(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewConsumerGroupManager(cluster)

	manager.mutex.Lock()
	assert.NotNil(t, manager.cluster)
	manager.mutex.Unlock()
}

func TestKafkaClusterMonitor_Mutex(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	monitor := NewKafkaClusterMonitor(cluster)

	monitor.mutex.Lock()
	assert.NotNil(t, monitor.cluster)
	monitor.mutex.Unlock()
}

func TestKafkaTopicMetricsStructure(t *testing.T) {
	t.Parallel()
	now := time.Now()
	metrics := TopicMetrics{
		Topic:             "test-topic",
		PartitionCount:    3,
		ReplicationFactor: 2,
		Timestamp:         now,
	}

	assert.Equal(t, "test-topic", metrics.Topic)
	assert.Equal(t, 3, metrics.PartitionCount)
	assert.Equal(t, 2, metrics.ReplicationFactor)
	assert.Equal(t, now, metrics.Timestamp)
}

func TestConsumerGroupStatusStructure(t *testing.T) {
	t.Parallel()
	now := time.Now()
	status := ConsumerGroupStatus{
		GroupID:   "test-group",
		Members:   3,
		Topics:    []string{"topic-1", "topic-2"},
		Lag:       100,
		Timestamp: now,
	}

	assert.Equal(t, "test-group", status.GroupID)
	assert.Equal(t, 3, status.Members)
	assert.Equal(t, 2, len(status.Topics))
	assert.Equal(t, int64(100), status.Lag)
	assert.Equal(t, now, status.Timestamp)
}

func TestBrokerHealthStructure(t *testing.T) {
	t.Parallel()
	health := BrokerHealth{
		Address: "broker1:9092",
		Healthy: true,
		Error:   "",
	}

	assert.Equal(t, "broker1:9092", health.Address)
	assert.True(t, health.Healthy)
	assert.Empty(t, health.Error)
}

func TestClusterStatusStructure(t *testing.T) {
	t.Parallel()
	now := time.Now()
	status := ClusterStatus{
		Timestamp:      now,
		Healthy:        true,
		HealthyBrokers: 3,
		TotalBrokers:   3,
	}

	assert.True(t, status.Healthy)
	assert.Equal(t, 3, status.HealthyBrokers)
	assert.Equal(t, 3, status.TotalBrokers)
	assert.Equal(t, now, status.Timestamp)
}

func TestConsumerGroupManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	cluster := &KafkaCluster{}
	manager := NewConsumerGroupManager(cluster)

	done := make(chan bool, 10)
	for i := 0; i < 5; i++ {
		go func() {
			_ = manager.CreateConsumerGroup(context.Background(), "g", "t", 1)
			done <- true
		}()
	}
	for i := 0; i < 5; i++ {
		go func() {
			_, _ = manager.GetConsumerGroupStatus(context.Background(), "g")
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestKafkaClusterMonitor_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	monitor := NewKafkaClusterMonitor(&KafkaCluster{brokers: []string{}})

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = monitor.GetClusterStatus(context.Background())
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
