package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/infrastructure/discovery"
)

// ServiceRegistryInterface defines the interface for service registry operations
type ServiceRegistryInterface interface {
	RegisterService(ctx context.Context, service *discovery.ServiceInfo) error
	DeregisterService(ctx context.Context, serviceID string) error
	UpdateServiceStatus(ctx context.Context, serviceID, status string) error
	GetService(ctx context.Context, serviceID string) (*discovery.ServiceInfo, error)
	DiscoverService(ctx context.Context, serviceName string) ([]*discovery.ServiceInfo, error)
}

// HealthCheckResult represents the result of a health check.
//
// Renaming would break many external uses.
type HealthCheckResult struct {
	ServiceID    string
	ServiceName  string
	Healthy      bool
	Message      string
	Timestamp    time.Time
	ResponseTime time.Duration
}

// HealthCheckEndpoint represents a health check endpoint.
//
// Renaming would break many external uses.
type HealthCheckEndpoint struct {
	ServiceID string
	URL       string
	Interval  time.Duration
	Timeout   time.Duration
}

// HealthCheckSystem manages health checks for all services.
//
// Renaming would break many external uses.
type HealthCheckSystem struct {
	registry  ServiceRegistryInterface
	endpoints map[string]*HealthCheckEndpoint
	results   map[string]*HealthCheckResult
	mutex     sync.RWMutex
	running   bool
	wg        sync.WaitGroup
	stopCh    chan struct{}
}

// NewHealthCheckSystem creates a new health check system
func NewHealthCheckSystem(registry ServiceRegistryInterface) *HealthCheckSystem {
	return &HealthCheckSystem{
		registry:  registry,
		endpoints: make(map[string]*HealthCheckEndpoint),
		results:   make(map[string]*HealthCheckResult),
	}
}

// RegisterHealthCheckEndpoint registers a health check endpoint
func (hcs *HealthCheckSystem) RegisterHealthCheckEndpoint(ctx context.Context, endpoint HealthCheckEndpoint) error {
	hcs.mutex.Lock()
	defer hcs.mutex.Unlock()

	hcs.endpoints[endpoint.ServiceID] = &endpoint

	return nil
}

// Start starts the health check system
func (hcs *HealthCheckSystem) Start(ctx context.Context) error {
	hcs.mutex.Lock()
	if hcs.running {
		hcs.mutex.Unlock()
		return fmt.Errorf("health check system already running")
	}
	hcs.running = true
	hcs.stopCh = make(chan struct{})
	hcs.mutex.Unlock()

	hcs.wg.Add(1)
	go hcs.checkLoop(ctx)
	return nil
}

// Stop stops the health check system and waits for the check loop to exit
func (hcs *HealthCheckSystem) Stop() {
	hcs.mutex.Lock()
	if !hcs.running {
		hcs.mutex.Unlock()
		return
	}
	hcs.running = false
	close(hcs.stopCh)
	hcs.mutex.Unlock()

	hcs.wg.Wait()
}

// checkLoop performs periodic health checks
func (hcs *HealthCheckSystem) checkLoop(ctx context.Context) {
	defer hcs.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Run once immediately
	hcs.performAllHealthChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hcs.stopCh:
			return
		case <-ticker.C:
			hcs.performAllHealthChecks(ctx)
		}
	}
}

// performAllHealthChecks performs health checks for all services
func (hcs *HealthCheckSystem) performAllHealthChecks(ctx context.Context) {
	hcs.mutex.RLock()
	endpoints := make(map[string]*HealthCheckEndpoint)
	for k, v := range hcs.endpoints {
		endpoints[k] = v
	}
	hcs.mutex.RUnlock()

	for serviceID, endpoint := range endpoints {
		hcs.wg.Add(1)
		go func(sid string, ep *HealthCheckEndpoint) {
			defer hcs.wg.Done()
			hcs.performHealthCheck(ctx, sid, ep)
		}(serviceID, endpoint)
	}
}

// performHealthCheck performs a health check for a single service
func (hcs *HealthCheckSystem) performHealthCheck(ctx context.Context, serviceID string, endpoint *HealthCheckEndpoint) {
	checkCtx, cancel := context.WithTimeout(ctx, endpoint.Timeout)
	defer cancel()

	start := time.Now()
	healthy := hcs.checkEndpoint(checkCtx, endpoint.URL)
	duration := time.Since(start)

	result := &HealthCheckResult{
		ServiceID:    serviceID,
		Healthy:      healthy,
		Timestamp:    time.Now(),
		ResponseTime: duration,
	}

	if healthy {
		result.Message = "Service is healthy"
	} else {
		result.Message = "Service is unhealthy"
	}

	hcs.mutex.Lock()
	hcs.results[serviceID] = result
	hcs.mutex.Unlock()

	// Update service status in registry
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}
	_ = hcs.registry.UpdateServiceStatus(ctx, serviceID, status)

	// Deregister if unhealthy for too long
	if !healthy {
		hcs.handleUnhealthyService(ctx, serviceID)
	}
}

