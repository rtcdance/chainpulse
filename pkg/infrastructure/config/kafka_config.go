package config

import (
	"context"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers           []string
	Topic             string
	ConsumerGroup     string
	PartitionCount    int
	ReplicationFactor int
}

// KafkaCluster manages Kafka cluster operations
type KafkaCluster struct {
	config  *KafkaConfig
	brokers []string
}

// NewKafkaCluster creates a new Kafka cluster manager
func NewKafkaCluster(cfg *KafkaConfig) (*KafkaCluster, error) {
	if cfg == nil {
		cfg = &KafkaConfig{
			Brokers:           []string{"localhost:9092"},
			PartitionCount:    3,
			ReplicationFactor: 1,
		}
	}

	return &KafkaCluster{
		config:  cfg,
		brokers: cfg.Brokers,
	}, nil
}

// CreateTopic creates a Kafka topic
func (k *KafkaCluster) CreateTopic(ctx context.Context, topic string) error {
	return k.CreateTopicWithConfig(ctx, topic, k.config.PartitionCount, k.config.ReplicationFactor)
}

// CreateTopicWithConfig creates a Kafka topic with specific configuration
func (k *KafkaCluster) CreateTopicWithConfig(ctx context.Context, topic string, partitions, replication int) error {
	conn, err := kafkago.Dial("tcp", k.brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafkago.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer func() {
		_ = controllerConn.Close()
	}()

	topicConfigs := []kafkago.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replication,
		},
	}

	if err := controllerConn.CreateTopics(topicConfigs...); err != nil {
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	return nil
}

// DeleteTopic deletes a Kafka topic
func (k *KafkaCluster) DeleteTopic(ctx context.Context, topic string) error {
	conn, err := kafkago.Dial("tcp", k.brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	if err := conn.DeleteTopics(topic); err != nil {
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}
	return nil
}

// ListTopics lists all Kafka topics
func (k *KafkaCluster) ListTopics(ctx context.Context) ([]string, error) {
	conn, err := kafkago.Dial("tcp", k.brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	topics := make(map[string]bool)
	for _, p := range partitions {
		topics[p.Topic] = true
	}

	result := make([]string, 0, len(topics))
	for topic := range topics {
		result = append(result, topic)
	}

	return result, nil
}

// GetTopicMetadata gets metadata for a topic
func (k *KafkaCluster) GetTopicMetadata(ctx context.Context, topic string) (*TopicMetadata, error) {
	conn, err := kafkago.Dial("tcp", k.brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic metadata: %w", err)
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("topic not found: %s", topic)
	}

	return &TopicMetadata{
		Topic:      topic,
		Partitions: partitions,
	}, nil
}

// Health checks Kafka cluster health
func (k *KafkaCluster) Health(ctx context.Context) error {
	conn, err := kafkago.Dial("tcp", k.brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	brokers, err := conn.Brokers()
	if err != nil {
		return fmt.Errorf("failed to get brokers: %w", err)
	}

	if len(brokers) == 0 {
		return fmt.Errorf("no brokers available")
	}

	return nil
}

// Close closes the Kafka client
func (k *KafkaCluster) Close() error {
	return nil
}

// WaitForKafka waits for Kafka to be available
func WaitForKafka(ctx context.Context, cfg *KafkaConfig, timeout time.Duration) error {
	cluster, err := NewKafkaCluster(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := cluster.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for Kafka")
		}

		healthCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := cluster.Health(healthCtx)
		cancel()

		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

// TopicMetadata holds topic metadata
type TopicMetadata struct {
	Topic      string
	Partitions []kafkago.Partition
}
