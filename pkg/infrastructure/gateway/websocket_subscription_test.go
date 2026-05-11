package gateway

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"chainpulse/pkg/core"
	"github.com/stretchr/testify/assert"
)

// TestNewSubscriptionManager tests subscription manager creation
func TestNewSubscriptionManager(t *testing.T) {
	manager := NewSubscriptionManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.subscriptions)
	assert.NotNil(t, manager.clients)
	assert.Equal(t, 0, len(manager.subscriptions))
}

// TestSubscribe tests creating a subscription
func TestSubscribe(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	sub, err := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")

	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, "client-1", sub.ClientID)
	assert.Equal(t, "ethereum", sub.ChainID)
	assert.Equal(t, "Transfer", sub.EventType)
	assert.True(t, sub.Active)
}

// TestUnsubscribe tests removing a subscription
func TestUnsubscribe(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	sub, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	err := manager.Unsubscribe(ctx, sub.ID)

	assert.NoError(t, err)
}

// TestUnsubscribeNotFound tests unsubscribing non-existent subscription
func TestUnsubscribeNotFound(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	err := manager.Unsubscribe(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetSubscription tests getting a subscription
func TestGetSubscription(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	sub, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	retrieved, err := manager.GetSubscription(ctx, sub.ID)

	assert.NoError(t, err)
	assert.Equal(t, sub.ID, retrieved.ID)
}

// TestGetSubscriptionNotFound tests getting non-existent subscription
func TestGetSubscriptionNotFound(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	_, err := manager.GetSubscription(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetClientSubscriptions tests getting client subscriptions
func TestGetClientSubscriptions(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	_, _ = manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	_, _ = manager.Subscribe(ctx, "client-1", "polygon", "Swap")

	subs, err := manager.GetClientSubscriptions(ctx, "client-1")

	assert.NoError(t, err)
	assert.Equal(t, 2, len(subs))
}

// TestGetClientSubscriptionsEmpty tests getting subscriptions for client with none
func TestGetClientSubscriptionsEmpty(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	subs, err := manager.GetClientSubscriptions(ctx, "nonexistent")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(subs))
}

// TestGetChainSubscriptions tests getting chain subscriptions
func TestGetChainSubscriptions(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	_, _ = manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	_, _ = manager.Subscribe(ctx, "client-2", "ethereum", "Swap")
	_, _ = manager.Subscribe(ctx, "client-3", "polygon", "Transfer")

	subs, err := manager.GetChainSubscriptions(ctx, "ethereum")

	assert.NoError(t, err)
	assert.Equal(t, 2, len(subs))
}

// TestGetChainSubscriptionsEmpty tests getting subscriptions for chain with none
func TestGetChainSubscriptionsEmpty(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	subs, err := manager.GetChainSubscriptions(ctx, "nonexistent")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(subs))
}

// TestSubscriptionFields tests subscription fields
func TestSubscriptionFields(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	sub, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")

	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, "client-1", sub.ClientID)
	assert.Equal(t, "ethereum", sub.ChainID)
	assert.Equal(t, "Transfer", sub.EventType)
	assert.False(t, sub.CreatedAt.IsZero())
	assert.True(t, sub.Active)
}

// TestNewEventDeliveryManager tests event delivery manager creation
func TestNewEventDeliveryManager(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)

	assert.NotNil(t, deliveryMgr)
	assert.Equal(t, subMgr, deliveryMgr.subscriptionMgr)
	assert.NotNil(t, deliveryMgr.deliveryChans)
}

// TestRegisterDeliveryChannel tests registering delivery channel
func TestRegisterDeliveryChannel(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	sub, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	ch := make(chan *core.BlockchainEvent, 10)

	err := deliveryMgr.RegisterDeliveryChannel(sub.ID, ch)

	assert.NoError(t, err)
}

