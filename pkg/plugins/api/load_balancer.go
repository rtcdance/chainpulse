package api

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// LoadBalancer distributes requests across handlers
type LoadBalancer struct {
	handlers      []*RequestHandler
	currentIndex  int64
	mu            sync.RWMutex
	algorithm     string
	metrics       *LoadBalancerMetrics
}

// LoadBalancerMetrics represents load balancer metrics
type LoadBalancerMetrics struct {
	TotalRequests    int64
	DistributedCount map[string]int64
	mu               sync.RWMutex
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(algorithm string) *LoadBalancer {
	if algorithm == "" {
		algorithm = "round-robin"
	}

	return &LoadBalancer{
		handlers:     make([]*RequestHandler, 0),
		currentIndex: 0,
		algorithm:    algorithm,
		metrics: &LoadBalancerMetrics{
			DistributedCount: make(map[string]int64),
		},
	}
}

// AddHandler adds a handler to the load balancer
func (lb *LoadBalancer) AddHandler(handler *RequestHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Check if handler already exists
	for _, h := range lb.handlers {
		if h.ID == handler.ID {
			return fmt.Errorf("handler %s already exists", handler.ID)
		}
	}

	lb.handlers = append(lb.handlers, handler)
	return nil
}

// RemoveHandler removes a handler from the load balancer
func (lb *LoadBalancer) RemoveHandler(handlerID string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, h := range lb.handlers {
		if h.ID == handlerID {
			lb.handlers = append(lb.handlers[:i], lb.handlers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("handler %s not found", handlerID)
}

// SelectHandler selects a handler using the configured algorithm
func (lb *LoadBalancer) SelectHandler() (*RequestHandler, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.handlers) == 0 {
		return nil, fmt.Errorf("no handlers available")
	}

	switch lb.algorithm {
	case "round-robin":
		return lb.selectRoundRobin()
	case "weighted":
		return lb.selectWeighted()
	case "least-connections":
		return lb.selectLeastConnections()
	default:
		return lb.selectRoundRobin()
	}
}

// selectRoundRobin selects a handler using round-robin algorithm
func (lb *LoadBalancer) selectRoundRobin() (*RequestHandler, error) {
	// Find available handlers
	availableHandlers := make([]*RequestHandler, 0)
	for _, h := range lb.handlers {
		if h.IsAvailable() {
			availableHandlers = append(availableHandlers, h)
		}
	}

	if len(availableHandlers) == 0 {
		return nil, fmt.Errorf("no available handlers")
	}

	// Select handler using round-robin
	index := atomic.AddInt64(&lb.currentIndex, 1) % int64(len(availableHandlers))
	handler := availableHandlers[index]

	// Record distribution
	lb.recordDistribution(handler.ID)

	return handler, nil
}

// selectWeighted selects a handler using weighted algorithm
func (lb *LoadBalancer) selectWeighted() (*RequestHandler, error) {
	// Find available handlers
	availableHandlers := make([]*RequestHandler, 0)
	totalWeight := 0

	for _, h := range lb.handlers {
		if h.IsAvailable() {
			availableHandlers = append(availableHandlers, h)
			totalWeight += h.Weight
		}
	}

	if len(availableHandlers) == 0 {
		return nil, fmt.Errorf("no available handlers")
	}

	if totalWeight == 0 {
		// If all weights are 0, use round-robin
		index := atomic.AddInt64(&lb.currentIndex, 1) % int64(len(availableHandlers))
		handler := availableHandlers[index]
		lb.recordDistribution(handler.ID)
		return handler, nil
	}

	// Select handler based on weight
	index := atomic.AddInt64(&lb.currentIndex, 1) % int64(totalWeight)
	currentWeight := int64(0)

	for _, h := range availableHandlers {
		currentWeight += int64(h.Weight)
		if index < currentWeight {
			lb.recordDistribution(h.ID)
			return h, nil
		}
	}

	// Fallback to first available handler
	handler := availableHandlers[0]
	lb.recordDistribution(handler.ID)
	return handler, nil
}

// selectLeastConnections selects a handler with least connections
func (lb *LoadBalancer) selectLeastConnections() (*RequestHandler, error) {
	// Find available handlers
	availableHandlers := make([]*RequestHandler, 0)
	for _, h := range lb.handlers {
		if h.IsAvailable() {
			availableHandlers = append(availableHandlers, h)
		}
	}

	if len(availableHandlers) == 0 {
		return nil, fmt.Errorf("no available handlers")
	}

	// Find handler with least requests
	var selectedHandler *RequestHandler
	minRequests := int64(^uint64(0) >> 1) // Max int64

	for _, h := range availableHandlers {
		metrics := h.GetMetrics()
		if metrics.RequestCount < minRequests {
			minRequests = metrics.RequestCount
			selectedHandler = h
		}
	}

	if selectedHandler == nil {
		selectedHandler = availableHandlers[0]
	}

	lb.recordDistribution(selectedHandler.ID)
	return selectedHandler, nil
}

// recordDistribution records handler distribution
func (lb *LoadBalancer) recordDistribution(handlerID string) {
	lb.metrics.mu.Lock()
	defer lb.metrics.mu.Unlock()

	atomic.AddInt64(&lb.metrics.TotalRequests, 1)
	lb.metrics.DistributedCount[handlerID]++
}

// GetMetrics returns load balancer metrics
func (lb *LoadBalancer) GetMetrics() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	lb.metrics.mu.RLock()
	defer lb.metrics.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["total_requests"] = atomic.LoadInt64(&lb.metrics.TotalRequests)
	metrics["algorithm"] = lb.algorithm
	metrics["handler_count"] = len(lb.handlers)

	// Calculate distribution percentages
	distribution := make(map[string]float64)
	totalRequests := atomic.LoadInt64(&lb.metrics.TotalRequests)
	if totalRequests > 0 {
		for handlerID, count := range lb.metrics.DistributedCount {
			distribution[handlerID] = (float64(count) / float64(totalRequests)) * 100.0
		}
	}
	metrics["distribution"] = distribution

	return metrics
}

// GetHandlers returns a copy of the handlers list
func (lb *LoadBalancer) GetHandlers() []*RequestHandler {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	handlers := make([]*RequestHandler, len(lb.handlers))
	copy(handlers, lb.handlers)
	return handlers
}

// GetAvailableHandlers returns available handlers
func (lb *LoadBalancer) GetAvailableHandlers() []*RequestHandler {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	availableHandlers := make([]*RequestHandler, 0)
	for _, h := range lb.handlers {
		if h.IsAvailable() {
			availableHandlers = append(availableHandlers, h)
		}
	}

	return availableHandlers
}

// SetAlgorithm sets the load balancing algorithm
func (lb *LoadBalancer) SetAlgorithm(algorithm string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.algorithm = algorithm
}

// GetAlgorithm returns the current algorithm
func (lb *LoadBalancer) GetAlgorithm() string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.algorithm
}
