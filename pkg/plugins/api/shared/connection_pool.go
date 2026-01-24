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
	created      int64
	reused       int64
	closed       int64
	errors       int64
	currentSize  int64
	maxSize      int64
	mu           sync.RWMutex
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

	// No available connections, check if we can create a new one
	p.mu.RLock()
	currentSize := int64(len(p.inUse))
	p.mu.RUnlock()

	if currentSize < int64(p.maxSize) {
		conn, err := p.factory.Create(ctx)
		if err != nil {
			p.recordError()
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}
		p.recordCreated()
		p.recordInUse(conn)
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
	defer p.metrics.mu.RUnlock()

	p.mu.RLock()
	currentSize := int64(len(p.inUse))
	p.mu.RUnlock()

	return map[string]interface{}{
		"pool_name":    p.name,
		"created":      p.metrics.created,
		"reused":       p.metrics.reused,
		"closed":       p.metrics.closed,
		"errors":       p.metrics.errors,
		"current_size": currentSize,
		"max_size":     p.metrics.maxSize,
		"available":    int64(len(p.available)),
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
