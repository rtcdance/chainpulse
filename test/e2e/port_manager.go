package e2e

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// PortManager manages dynamic port allocation for concurrent tests
type PortManager interface {
	// Allocate returns an available port
	Allocate(ctx context.Context) (int, error)

	// Release returns a port to the pool
	Release(port int) error

	// IsAvailable checks if a port is available
	IsAvailable(port int) bool

	// GetStats returns allocation statistics
	GetStats() PortStats
}

// PortStats contains port allocation statistics
type PortStats struct {
	TotalPorts     int
	AllocatedPorts int
	AvailablePorts int
	LastAllocated  time.Time
}

// DefaultPortManager implements PortManager with a pool-based approach
type DefaultPortManager struct {
	mu         sync.Mutex
	available  []int
	allocated  map[int]*PortAllocation
	minPort    int
	maxPort    int
	lastAlloc  time.Time
}

// PortAllocation tracks information about an allocated port
type PortAllocation struct {
	Port        int
	TestID      string
	AllocTime   time.Time
	ReleaseTime *time.Time
}

// NewPortManager creates a new port manager with a pool of available ports
func NewPortManager(minPort, maxPort int) (PortManager, error) {
	if minPort >= maxPort {
		return nil, fmt.Errorf("invalid port range: min=%d must be less than max=%d", minPort, maxPort)
	}

	pm := &DefaultPortManager{
		available: make([]int, 0, maxPort-minPort),
		allocated: make(map[int]*PortAllocation),
		minPort:   minPort,
		maxPort:   maxPort,
	}

	// Pre-allocate ports in the range
	for port := minPort; port < maxPort; port++ {
		pm.available = append(pm.available, port)
	}

	return pm, nil
}

// Allocate returns an available port from the pool
func (pm *DefaultPortManager) Allocate(ctx context.Context) (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Try to find an available port
	for i := 0; i < len(pm.available); i++ {
		port := pm.available[i]

		// Verify the port is actually available
		if pm.isPortAvailable(port) {
			// Remove from available list
			pm.available = append(pm.available[:i], pm.available[i+1:]...)

			// Add to allocated map
			pm.allocated[port] = &PortAllocation{
				Port:      port,
				AllocTime: time.Now(),
			}

			pm.lastAlloc = time.Now()
			return port, nil
		}
	}

	// No available ports
	return 0, fmt.Errorf("no available ports in range [%d, %d); allocated: %d, available: %d",
		pm.minPort, pm.maxPort, len(pm.allocated), len(pm.available))
}

// Release returns a port to the available pool
func (pm *DefaultPortManager) Release(port int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if port is allocated
	alloc, exists := pm.allocated[port]
	if !exists {
		return fmt.Errorf("port %d is not allocated", port)
	}

	// Mark release time
	now := time.Now()
	alloc.ReleaseTime = &now

	// Remove from allocated
	delete(pm.allocated, port)

	// Add back to available
	pm.available = append(pm.available, port)

	return nil
}

// IsAvailable checks if a port is available (not allocated)
func (pm *DefaultPortManager) IsAvailable(port int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, allocated := pm.allocated[port]
	return !allocated
}

// GetStats returns current allocation statistics
func (pm *DefaultPortManager) GetStats() PortStats {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return PortStats{
		TotalPorts:     pm.maxPort - pm.minPort,
		AllocatedPorts: len(pm.allocated),
		AvailablePorts: len(pm.available),
		LastAllocated:  pm.lastAlloc,
	}
}

// isPortAvailable checks if a port can be bound to (internal, must hold lock)
func (pm *DefaultPortManager) isPortAvailable(port int) bool {
	// Try to bind to the port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

// GetAllocatedPorts returns a map of all allocated ports (for testing/debugging)
func (pm *DefaultPortManager) GetAllocatedPorts() map[int]*PortAllocation {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	result := make(map[int]*PortAllocation)
	for port, alloc := range pm.allocated {
		result[port] = alloc
	}
	return result
}

// GetAvailablePorts returns a slice of available ports (for testing/debugging)
func (pm *DefaultPortManager) GetAvailablePorts() []int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	result := make([]int, len(pm.available))
	copy(result, pm.available)
	return result
}
