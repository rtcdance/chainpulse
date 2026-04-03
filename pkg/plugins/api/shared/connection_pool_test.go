package shared

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockConnection implements Connection interface for testing
type MockConnection struct {
	id      string
	healthy bool
	closed  bool
	mu      sync.Mutex
}

func (m *MockConnection) IsHealthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.healthy && !m.closed
}

func (m *MockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockConnection) GetID() string {
	return m.id
}

// MockConnectionFactory implements ConnectionFactory for testing
type MockConnectionFactory struct {
	count int
	mu    sync.Mutex
}

func (f *MockConnectionFactory) Create(ctx context.Context) (Connection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	return &MockConnection{
		id:      fmt.Sprintf("conn-%d", f.count),
		healthy: true,
		closed:  false,
	}, nil
}

func TestConnectionPoolAcquire(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	// Acquire a connection
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}

	if conn == nil {
		t.Fatal("acquired connection is nil")
	}

	// Release the connection
	if err := pool.Release(conn); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}
}

func TestConnectionPoolReuse(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	// Acquire and release a connection
	conn1, _ := pool.Acquire(context.Background())
	id1 := conn1.GetID()
	if err := pool.Release(conn1); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	// Acquire again - should reuse the same connection
	conn2, _ := pool.Acquire(context.Background())
	id2 := conn2.GetID()

	if id1 != id2 {
		t.Errorf("expected reused connection, got different IDs: %s vs %s", id1, id2)
	}

	if err := pool.Release(conn2); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}
}

func TestConnectionPoolMaxSize(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 2, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	// Acquire max connections
	conn1, _ := pool.Acquire(context.Background())
	conn2, _ := pool.Acquire(context.Background())

	// Try to acquire one more with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := pool.Acquire(ctx)
	if err == nil {
		t.Fatal("expected timeout error when pool is full")
	}

	if err := pool.Release(conn1); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}
	if err := pool.Release(conn2); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}
}

func TestConnectionPoolConcurrent(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 10, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := pool.Acquire(context.Background())
			if err != nil {
				t.Errorf("failed to acquire connection: %v", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
			if err := pool.Release(conn); err != nil {
				t.Errorf("failed to release connection: %v", err)
			}
		}()
	}

	wg.Wait()

	metrics := pool.GetMetrics()
	if metrics["created"] == nil {
		t.Fatal("metrics missing 'created' field")
	}
}

func TestConnectionPoolMetrics(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	conn, _ := pool.Acquire(context.Background())
	if err := pool.Release(conn); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	// Acquire again to test reuse
	conn2, _ := pool.Acquire(context.Background())
	if err := pool.Release(conn2); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	metrics := pool.GetMetrics()
	if metrics["pool_name"] != "test" {
		t.Errorf("expected pool_name 'test', got %v", metrics["pool_name"])
	}

	if metrics["created"].(int64) < 1 {
		t.Errorf("expected created >= 1, got %v", metrics["created"])
	}

	if metrics["reused"].(int64) < 1 {
		t.Errorf("expected reused >= 1, got %v", metrics["reused"])
	}
}

func TestConnectionPoolUnhealthyConnection(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	conn, _ := pool.Acquire(context.Background())
	mockConn := conn.(*MockConnection)
	mockConn.healthy = false

	if err := pool.Release(conn); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	// Next acquire should create a new connection
	conn2, _ := pool.Acquire(context.Background())
	if conn2.GetID() == conn.GetID() {
		t.Fatal("expected new connection for unhealthy connection")
	}

	if err := pool.Release(conn2); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}
}

func TestConnectionPoolClose(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)

	conn, _ := pool.Acquire(context.Background())
	if err := pool.Release(conn); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	err := pool.Close()
	if err != nil {
		t.Fatalf("failed to close pool: %v", err)
	}

	// Try to acquire after close
	_, err = pool.Acquire(context.Background())
	if err == nil {
		t.Fatal("expected error when acquiring from closed pool")
	}
}

func TestConnectionPoolRuntimeMetricsHealthy(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	if err := pool.Release(conn); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	metrics := pool.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "pool-available" {
		t.Fatalf("expected pool-available coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["capacity_posture"] != "pool-available" {
		t.Fatalf("expected pool-available capacity, got %v", metrics["capacity_posture"])
	}
	if metrics["runtime_posture"] != "pool-healthy" {
		t.Fatalf("expected pool-healthy, got %v", metrics["runtime_posture"])
	}
}

func TestConnectionPoolMetricsIncludesPostureFields(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	if err := pool.Release(conn); err != nil {
		t.Fatalf("failed to release connection: %v", err)
	}

	metrics := pool.GetMetrics()
	if metrics["coverage_posture"] != "pool-available" {
		t.Fatalf("expected pool-available coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["capacity_posture"] != "pool-available" {
		t.Fatalf("expected pool-available capacity, got %v", metrics["capacity_posture"])
	}
	if metrics["runtime_posture"] != "pool-healthy" {
		t.Fatalf("expected pool-healthy runtime, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "connection pool is healthy with available capacity" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestConnectionPoolRuntimeMetricsUnobserved(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	metrics := pool.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "pool-idle" {
		t.Fatalf("expected pool-idle coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "pool-unobserved" {
		t.Fatalf("expected pool-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestConnectionPoolRuntimeMetricsDegraded(t *testing.T) {
	factory := &MockConnectionFactory{}
	pool := NewConnectionPool("test", factory, 5, 1*time.Minute)
	defer func() {
		if err := pool.Close(); err != nil {
			t.Logf("failed to close pool: %v", err)
		}
	}()

	pool.recordError()

	metrics := pool.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "pool-idle" {
		t.Fatalf("expected pool-idle coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "pool-degraded" {
		t.Fatalf("expected pool-degraded, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "connection pool is degraded; inspect factory failures and unhealthy connection churn" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}
