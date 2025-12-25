package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaPlugin implements MQPlugin for Kafka
type KafkaPlugin struct {
	writer           *kafka.Writer
	reader           *kafka.Reader
	metricsCollector *MetricsCollector
	config           KafkaConfig
}

// NewKafkaPlugin creates a new Kafka plugin instance
func NewKafkaPlugin() *KafkaPlugin {
	return &KafkaPlugin{}
}

// Initialize initializes the Kafka plugin with configuration
func (k *KafkaPlugin) Initialize(config map[string]interface{}) error {
	// Extract brokers from config
	brokersInterface, exists := config["brokers"]
	if !exists {
		return fmt.Errorf("brokers configuration is required for Kafka plugin")
	}

	var brokers []string
	switch v := brokersInterface.(type) {
	case []interface{}:
		for _, broker := range v {
			if str, ok := broker.(string); ok {
				brokers = append(brokers, str)
			}
		}
	case []string:
		brokers = v
	case string:
		brokers = []string{v}
	default:
		return fmt.Errorf("invalid brokers configuration type: %T", brokersInterface)
	}

	if len(brokers) == 0 {
		return fmt.Errorf("at least one broker is required for Kafka plugin")
	}

	k.config = KafkaConfig{
		Brokers: brokers,
	}

	// Create Kafka writer with configuration
	k.writer = &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		AllowAutoTopicCreation: true,
		Balancer:               &kafka.LeastBytes{},
		WriteBackoffMin:        100 * time.Millisecond,
		WriteBackoffMax:        1 * time.Second,
		MaxAttempts:            5,
		BatchSize:              100,
		BatchTimeout:           100 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
	}

	return nil
}

// GetName returns the name of the plugin
func (k *KafkaPlugin) GetName() string {
	return "kafka"
}

// SetMetricsCollector sets the metrics collector for the plugin
func (k *KafkaPlugin) SetMetricsCollector(collector *MetricsCollector) {
	k.metricsCollector = collector
}

// Publish sends a message to the specified topic
func (k *KafkaPlugin) Publish(topic string, message interface{}) error {
	startTime := time.Now()

	data, err := json.Marshal(message)
	if err != nil {
		if k.metricsCollector != nil {
			k.metricsCollector.RecordRequest("kafka", time.Since(startTime), err)
		}
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Value: data,
		Time:  time.Now(),
	}

	err = k.writer.WriteMessages(context.Background(), msg)

	if k.metricsCollector != nil {
		k.metricsCollector.RecordRequest("kafka", time.Since(startTime), err)
	}

	if err != nil {
		return fmt.Errorf("failed to publish message to Kafka: %w", err)
	}

	return nil
}

// Consume reads messages from the specified topic and handles them
func (k *KafkaPlugin) Consume(ctx context.Context, topic string, handler MessageHandler) error {
	k.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:         k.config.Brokers,
		Topic:           topic,
		GroupID:         "chainpulse-consumer-group",
		MinBytes:        10e3, // 10KB
		MaxBytes:        10e6, // 10MB
		MaxWait:         1 * time.Second,
		ReadLagInterval: -1,
		StartOffset:     kafka.FirstOffset,
	})

	defer k.reader.Close()

	// Create a worker pool for concurrent message processing
	const numWorkers = 10
	tasks := make(chan kafka.Message, numWorkers*2)

	// Start worker goroutines
	var workersDone int
	workersDoneChan := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() {
				workersDone++
				workersDoneChan <- true
			}()

			for msg := range tasks {
				startTime := time.Now()
				err := handler(msg.Value)

				if k.metricsCollector != nil {
					k.metricsCollector.RecordRequest("kafka", time.Since(startTime), err)
				}

				if err != nil {
					log.Printf("Error handling message in worker %d: %v", workerID, err)
					continue
				}

				// Commit the message after successful processing
				if err := k.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Error committing message from worker %d: %v", workerID, err)
				}
			}
		}(i)
	}

	for {
		select {
		case <-ctx.Done():
			close(tasks)
			for i := 0; i < numWorkers; i++ {
				<-workersDoneChan
			}
			return ctx.Err()
		default:
			m, err := k.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					close(tasks)
					for i := 0; i < numWorkers; i++ {
						<-workersDoneChan
					}
					return ctx.Err()
				}
				log.Printf("Error fetching message: %v", err)
				continue
			}

			select {
			case tasks <- m:
			case <-ctx.Done():
				close(tasks)
				for i := 0; i < numWorkers; i++ {
					<-workersDoneChan
				}
				return ctx.Err()
			}
		}
	}
}

// Close closes the Kafka connections
func (k *KafkaPlugin) Close() error {
	if k.writer != nil {
		return k.writer.Close()
	}
	return nil
}
