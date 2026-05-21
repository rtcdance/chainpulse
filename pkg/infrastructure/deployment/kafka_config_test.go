package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewKafkaConfig tests creating a new Kafka configuration
func TestNewKafkaConfig(t *testing.T) {
	t.Parallel()
	config := NewKafkaConfig()

	assert.NotNil(t, config)
	assert.Equal(t, []string{"localhost:9092"}, config.Brokers)
	assert.Equal(t, "events", config.Topic)
	assert.Equal(t, "chainpulse-consumer", config.ConsumerGroup)
	assert.Equal(t, 3, config.Partitions)
	assert.Equal(t, 1, config.ReplicationFactor)
}

// TestKafkaConfigDefaults tests Kafka configuration defaults
func TestKafkaConfigDefaults(t *testing.T) {
	t.Parallel()
	config := NewKafkaConfig()

	assert.Equal(t, int64(604800000), config.RetentionMs)
	assert.Equal(t, "snappy", config.CompressionType)
	assert.Equal(t, "PLAINTEXT", config.SecurityProtocol)
	assert.Equal(t, 10000, config.ConnectTimeoutMs)
	assert.Equal(t, 30000, config.RequestTimeoutMs)
	assert.Equal(t, 10000, config.SessionTimeoutMs)
	assert.Equal(t, 3000, config.HeartbeatIntervalMs)
	assert.Equal(t, 300000, config.MaxPollIntervalMs)
	assert.Equal(t, 500, config.MaxPollRecords)
	assert.Equal(t, 1, config.FetchMinBytes)
	assert.Equal(t, 500, config.FetchMaxWaitMs)
}

// TestKafkaConfigValidate tests Kafka configuration validation
func TestKafkaConfigValidate(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, []string{"localhost:9092"}, config.Brokers)
	assert.Equal(t, "events", config.Topic)
	assert.Equal(t, "chainpulse-consumer", config.ConsumerGroup)
	assert.Equal(t, 3, config.Partitions)
	assert.Equal(t, 1, config.ReplicationFactor)
}

// TestKafkaConfigValidateWithExistingValues tests validation preserves existing values
func TestKafkaConfigValidateWithExistingValues(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers:           []string{"broker1:9092", "broker2:9092"},
		Topic:             "custom-topic",
		ConsumerGroup:     "custom-group",
		Partitions:        5,
		ReplicationFactor: 2,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, []string{"broker1:9092", "broker2:9092"}, config.Brokers)
	assert.Equal(t, "custom-topic", config.Topic)
	assert.Equal(t, "custom-group", config.ConsumerGroup)
	assert.Equal(t, 5, config.Partitions)
	assert.Equal(t, 2, config.ReplicationFactor)
}

// TestKafkaConfigWithCustomBrokers tests Kafka config with custom brokers
func TestKafkaConfigWithCustomBrokers(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers: []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 3, len(config.Brokers))
}

// TestKafkaConfigWithCustomTopic tests Kafka config with custom topic
func TestKafkaConfigWithCustomTopic(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Topic: "blockchain-events",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "blockchain-events", config.Topic)
}

// TestKafkaConfigWithCustomConsumerGroup tests Kafka config with custom consumer group
func TestKafkaConfigWithCustomConsumerGroup(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		ConsumerGroup: "my-consumer-group",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "my-consumer-group", config.ConsumerGroup)
}

// TestKafkaConfigWithCustomPartitions tests Kafka config with custom partitions
func TestKafkaConfigWithCustomPartitions(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Partitions: 10,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 10, config.Partitions)
}

// TestKafkaConfigWithCustomReplicationFactor tests Kafka config with custom replication factor
func TestKafkaConfigWithCustomReplicationFactor(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		ReplicationFactor: 3,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 3, config.ReplicationFactor)
}

// TestKafkaConfigWithSASL tests Kafka config with SASL authentication
func TestKafkaConfigWithSASL(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		SecurityProtocol: "SASL_SSL",
		SASLMechanism:    "PLAIN",
		SASLUsername:     "user",
		SASLPassword:     "password",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "SASL_SSL", config.SecurityProtocol)
	assert.Equal(t, "PLAIN", config.SASLMechanism)
}

