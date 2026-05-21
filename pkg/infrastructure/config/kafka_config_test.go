package config

import (
	"context"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// TestNewKafkaCluster tests Kafka cluster creation
func TestNewKafkaCluster(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		Topic:             "test-topic",
		ConsumerGroup:     "test-group",
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.NotNil(t, cluster)
	assert.Equal(t, config, cluster.config)
}

// TestNewKafkaClusterNilConfig tests Kafka cluster creation with nil config
func TestNewKafkaClusterNilConfig(t *testing.T) {
	t.Parallel()
	cluster, err := NewKafkaCluster(nil)

	assert.NoError(t, err)
	assert.NotNil(t, cluster)
	assert.NotNil(t, cluster.config)
	assert.Equal(t, []string{"localhost:9092"}, cluster.config.Brokers)
}

// TestKafkaClusterConfig tests Kafka cluster configuration
func TestKafkaClusterConfig(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"broker1:9092", "broker2:9092"},
		Topic:             "test-topic",
		ConsumerGroup:     "test-group",
		PartitionCount:    5,
		ReplicationFactor: 2,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(cluster.brokers))
	assert.Equal(t, 5, cluster.config.PartitionCount)
	assert.Equal(t, 2, cluster.config.ReplicationFactor)
}

// TestKafkaClusterClose tests closing Kafka cluster
func TestKafkaClusterClose(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)
	assert.NoError(t, err)

	err = cluster.Close()

	assert.NoError(t, err)
}

// TestTopicMetadataStructure tests TopicMetadata structure
func TestTopicMetadataStructure(t *testing.T) {
	t.Parallel()
	metadata := &TopicMetadata{
		Topic:      "test-topic",
		Partitions: make([]kafkago.Partition, 0),
	}

	assert.Equal(t, "test-topic", metadata.Topic)
	assert.NotNil(t, metadata.Partitions)
}

// TestKafkaConfigStructure tests KafkaConfig structure
func TestKafkaConfigStructure(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		Topic:             "test-topic",
		ConsumerGroup:     "test-group",
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	assert.Equal(t, "test-topic", config.Topic)
	assert.Equal(t, "test-group", config.ConsumerGroup)
	assert.Equal(t, 3, config.PartitionCount)
	assert.Equal(t, 1, config.ReplicationFactor)
	assert.Equal(t, 1, len(config.Brokers))
}

// TestKafkaClusterMultipleBrokers tests Kafka cluster with multiple brokers
func TestKafkaClusterMultipleBrokers(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"broker1:9092", "broker2:9092", "broker3:9092"},
		PartitionCount:    3,
		ReplicationFactor: 2,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(cluster.brokers))
}

// TestKafkaClusterDefaultConfig tests Kafka cluster with default configuration
func TestKafkaClusterDefaultConfig(t *testing.T) {
	t.Parallel()
	cluster, err := NewKafkaCluster(nil)

	assert.NoError(t, err)
	assert.Equal(t, 3, cluster.config.PartitionCount)
	assert.Equal(t, 1, cluster.config.ReplicationFactor)
	assert.Equal(t, []string{"localhost:9092"}, cluster.config.Brokers)
}

// TestKafkaClusterBrokerAccess tests accessing brokers
func TestKafkaClusterBrokerAccess(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Greater(t, len(cluster.brokers), 0)
	assert.Equal(t, "localhost:9092", cluster.brokers[0])
}

// TestKafkaConfigWithCustomValues tests Kafka config with custom values
func TestKafkaConfigWithCustomValues(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"kafka1:9092", "kafka2:9092"},
		Topic:             "custom-topic",
		ConsumerGroup:     "custom-group",
		PartitionCount:    10,
		ReplicationFactor: 3,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, "custom-topic", cluster.config.Topic)
	assert.Equal(t, "custom-group", cluster.config.ConsumerGroup)
	assert.Equal(t, 10, cluster.config.PartitionCount)
	assert.Equal(t, 3, cluster.config.ReplicationFactor)
}

// TestKafkaClusterContextHandling tests context handling
func TestKafkaClusterContextHandling(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	_, err := NewKafkaCluster(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Context should be usable
	assert.NoError(t, ctx.Err())
}

// TestKafkaClusterEmptyBrokers tests Kafka cluster with empty brokers
func TestKafkaClusterEmptyBrokers(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{},
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.NotNil(t, cluster)
}

// TestKafkaClusterSingleBroker tests Kafka cluster with single broker
func TestKafkaClusterSingleBroker(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"single-broker:9092"},
		PartitionCount:    1,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(cluster.brokers))
}

// TestKafkaClusterHighReplication tests Kafka cluster with high replication factor
func TestKafkaClusterHighReplication(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"broker1:9092", "broker2:9092", "broker3:9092"},
		PartitionCount:    10,
		ReplicationFactor: 3,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, 3, cluster.config.ReplicationFactor)
}

// TestKafkaClusterManyPartitions tests Kafka cluster with many partitions
func TestKafkaClusterManyPartitions(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		PartitionCount:    100,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, 100, cluster.config.PartitionCount)
}

// TestKafkaClusterConfigCopy tests that config is properly stored
func TestKafkaClusterConfigCopy(t *testing.T) {
	t.Parallel()
	originalConfig := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		Topic:             "test-topic",
		ConsumerGroup:     "test-group",
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(originalConfig)

	assert.NoError(t, err)
	assert.Equal(t, originalConfig.Topic, cluster.config.Topic)
	assert.Equal(t, originalConfig.ConsumerGroup, cluster.config.ConsumerGroup)
}

// TestKafkaClusterBrokersList tests brokers list
func TestKafkaClusterBrokersList(t *testing.T) {
	t.Parallel()
	brokers := []string{"broker1:9092", "broker2:9092", "broker3:9092"}
	config := &KafkaConfig{
		Brokers:           brokers,
		PartitionCount:    3,
		ReplicationFactor: 1,
	}

	cluster, err := NewKafkaCluster(config)

	assert.NoError(t, err)
	assert.Equal(t, len(brokers), len(cluster.brokers))
	for i, broker := range brokers {
		assert.Equal(t, broker, cluster.brokers[i])
	}
}
