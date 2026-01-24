package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GracefulShutdownManager manages graceful shutdown of services
type GracefulShutdownManager struct {
	mu                    sync.RWMutex
	id                    string
	services              map[string]*ShutdownInfo
	connectionDrainTime   time.Duration
	requestCompletionWait time.Duration
	metrics               *ShutdownMetrics
	shutdownInProgress    bool
}

// ShutdownInfo tracks shutdown information
type ShutdownInfo struct {
	ServiceID              string
	Status                 string // "running", "draining", "stopped"
	ActiveConnections      int
	PendingRequests        int
	DrainStartTime         time.Time
	DrainCompleteTime      time.Time
	LastConnectionClosed   time.Time
	LastRequestCompleted   time.Time
}

// ShutdownMetrics tracks shutdown metrics
type ShutdownMetrics struct {
	mu                    sync.RWMutex
	ShutdownsInitiated    int64
	ShutdownsCompleted    int64
	ShutdownsFailed       int64
	ConnectionsDrained    int64
	RequestsCompleted     int64
	AverageShutdownTime   time.Duration
	TotalShutdownTime     time.Duration
	LastShutdownTime      time.Time
	ForcedTerminations    int64
}

// NewGracefulShutdownManager creates a new graceful shutdown manager
func NewGracefulShutdownManager(id string) *GracefulShutdownManager {
	return &GracefulShutdownManager{
		id:                    id,
		services:              make(map[string]*ShutdownInfo),
		connectionDrainTime:   30 * time.Second,
		requestCompletionWait: 60 * time.Second,
		metrics: &ShutdownMetrics{
			LastShutdownTime: time.Now(),
		},
	}
}

// RegisterService registers a service for shutdown management
func (gsm *GracefulShutdownManager) RegisterService(serviceID string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	gsm.services[serviceID] = &ShutdownInfo{
		ServiceID: serviceID,
		Status:    "running",
	}
}

// InitiateShutdown initiates graceful shutdown
func (gsm *GracefulShutdownManager) InitiateShutdown(ctx context.Context) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if gsm.shutdownInProgress {
		return fmt.Errorf("shutdown already in progress")
	}

	gsm.shutdownInProgress = true
	start := time.Now()
	defer func() {
		gsm.recordShutdownTime(time.Since(start))
		gsm.shutdownInProgress = false
	}()

	gsm.metrics.mu.Lock()
	gsm.metrics.ShutdownsInitiated++
	gsm.metrics.mu.Unlock()

	// Start draining connections for all services
	for serviceID, info := range gsm.services {
		info.Status = "draining"
		info.DrainStartTime = time.Now()

		// Drain connections
		gsm.drainConnections(ctx, serviceID, info)
	}

	// Wait for all requests to complete
	gsm.waitForRequestCompletion(ctx)

	// Mark all services as stopped
	for _, info := range gsm.services {
		info.Status = "stopped"
		info.DrainCompleteTime = time.Now()
	}

	gsm.metrics.mu.Lock()
	gsm.metrics.ShutdownsCompleted++
	gsm.metrics.mu.Unlock()

	return nil
}

// drainConnections drains connections for a service
func (gsm *GracefulShutdownManager) drainConnections(ctx context.Context, serviceID string, info *ShutdownInfo) {
	deadline := time.Now().Add(gsm.connectionDrainTime)

	for {
		if info.ActiveConnections == 0 {
			gsm.metrics.mu.Lock()
			gsm.metrics.ConnectionsDrained++
			gsm.metrics.mu.Unlock()
			break
		}

		// Check if deadline exceeded
		if time.Now().After(deadline) {
			gsm.metrics.mu.Lock()
			gsm.metrics.ForcedTerminations++
			gsm.metrics.mu.Unlock()
			break
		}

		// Wait a bit before checking again
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			// Continue draining
		}
	}
}

