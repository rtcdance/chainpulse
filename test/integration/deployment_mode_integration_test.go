package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/infrastructure/deployment"
)

// Integration test for deployment mode support
func TestDeploymentMode_MonolithicInitialization(t *testing.T) {
	config := &deployment.DeploymentConfig{
		Mode:                    deployment.MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	manager := deployment.NewDeploymentModeManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify initialization
	if !manager.IsInitialized() {
		t.Error("Manager should be initialized")
	}

	if manager.GetMode() != deployment.MonolithicMode {
		t.Errorf("Expected mode %s, got %s", deployment.MonolithicMode, manager.GetMode())
	}

	// Verify feature parity
	err = manager.ValidateFeatureParity(ctx)
	if err != nil {
		t.Fatalf("ValidateFeatureParity failed: %v", err)
	}

	// Shutdown
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if manager.IsInitialized() {
		t.Error("Manager should not be initialized after shutdown")
	}
}

// Integration test for microservice mode
func TestDeploymentMode_MicroserviceInitialization(t *testing.T) {
	config := &deployment.DeploymentConfig{
		Mode:                    deployment.MicroserviceMode,
		ServiceName:             "test-service",
		EnableServiceRegistry:   true,
		EnableDistributedCache:  true,
		EnableMessageQueue:      true,
		GracefulShutdownTimeout: 5 * time.Second,
	}

	manager := deployment.NewDeploymentModeManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify initialization
	if !manager.IsInitialized() {
		t.Error("Manager should be initialized")
	}

	if manager.GetMode() != deployment.MicroserviceMode {
		t.Errorf("Expected mode %s, got %s", deployment.MicroserviceMode, manager.GetMode())
	}

	// Shutdown
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// Integration test for monolithic initializer
func TestMonolithicInitializer_FullLifecycle(t *testing.T) {
	config := &deployment.DeploymentConfig{
		Mode:                    deployment.MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := deployment.NewMonolithicInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify metrics
	metrics := initializer.GetMetrics()
	if metrics["components_ready"] == 0 {
		t.Error("Expected components to be ready")
	}

	// Health check
	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	// Verify health metrics
	metrics = initializer.GetMetrics()
	if metrics["health_checks_passed"] == 0 {
		t.Error("Expected health checks to pass")
	}

	// Shutdown
	err = initializer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// Integration test for microservice initializer
func TestMicroserviceInitializer_FullLifecycle(t *testing.T) {
	config := &deployment.DeploymentConfig{
		Mode:                    deployment.MicroserviceMode,
		ServiceName:             "test-service",
		EnableServiceRegistry:   true,
		EnableDistributedCache:  true,
		EnableMessageQueue:      true,
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := deployment.NewMicroserviceInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify metrics
	metrics := initializer.GetMetrics()
	if metrics["services_registered"] == 0 {
		t.Error("Expected services to be registered")
	}

	// Health check
	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	// Get registered services
	services := initializer.GetRegisteredServices()
	if len(services) == 0 {
		t.Error("Expected registered services")
	}

	// Shutdown
	err = initializer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify services are deregistered
	services = initializer.GetRegisteredServices()
	if len(services) != 0 {
		t.Error("Expected no services after shutdown")
	}
}

// Integration test for feature parity
func TestDeploymentMode_FeatureParity(t *testing.T) {
	monolithicConfig := &deployment.DeploymentConfig{
		Mode:                    deployment.MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	microserviceConfig := &deployment.DeploymentConfig{
		Mode:                    deployment.MicroserviceMode,
		ServiceName:             "test-service",
		EnableServiceRegistry:   true,
		EnableDistributedCache:  true,
		EnableMessageQueue:      true,
		GracefulShutdownTimeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize both modes
	monolithicManager := deployment.NewDeploymentModeManager(monolithicConfig)
	err := monolithicManager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Monolithic initialize failed: %v", err)
	}

	microserviceManager := deployment.NewDeploymentModeManager(microserviceConfig)
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

	// Both should have components/services
	monolithicStatus := monolithicManager.GetComponentStatus()
	if len(monolithicStatus) == 0 {
		t.Error("Monolithic should have components")
	}

	// Cleanup
	_ = monolithicManager.Shutdown(ctx)
	_ = microserviceManager.Shutdown(ctx)
}

// Integration test for graceful shutdown
func TestDeploymentMode_GracefulShutdown(t *testing.T) {
	config := &deployment.DeploymentConfig{
		Mode:                    deployment.MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	manager := deployment.NewDeploymentModeManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// First shutdown should succeed
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("First shutdown failed: %v", err)
	}

	// Second shutdown should fail (not initialized)
	err = manager.Shutdown(ctx)
	if err == nil {
		t.Error("Second shutdown should fail (not initialized)")
	}
}
