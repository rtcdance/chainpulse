package api

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLoadBalancer tests load balancer initialization
func TestNewLoadBalancer(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	require.NotNil(t, lb)
	assert.Equal(t, "round-robin", lb.algorithm)
	assert.Equal(t, 0, len(lb.handlers))
}

// TestNewLoadBalancerDefaultAlgorithm tests default algorithm
func TestNewLoadBalancerDefaultAlgorithm(t *testing.T) {
	lb := NewLoadBalancer("")

	require.NotNil(t, lb)
	assert.Equal(t, "round-robin", lb.algorithm)
}

// TestAddHandler tests adding a handler
func TestAddHandler(t *testing.T) {
	lb := NewLoadBalancer("round-robin")
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	err := lb.AddHandler(handler)

	require.NoError(t, err)
	assert.Equal(t, 1, len(lb.handlers))
}

// TestAddHandlerNil tests adding nil handler
func TestAddHandlerNil(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	err := lb.AddHandler(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestAddHandlerDuplicate tests adding duplicate handler
func TestAddHandlerDuplicate(t *testing.T) {
	lb := NewLoadBalancer("round-robin")
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	err := lb.AddHandler(handler)
	require.NoError(t, err)
	err = lb.AddHandler(handler)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestRemoveHandler tests removing a handler
func TestRemoveHandler(t *testing.T) {
	lb := NewLoadBalancer("round-robin")
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	err := lb.AddHandler(handler)
	require.NoError(t, err)
	err = lb.RemoveHandler("h1")

	require.NoError(t, err)
	assert.Equal(t, 0, len(lb.handlers))
}

// TestRemoveHandlerNotFound tests removing nonexistent handler
func TestRemoveHandlerNotFound(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	err := lb.RemoveHandler("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestSelectHandlerNoHandlers tests selecting with no handlers
func TestSelectHandlerNoHandlers(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	handler, err := lb.SelectHandler()

	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "no handlers available")
}

// TestSelectHandlerRoundRobin tests round-robin selection
func TestSelectHandlerRoundRobin(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h3 := NewHandler("h3", "Handler 3", "http://localhost:8003")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)
	err = lb.AddHandler(h3)
	require.NoError(t, err)

	// Select handlers and verify round-robin distribution
	selected := make(map[string]int)
	for i := 0; i < 9; i++ {
		handler, err := lb.SelectHandler()
		require.NoError(t, err)
		selected[handler.ID]++
	}

	// Each handler should be selected 3 times
	assert.Equal(t, 3, selected["h1"])
	assert.Equal(t, 3, selected["h2"])
	assert.Equal(t, 3, selected["h3"])
}

// TestSelectHandlerWeighted tests weighted selection
func TestSelectHandlerWeighted(t *testing.T) {
	lb := NewLoadBalancer("weighted")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h1.Weight = 1

	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h2.Weight = 2

	h3 := NewHandler("h3", "Handler 3", "http://localhost:8003")
	h3.Weight = 3

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)
	err = lb.AddHandler(h3)
	require.NoError(t, err)

	// Select handlers and verify weighted distribution
	selected := make(map[string]int)
	for i := 0; i < 60; i++ {
		handler, err := lb.SelectHandler()
		require.NoError(t, err)
		selected[handler.ID]++
	}

	// Verify approximate weighted distribution
	// h1: 1/6 ≈ 10, h2: 2/6 ≈ 20, h3: 3/6 ≈ 30
	assert.Greater(t, selected["h1"], 5)
	assert.Greater(t, selected["h2"], 15)
	assert.Greater(t, selected["h3"], 25)
}

// TestSelectHandlerLeastConnections tests least connections selection
func TestSelectHandlerLeastConnections(t *testing.T) {
	lb := NewLoadBalancer("least-connections")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h3 := NewHandler("h3", "Handler 3", "http://localhost:8003")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)
	err = lb.AddHandler(h3)
	require.NoError(t, err)

	// Record requests to create different connection counts
	h1.RecordRequest(10*time.Millisecond, true)
	h1.RecordRequest(10*time.Millisecond, true)
	h1.RecordRequest(10*time.Millisecond, true)

	h2.RecordRequest(10*time.Millisecond, true)
	h2.RecordRequest(10*time.Millisecond, true)

	// h3 has no requests, should be selected
	handler, err := lb.SelectHandler()
	require.NoError(t, err)
	assert.Equal(t, "h3", handler.ID)
}

// TestSelectHandlerUnavailable tests selecting with unavailable handlers
func TestSelectHandlerUnavailable(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	// Mark h1 as unavailable
	h1.SetAvailable(false)

	// Should select h2
	handler, err := lb.SelectHandler()
	require.NoError(t, err)
	assert.Equal(t, "h2", handler.ID)
}

// TestSelectHandlerAllUnavailable tests selecting when all handlers unavailable
func TestSelectHandlerAllUnavailable(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	h1.SetAvailable(false)
	h2.SetAvailable(false)

	handler, err := lb.SelectHandler()

	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "no available handlers")
}

