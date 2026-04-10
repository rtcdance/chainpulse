package deployment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// MicroserviceDeployment represents a microservice deployment mode where services run independently
type MicroserviceDeployment struct {
	config             core.Config
	registry           core.PluginRegistry
	configManager      core.ConfigManager
	eventBus           core.EventBus
	logger             core.Logger
	metricsCollector   core.MetricsCollector
	healthChecker      core.HealthChecker
	mqPlugin           core.MQPlugin
	mu                 sync.RWMutex
	isRunning          bool
	shutdownChan       chan struct{}
	shutdownWaitGroup  sync.WaitGroup
	serviceName        string
	serviceInitializer func() error
	serviceStarter     func() error
	serviceStopper     func() error
	coordinationTopic  string
	heartbeatInterval  time.Duration
	lastHeartbeat      time.Time
	instanceID         string
}

// NewMicroserviceDeployment creates a new microservice deployment
func NewMicroserviceDeployment(
	config core.Config,
	registry core.PluginRegistry,
	configManager core.ConfigManager,
	eventBus core.EventBus,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	healthChecker core.HealthChecker,
	mqPlugin core.MQPlugin,
) *MicroserviceDeployment {
	return &MicroserviceDeployment{
		config:            config,
		registry:          registry,
		configManager:     configManager,
		eventBus:          eventBus,
		logger:            logger,
		metricsCollector:  metricsCollector,
		healthChecker:     healthChecker,
		mqPlugin:          mqPlugin,
		isRunning:         false,
		shutdownChan:      make(chan struct{}),
		serviceName:       config.ServiceName,
		coordinationTopic: fmt.Sprintf("chainpulse.coordination.%s", config.ServiceName),
		heartbeatInterval: 30 * time.Second,
		instanceID:        generateInstanceID(),
	}
}

// RegisterService registers the service with lifecycle functions
func (md *MicroserviceDeployment) RegisterService(
	initializer func() error,
	starter func() error,
	stopper func() error,
) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if initializer == nil || starter == nil || stopper == nil {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			"service functions cannot be nil",
			nil,
		)
	}

	md.serviceInitializer = initializer
	md.serviceStarter = starter
	md.serviceStopper = stopper

	if md.logger != nil {
		md.logger.Info("microservice registered",
			"service", md.serviceName,
			"instance_id", md.instanceID,
		)
	}

	return nil
}

// Initialize initializes the microservice
func (md *MicroserviceDeployment) Initialize(ctx context.Context) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if md.isRunning {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"microservice is already running",
			nil,
		)
	}

	if md.serviceInitializer == nil {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			"service initializer not registered",
			nil,
		)
	}

	if md.logger != nil {
		md.logger.Info("initializing microservice",
			"service", md.serviceName,
			"instance_id", md.instanceID,
		)
	}

	if err := md.serviceInitializer(); err != nil {
		if md.logger != nil {
			md.logger.Error("failed to initialize microservice",
				"service", md.serviceName,
				"error", err.Error(),
			)
		}
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeConfigError,
			fmt.Sprintf("failed to initialize microservice %s: %v", md.serviceName, err),
			err,
		)
	}

	if md.logger != nil {
		md.logger.Info("microservice initialized",
			"service", md.serviceName,
		)
	}

	return nil
}

// Start starts the microservice
func (md *MicroserviceDeployment) Start(ctx context.Context) error {
	md.mu.Lock()
	if md.isRunning {
		md.mu.Unlock()
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"microservice is already running",
			nil,
		)
	}
	md.isRunning = true
	md.lastHeartbeat = time.Now()
	md.mu.Unlock()

	if md.logger != nil {
		md.logger.Info("starting microservice",
			"service", md.serviceName,
			"instance_id", md.instanceID,
		)
	}

	// Start heartbeat goroutine
	md.shutdownWaitGroup.Add(1)
	go md.heartbeatLoop()

	// Start service
	md.shutdownWaitGroup.Add(1)
	go func() {
		defer md.shutdownWaitGroup.Done()

		if err := md.serviceStarter(); err != nil {
			if md.logger != nil {
				md.logger.Error("microservice error",
					"service", md.serviceName,
					"error", err.Error(),
				)
			}
			// Trigger shutdown on service error
			select {
			case md.shutdownChan <- struct{}{}:
			default:
			}
		}
	}()

	if md.logger != nil {
		md.logger.Info("microservice started",
			"service", md.serviceName,
		)
	}

	// Wait for shutdown signal
	<-md.shutdownChan

	return md.Stop(ctx)
}

