package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-zeromq/zmq4"
)

// ZeroMQPlugin implements MQPlugin for ZeroMQ
type ZeroMQPlugin struct {
	publisher        zmq4.Socket
	subscriber       zmq4.Socket
	metricsCollector *MetricsCollector
	config           ZeroMQConfig
}

// ZeroMQConfig holds configuration for ZeroMQ connection
type ZeroMQConfig struct {
	PublishAddr   string
	SubscribeAddr string
}

// NewZeroMQPlugin creates a new ZeroMQ plugin instance
func NewZeroMQPlugin() *ZeroMQPlugin {
	return &ZeroMQPlugin{}
}

// Initialize initializes the ZeroMQ plugin with configuration
func (z *ZeroMQPlugin) Initialize(config map[string]interface{}) error {
	publishAddrInterface, exists := config["publish_addr"]
	if !exists {
		return fmt.Errorf("publish_addr configuration is required for ZeroMQ plugin")
	}

	publishAddr, ok := publishAddrInterface.(string)
	if !ok {
		return fmt.Errorf("publish_addr must be a string")
	}

	subscribeAddrInterface, exists := config["subscribe_addr"]
	if !exists {
		return fmt.Errorf("subscribe_addr configuration is required for ZeroMQ plugin")
	}

	subscribeAddr, ok := subscribeAddrInterface.(string)
	if !ok {
		return fmt.Errorf("subscribe_addr must be a string")
	}

	z.config = ZeroMQConfig{
		PublishAddr:   publishAddr,
		SubscribeAddr: subscribeAddr,
	}

	// Create publisher socket
	z.publisher = zmq4.NewPub(context.Background())

	// Create subscriber socket
	z.subscriber = zmq4.NewSub(context.Background())

	return nil
}

// GetName returns the name of the plugin
func (z *ZeroMQPlugin) GetName() string {
	return "zeromq"
}

// SetMetricsCollector sets the metrics collector for the plugin
func (z *ZeroMQPlugin) SetMetricsCollector(collector *MetricsCollector) {
	z.metricsCollector = collector
}

// Publish sends a message to the specified topic using ZeroMQ
func (z *ZeroMQPlugin) Publish(topic string, message interface{}) error {
	startTime := time.Now()

	data, err := json.Marshal(message)
	if err != nil {
		if z.metricsCollector != nil {
			z.metricsCollector.RecordRequest("zeromq", time.Since(startTime), err)
		}
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Connect publisher if not already connected
	if err := z.publisher.Dial(z.config.PublishAddr); err != nil {
		if z.metricsCollector != nil {
			z.metricsCollector.RecordRequest("zeromq", time.Since(startTime), err)
		}
		return fmt.Errorf("failed to connect publisher: %w", err)
	}

	// Format message as topic:message
	msg := zmq4.Msg{Frames: [][]byte{[]byte(topic + ":"), data}}

	err = z.publisher.Send(msg)

	if z.metricsCollector != nil {
		z.metricsCollector.RecordRequest("zeromq", time.Since(startTime), err)
	}

	if err != nil {
		return fmt.Errorf("failed to publish message to ZeroMQ: %w", err)
	}

	return nil
}

// Consume reads messages from the specified topic and handles them using ZeroMQ
func (z *ZeroMQPlugin) Consume(ctx context.Context, topic string, handler MessageHandler) error {
	// Connect subscriber
	if err := z.subscriber.Dial(z.config.SubscribeAddr); err != nil {
		return fmt.Errorf("failed to connect subscriber: %w", err)
	}

	// Subscribe to the topic
	z.subscriber.SetOption(zmq4.OptionSubscribe, topic)

	// Create a worker pool for concurrent message processing
	const numWorkers = 5
	tasks := make(chan []byte, numWorkers*2)

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
				err := handler(msg)

				if z.metricsCollector != nil {
					z.metricsCollector.RecordRequest("zeromq", time.Since(startTime), err)
				}

				if err != nil {
					log.Printf("Error handling message in worker %d: %v", workerID, err)
				}
			}
		}(i)
	}

	// Goroutine to fetch messages from ZeroMQ
	go func() {
		defer close(tasks)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Receive message from ZeroMQ
				msg, err := z.subscriber.Recv()
				if err != nil {
					log.Printf("Error receiving message from ZeroMQ: %v", err)
					time.Sleep(100 * time.Millisecond) // Brief pause before retrying
					continue
				}

				// Extract the actual message content (skip topic prefix)
				messageData := msg.Frames[0]
				if len(messageData) > len(topic)+1 { // +1 for ':'
					if string(messageData[:len(topic)]) == topic {
						// Skip the topic part and keep the actual message
						actualMessage := messageData[len(topic)+1:]
						select {
						case tasks <- actualMessage:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	// Wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		<-workersDoneChan
	}

	return ctx.Err()
}

// Close closes the ZeroMQ connections
func (z *ZeroMQPlugin) Close() error {
	if z.publisher != nil {
		z.publisher.Close()
	}
	if z.subscriber != nil {
		z.subscriber.Close()
	}
	return nil
}