// TestRegisterDeliveryChannelDuplicate tests registering duplicate channel
func TestRegisterDeliveryChannelDuplicate(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	sub, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	ch1 := make(chan *core.BlockchainEvent, 10)
	ch2 := make(chan *core.BlockchainEvent, 10)

	_ = deliveryMgr.RegisterDeliveryChannel(sub.ID, ch1)
	err := deliveryMgr.RegisterDeliveryChannel(sub.ID, ch2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestUnregisterDeliveryChannel tests unregistering delivery channel
func TestUnregisterDeliveryChannel(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	sub, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	ch := make(chan *core.BlockchainEvent, 10)

	_ = deliveryMgr.RegisterDeliveryChannel(sub.ID, ch)
	err := deliveryMgr.UnregisterDeliveryChannel(sub.ID)

	assert.NoError(t, err)
}

// TestDeliverEvent tests delivering an event
func TestDeliverEvent(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	sub, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	ch := make(chan *core.BlockchainEvent, 10)
	_ = deliveryMgr.RegisterDeliveryChannel(sub.ID, ch)

	event := &core.BlockchainEvent{
		ChainID: "ethereum",
	}

	err := deliveryMgr.DeliverEvent(ctx, event)

	assert.NoError(t, err)
}

// TestDeliverEventMultipleSubscribers tests delivering to multiple subscribers
func TestDeliverEventMultipleSubscribers(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	sub1, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	sub2, _ := subMgr.Subscribe(ctx, "client-2", "ethereum", "Transfer")

	ch1 := make(chan *core.BlockchainEvent, 10)
	ch2 := make(chan *core.BlockchainEvent, 10)

	_ = deliveryMgr.RegisterDeliveryChannel(sub1.ID, ch1)
	_ = deliveryMgr.RegisterDeliveryChannel(sub2.ID, ch2)

	event := &core.BlockchainEvent{
		ChainID: "ethereum",
	}

	err := deliveryMgr.DeliverEvent(ctx, event)

	assert.NoError(t, err)
}

// TestNewConnectionPoolManager tests connection pool manager creation
func TestNewConnectionPoolManager(t *testing.T) {
	manager := NewConnectionPoolManager(100)

	assert.NotNil(t, manager)
	assert.Equal(t, 100, manager.maxConns)
	assert.NotNil(t, manager.connections)
}

// TestAddConnection tests adding a connection
func TestAddConnection(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	conn, err := manager.AddConnection(ctx, "client-1")

	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, "client-1", conn.ClientID)
	assert.True(t, conn.Active)
}

// TestAddConnectionMaxReached tests adding connection when max reached
func TestAddConnectionMaxReached(t *testing.T) {
	manager := NewConnectionPoolManager(1)
	ctx := context.Background()

	_, _ = manager.AddConnection(ctx, "client-1")
	_, err := manager.AddConnection(ctx, "client-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max connections reached")
}

// TestRemoveConnection tests removing a connection
func TestRemoveConnection(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	conn, _ := manager.AddConnection(ctx, "client-1")
	err := manager.RemoveConnection(ctx, conn.ID)

	assert.NoError(t, err)
}

// TestRemoveConnectionNotFound tests removing non-existent connection
func TestRemoveConnectionNotFound(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	err := manager.RemoveConnection(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetConnection tests getting a connection
func TestGetConnection(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	conn, _ := manager.AddConnection(ctx, "client-1")
	retrieved, err := manager.GetConnection(ctx, conn.ID)

	assert.NoError(t, err)
	assert.Equal(t, conn.ID, retrieved.ID)
}

// TestGetConnectionNotFound tests getting non-existent connection
func TestGetConnectionNotFound(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	_, err := manager.GetConnection(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetClientConnections tests getting client connections
func TestGetClientConnections(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	_, _ = manager.AddConnection(ctx, "client-1")
	_, _ = manager.AddConnection(ctx, "client-1")

	conns, err := manager.GetClientConnections(ctx, "client-1")

	assert.NoError(t, err)
	assert.Equal(t, 2, len(conns))
}

// TestGetClientConnectionsEmpty tests getting connections for client with none
func TestGetClientConnectionsEmpty(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	conns, err := manager.GetClientConnections(ctx, "nonexistent")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(conns))
}

// TestGetConnectionCount tests getting connection count
func TestGetConnectionCount(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	_, _ = manager.AddConnection(ctx, "client-1")
	_, _ = manager.AddConnection(ctx, "client-2")

	count := manager.GetConnectionCount()

	assert.Equal(t, 2, count)
}

// TestConnectionFields tests connection fields
func TestConnectionFields(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	conn, _ := manager.AddConnection(ctx, "client-1")

	assert.NotEmpty(t, conn.ID)
	assert.Equal(t, "client-1", conn.ClientID)
	assert.False(t, conn.CreatedAt.IsZero())
	assert.True(t, conn.Active)
}

// TestConcurrentSubscriptions tests concurrent subscriptions
func TestConcurrentSubscriptions(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	var wg sync.WaitGroup
	var subCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := manager.Subscribe(ctx, fmt.Sprintf("client-%d", id), "ethereum", "Transfer")
			if err == nil {
				atomic.AddInt32(&subCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&subCount))
}

// TestConcurrentConnections tests concurrent connections
func TestConcurrentConnections(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	var wg sync.WaitGroup
	var connCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := manager.AddConnection(ctx, fmt.Sprintf("client-%d", id))
			if err == nil {
				atomic.AddInt32(&connCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&connCount))
}

// TestSubscriptionLifecycle tests subscription lifecycle
func TestSubscriptionLifecycle(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	// Create subscription
	sub, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	assert.True(t, sub.Active)

	// Get subscription
	retrieved, _ := manager.GetSubscription(ctx, sub.ID)
	assert.Equal(t, sub.ID, retrieved.ID)

	// Unsubscribe
	_ = manager.Unsubscribe(ctx, sub.ID)

	// Verify removed
	_, err := manager.GetSubscription(ctx, sub.ID)
	assert.Error(t, err)
}

// TestConnectionLifecycle tests connection lifecycle
func TestConnectionLifecycle(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	// Add connection
	conn, _ := manager.AddConnection(ctx, "client-1")
	assert.True(t, conn.Active)

	// Get connection
	retrieved, _ := manager.GetConnection(ctx, conn.ID)
	assert.Equal(t, conn.ID, retrieved.ID)

	// Remove connection
	_ = manager.RemoveConnection(ctx, conn.ID)

	// Verify removed
	_, err := manager.GetConnection(ctx, conn.ID)
	assert.Error(t, err)
}

// TestMultipleChainSubscriptions tests subscriptions across multiple chains
func TestMultipleChainSubscriptions(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	_, _ = manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	_, _ = manager.Subscribe(ctx, "client-1", "polygon", "Transfer")
	_, _ = manager.Subscribe(ctx, "client-1", "arbitrum", "Transfer")

	ethSubs, _ := manager.GetChainSubscriptions(ctx, "ethereum")
	polySubs, _ := manager.GetChainSubscriptions(ctx, "polygon")
	arbSubs, _ := manager.GetChainSubscriptions(ctx, "arbitrum")

	assert.Equal(t, 1, len(ethSubs))
	assert.Equal(t, 1, len(polySubs))
	assert.Equal(t, 1, len(arbSubs))
}

// TestEventDeliveryChannelFull tests event delivery with full channel
func TestEventDeliveryChannelFull(t *testing.T) {
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	sub, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	ch := make(chan *core.BlockchainEvent, 1)
	_ = deliveryMgr.RegisterDeliveryChannel(sub.ID, ch)

	// Fill channel
	ch <- &core.BlockchainEvent{ChainID: "ethereum"}

	// Try to deliver — channel is full, will wait briefly then count as dropped
	event := &core.BlockchainEvent{ChainID: "ethereum"}
	err := deliveryMgr.DeliverEvent(ctx, event)

	assert.NoError(t, err)
	// Event should be counted as dropped since the channel stays full
	assert.Equal(t, int64(1), deliveryMgr.GetDroppedCount())
}

// TestConnectionPoolMaxConnections tests connection pool max connections
func TestConnectionPoolMaxConnections(t *testing.T) {
	manager := NewConnectionPoolManager(5)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := manager.AddConnection(ctx, fmt.Sprintf("client-%d", i))
		assert.NoError(t, err)
	}

	// 6th connection should fail
	_, err := manager.AddConnection(ctx, "client-6")
	assert.Error(t, err)
}

// TestSubscriptionTimestamp tests subscription timestamp
func TestSubscriptionTimestamp(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	before := time.Now()
	sub, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	after := time.Now()

	assert.True(t, sub.CreatedAt.After(before) || sub.CreatedAt.Equal(before))
	assert.True(t, sub.CreatedAt.Before(after) || sub.CreatedAt.Equal(after))
}

// TestConnectionTimestamp tests connection timestamp
func TestConnectionTimestamp(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	before := time.Now()
	conn, _ := manager.AddConnection(ctx, "client-1")
	after := time.Now()

	assert.True(t, conn.CreatedAt.After(before) || conn.CreatedAt.Equal(before))
	assert.True(t, conn.CreatedAt.Before(after) || conn.CreatedAt.Equal(after))
}

// TestSubscriptionIDUniqueness tests subscription ID uniqueness
func TestSubscriptionIDUniqueness(t *testing.T) {
	manager := NewSubscriptionManager()
	ctx := context.Background()

	sub1, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")
	sub2, _ := manager.Subscribe(ctx, "client-1", "ethereum", "Transfer")

	assert.NotEqual(t, sub1.ID, sub2.ID)
}

// TestConnectionIDUniqueness tests connection ID uniqueness
func TestConnectionIDUniqueness(t *testing.T) {
	manager := NewConnectionPoolManager(100)
	ctx := context.Background()

	conn1, _ := manager.AddConnection(ctx, "client-1")
	conn2, _ := manager.AddConnection(ctx, "client-1")

	assert.NotEqual(t, conn1.ID, conn2.ID)
}

func TestConcurrentUnregisterAndDeliverNoPanic(t *testing.T) {
	// Regression test: UnregisterDeliveryChannel used to close the channel,
	// which could cause DeliverEvent to panic on send-to-closed-channel.
	subMgr := NewSubscriptionManager()
	deliveryMgr := NewEventDeliveryManager(subMgr)
	ctx := context.Background()

	const numSubs = 100
	var wg sync.WaitGroup

	// Register many subscriptions
	for i := 0; i < numSubs; i++ {
		sub, _ := subMgr.Subscribe(ctx, "client-1", "ethereum", "Transfer")
		ch := make(chan *core.BlockchainEvent, 10)
		_ = deliveryMgr.RegisterDeliveryChannel(sub.ID, ch)

		// Concurrently unregister and deliver
		wg.Add(2)
		go func(subID string) {
			defer wg.Done()
			time.Sleep(time.Duration(i%10) * time.Millisecond)
			_ = deliveryMgr.UnregisterDeliveryChannel(subID)
		}(sub.ID)
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(i%10) * time.Millisecond)
			_ = deliveryMgr.DeliverEvent(ctx, &core.BlockchainEvent{ChainID: "ethereum"})
		}()
	}

	wg.Wait()
	// If we get here without panicking, the test passes.
}
