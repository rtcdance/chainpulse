package e2e

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestPortManagerPropertyUniqueness validates that concurrent allocations always return unique ports
// Feature: e2e-test-port-contract-fixes, Property 1: Port Uniqueness
// Validates: Requirements 1.1, 1.2
func TestPortManagerPropertyUniqueness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("concurrent allocations return unique ports", prop.ForAll(
		func(numAllocations int) bool {
			// Constrain to reasonable range
			if numAllocations < 1 || numAllocations > 50 {
				return true
			}

			pm, err := NewPortManager(10000, 10000+numAllocations+10)
			if err != nil {
				return false
			}

			ctx := context.Background()
			portsChan := make(chan int, numAllocations)
			var wg sync.WaitGroup

			// Allocate ports concurrently
			for i := 0; i < numAllocations; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					port, err := pm.Allocate(ctx)
					if err == nil {
						portsChan <- port
					}
				}()
			}

			wg.Wait()
			close(portsChan)

			// Verify uniqueness
			ports := make(map[int]bool)
			for port := range portsChan {
				if ports[port] {
					return false // Duplicate port found
				}
				ports[port] = true
			}

			return len(ports) == numAllocations
		},
		gen.IntRange(1, 50),
	))

	if !properties.Run(gopter.ConsoleReporter(true)) {
		t.Fail()
	}
}

// TestPortManagerPropertyAllocationWithinRange validates that all allocated ports are within the specified range
// Feature: e2e-test-port-contract-fixes, Property 1: Port Uniqueness
// Validates: Requirements 1.1
func TestPortManagerPropertyAllocationWithinRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("allocated ports are within specified range", prop.ForAll(
		func(minPort, rangeSize int) bool {
			// Constrain to reasonable ranges
			if minPort < 1024 || minPort > 60000 {
				return true
			}
			if rangeSize < 5 || rangeSize > 100 {
				return true
			}

			maxPort := minPort + rangeSize
			pm, err := NewPortManager(minPort, maxPort)
			if err != nil {
				return false
			}

			ctx := context.Background()
			numAllocations := rangeSize / 2

			for i := 0; i < numAllocations; i++ {
				port, err := pm.Allocate(ctx)
				if err != nil {
					return false
				}
				if port < minPort || port >= maxPort {
					return false
				}
			}

			return true
		},
		gen.IntRange(1024, 60000),
		gen.IntRange(5, 100),
	))

	if !properties.Run(gopter.ConsoleReporter(true)) {
		t.Fail()
	}
}

// TestPortManagerPropertyReleaseRestoresAvailability validates that released ports become available again
// Feature: e2e-test-port-contract-fixes, Property 1: Port Uniqueness
// Validates: Requirements 1.3
func TestPortManagerPropertyReleaseRestoresAvailability(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("released ports become available for reallocation", prop.ForAll(
		func(numCycles int) bool {
			// Constrain to reasonable range
			if numCycles < 1 || numCycles > 20 {
				return true
			}

			pm, err := NewPortManager(11000, 11010)
			if err != nil {
				return false
			}

			ctx := context.Background()

			for cycle := 0; cycle < numCycles; cycle++ {
				// Allocate a port
				port, err := pm.Allocate(ctx)
				if err != nil {
					return false
				}

				// Verify it's not available
				if pm.IsAvailable(port) {
					return false
				}

				// Release it
				err = pm.Release(port)
				if err != nil {
					return false
				}

				// Verify it's available again
				if !pm.IsAvailable(port) {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 20),
	))

	if !properties.Run(gopter.ConsoleReporter(true)) {
		t.Fail()
	}
}

// TestPortManagerPropertyConcurrentReleaseAndAllocate validates that concurrent release/allocate operations maintain consistency
// Feature: e2e-test-port-contract-fixes, Property 1: Port Uniqueness
// Validates: Requirements 1.2, 1.5
func TestPortManagerPropertyConcurrentReleaseAndAllocate(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("concurrent release and allocate maintain port uniqueness", prop.ForAll(
		func(numWorkers, numOperations int) bool {
			// Constrain to reasonable ranges
			if numWorkers < 2 || numWorkers > 20 {
				return true
			}
			if numOperations < 1 || numOperations > 10 {
				return true
			}

			pm, err := NewPortManager(12000, 12000+numWorkers+10)
			if err != nil {
				return false
			}

			ctx := context.Background()

			// Pre-allocate ports for workers
			initialPorts := make([]int, numWorkers)
			for i := 0; i < numWorkers; i++ {
				port, err := pm.Allocate(ctx)
				if err != nil {
					return false
				}
				initialPorts[i] = port
			}

			// Concurrent release and allocate
			var wg sync.WaitGroup
			var successCount int32
			var errorCount int32

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for op := 0; op < numOperations; op++ {
						port := initialPorts[workerID]

						// Release
						err := pm.Release(port)
						if err != nil {
							atomic.AddInt32(&errorCount, 1)
							return
						}

						// Allocate
						newPort, err := pm.Allocate(ctx)
						if err != nil {
							atomic.AddInt32(&errorCount, 1)
							return
						}

						initialPorts[workerID] = newPort
						atomic.AddInt32(&successCount, 1)
					}
				}(i)
			}

			wg.Wait()

			// Verify all operations succeeded
			if errorCount > 0 {
				return false
			}

			expectedSuccesses := int64(numWorkers) * int64(numOperations)
			if int64(successCount) != expectedSuccesses {
				return false
			}

			// Verify final state consistency
			stats := pm.GetStats()
			return stats.AllocatedPorts == numWorkers
		},
		gen.IntRange(2, 20),
		gen.IntRange(1, 10),
	))

	if !properties.Run(gopter.ConsoleReporter(true)) {
		t.Fail()
	}
}

