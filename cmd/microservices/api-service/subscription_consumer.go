package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

type subscriptionQuerySource interface {
	SubscriptionStore() ports.SubscriptionDeliveryStore
	SubscriptionChains() []string
}

type subscriptionConsumerState struct {
	mu sync.RWMutex
	// topics tracks generation assignment per topic; failures tracks the last
	// effect outcome per partition so one partition's success can never hide
	// another partition's failure.
	topics   map[string]bool
	failures map[string]map[int]bool
	metrics  core.MetricsCollector
}

func newSubscriptionConsumerState(topics []string, metrics core.MetricsCollector) *subscriptionConsumerState {
	s := &subscriptionConsumerState{
		topics:   make(map[string]bool, len(topics)),
		failures: make(map[string]map[int]bool, len(topics)),
		metrics:  metrics,
	}
	for _, topic := range topics {
		s.topics[topic] = false
	}
	return s
}

// assigned marks a topic's generation ownership. Revocation clears every
// recorded failure because it ends all handlers of the old generation.
func (s *subscriptionConsumerState) assigned(topic string, ok bool) {
	s.mu.Lock()
	s.topics[topic] = ok
	if !ok {
		delete(s.failures, topic)
	}
	s.mu.Unlock()
	s.recordReadyGauge(topic)
}

// effect records one partition's handler outcome for a topic.
func (s *subscriptionConsumerState) effect(topic string, partition int, err error) {
	s.mu.Lock()
	if err != nil {
		if s.failures[topic] == nil {
			s.failures[topic] = make(map[int]bool)
		}
		s.failures[topic][partition] = true
	} else if tracked := s.failures[topic]; tracked != nil {
		delete(tracked, partition)
	}
	s.mu.Unlock()
	s.recordReadyGauge(topic)
}

func (s *subscriptionConsumerState) topicReady(topic string) bool {
	assigned := s.topics[topic]
	return assigned && len(s.failures[topic]) == 0
}

func (s *subscriptionConsumerState) recordReadyGauge(topic string) {
	if s.metrics == nil {
		return
	}
	s.mu.RLock()
	ready := s.topicReady(topic)
	s.mu.RUnlock()
	value := float64(0)
	if ready {
		value = 1
	}
	s.metrics.RecordGauge("subscription_consumer_ready", value, map[string]string{"service": "api-service", "topic": topic})
}

func (s *subscriptionConsumerState) ready(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.topics) == 0 {
		return false
	}
	for topic := range s.topics {
		if !s.topicReady(topic) {
			return false
		}
	}
	return true
}

func validSubscriptionTopic(topic string) bool {
	if topic == "" || len(topic) > 249 || topic == "." || topic == ".." {
		return false
	}
	for _, r := range topic {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func startEventPushConsumer(ctx context.Context, wg *sync.WaitGroup, config APIServiceConfig, handler *api.EventSubscriptionHandler, logger ports.Logger, metrics core.MetricsCollector, store ports.SubscriptionDeliveryStore, chains []string) (func(context.Context) bool, error) {
	if config.CommittedTopic == "" || config.ReorgTopic == "" || config.ConsumerGroup == "" || len(config.ConsumerGroup) > 128 {
		return nil, fmt.Errorf("subscription consumer requires bounded group and committed/reorg topics")
	}
	if !validSubscriptionTopic(config.CommittedTopic) || !validSubscriptionTopic(config.ReorgTopic) {
		return nil, fmt.Errorf("subscription consumer requires valid Kafka topic names, got committed=%q reorg=%q", config.CommittedTopic, config.ReorgTopic)
	}
	delivery, err := appindexing.NewSubscriptionDelivery(store, handler, chains, metrics)
	if err != nil {
		return nil, err
	}
	topics := []string{config.CommittedTopic}
	if config.ReorgTopic != config.CommittedTopic {
		topics = append(topics, config.ReorgTopic)
	}
	state := newSubscriptionConsumerState(topics, metrics)
	// Each API instance needs every partition, rather than competing with
	// another instance whose connected clients cannot see its local hub.
	group := fmt.Sprintf("%s-sub-%x", config.ConsumerGroup, sha256.Sum256([]byte(config.InstanceID)))
	kafka := mq.NewKafkaMQPlugin("api-service-push-consumer", "1.0.0", &core.Config{APIType: "kafka", LogLevel: config.LogLevel}, logger, metrics, nil, config.KafkaBrokers, group)
	if err := kafka.Initialize(ctx, core.Config{}); err != nil {
		return nil, fmt.Errorf("initialize subscription Kafka: %w", err)
	}
	if err := kafka.Start(ctx); err != nil {
		return nil, fmt.Errorf("start subscription Kafka: %w", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var loops sync.WaitGroup
		for _, topic := range topics {
			loops.Go(func() { consumeSubscriptionTopic(ctx, topic, kafka, delivery, state, logger) })
		}
		loops.Wait()
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := kafka.Stop(stopCtx); err != nil {
			logger.Error("Stop subscription Kafka", "error", err.Error())
		}
	}()
	return func(probe context.Context) bool { return ctx.Err() == nil && state.ready(probe) }, nil
}

func consumeSubscriptionTopic(ctx context.Context, topic string, kafka *mq.KafkaMQPlugin, delivery *appindexing.SubscriptionDelivery, state *subscriptionConsumerState, logger ports.Logger) {
	defer state.assigned(topic, false)
	for ctx.Err() == nil {
		err := kafka.ConsumeOwnedMessages(ctx, topic, func(active context.Context, message core.MessageQueueMessage) error {
			err := delivery.Handle(active, ports.SubscriptionDeliveryMessage{ID: message.ID, Topic: message.Topic, PartitionKey: message.PartitionKey, Offset: message.Offset, Payload: message.Payload})
			state.effect(topic, message.Partition, err)
			return err
		}, mq.ConsumerOwnershipHooks{
			Assigned: func(context.Context, string) error { state.assigned(topic, true); return nil },
			Revoked:  func(context.Context) error { state.assigned(topic, false); return nil },
		})
		state.assigned(topic, false)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			logger.Error("Subscription consumer stopped; retrying", "topic", topic, "error", err.Error())
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
