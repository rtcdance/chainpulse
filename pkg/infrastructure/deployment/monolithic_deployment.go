package deployment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

// MonolithicDeployment represents a monolithic deployment mode where all services run in a single binary
type MonolithicDeployment struct {
	config              core.Config
	registry            core.PluginRegistry
	configManager       core.ConfigManager
	eventBus            core.EventBus
	logger              core.Logger
	metricsCollector    core.MetricsCollector
	healthChecker       core.HealthChecker
	mu                  sync.RWMutex
	isRunning           bool
	shutdownChan        chan struct{}
	shutdownWaitGroup   sync.WaitGroup
	serviceInitializers map[string]func() error
	serviceStarters     map[string]func() error
	serviceStoppers     map[string]func() error
}

// NewMonolithicDeployment creates a new monolithic deployment
func NewMonolithicDeployment(
	config core.Config,
	registry core.PluginRegistry,
	configManager core.ConfigManager,
	eventBus core.EventBus,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	healthChecker core.HealthChecker,
) *MonolithicDeployment {
	return &MonolithicDeployment{
		config:              config,
		registry:            registry,
		configManager:       configManager,
		eventBus:            eventBus,
		logger:              logger,
		metricsCollector:    metricsCollector,
		healthChecker:       healthChecker,
		isRunning:           false,
		shutdownChan:        make(chan struct{}),
		serviceInitializers: make(map[string]func() error),
		serviceStarters:     make(map[string]func() error),
		serviceStoppers:     make(map[string]func() error),
	}
}

// RegisterService registers a service with initializer, starter, and stopper functions
func (md *MonolithicDeployment) RegisterService(
	name string,
	initializer func() error,
	starter func() error,
	stopper func() error,
) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if name == "" {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			"service name cannot be empty",
			nil,
		)
	}

	if initializer == nil || starter == nil || stopper == nil {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			"service functions cannot be nil",
			nil,
		)
	}

	md.serviceInitializers[name] = initializer
	md.serviceStarters[name] = starter
	md.serviceStoppers[name] = stopper

	if md.logger != nil {
		md.logger.Info(
			"service registered",
			"service", name,
		)
	}

	return nil
}

// Initialize initializes all registered services
func (md *MonolithicDeployment) Initialize(ctx context.Context) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if md.isRunning {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"deployment is already running",
			nil,
		)
	}

	if md.logger != nil {
		md.logger.Info(
			"initializing monolithic deployment",
			"service_count", len(md.serviceInitializers),
		)
	}

	// Initialize all services
	for name, initializer := range md.serviceInitializers {
		if err := initializer(); err != nil {
			if md.logger != nil {
				md.logger.Error(
					"failed to initialize service",
					"service", name,
					"error", err.Error(),
				)
			}
			return core.NewSystemError(
				core.ErrorTypePermanent,
				core.ErrorCodeConfigError,
				fmt.Sprintf("failed to initialize service %s: %v", name, err),
				err,
			)
		}

		if md.logger != nil {
			md.logger.Info(
				"service initialized",
				"service", name,
			)
		}
	}

	return nil
}

// Start starts all registered services
func (md *MonolithicDeployment) Start(ctx context.Context) error {
	md.mu.Lock()
	if md.isRunning {
		md.mu.Unlock()
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"deployment is already running",
			nil,
		)
	}
	md.isRunning = true
	md.mu.Unlock()

	if md.logger != nil {
		md.logger.Info(
			"starting monolithic deployment",
			"service_count", len(md.serviceStarters),
		)
	}

	// Start all services
	for name, starter := range md.serviceStarters {
		md.shutdownWaitGroup.Add(1)
		go func(serviceName string, startFunc func() error) {
			defer md.shutdownWaitGroup.Done()

			if err := startFunc(); err != nil {
				if md.logger != nil {
					md.logger.Error(
						"service error",
						"service", serviceName,
						"error", err.Error(),
					)
				}
				// Trigger shutdown on service error
				select {
				case md.shutdownChan <- struct{}{}:
				default:
				}
			}
		}(name, starter)

		if md.logger != nil {
			md.logger.Info(
				"service started",
				"service", name,
			)
		}
	}

	// Wait for shutdown signal
	<-md.shutdownChan

	return md.Stop(ctx)
}

