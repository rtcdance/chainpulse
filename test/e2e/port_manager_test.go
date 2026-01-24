package e2e

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPortManagerAllocation(t *testing.T) {
	pm, err := NewPortManager(9000, 9010)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	ctx := context.Background()

	// Allocate a port
	port1, err := pm.Allocate(ctx)
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	if port1 < 9000 || port1 >= 9010 {
		t.Errorf("Port %d out of range [9000, 9010)", port1)
	}

	// Verify it's not available
	if pm.IsAvailable(port1) {
		t.Errorf("Port %d should not be available after allocation", port1)
	}

	// Release the port
	err = pm.Release(port1)
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Verify it's available again
	if !pm.IsAvailable(port1) {
		t.Errorf("Port %d should be available after release", port1)
	}
}

func TestPortManagerUniqueness(t *testing.T) {
	pm, err := NewPortManager(9000, 9020)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	ctx := context.Background()
	ports := make(map[int]bool)

	// Allocate multiple ports
	for i := 0; i < 10; i++ {
		port, err := pm.Allocate(ctx)
		if err != nil {
			t.Fatalf("Failed to allocate port %d: %v", i, err)
		}

		if ports[port] {
			t.Errorf("Port %d allocated twice", port)
		}
		ports[port] = true
	}

	// Verify we got 10 unique ports
	if len(ports) != 10 {
		t.Errorf("Expected 10 unique ports, got %d", len(ports))
	}
}

func TestPortManagerExhaustion(t *testing.T) {
	pm, err := NewPortManager(9000, 9005)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	ctx := context.Background()
	ports := make([]int, 0)

	// Allocate all available ports
	for i := 0; i < 5; i++ {
		port, err := pm.Allocate(ctx)
		if err != nil {
			t.Fatalf("Failed to allocate port %d: %v", i, err)
		}
		ports = append(ports, port)
	}

	// Try to allocate one more (should fail)
	_, err = pm.Allocate(ctx)
	if err == nil {
		t.Error("Expected error when allocating from exhausted pool")
	}

	// Release one port
	err = pm.Release(ports[0])
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Now allocation should succeed
	port, err := pm.Allocate(ctx)
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}

	if port != ports[0] {
		t.Errorf("Expected port %d, got %d", ports[0], port)
	}
}

func TestPortManagerConcurrentAllocation(t *testing.T) {
	pm, err := NewPortManager(9000, 9100)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	ctx := context.Background()
	numGoroutines := 50
	portsPerGoroutine := 1

	var wg sync.WaitGroup
	portsChan := make(chan int, numGoroutines*portsPerGoroutine)
	errorsChan := make(chan error, numGoroutines*portsPerGoroutine)

	// Launch concurrent allocations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < portsPerGoroutine; j++ {
				port, err := pm.Allocate(ctx)
				if err != nil {
					errorsChan <- err
					return
				}
				portsChan <- port
			}
		}()
	}

	wg.Wait()
	close(portsChan)
	close(errorsChan)

	// Check for errors
	for err := range errorsChan {
		t.Errorf("Allocation error: %v", err)
	}

	// Verify uniqueness
	ports := make(map[int]bool)
	for port := range portsChan {
		if ports[port] {
			t.Errorf("Port %d allocated multiple times", port)
		}
		ports[port] = true
	}

	if len(ports) != numGoroutines*portsPerGoroutine {
		t.Errorf("Expected %d unique ports, got %d", numGoroutines*portsPerGoroutine, len(ports))
	}
}

