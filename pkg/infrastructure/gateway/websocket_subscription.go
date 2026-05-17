package gateway

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Subscription represents a WebSocket subscription
type Subscription struct {
	ID        string
	ClientID  string
	ChainID   string
	EventType string
	CreatedAt time.Time
	Active    bool
}

// SubscriptionManager manages WebSocket subscriptions
type SubscriptionManager struct {
	subscriptions map[string]*Subscription
	clients       map[string][]string // clientID -> subscriptionIDs
	mutex         sync.RWMutex
	nextSubID     atomic.Int64
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
		clients:       make(map[string][]string),
	}
}

// Subscribe creates a new subscription
func (sm *SubscriptionManager) Subscribe(ctx context.Context, clientID string, chainID string, eventType string) (*Subscription, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Generate subscription ID
	subscriptionID := fmt.Sprintf("sub-%s-%s-%d", clientID, chainID, sm.nextSubID.Add(1))

	subscription := &Subscription{
		ID:        subscriptionID,
		ClientID:  clientID,
		ChainID:   chainID,
		EventType: eventType,
		CreatedAt: time.Now(),
		Active:    true,
	}

	sm.subscriptions[subscriptionID] = subscription
	sm.clients[clientID] = append(sm.clients[clientID], subscriptionID)

	return subscription, nil
}