// TestPortManagerPropertyStatsConsistency validates that stats remain consistent with actual state
// Feature: e2e-test-port-contract-fixes, Property 1: Port Uniqueness
// Validates: Requirements 1.1, 1.3
func TestPortManagerPropertyStatsConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("stats accurately reflect allocation state", prop.ForAll(
		func(numAllocations int) bool {
			// Constrain to reasonable range
			if numAllocations < 0 || numAllocations > 30 {
				return true
			}

			pm, err := NewPortManager(13000, 13000+numAllocations+10)
			if err != nil {
				return false
			}

			ctx := context.Background()
			allocatedPorts := make([]int, 0)

			// Allocate ports
			for i := 0; i < numAllocations; i++ {
				port, err := pm.Allocate(ctx)
				if err != nil {
					return false
				}
				allocatedPorts = append(allocatedPorts, port)
			}

			// Check stats
			stats := pm.GetStats()
			if stats.AllocatedPorts != numAllocations {
				fmt.Printf("Expected %d allocated, got %d\n", numAllocations, stats.AllocatedPorts)
				return false
			}

			expectedAvailable := stats.TotalPorts - numAllocations
			if stats.AvailablePorts != expectedAvailable {
				fmt.Printf("Expected %d available, got %d\n", expectedAvailable, stats.AvailablePorts)
				return false
			}

			// Release half the ports
			numToRelease := numAllocations / 2
			for i := 0; i < numToRelease; i++ {
				err := pm.Release(allocatedPorts[i])
				if err != nil {
					return false
				}
			}

			// Check stats again
			stats = pm.GetStats()
			expectedAllocated := numAllocations - numToRelease
			if stats.AllocatedPorts != expectedAllocated {
				fmt.Printf("After release: expected %d allocated, got %d\n", expectedAllocated, stats.AllocatedPorts)
				return false
			}

			expectedAvailable = stats.TotalPorts - expectedAllocated
			if stats.AvailablePorts != expectedAvailable {
				fmt.Printf("After release: expected %d available, got %d\n", expectedAvailable, stats.AvailablePorts)
				return false
			}

			return true
		},
		gen.IntRange(0, 30),
	))

	if !properties.Run(gopter.ConsoleReporter(true)) {
		t.Fail()
	}
}

// TestPortManagerPropertyNoPortLeaks validates that all allocated ports can eventually be released
// Feature: e2e-test-port-contract-fixes, Property 1: Port Uniqueness
// Validates: Requirements 1.3, 1.5
func TestPortManagerPropertyNoPortLeaks(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("all allocated ports can be released without errors", prop.ForAll(
		func(numAllocations int) bool {
			// Constrain to reasonable range
			if numAllocations < 1 || numAllocations > 40 {
				return true
			}

			pm, err := NewPortManager(14000, 14000+numAllocations+10)
			if err != nil {
				return false
			}

			ctx := context.Background()
			allocatedPorts := make([]int, 0)

			// Allocate ports
			for i := 0; i < numAllocations; i++ {
				port, err := pm.Allocate(ctx)
				if err != nil {
					return false
				}
				allocatedPorts = append(allocatedPorts, port)
			}

			// Release all ports
			for _, port := range allocatedPorts {
				err := pm.Release(port)
				if err != nil {
					return false
				}
			}

			// Verify all ports are available
			stats := pm.GetStats()
			if stats.AllocatedPorts != 0 {
				return false
			}
			if stats.AvailablePorts != stats.TotalPorts {
				return false
			}

			return true
		},
		gen.IntRange(1, 40),
	))

	if !properties.Run(gopter.ConsoleReporter(true)) {
		t.Fail()
	}
}
