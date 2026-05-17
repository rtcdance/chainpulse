package graphql

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// SubscriptionManager manages GraphQL subscriptions
type SubscriptionManager struct {
	subscribers map[string][]chan any
	mu          sync.RWMutex
	logger      core.Logger
	metrics     core.MetricsCollector
	eventBus    core.EventBus
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(
	logger core.Logger,
	metrics core.MetricsCollector,
	eventBus core.EventBus,
) *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[string][]chan any),
		logger:      logger,
		metrics:     metrics,
		eventBus:    eventBus,
	}
}

// Subscription represents a GraphQL subscription
type Subscription struct {
	ID        string
	Topic     string
	Channel   chan any
	Context   context.Context
	Cancel    context.CancelFunc
	CreatedAt time.Time
}

// Subscribe creates a new subscription
func (sm *SubscriptionManager) Subscribe(topic string) (*Subscription, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	channel := make(chan any, 100)

	subscription := &Subscription{
		ID:        fmt.Sprintf("%s:%d", topic, time.Now().UnixNano()),
		Topic:     topic,
		Channel:   channel,
		Context:   ctx,
		Cancel:    cancel,
		CreatedAt: time.Now(),
	}

	sm.mu.Lock()
	sm.subscribers[topic] = append(sm.subscribers[topic], channel)
	sm.mu.Unlock()

	sm.logger.Info("Subscription created", "topic", topic, "subscriptionId", subscription.ID)
	sm.metrics.RecordCounter("graphql_subscription_created", 1, nil)

	return subscription, nil
}

// Unsubscribe removes a subscription
func (sm *SubscriptionManager) Unsubscribe(subscription *Subscription) error {
	if subscription == nil {
		return fmt.Errorf("subscription is required")
	}

	subscription.Cancel()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	channels, ok := sm.subscribers[subscription.Topic]
	if !ok {
		return nil
	}

	// Remove channel from subscribers
	for i, ch := range channels {
		if ch == subscription.Channel {
			sm.subscribers[subscription.Topic] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty topic
	if len(sm.subscribers[subscription.Topic]) == 0 {
		delete(sm.subscribers, subscription.Topic)
	}

	sm.logger.Info("Subscription removed", "topic", subscription.Topic, "subscriptionId", subscription.ID)
	sm.metrics.RecordCounter("graphql_subscription_removed", 1, nil)

	return nil
}

// Publish publishes a message to all subscribers of a topic
func (sm *SubscriptionManager) Publish(topic string, data any) error {
	if topic == "" {
		return fmt.Errorf("topic is required")
	}

	sm.mu.RLock()
	channels, ok := sm.subscribers[topic]
	sm.mu.RUnlock()

	if !ok || len(channels) == 0 {
		return nil
	}

	// Send to all subscribers
	for _, ch := range channels {
		select {
		case ch <- data:
			sm.metrics.RecordCounter("graphql_subscription_message_sent", 1, nil)
		case <-time.After(1 * time.Second):
			// Timeout sending to subscriber
			sm.logger.Warn("Timeout sending to subscriber", "topic", topic)
			sm.metrics.RecordCounter("graphql_subscription_message_timeout", 1, nil)
		}
	}

	return nil
}

// GetSubscriberCount returns the number of subscribers for a topic
func (sm *SubscriptionManager) GetSubscriberCount(topic string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	channels, ok := sm.subscribers[topic]
	if !ok {
		return 0
	}

	return len(channels)
}

// GetAllSubscriptions returns all active subscriptions
func (sm *SubscriptionManager) GetAllSubscriptions() map[string]int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]int)
	for topic, channels := range sm.subscribers {
		result[topic] = len(channels)
	}

	return result
}

// SubscriptionTopics defines available subscription topics
type SubscriptionTopics struct {
	EventCreated     string
	EventUpdated     string
	EventDeleted     string
	EventConfirmed   string
	EventFailed      string
	CacheInvalidated string
}

// NewSubscriptionTopics creates new subscription topics
func NewSubscriptionTopics() *SubscriptionTopics {
	return &SubscriptionTopics{
		EventCreated:     "event:created",
		EventUpdated:     "event:updated",
		EventDeleted:     "event:deleted",
		EventConfirmed:   "event:confirmed",
		EventFailed:      "event:failed",
		CacheInvalidated: "cache:invalidated",
	}
}