// TestGetMetrics tests getting load balancer metrics
func TestGetMetrics(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	// Make some selections
	for i := 0; i < 10; i++ {
		_, _ = lb.SelectHandler()
	}

	metrics := lb.GetMetrics()

	assert.Equal(t, int64(10), metrics["total_requests"])
	assert.Equal(t, "round-robin", metrics["algorithm"])
	assert.Equal(t, 2, metrics["handler_count"])
	assert.NotNil(t, metrics["distribution"])
}

// TestGetHandlers tests getting handlers list
func TestGetHandlers(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	handlers := lb.GetHandlers()

	assert.Equal(t, 2, len(handlers))
	assert.Equal(t, "h1", handlers[0].ID)
	assert.Equal(t, "h2", handlers[1].ID)
}

// TestGetAvailableHandlers tests getting available handlers
func TestGetAvailableHandlers(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h3 := NewHandler("h3", "Handler 3", "http://localhost:8003")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)
	err = lb.AddHandler(h3)
	require.NoError(t, err)

	h2.SetAvailable(false)

	available := lb.GetAvailableHandlers()

	assert.Equal(t, 2, len(available))
	assert.Equal(t, "h1", available[0].ID)
	assert.Equal(t, "h3", available[1].ID)
}

// TestSetAlgorithm tests setting algorithm
func TestSetAlgorithm(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	lb.SetAlgorithm("weighted")

	assert.Equal(t, "weighted", lb.GetAlgorithm())
}

// TestGetAlgorithm tests getting algorithm
func TestGetAlgorithm(t *testing.T) {
	lb := NewLoadBalancer("least-connections")

	algorithm := lb.GetAlgorithm()

	assert.Equal(t, "least-connections", algorithm)
}

// TestConcurrentSelection tests concurrent handler selection
func TestConcurrentSelection(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h3 := NewHandler("h3", "Handler 3", "http://localhost:8003")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)
	err = lb.AddHandler(h3)
	require.NoError(t, err)

	var wg sync.WaitGroup
	selected := make(map[string]int)
	mu := sync.Mutex{}

	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			handler, err := lb.SelectHandler()
			require.NoError(t, err)

			mu.Lock()
			selected[handler.ID]++
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Verify all handlers were selected
	assert.Greater(t, selected["h1"], 0)
	assert.Greater(t, selected["h2"], 0)
	assert.Greater(t, selected["h3"], 0)
	assert.Equal(t, 300, selected["h1"]+selected["h2"]+selected["h3"])
}

// TestConcurrentAddRemove tests concurrent add/remove operations
func TestConcurrentAddRemove(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	var wg sync.WaitGroup

	// Add handlers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := NewHandler(fmt.Sprintf("h%d", id), fmt.Sprintf("Handler %d", id), fmt.Sprintf("http://localhost:%d", 8000+id))
			_ = lb.AddHandler(handler)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 10, len(lb.GetHandlers()))

	// Remove handlers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = lb.RemoveHandler(fmt.Sprintf("h%d", id))
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, len(lb.GetHandlers()))
}

// TestDistributionMetrics tests distribution metrics calculation
func TestDistributionMetrics(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	// Make selections
	for i := 0; i < 100; i++ {
		_, _ = lb.SelectHandler()
	}

	metrics := lb.GetMetrics()
	distribution := metrics["distribution"].(map[string]float64)

	// Each handler should have ~50% distribution
	assert.Greater(t, distribution["h1"], 40.0)
	assert.Less(t, distribution["h1"], 60.0)
	assert.Greater(t, distribution["h2"], 40.0)
	assert.Less(t, distribution["h2"], 60.0)
}

// TestWeightedWithZeroWeights tests weighted algorithm with zero weights
func TestWeightedWithZeroWeights(t *testing.T) {
	lb := NewLoadBalancer("weighted")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h1.Weight = 0

	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h2.Weight = 0

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	// Should fall back to round-robin
	handler, err := lb.SelectHandler()
	require.NoError(t, err)
	assert.NotNil(t, handler)
}