// Stop stops the microservice
func (md *MicroserviceDeployment) Stop(ctx context.Context) error {
	md.mu.Lock()
	if !md.isRunning {
		md.mu.Unlock()
		return nil
	}
	md.isRunning = false
	md.mu.Unlock()

	if md.logger != nil {
		md.logger.Info("stopping microservice",
			"service", md.serviceName,
		)
	}

	if md.serviceStopper != nil {
		if err := md.serviceStopper(); err != nil {
			if md.logger != nil {
				md.logger.Error("failed to stop microservice",
					"service", md.serviceName,
					"error", err.Error(),
				)
			}
		}
	}

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		md.shutdownWaitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		if md.logger != nil {
			md.logger.Info("microservice stopped",
				"service", md.serviceName,
			)
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

// IsRunning returns whether the microservice is running
func (md *MicroserviceDeployment) IsRunning() bool {
	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.isRunning
}

// GetInstanceID returns the instance ID
func (md *MicroserviceDeployment) GetInstanceID() string {
	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.instanceID
}

// GetServiceName returns the service name
func (md *MicroserviceDeployment) GetServiceName() string {
	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.serviceName
}

// Shutdown triggers graceful shutdown
func (md *MicroserviceDeployment) Shutdown() error {
	md.mu.RLock()
	isRunning := md.isRunning
	md.mu.RUnlock()

	if !isRunning {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeInternalError,
			"microservice is not running",
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

// GetHealth returns the health status of the microservice
func (md *MicroserviceDeployment) GetHealth(ctx context.Context) (core.HealthStatus, error) {
	md.mu.RLock()
	isRunning := md.isRunning
	lastHeartbeat := md.lastHeartbeat
	md.mu.RUnlock()

	if !isRunning {
		return core.HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"reason": "microservice not running",
			},
		}, nil
	}

	// Check if heartbeat is recent
	timeSinceHeartbeat := time.Since(lastHeartbeat)
	status := "healthy"
	if timeSinceHeartbeat > md.heartbeatInterval*2 {
		status = "degraded"
	}

	return core.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"service":              md.serviceName,
			"instance_id":          md.instanceID,
			"time_since_heartbeat": timeSinceHeartbeat.String(),
		},
	}, nil
}

// GetMetrics returns metrics for the microservice
func (md *MicroserviceDeployment) GetMetrics() map[string]interface{} {
	md.mu.RLock()
	defer md.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["is_running"] = md.isRunning
	metrics["service_name"] = md.serviceName
	metrics["instance_id"] = md.instanceID
	metrics["deployment_mode"] = "microservice"
	metrics["last_heartbeat"] = md.lastHeartbeat

	if md.metricsCollector != nil {
		exported := md.metricsCollector.GetMetrics()
		metrics["system_metrics"] = exported
	}

	return metrics
}

// heartbeatLoop sends periodic heartbeats to the coordination topic
func (md *MicroserviceDeployment) heartbeatLoop() {
	defer md.shutdownWaitGroup.Done()

	ticker := time.NewTicker(md.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			md.mu.Lock()
			md.lastHeartbeat = time.Now()
			md.mu.Unlock()

			// Send heartbeat to coordination topic
			if md.mqPlugin != nil {
				msgBytes := []byte(fmt.Sprintf(`{"service":"%s","instance":"%s","timestamp":%d,"status":"healthy"}`,
					md.serviceName, md.instanceID, time.Now().Unix()))

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = md.mqPlugin.Publish(ctx, md.coordinationTopic, msgBytes)
				cancel()
			}

		case <-md.shutdownChan:
			return
		}
	}
}

// WaitForShutdown waits for the microservice to shutdown
func (md *MicroserviceDeployment) WaitForShutdown() {
	md.shutdownWaitGroup.Wait()
}

// SetHeartbeatInterval sets the heartbeat interval
func (md *MicroserviceDeployment) SetHeartbeatInterval(interval time.Duration) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if interval <= 0 {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			"heartbeat interval must be positive",
			nil,
		)
	}

	md.heartbeatInterval = interval
	return nil
}

// GetHeartbeatInterval returns the heartbeat interval
func (md *MicroserviceDeployment) GetHeartbeatInterval() time.Duration {
	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.heartbeatInterval
}

// generateInstanceID generates a unique instance ID
func generateInstanceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