// Stop stops all registered services
func (md *MonolithicDeployment) Stop(ctx context.Context) error {
	md.mu.Lock()
	if !md.isRunning {
		md.mu.Unlock()
		return nil
	}
	md.isRunning = false
	md.mu.Unlock()

	if md.logger != nil {
		md.logger.Info(
			"stopping monolithic deployment",
			"service_count", len(md.serviceStoppers),
		)
	}

	// Stop all services in reverse order
	serviceNames := make([]string, 0, len(md.serviceStoppers))
	for name := range md.serviceStoppers {
		serviceNames = append(serviceNames, name)
	}

	// Reverse order for graceful shutdown
	for i := len(serviceNames) - 1; i >= 0; i-- {
		name := serviceNames[i]
		stopper := md.serviceStoppers[name]

		if err := stopper(); err != nil {
			if md.logger != nil {
				md.logger.Error(
					"failed to stop service",
					"service", name,
					"error", err.Error(),
				)
			}
		} else {
			if md.logger != nil {
				md.logger.Info(
					"service stopped",
					"service", name,
				)
			}
		}
	}

	// Wait for all services to finish
	done := make(chan struct{})
	go func() {
		md.shutdownWaitGroup.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		if md.logger != nil {
			md.logger.Info("monolithic deployment stopped")
		}
		return nil
	case <-ctx.Done():
		return core.NewSystemError(
			core.ErrorTypeTransient,
			core.ErrorCodeTimeout,
			"shutdown timeout exceeded",
			ctx.Err(),
		)
	}
}

// IsRunning returns whether the deployment is running
func (md *MonolithicDeployment) IsRunning() bool {
	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.isRunning
}

// GetServiceCount returns the number of registered services
func (md *MonolithicDeployment) GetServiceCount() int {
	md.mu.RLock()
	defer md.mu.RUnlock()

	return len(md.serviceInitializers)
}

// GetServices returns the names of all registered services
func (md *MonolithicDeployment) GetServices() []string {
	md.mu.RLock()
	defer md.mu.RUnlock()

	services := make([]string, 0, len(md.serviceInitializers))
	for name := range md.serviceInitializers {
		services = append(services, name)
	}
	return services
}

// Shutdown triggers graceful shutdown
func (md *MonolithicDeployment) Shutdown() error {
	md.mu.RLock()
	isRunning := md.isRunning
	md.mu.RUnlock()

	if !isRunning {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"deployment is not running",
			nil,
		)
	}

	select {
	case md.shutdownChan <- struct{}{}:
		return nil
	default:
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"shutdown already in progress",
			nil,
		)
	}
}

// GetHealth returns the health status of the deployment
func (md *MonolithicDeployment) GetHealth(ctx context.Context) (core.HealthStatus, error) {
	md.mu.RLock()
	isRunning := md.isRunning
	md.mu.RUnlock()

	if !isRunning {
		return core.HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now(),
			Details: map[string]any{
				"reason": "deployment not running",
			},
		}, nil
	}

	// Check health of all services through registry
	if md.registry == nil {
		return core.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
			Details: map[string]any{
				"service_count": md.GetServiceCount(),
			},
		}, nil
	}

	plugins := md.registry.List()
	healthyCount := 0
	unhealthyServices := make([]string, 0)

	for _, plugin := range plugins {
		if hp, ok := plugin.(ports.HealthPlugin); ok {
			if err := hp.Health(context.Background()); err != nil {
				unhealthyServices = append(unhealthyServices, plugin.Name())
			} else {
				healthyCount++
			}
		}
	}

	status := "healthy"
	if healthyCount < len(plugins) {
		status = "degraded"
	}

	return core.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Details: map[string]any{
			"service_count":      md.GetServiceCount(),
			"healthy_count":      healthyCount,
			"unhealthy_services": unhealthyServices,
		},
	}, nil
}

// GetMetrics returns metrics for the deployment
func (md *MonolithicDeployment) GetMetrics() map[string]any {
	md.mu.RLock()
	defer md.mu.RUnlock()

	metrics := make(map[string]any)
	metrics["is_running"] = md.isRunning
	metrics["service_count"] = len(md.serviceInitializers)
	metrics["deployment_mode"] = "monolithic"

	if md.metricsCollector != nil {
		exported := md.metricsCollector.GetMetrics()
		metrics["system_metrics"] = exported
	}

	return metrics
}

// WaitForShutdown waits for the deployment to shutdown
func (md *MonolithicDeployment) WaitForShutdown() {
	md.shutdownWaitGroup.Wait()
}
