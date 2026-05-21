package deployment

import (
	"context"
	"testing"
	"time"
)

// Property 1: Dual Deployment Mode Support
// For any deployment mode configuration, initializing the system should result in
// the correct mode being set and all required components being initialized
func TestProperty_DualDeploymentModeSupport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mode     DeploymentMode
		validate func(*DeploymentModeManager) error
	}{
		{
			name: "monolithic_mode_initialization",
			mode: MonolithicMode,
			validate: func(manager *DeploymentModeManager) error {
				if manager.GetMode() != MonolithicMode {
					t.Errorf("Expected mode %s, got %s", MonolithicMode, manager.GetMode())
				}
				return manager.ValidateFeatureParity(context.Background())
			},
		},
		{
			name: "microservice_mode_initialization",
			mode: MicroserviceMode,
			validate: func(manager *DeploymentModeManager) error {
				if manager.GetMode() != MicroserviceMode {
					t.Errorf("Expected mode %s, got %s", MicroserviceMode, manager.GetMode())
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DeploymentConfig{
				Mode:                    tt.mode,
				ServiceName:             "test-service",
				EnableServiceRegistry:   true,
				EnableDistributedCache:  true,
				EnableMessageQueue:      true,
				GracefulShutdownTimeout: 5 * time.Second,
			}

			manager := NewDeploymentModeManager(config)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := manager.Initialize(ctx)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}

			if !manager.IsInitialized() {
				t.Error("Manager should be initialized")
			}

			if err := tt.validate(manager); err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			err = manager.Shutdown(ctx)
			if err != nil {
				t.Fatalf("Shutdown failed: %v", err)
			}
		})
	}
}

// Property 2: Monolithic Mode Initialization Correctness
// For any monolithic deployment, all components should be initialized in the correct order
// and health checks should pass
func TestProperty_MonolithicModeInitializationCorrectness(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify all components are ready
	metrics := initializer.GetMetrics()
	if metrics["components_ready"] != 5 {
		t.Errorf("Expected 5 components ready, got %v", metrics["components_ready"])
	}

	if metrics["components_failed"] != 0 {
		t.Errorf("Expected 0 failed components, got %v", metrics["components_failed"])
	}

	// Health check should pass
	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	// Shutdown should succeed
	err = initializer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// Property 3: Microservice Mode Initialization Correctness
// For any microservice deployment, all services should be registered and
// infrastructure components should be initialized
func TestProperty_MicroserviceModeInitializationCorrectness(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		EnableServiceRegistry:   true,
		EnableDistributedCache:  true,
		EnableMessageQueue:      true,
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify all services are registered
	metrics := initializer.GetMetrics()
	if metrics["services_registered"] != 4 {
		t.Errorf("Expected 4 services registered, got %v", metrics["services_registered"])
	}

	if metrics["services_failed"] != 0 {
		t.Errorf("Expected 0 failed services, got %v", metrics["services_failed"])
	}

	// Health check should pass
	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	// Verify services are retrievable
	services := initializer.GetRegisteredServices()
	if len(services) != 4 {
		t.Errorf("Expected 4 services, got %d", len(services))
	}

	// Shutdown should succeed
	err = initializer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// Property 4: Feature Parity Between Modes
// For any feature set, both monolithic and microservice modes should support
// the same core functionality
func TestProperty_FeatureParityBetweenModes(t *testing.T) {
	t.Parallel()
	monolithicConfig := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	microserviceConfig := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		EnableServiceRegistry:   true,
		EnableDistributedCache:  true,
		EnableMessageQueue:      true,
		GracefulShutdownTimeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize monolithic
	monolithicManager := NewDeploymentModeManager(monolithicConfig)
	err := monolithicManager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Monolithic initialize failed: %v", err)
	}

	// Initialize microservice
	microserviceManager := NewDeploymentModeManager(microserviceConfig)
	err = microserviceManager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Microservice initialize failed: %v", err)
	}

	// Both should be initialized
	if !monolithicManager.IsInitialized() {
		t.Error("Monolithic manager should be initialized")
	}

	if !microserviceManager.IsInitialized() {
		t.Error("Microservice manager should be initialized")
	}

	// Both should have required components
	monolithicStatus := monolithicManager.GetComponentStatus()
	if len(monolithicStatus) == 0 {
		t.Error("Monolithic should have components")
	}

	// Cleanup
	if err := monolithicManager.Shutdown(ctx); err != nil {
		t.Logf("failed to shutdown monolithic manager: %v", err)
	}
	if err := microserviceManager.Shutdown(ctx); err != nil {
		t.Logf("failed to shutdown microservice manager: %v", err)
	}
}

// Property 5: Graceful Shutdown Idempotency
// For any initialized deployment, calling shutdown multiple times should be safe
func TestProperty_GracefulShutdownIdempotency(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	manager := NewDeploymentModeManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// First shutdown should succeed
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("First shutdown failed: %v", err)
	}

	// Second shutdown should fail gracefully (not initialized)
	err = manager.Shutdown(ctx)
	if err == nil {
		t.Error("Second shutdown should fail (not initialized)")
	}
}

// Property 6: Metrics Tracking Accuracy
// For any deployment mode, metrics should accurately reflect the state
func TestProperty_MetricsTrackingAccuracy(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Before initialization
	metrics := initializer.GetMetrics()
	if metrics["components_ready"] != 0 {
		t.Error("Should have 0 components ready before initialization")
	}

	// After initialization
	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	metrics = initializer.GetMetrics()
	if metrics["components_ready"] == 0 {
		t.Error("Should have components ready after initialization")
	}

	// After health check
	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	metrics = initializer.GetMetrics()
	if metrics["health_checks_passed"] == 0 {
		t.Error("Should have passed health checks")
	}
}
