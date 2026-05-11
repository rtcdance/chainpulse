package shared

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Connection represents a pooled connection
type Connection interface {
	// IsHealthy checks if the connection is healthy
	IsHealthy() bool
	// Close closes the connection
	Close() error
	// GetID returns the connection ID
	GetID() string
}

// ConnectionPool manages a pool of reusable connections
type ConnectionPool struct {
	name          string
	factory       ConnectionFactory
	maxSize       int
	idleTimeout   time.Duration
	available     chan Connection
	inUse         map[string]Connection
	mu            sync.RWMutex
	metrics       *PoolMetrics
	ctx           context.Context
	cancel        context.CancelFunc
	cleanupTicker *time.Ticker
}

// ConnectionFactory creates new connections
type ConnectionFactory interface {
	Create(ctx context.Context) (Connection, error)
}

// PoolMetrics tracks connection pool metrics
type PoolMetrics struct {
	created     int64
	reused      int64
	closed      int64
	errors      int64
	currentSize int64
	maxSize     int64
	mu          sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(name string, factory ConnectionFactory, maxSize int, idleTimeout time.Duration) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &ConnectionPool{
		name:        name,
		factory:     factory,
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		available:   make(chan Connection, maxSize),
		inUse:       make(map[string]Connection),
		metrics: &PoolMetrics{
			maxSize: int64(maxSize),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine
	pool.cleanupTicker = time.NewTicker(idleTimeout / 2)
	go pool.cleanupLoop()

	return pool
}

// Acquire gets a connection from the pool
func (p *ConnectionPool) Acquire(ctx context.Context) (Connection, error) {
	// Check if pool is closed
	select {
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	default:
	}

	// First try to get from available connections
	select {
	case conn := <-p.available:
		if conn == nil {
			return nil, fmt.Errorf("pool is closed")
		}
		if conn.IsHealthy() {
			p.recordReuse()
			p.recordInUse(conn)
			return conn, nil
		}
		// Connection is unhealthy, close it and create a new one
		_ = conn.Close()
		p.recordClosed()
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	default:
	}

	// No available connections, check if we can create a new one.
	// Hold write lock for the check + reserve to prevent TOCTOU race:
	// multiple goroutines could otherwise read the same size and all create.
	p.mu.Lock()
	if int64(len(p.inUse)) >= int64(p.maxSize) {
		p.mu.Unlock()
	} else {
		// Reserve a slot with a nil placeholder to prevent over-allocation
		slotID := fmt.Sprintf("creating-%d", time.Now().UnixNano())
		p.inUse[slotID] = nil
		p.mu.Unlock()

		conn, err := p.factory.Create(ctx)
		if err != nil {
			// Remove the placeholder on failure
			p.mu.Lock()
			delete(p.inUse, slotID)
			p.mu.Unlock()
			p.recordError()
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}
		p.recordCreated()
		// Replace placeholder with real connection
		p.mu.Lock()
		delete(p.inUse, slotID)
		p.inUse[conn.GetID()] = conn
		p.mu.Unlock()
		return conn, nil
	}

	// Wait for available connection
	select {
	case conn := <-p.available:
		if conn == nil {
			return nil, fmt.Errorf("pool is closed")
		}
		if conn.IsHealthy() {
			p.recordReuse()
			p.recordInUse(conn)
			return conn, nil
		}
		_ = conn.Close()
		p.recordClosed()
		// Retry
		return p.Acquire(ctx)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pool is closed")
	}
}

// Release returns a connection to the pool
func (p *ConnectionPool) Release(conn Connection) error {
	if conn == nil {
		return fmt.Errorf("cannot release nil connection")
	}

	p.mu.Lock()
	delete(p.inUse, conn.GetID())
	p.mu.Unlock()

	if !conn.IsHealthy() {
		_ = conn.Close()
		p.recordClosed()
		return nil
	}

	select {
	case p.available <- conn:
		return nil
	case <-p.ctx.Done():
		_ = conn.Close()
		p.recordClosed()
		return fmt.Errorf("pool is closed")
	default:
		// Pool is full, close the connection
		_ = conn.Close()
		p.recordClosed()
		return nil
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.cancel()
	p.cleanupTicker.Stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Close all in-use connections
	for _, conn := range p.inUse {
		_ = conn.Close()
		p.recordClosed()
	}
	p.inUse = make(map[string]Connection)

	// Close all available connections
	close(p.available)
	for conn := range p.available {
		_ = conn.Close()
		p.recordClosed()
	}

	return nil
}

// GetMetrics returns pool metrics
func (p *ConnectionPool) GetMetrics() map[string]interface{} {
	p.metrics.mu.RLock()
	created := p.metrics.created
	reused := p.metrics.reused
	closed := p.metrics.closed
	errors := p.metrics.errors
	maxSize := p.metrics.maxSize
	p.metrics.mu.RUnlock()

	p.mu.RLock()
	currentSize := int64(len(p.inUse))
	available := int64(len(p.available))
	p.mu.RUnlock()

	capacityPosture := classifyPoolCapacityPosture(currentSize, maxSize, available)
	runtimePosture := classifyPoolRuntimePosture(created, errors, capacityPosture)

	return map[string]interface{}{
		"pool_name":        p.name,
		"created":          created,
		"reused":           reused,
		"closed":           closed,
		"errors":           errors,
		"current_size":     currentSize,
		"max_size":         maxSize,
		"available":        available,
		"coverage_posture": capacityPosture,
		"capacity_posture": capacityPosture,
		"runtime_posture":  runtimePosture,
		"reliability_hint": buildPoolReliabilityHint(runtimePosture, capacityPosture),
	}
}

// GetRuntimeMetrics returns a compact runtime surface for pool capacity and
// reliability posture on top of the raw metrics.
func (p *ConnectionPool) GetRuntimeMetrics() map[string]interface{} {
	metrics := p.GetMetrics()

	created, _ := metrics["created"].(int64)
	reused, _ := metrics["reused"].(int64)
	closed, _ := metrics["closed"].(int64)
	errors, _ := metrics["errors"].(int64)
	currentSize, _ := metrics["current_size"].(int64)
	maxSize, _ := metrics["max_size"].(int64)
	available, _ := metrics["available"].(int64)

	return map[string]interface{}{
		"pool_name":        p.name,
		"created":          created,
		"reused":           reused,
		"closed":           closed,
		"errors":           errors,
		"current_size":     currentSize,
		"max_size":         maxSize,
		"available":        available,
		"coverage_posture": metrics["coverage_posture"],
		"capacity_posture": metrics["capacity_posture"],
		"runtime_posture":  metrics["runtime_posture"],
		"reliability_hint": metrics["reliability_hint"],
	}
}

// Helper methods

func (p *ConnectionPool) recordCreated() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.created++
	p.metrics.currentSize++
}

func (p *ConnectionPool) recordReuse() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.reused++
}

func (p *ConnectionPool) recordClosed() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.closed++
	p.metrics.currentSize--
}

