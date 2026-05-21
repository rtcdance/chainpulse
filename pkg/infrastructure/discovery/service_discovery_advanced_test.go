package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRoutingUpdateStructure tests RoutingUpdate structure
func TestRoutingUpdateStructure(t *testing.T) {
	t.Parallel()
	services := []*ServiceInfo{
		{ID: "service-1", Name: "api", Port: 8080},
		{ID: "service-2", Name: "api", Port: 8081},
	}

	update := RoutingUpdate{
		ServiceName: "api",
		Services:    services,
		Timestamp:   time.Now(),
	}

	assert.Equal(t, "api", update.ServiceName)
	assert.Equal(t, 2, len(update.Services))
	assert.NotZero(t, update.Timestamp)
}

// TestNewAdvancedServiceDiscoveryClient tests creating a new advanced service discovery client
func TestNewAdvancedServiceDiscoveryClient(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}

	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	assert.NotNil(t, client)
	assert.Equal(t, discoveryClient, client.discoveryClient)
	assert.Equal(t, registry, client.registry)
	assert.NotNil(t, client.listeners)
	assert.False(t, client.running)
}

// TestRegisterRoutingUpdateListener tests registering a routing update listener
func TestRegisterRoutingUpdateListener(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}
	client.RegisterRoutingUpdateListener("api", listener)

	client.listenerMutex.RLock()
	assert.Equal(t, 1, len(client.listeners["api"]))
	client.listenerMutex.RUnlock()
}

// TestRegisterMultipleListeners tests registering multiple listeners
func TestRegisterMultipleListeners(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener1 := func(update RoutingUpdate) {}
	listener2 := func(update RoutingUpdate) {}

	client.RegisterRoutingUpdateListener("api", listener1)
	client.RegisterRoutingUpdateListener("api", listener2)

	client.listenerMutex.RLock()
	assert.Equal(t, 2, len(client.listeners["api"]))
	client.listenerMutex.RUnlock()
}

// TestRegisterListenersForDifferentServices tests registering listeners for different services
func TestRegisterListenersForDifferentServices(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}

	client.RegisterRoutingUpdateListener("api", listener)
	client.RegisterRoutingUpdateListener("auth", listener)
	client.RegisterRoutingUpdateListener("database", listener)

	client.listenerMutex.RLock()
	assert.Equal(t, 1, len(client.listeners["api"]))
	assert.Equal(t, 1, len(client.listeners["auth"]))
	assert.Equal(t, 1, len(client.listeners["database"]))
	client.listenerMutex.RUnlock()
}

// TestStartAdvancedServiceDiscoveryClient tests starting the client
func TestStartAdvancedServiceDiscoveryClient(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	err := client.Start(ctx)

	assert.NoError(t, err)
	client.runningMutex.RLock()
	assert.True(t, client.running)
	client.runningMutex.RUnlock()

	client.Stop()
}

// TestStartAlreadyRunning tests starting when already running
func TestStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	_ = client.Start(ctx)

	err := client.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	client.Stop()
}

// TestStopAdvancedServiceDiscoveryClient tests stopping the client
func TestStopAdvancedServiceDiscoveryClient(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	_ = client.Start(ctx)
	client.Stop()

	client.runningMutex.RLock()
	assert.False(t, client.running)
	client.runningMutex.RUnlock()
}

// TestRoutingUpdateTimestamp tests routing update timestamp
func TestRoutingUpdateTimestamp(t *testing.T) {
	t.Parallel()
	before := time.Now()
	update := RoutingUpdate{
		ServiceName: "api",
		Timestamp:   time.Now(),
	}
	after := time.Now()

	assert.True(t, update.Timestamp.After(before) || update.Timestamp.Equal(before))
	assert.True(t, update.Timestamp.Before(after) || update.Timestamp.Equal(after))
}

// TestAdvancedServiceDiscoveryClientListenerStorage tests listener storage
func TestAdvancedServiceDiscoveryClientListenerStorage(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}

	client.RegisterRoutingUpdateListener("service-1", listener)
	client.RegisterRoutingUpdateListener("service-2", listener)

	client.listenerMutex.RLock()
	assert.Equal(t, 2, len(client.listeners))
	client.listenerMutex.RUnlock()
}

// TestAdvancedServiceDiscoveryClientConcurrentListenerRegistration tests concurrent listener registration
func TestAdvancedServiceDiscoveryClientConcurrentListenerRegistration(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			client.RegisterRoutingUpdateListener("api", listener)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	client.listenerMutex.RLock()
	assert.Equal(t, 10, len(client.listeners["api"]))
	client.listenerMutex.RUnlock()
}

// TestRoutingUpdateEmptyServices tests routing update with empty services
func TestRoutingUpdateEmptyServices(t *testing.T) {
	t.Parallel()
	update := RoutingUpdate{
		ServiceName: "api",
		Services:    []*ServiceInfo{},
		Timestamp:   time.Now(),
	}

	assert.Equal(t, 0, len(update.Services))
}

// TestRoutingUpdateMultipleServices tests routing update with multiple services
func TestRoutingUpdateMultipleServices(t *testing.T) {
	t.Parallel()
	services := make([]*ServiceInfo, 10)
	for i := 0; i < 10; i++ {
		services[i] = &ServiceInfo{ID: "service-" + string(rune(i))}
	}

	update := RoutingUpdate{
		ServiceName: "api",
		Services:    services,
		Timestamp:   time.Now(),
	}

	assert.Equal(t, 10, len(update.Services))
}

// TestAdvancedServiceDiscoveryClientInitialState tests initial state
func TestAdvancedServiceDiscoveryClientInitialState(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	client.runningMutex.RLock()
	assert.False(t, client.running)
	client.runningMutex.RUnlock()

	client.listenerMutex.RLock()
	assert.Equal(t, 0, len(client.listeners))
	client.listenerMutex.RUnlock()
}

// TestAdvancedServiceDiscoveryClientReferences tests client references
func TestAdvancedServiceDiscoveryClientReferences(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	assert.Equal(t, discoveryClient, client.discoveryClient)
	assert.Equal(t, registry, client.registry)
}

// TestRoutingUpdateServiceName tests routing update service name
func TestRoutingUpdateServiceName(t *testing.T) {
	t.Parallel()
	serviceNames := []string{"api", "auth", "database", "cache"}

	for _, name := range serviceNames {
		update := RoutingUpdate{
			ServiceName: name,
		}
		assert.Equal(t, name, update.ServiceName)
	}
}

// TestAdvancedServiceDiscoveryClientStopWhenNotRunning tests stopping when not running
func TestAdvancedServiceDiscoveryClientStopWhenNotRunning(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	// Should not panic
	client.Stop()

	client.runningMutex.RLock()
	assert.False(t, client.running)
	client.runningMutex.RUnlock()
}

// TestAdvancedServiceDiscoveryClientMultipleStarts tests multiple starts and stops
func TestAdvancedServiceDiscoveryClientMultipleStarts(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	_ = client.Start(ctx)
	client.Stop()

	err := client.Start(ctx)
	assert.NoError(t, err)

	client.Stop()
}