// TestKafkaConfigWithSSL tests Kafka config with SSL
func TestKafkaConfigWithSSL(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		SecurityProtocol: "SSL",
		SSLCALocation:    "/path/to/ca.pem",
		SSLCertLocation:  "/path/to/cert.pem",
		SSLKeyLocation:   "/path/to/key.pem",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "SSL", config.SecurityProtocol)
}

// TestKafkaConfigWithCustomRetention tests Kafka config with custom retention
func TestKafkaConfigWithCustomRetention(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		RetentionMs: 86400000, // 1 day
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, int64(86400000), config.RetentionMs)
}

// TestKafkaConfigWithCustomCompression tests Kafka config with custom compression
func TestKafkaConfigWithCustomCompression(t *testing.T) {
	t.Parallel()
	compressionTypes := []string{"gzip", "snappy", "lz4", "zstd", "uncompressed"}

	for _, compression := range compressionTypes {
		config := &KafkaConfig{
			CompressionType: compression,
		}

		err := config.Validate()

		assert.NoError(t, err)
		assert.Equal(t, compression, config.CompressionType)
	}
}

// TestKafkaConfigWithCustomTimeouts tests Kafka config with custom timeouts
func TestKafkaConfigWithCustomTimeouts(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		ConnectTimeoutMs:    5000,
		RequestTimeoutMs:    15000,
		SessionTimeoutMs:    6000,
		HeartbeatIntervalMs: 2000,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 5000, config.ConnectTimeoutMs)
	assert.Equal(t, 15000, config.RequestTimeoutMs)
	assert.Equal(t, 6000, config.SessionTimeoutMs)
	assert.Equal(t, 2000, config.HeartbeatIntervalMs)
}

// TestKafkaConfigWithCustomMaxPoll tests Kafka config with custom max poll settings
func TestKafkaConfigWithCustomMaxPoll(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		MaxPollIntervalMs: 600000,
		MaxPollRecords:    1000,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 600000, config.MaxPollIntervalMs)
	assert.Equal(t, 1000, config.MaxPollRecords)
}

// TestKafkaConfigWithCustomFetch tests Kafka config with custom fetch settings
func TestKafkaConfigWithCustomFetch(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		FetchMinBytes:  1024,
		FetchMaxWaitMs: 1000,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 1024, config.FetchMinBytes)
	assert.Equal(t, 1000, config.FetchMaxWaitMs)
}

// TestKafkaConfigMultipleInstances tests creating multiple Kafka config instances
func TestKafkaConfigMultipleInstances(t *testing.T) {
	t.Parallel()
	config1 := NewKafkaConfig()
	config2 := NewKafkaConfig()

	assert.Equal(t, config1.Topic, config2.Topic)
	assert.Equal(t, config1.ConsumerGroup, config2.ConsumerGroup)

	config2.Topic = "different-topic"
	assert.NotEqual(t, config1.Topic, config2.Topic)
}

// TestKafkaConfigValidateEmptyBrokers tests validation with empty brokers
func TestKafkaConfigValidateEmptyBrokers(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Brokers: []string{},
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, []string{"localhost:9092"}, config.Brokers)
}

// TestKafkaConfigValidateEmptyTopic tests validation with empty topic
func TestKafkaConfigValidateEmptyTopic(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Topic: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "events", config.Topic)
}

// TestKafkaConfigValidateEmptyConsumerGroup tests validation with empty consumer group
func TestKafkaConfigValidateEmptyConsumerGroup(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		ConsumerGroup: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "chainpulse-consumer", config.ConsumerGroup)
}

// TestKafkaConfigValidateZeroPartitions tests validation with zero partitions
func TestKafkaConfigValidateZeroPartitions(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		Partitions: 0,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 3, config.Partitions)
}

// TestKafkaConfigValidateZeroReplicationFactor tests validation with zero replication factor
func TestKafkaConfigValidateZeroReplicationFactor(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		ReplicationFactor: 0,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 1, config.ReplicationFactor)
}

// TestKafkaConfigValidateEmptyCompressionType tests validation with empty compression type
func TestKafkaConfigValidateEmptyCompressionType(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		CompressionType: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "snappy", config.CompressionType)
}

// TestKafkaConfigValidateEmptySecurityProtocol tests validation with empty security protocol
func TestKafkaConfigValidateEmptySecurityProtocol(t *testing.T) {
	t.Parallel()
	config := &KafkaConfig{
		SecurityProtocol: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "PLAINTEXT", config.SecurityProtocol)
}