// EventSubscriptionPayload represents the payload for event subscriptions
type EventSubscriptionPayload struct {
	Type            string         `json:"type"`
	EventID         string         `json:"eventId"`
	ChainID         string         `json:"chainId,omitempty"`
	ContractAddress string         `json:"contractAddress,omitempty"`
	EventName       string         `json:"eventName,omitempty"`
	BlockNumber     int64          `json:"blockNumber,omitempty"`
	Event           map[string]any `json:"event,omitempty"`
	Timestamp       int64          `json:"timestamp"`
}

// SubscriptionHandler handles subscription operations
type SubscriptionHandler struct {
	manager *SubscriptionManager
	topics  *SubscriptionTopics
	logger  core.Logger
	metrics core.MetricsCollector
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(
	manager *SubscriptionManager,
	logger core.Logger,
	metrics core.MetricsCollector,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		manager: manager,
		topics:  NewSubscriptionTopics(),
		logger:  logger,
		metrics: metrics,
	}
}

// OnEventCreated publishes event created notification
func (sh *SubscriptionHandler) OnEventCreated(event *core.BlockchainEvent) error {
	payload := EventSubscriptionPayload{
		Type:      "created",
		EventID:   event.ID,
		Event:     eventToGraphQL(event),
		Timestamp: time.Now().Unix(),
	}

	return sh.manager.Publish(sh.topics.EventCreated, payload)
}

// OnEventUpdated publishes event updated notification
func (sh *SubscriptionHandler) OnEventUpdated(event *core.BlockchainEvent) error {
	payload := EventSubscriptionPayload{
		Type:      "updated",
		EventID:   event.ID,
		Event:     eventToGraphQL(event),
		Timestamp: time.Now().Unix(),
	}

	return sh.manager.Publish(sh.topics.EventUpdated, payload)
}

// OnEventDeleted publishes event deleted notification
func (sh *SubscriptionHandler) OnEventDeleted(eventID string) error {
	payload := EventSubscriptionPayload{
		Type:      "deleted",
		EventID:   eventID,
		Timestamp: time.Now().Unix(),
	}

	return sh.manager.Publish(sh.topics.EventDeleted, payload)
}

// OnEventConfirmed publishes event confirmed notification
func (sh *SubscriptionHandler) OnEventConfirmed(event *core.BlockchainEvent) error {
	payload := EventSubscriptionPayload{
		Type:      "confirmed",
		EventID:   event.ID,
		Event:     eventToGraphQL(event),
		Timestamp: time.Now().Unix(),
	}

	return sh.manager.Publish(sh.topics.EventConfirmed, payload)
}

// OnEventFailed publishes event failed notification
func (sh *SubscriptionHandler) OnEventFailed(event *core.BlockchainEvent) error {
	payload := EventSubscriptionPayload{
		Type:      "failed",
		EventID:   event.ID,
		Event:     eventToGraphQL(event),
		Timestamp: time.Now().Unix(),
	}

	return sh.manager.Publish(sh.topics.EventFailed, payload)
}

// OnCacheInvalidated publishes cache invalidated notification
func (sh *SubscriptionHandler) OnCacheInvalidated(pattern string) error {
	payload := map[string]any{
		"type":      "invalidated",
		"pattern":   pattern,
		"timestamp": time.Now(),
	}

	return sh.manager.Publish(sh.topics.CacheInvalidated, payload)
}

// SubscriptionStats represents subscription statistics
type SubscriptionStats struct {
	TotalSubscriptions int64
	ActiveTopics       int
	TopicSubscriptions map[string]int
	CreatedAt          time.Time
}

// GetStats returns subscription statistics
func (sm *SubscriptionManager) GetStats() SubscriptionStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	totalSubs := int64(0)
	topicSubs := make(map[string]int)

	for topic, channels := range sm.subscribers {
		count := len(channels)
		totalSubs += int64(count)
		topicSubs[topic] = count
	}

	return SubscriptionStats{
		TotalSubscriptions: totalSubs,
		ActiveTopics:       len(sm.subscribers),
		TopicSubscriptions: topicSubs,
		CreatedAt:          time.Now(),
	}
}