// Unsubscribe removes a subscription
func (sm *SubscriptionManager) Unsubscribe(ctx context.Context, subscriptionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	// Remove from subscriptions
	delete(sm.subscriptions, subscriptionID)

	// Remove from client subscriptions
	clientID := subscription.ClientID
	if subs, exists := sm.clients[clientID]; exists {
		for i, subID := range subs {
			if subID == subscriptionID {
				sm.clients[clientID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}

	return nil
}

// GetSubscription gets a subscription
func (sm *SubscriptionManager) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	return subscription, nil
}

// GetClientSubscriptions gets all subscriptions for a client
func (sm *SubscriptionManager) GetClientSubscriptions(ctx context.Context, clientID string) ([]*Subscription, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	subscriptionIDs, exists := sm.clients[clientID]
	if !exists {
		return []*Subscription{}, nil
	}

	subscriptions := make([]*Subscription, 0)
	for _, subID := range subscriptionIDs {
		if sub, exists := sm.subscriptions[subID]; exists {
			subscriptions = append(subscriptions, sub)
		}
	}

	return subscriptions, nil
}

// GetChainSubscriptions gets all subscriptions for a chain
func (sm *SubscriptionManager) GetChainSubscriptions(ctx context.Context, chainID string) ([]*Subscription, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	subscriptions := make([]*Subscription, 0)
	for _, sub := range sm.subscriptions {
		if sub.ChainID == chainID && sub.Active {
			subscriptions = append(subscriptions, sub)
		}
	}

	return subscriptions, nil
}

// EventDeliveryManager manages event delivery to subscribers
type EventDeliveryManager struct {
	subscriptionMgr *SubscriptionManager
	deliveryChans   map[string]chan *core.BlockchainEvent
	mutex           sync.RWMutex
	eventsDropped   atomic.Int64
}

// NewEventDeliveryManager creates a new event delivery manager
func NewEventDeliveryManager(subscriptionMgr *SubscriptionManager) *EventDeliveryManager {
	return &EventDeliveryManager{
		subscriptionMgr: subscriptionMgr,
		deliveryChans:   make(map[string]chan *core.BlockchainEvent),
	}
}

// RegisterDeliveryChannel registers a delivery channel for a subscription
func (edm *EventDeliveryManager) RegisterDeliveryChannel(subscriptionID string, ch chan *core.BlockchainEvent) error {
	edm.mutex.Lock()
	defer edm.mutex.Unlock()

	if _, exists := edm.deliveryChans[subscriptionID]; exists {
		return fmt.Errorf("delivery channel already registered for subscription: %s", subscriptionID)
	}

	edm.deliveryChans[subscriptionID] = ch
	return nil
}

// UnregisterDeliveryChannel unregisters a delivery channel.
// The channel is removed from the map but NOT closed — DeliverEvent may hold
// a stale reference and sending to a closed channel panics. The subscriber's
// goroutine will exit via context cancellation.
func (edm *EventDeliveryManager) UnregisterDeliveryChannel(subscriptionID string) error {
	edm.mutex.Lock()
	defer edm.mutex.Unlock()

	delete(edm.deliveryChans, subscriptionID)

	return nil
}

// DeliverEvent delivers an event to all subscribers
func (edm *EventDeliveryManager) DeliverEvent(ctx context.Context, event *core.BlockchainEvent) error {
	edm.mutex.RLock()
	defer edm.mutex.RUnlock()

	// Get all subscriptions for the chain
	subscriptions, err := edm.subscriptionMgr.GetChainSubscriptions(ctx, event.ChainID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	// Deliver event to each subscription
	for _, sub := range subscriptions {
		if ch, exists := edm.deliveryChans[sub.ID]; exists {
			if !safeSend(ch, event, ctx) {
				edm.eventsDropped.Add(1)
			}
		}
	}

	return nil
}

// safeSend attempts to send an event on a channel, recovering from send-to-closed-channel.
// This guards against the race where UnregisterDeliveryChannel removes the channel
// from the map but DeliverEvent already holds a stale reference.
func safeSend(ch chan *core.BlockchainEvent, event *core.BlockchainEvent, ctx context.Context) (sent bool) { //nolint:revive // ctx cannot be first; chan and event are primary params
	defer func() {
		if r := recover(); r != nil {
			// Expected: send on closed channel (channel removed from map while DeliverEvent held stale ref).
			// Unexpected: log to stderr so the panic is not silently swallowed.
			if r != "send on closed channel" {
				log.Printf("safeSend: unexpected panic recovered: %v", r)
			}
		}
	}()
	select {
	case ch <- event:
		return true
	case <-ctx.Done():
		return false
	default:
		// Channel full — briefly wait before giving up
		select {
		case ch <- event:
			return true
		case <-time.After(100 * time.Millisecond):
			return false
		case <-ctx.Done():
			return false
		}
	}
}

// GetDroppedCount returns the total number of events dropped due to full channels.
func (edm *EventDeliveryManager) GetDroppedCount() int64 {
	return edm.eventsDropped.Load()
}

// ConnectionPoolManager manages WebSocket connections
type ConnectionPoolManager struct {
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
	maxConns    int
	nextConnID  atomic.Int64
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	ID        string
	ClientID  string
	CreatedAt time.Time
	Active    bool
}

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(maxConns int) *ConnectionPoolManager {
	return &ConnectionPoolManager{
		connections: make(map[string]*WebSocketConnection),
		maxConns:    maxConns,
	}
}

// AddConnection adds a WebSocket connection
func (cpm *ConnectionPoolManager) AddConnection(ctx context.Context, clientID string) (*WebSocketConnection, error) {
	cpm.mutex.Lock()
	defer cpm.mutex.Unlock()

	// Check max connections
	if len(cpm.connections) >= cpm.maxConns {
		return nil, fmt.Errorf("max connections reached")
	}

	connID := fmt.Sprintf("conn-%s-%d", clientID, cpm.nextConnID.Add(1))
	conn := &WebSocketConnection{
		ID:        connID,
		ClientID:  clientID,
		CreatedAt: time.Now(),
		Active:    true,
	}

	cpm.connections[connID] = conn
	return conn, nil
}

// RemoveConnection removes a WebSocket connection
func (cpm *ConnectionPoolManager) RemoveConnection(ctx context.Context, connID string) error {
	cpm.mutex.Lock()
	defer cpm.mutex.Unlock()

	if _, exists := cpm.connections[connID]; !exists {
		return fmt.Errorf("connection not found: %s", connID)
	}

	delete(cpm.connections, connID)
	return nil
}

// GetConnection gets a WebSocket connection
func (cpm *ConnectionPoolManager) GetConnection(ctx context.Context, connID string) (*WebSocketConnection, error) {
	cpm.mutex.RLock()
	defer cpm.mutex.RUnlock()

	conn, exists := cpm.connections[connID]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", connID)
	}

	return conn, nil
}

// GetClientConnections gets all connections for a client
func (cpm *ConnectionPoolManager) GetClientConnections(ctx context.Context, clientID string) ([]*WebSocketConnection, error) {
	cpm.mutex.RLock()
	defer cpm.mutex.RUnlock()

	connections := make([]*WebSocketConnection, 0)
	for _, conn := range cpm.connections {
		if conn.ClientID == clientID && conn.Active {
			connections = append(connections, conn)
		}
	}

	return connections, nil
}

// GetConnectionCount returns the number of active connections
func (cpm *ConnectionPoolManager) GetConnectionCount() int {
	cpm.mutex.RLock()
	defer cpm.mutex.RUnlock()

	return len(cpm.connections)
}