// waitForRequestCompletion waits for all requests to complete
func (gsm *GracefulShutdownManager) waitForRequestCompletion(ctx context.Context) {
	deadline := time.Now().Add(gsm.requestCompletionWait)

	for {
		allComplete := true
		for _, info := range gsm.services {
			if info.PendingRequests > 0 {
				allComplete = false
				break
			}
		}

		if allComplete {
			gsm.metrics.mu.Lock()
			gsm.metrics.RequestsCompleted++
			gsm.metrics.mu.Unlock()
			break
		}

		// Check if deadline exceeded
		if time.Now().After(deadline) {
			gsm.metrics.mu.Lock()
			gsm.metrics.ForcedTerminations++
			gsm.metrics.mu.Unlock()
			break
		}

		// Wait a bit before checking again
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			// Continue waiting
		}
	}
}

// recordShutdownTime records shutdown execution time
func (gsm *GracefulShutdownManager) recordShutdownTime(duration time.Duration) {
	gsm.metrics.mu.Lock()
	defer gsm.metrics.mu.Unlock()

	gsm.metrics.TotalShutdownTime += duration
	if gsm.metrics.ShutdownsCompleted > 0 {
		gsm.metrics.AverageShutdownTime = gsm.metrics.TotalShutdownTime / time.Duration(gsm.metrics.ShutdownsCompleted)
	}
	gsm.metrics.LastShutdownTime = time.Now()
}

// UpdateConnectionCount updates the active connection count
func (gsm *GracefulShutdownManager) UpdateConnectionCount(serviceID string, count int) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	info, exists := gsm.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found")
	}

	info.ActiveConnections = count
	if count == 0 {
		info.LastConnectionClosed = time.Now()
	}

	return nil
}

// UpdatePendingRequests updates the pending request count
func (gsm *GracefulShutdownManager) UpdatePendingRequests(serviceID string, count int) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	info, exists := gsm.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found")
	}

	info.PendingRequests = count
	if count == 0 {
		info.LastRequestCompleted = time.Now()
	}

	return nil
}

// GetShutdownStatus returns the shutdown status
func (gsm *GracefulShutdownManager) GetShutdownStatus() map[string]interface{} {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	status := make(map[string]interface{})
	for serviceID, info := range gsm.services {
		status[serviceID] = map[string]interface{}{
			"status":              info.Status,
			"active_connections":  info.ActiveConnections,
			"pending_requests":    info.PendingRequests,
			"drain_start_time":    info.DrainStartTime,
			"drain_complete_time": info.DrainCompleteTime,
		}
	}

	return status
}

// GetMetrics returns shutdown metrics
func (gsm *GracefulShutdownManager) GetMetrics() map[string]interface{} {
	gsm.metrics.mu.RLock()
	defer gsm.metrics.mu.RUnlock()

	return map[string]interface{}{
		"shutdowns_initiated":    gsm.metrics.ShutdownsInitiated,
		"shutdowns_completed":    gsm.metrics.ShutdownsCompleted,
		"shutdowns_failed":       gsm.metrics.ShutdownsFailed,
		"connections_drained":    gsm.metrics.ConnectionsDrained,
		"requests_completed":     gsm.metrics.RequestsCompleted,
		"average_shutdown_time":  gsm.metrics.AverageShutdownTime.String(),
		"total_shutdown_time":    gsm.metrics.TotalShutdownTime.String(),
		"forced_terminations":    gsm.metrics.ForcedTerminations,
	}
}

// IsShutdownInProgress returns if shutdown is in progress
func (gsm *GracefulShutdownManager) IsShutdownInProgress() bool {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	return gsm.shutdownInProgress
}

// Deregister deregisters a service
func (gsm *GracefulShutdownManager) Deregister(serviceID string) error {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if _, exists := gsm.services[serviceID]; !exists {
		return fmt.Errorf("service not found")
	}

	delete(gsm.services, serviceID)
	return nil
}