func (p *ConnectionPool) recordError() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.errors++
}

func (p *ConnectionPool) recordInUse(conn Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inUse[conn.GetID()] = conn
}

func (p *ConnectionPool) cleanupLoop() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.cleanup()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *ConnectionPool) cleanup() {
	// Cleanup logic for idle connections
	// This is a placeholder for future implementation
}

func classifyPoolCapacityPosture(currentSize int64, maxSize int64, available int64) string {
	if maxSize == 0 {
		return "pool-unbounded"
	}
	if currentSize == 0 && available == 0 {
		return "pool-idle"
	}
	if currentSize >= maxSize && available == 0 {
		return "pool-saturated"
	}
	if available > 0 {
		return "pool-available"
	}
	return "pool-busy"
}

func classifyPoolRuntimePosture(created int64, errors int64, capacityPosture string) string {
	if errors > 0 {
		return "pool-degraded"
	}
	if created == 0 {
		return "pool-unobserved"
	}
	if capacityPosture == "pool-saturated" {
		return "pool-pressured"
	}
	return "pool-healthy"
}

func buildPoolReliabilityHint(runtimePosture string, capacityPosture string) string {
	switch runtimePosture {
	case "pool-degraded":
		return "connection pool is degraded; inspect factory failures and unhealthy connection churn"
	case "pool-pressured":
		return "connection pool is under pressure; monitor saturation and available capacity"
	case "pool-healthy":
		if capacityPosture == "pool-available" {
			return "connection pool is healthy with available capacity"
		}
		if capacityPosture == "pool-busy" {
			return "connection pool is healthy but currently busy"
		}
		return "connection pool is healthy"
	default:
		return "connection pool has not been exercised yet"
	}
}