// TestMultipleAlgorithms tests switching between algorithms
func TestMultipleAlgorithms(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	// Test round-robin
	lb.SetAlgorithm("round-robin")
	handler, _ := lb.SelectHandler()
	assert.NotNil(t, handler)

	// Test weighted
	lb.SetAlgorithm("weighted")
	handler, _ = lb.SelectHandler()
	assert.NotNil(t, handler)

	// Test least-connections
	lb.SetAlgorithm("least-connections")
	handler, _ = lb.SelectHandler()
	assert.NotNil(t, handler)
}

// TestHandlerMetricsRecording tests handler metrics recording
func TestHandlerMetricsRecording(t *testing.T) {
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	handler.RecordRequest(10*time.Millisecond, true)
	handler.RecordRequest(20*time.Millisecond, true)
	handler.RecordRequest(30*time.Millisecond, false)

	metrics := handler.GetMetrics()

	assert.Equal(t, int64(3), metrics.RequestCount)
	assert.Equal(t, int64(2), metrics.SuccessCount)
	assert.Equal(t, int64(1), metrics.ErrorCount)
	assert.Greater(t, metrics.AvgLatency, int64(0))
}

// TestHandlerSuccessRate tests handler success rate calculation
func TestHandlerSuccessRate(t *testing.T) {
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	handler.RecordRequest(10*time.Millisecond, true)
	handler.RecordRequest(10*time.Millisecond, true)
	handler.RecordRequest(10*time.Millisecond, false)

	rate := handler.GetSuccessRate()

	assert.Equal(t, 66.66666666666666, rate)
}

// TestHandlerAvailability tests handler availability
func TestHandlerAvailability(t *testing.T) {
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	assert.True(t, handler.IsAvailable())

	handler.SetAvailable(false)
	assert.False(t, handler.IsAvailable())

	handler.SetAvailable(true)
	assert.True(t, handler.IsAvailable())
}

// TestLoadBalancerMetricsAccuracy tests metrics accuracy
func TestLoadBalancerMetricsAccuracy(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")
	h3 := NewHandler("h3", "Handler 3", "http://localhost:8003")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)
	err = lb.AddHandler(h3)
	require.NoError(t, err)

	// Make selections
	for i := 0; i < 30; i++ {
		_, _ = lb.SelectHandler()
	}

	metrics := lb.GetMetrics()

	assert.Equal(t, int64(30), metrics["total_requests"])
	assert.Equal(t, 3, metrics["handler_count"])
}

// TestHandlerString tests handler string representation
func TestHandlerString(t *testing.T) {
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	str := handler.String()

	assert.Contains(t, str, "h1")
	assert.Contains(t, str, "Handler 1")
	assert.Contains(t, str, "http://localhost:8001")
}

// TestLoadBalancerWithSingleHandler tests load balancer with single handler
func TestLoadBalancerWithSingleHandler(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	err := lb.AddHandler(h1)
	require.NoError(t, err)

	// All selections should return the same handler
	for i := 0; i < 10; i++ {
		handler, err := lb.SelectHandler()
		require.NoError(t, err)
		assert.Equal(t, "h1", handler.ID)
	}
}

// TestHandlerErrorRecording tests handler error recording
func TestHandlerErrorRecording(t *testing.T) {
	handler := NewHandler("h1", "Handler 1", "http://localhost:8001")

	handler.RecordError("connection timeout")

	metrics := handler.GetMetrics()

	assert.Equal(t, "connection timeout", metrics.LastError)
	assert.False(t, metrics.LastErrorTime.IsZero())
}

// TestLoadBalancerMetricsReset tests metrics after handler removal
func TestLoadBalancerMetricsReset(t *testing.T) {
	lb := NewLoadBalancer("round-robin")

	h1 := NewHandler("h1", "Handler 1", "http://localhost:8001")
	h2 := NewHandler("h2", "Handler 2", "http://localhost:8002")

	err := lb.AddHandler(h1)
	require.NoError(t, err)
	err = lb.AddHandler(h2)
	require.NoError(t, err)

	// Make selections
	for i := 0; i < 20; i++ {
		_, _ = lb.SelectHandler()
	}

	metrics1 := lb.GetMetrics()
	assert.Equal(t, int64(20), metrics1["total_requests"])

	// Remove a handler
	_ = lb.RemoveHandler("h1")

	// Metrics should still reflect previous selections
	metrics2 := lb.GetMetrics()
	assert.Equal(t, int64(20), metrics2["total_requests"])
}