// checkEndpoint checks if an endpoint is healthy by making an HTTP GET
// request to its /health/live endpoint with a 5-second timeout.
func (hcs *HealthCheckSystem) checkEndpoint(ctx context.Context, url string) bool {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, url+"/health/live", nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close() //nolint:errcheck // defer close

	return resp.StatusCode == http.StatusOK
}

// handleUnhealthyService handles an unhealthy service
func (hcs *HealthCheckSystem) handleUnhealthyService(ctx context.Context, serviceID string) {
	// Check if service has been unhealthy for too long
	hcs.mutex.RLock()
	result, exists := hcs.results[serviceID]
	hcs.mutex.RUnlock()

	if !exists {
		return
	}

	// If unhealthy for more than 30 seconds, deregister
	if time.Since(result.Timestamp) > 30*time.Second {
		_ = hcs.registry.DeregisterService(ctx, serviceID)
	}
}

// GetHealthCheckResult retrieves the latest health check result for a service
func (hcs *HealthCheckSystem) GetHealthCheckResult(serviceID string) (*HealthCheckResult, error) {
	hcs.mutex.RLock()
	defer hcs.mutex.RUnlock()

	result, exists := hcs.results[serviceID]
	if !exists {
		return nil, fmt.Errorf("no health check result found for service: %s", serviceID)
	}

	return result, nil
}

// GetAllHealthCheckResults retrieves all health check results
func (hcs *HealthCheckSystem) GetAllHealthCheckResults() map[string]*HealthCheckResult {
	hcs.mutex.RLock()
	defer hcs.mutex.RUnlock()

	results := make(map[string]*HealthCheckResult)
	for k, v := range hcs.results {
		results[k] = v
	}

	return results
}

// FailureDetector detects service failures
type FailureDetector struct {
	healthCheckSystem *HealthCheckSystem
	failureThreshold  int
	failureCount      map[string]int
	mutex             sync.RWMutex
}

// NewFailureDetector creates a new failure detector
func NewFailureDetector(healthCheckSystem *HealthCheckSystem, failureThreshold int) *FailureDetector {
	return &FailureDetector{
		healthCheckSystem: healthCheckSystem,
		failureThreshold:  failureThreshold,
		failureCount:      make(map[string]int),
	}
}

// DetectFailures detects service failures
func (fd *FailureDetector) DetectFailures(ctx context.Context) []string {
	results := fd.healthCheckSystem.GetAllHealthCheckResults()

	var failedServices []string

	for serviceID, result := range results {
		if !result.Healthy {
			fd.mutex.Lock()
			fd.failureCount[serviceID]++
			count := fd.failureCount[serviceID]
			fd.mutex.Unlock()

			if count >= fd.failureThreshold {
				failedServices = append(failedServices, serviceID)
			}
		} else {
			fd.mutex.Lock()
			fd.failureCount[serviceID] = 0
			fd.mutex.Unlock()
		}
	}

	return failedServices
}

// ResetFailureCount resets the failure count for a service
func (fd *FailureDetector) ResetFailureCount(serviceID string) {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()
	fd.failureCount[serviceID] = 0
}

// AutomaticDeregistration automatically deregisters unhealthy services
type AutomaticDeregistration struct {
	registry          ServiceRegistryInterface
	failureDetector   *FailureDetector
	deregistrationTTL time.Duration
	mutex             sync.RWMutex
	running           bool
	wg                sync.WaitGroup
	stopCh            chan struct{}
}

// NewAutomaticDeregistration creates a new automatic deregistration system
func NewAutomaticDeregistration(registry ServiceRegistryInterface, failureDetector *FailureDetector, ttl time.Duration) *AutomaticDeregistration {
	return &AutomaticDeregistration{
		registry:          registry,
		failureDetector:   failureDetector,
		deregistrationTTL: ttl,
	}
}

// Start starts the automatic deregistration system
func (ad *AutomaticDeregistration) Start(ctx context.Context) error {
	ad.mutex.Lock()
	if ad.running {
		ad.mutex.Unlock()
		return fmt.Errorf("automatic deregistration already running")
	}
	ad.running = true
	ad.stopCh = make(chan struct{})
	ad.mutex.Unlock()

	ad.wg.Add(1)
	go ad.deregistrationLoop(ctx)
	return nil
}

// Stop stops the automatic deregistration system and waits for the loop to exit
func (ad *AutomaticDeregistration) Stop() {
	ad.mutex.Lock()
	if !ad.running {
		ad.mutex.Unlock()
		return
	}
	ad.running = false
	close(ad.stopCh)
	ad.mutex.Unlock()

	ad.wg.Wait()
}

// deregistrationLoop performs periodic deregistration checks
func (ad *AutomaticDeregistration) deregistrationLoop(ctx context.Context) {
	defer ad.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ad.stopCh:
			return
		case <-ticker.C:
			failedServices := ad.failureDetector.DetectFailures(ctx)
			for _, serviceID := range failedServices {
				_ = ad.registry.DeregisterService(ctx, serviceID)
			}
		}
	}
}