func TestPortManagerConcurrentReleaseAndAllocate(t *testing.T) {
	pm, err := NewPortManager(9000, 9050)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	ctx := context.Background()
	numWorkers := 10
	numIterations := 10

	var wg sync.WaitGroup
	var successCount int32

	// Allocate initial ports for each worker
	initialPorts := make([]int, numWorkers)
	for i := 0; i < numWorkers; i++ {
		port, err := pm.Allocate(ctx)
		if err != nil {
			t.Fatalf("Failed to allocate initial port: %v", err)
		}
		initialPorts[i] = port
	}

	// Concurrent release and allocate per worker
	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for iter := 0; iter < numIterations; iter++ {
				port := initialPorts[id]

				// Release
				err := pm.Release(port)
				if err != nil {
					t.Errorf("Worker %d: Failed to release port %d: %v", id, port, err)
					return
				}

				// Allocate
				newPort, err := pm.Allocate(ctx)
				if err != nil {
					t.Errorf("Worker %d: Failed to allocate after release: %v", id, err)
					return
				}

				if newPort < 9000 || newPort >= 9050 {
					t.Errorf("Worker %d: Port %d out of range", id, newPort)
					return
				}

				initialPorts[id] = newPort
				atomic.AddInt32(&successCount, 1)
			}
		}(workerID)
	}

	wg.Wait()

	expectedSuccesses := int32(numWorkers * numIterations)
	if successCount != expectedSuccesses {
		t.Errorf("Expected %d successful operations, got %d", expectedSuccesses, successCount)
	}
}

func TestPortManagerContextCancellation(t *testing.T) {
	pm, err := NewPortManager(9000, 9005)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	// Allocate all ports
	for i := 0; i < 5; i++ {
		_, err := pm.Allocate(context.Background())
		if err != nil {
			t.Fatalf("Failed to allocate port: %v", err)
		}
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to allocate with cancelled context
	_, err = pm.Allocate(ctx)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestPortManagerStats(t *testing.T) {
	pm, err := NewPortManager(9000, 9010)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	ctx := context.Background()

	// Check initial stats
	stats := pm.GetStats()
	if stats.TotalPorts != 10 {
		t.Errorf("Expected 10 total ports, got %d", stats.TotalPorts)
	}
	if stats.AllocatedPorts != 0 {
		t.Errorf("Expected 0 allocated ports, got %d", stats.AllocatedPorts)
	}
	if stats.AvailablePorts != 10 {
		t.Errorf("Expected 10 available ports, got %d", stats.AvailablePorts)
	}

	// Allocate a port
	port, err := pm.Allocate(ctx)
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	// Check stats after allocation
	stats = pm.GetStats()
	if stats.AllocatedPorts != 1 {
		t.Errorf("Expected 1 allocated port, got %d", stats.AllocatedPorts)
	}
	if stats.AvailablePorts != 9 {
		t.Errorf("Expected 9 available ports, got %d", stats.AvailablePorts)
	}
	if stats.LastAllocated.IsZero() {
		t.Error("Expected LastAllocated to be set")
	}

	// Release the port
	err = pm.Release(port)
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Check stats after release
	stats = pm.GetStats()
	if stats.AllocatedPorts != 0 {
		t.Errorf("Expected 0 allocated ports after release, got %d", stats.AllocatedPorts)
	}
	if stats.AvailablePorts != 10 {
		t.Errorf("Expected 10 available ports after release, got %d", stats.AvailablePorts)
	}
}

func TestPortManagerReleaseNonAllocated(t *testing.T) {
	pm, err := NewPortManager(9000, 9010)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	// Try to release a port that was never allocated
	err = pm.Release(9005)
	if err == nil {
		t.Error("Expected error when releasing non-allocated port")
	}
}

func TestPortManagerInvalidRange(t *testing.T) {
	_, err := NewPortManager(9010, 9000)
	if err == nil {
		t.Error("Expected error with invalid port range")
	}
}

func TestPortManagerTimeoutContext(t *testing.T) {
	pm, err := NewPortManager(9000, 9005)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	// Allocate all ports
	for i := 0; i < 5; i++ {
		_, err := pm.Allocate(context.Background())
		if err != nil {
			t.Fatalf("Failed to allocate port: %v", err)
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try to allocate with timeout (should fail immediately since pool is exhausted)
	_, err = pm.Allocate(ctx)
	if err == nil {
		t.Error("Expected error when allocating from exhausted pool")
	}
}
