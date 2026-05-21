package deployment

import (
	"context"
	"testing"
	"time"
)

func TestDeploymentModeManager_Initialize_Monolithic(t *testing.T) {
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

	if !manager.IsInitialized() {
		t.Error("Manager should be initialized")
	}

	if manager.GetMode() != MonolithicMode {
		t.Errorf("Expected mode %s, got %s", MonolithicMode, manager.GetMode())
	}

	metrics := manager.GetMetrics()
	if metrics["components_initialized"] == 0 {
		t.Error("Expected components to be initialized")
	}
}

func TestDeploymentModeManager_Initialize_Microservice(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
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

	if manager.GetMode() != MicroserviceMode {
		t.Errorf("Expected mode %s, got %s", MicroserviceMode, manager.GetMode())
	}
}

func TestDeploymentModeManager_ValidateFeatureParity(t *testing.T) {
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

	err = manager.ValidateFeatureParity(ctx)
	if err != nil {
		t.Fatalf("ValidateFeatureParity failed: %v", err)
	}
}

func TestDeploymentModeManager_Shutdown(t *testing.T) {
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

	err = manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if manager.IsInitialized() {
		t.Error("Manager should not be initialized after shutdown")
	}
}

func TestDeploymentModeManager_GetComponentStatus(t *testing.T) {
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

	status := manager.GetComponentStatus()
	if len(status) == 0 {
		t.Error("Expected components in status")
	}

	if !status["api_gateway"] {
		t.Error("Expected api_gateway to be initialized")
	}
}

func TestMonolithicInitializer_Initialize(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	metrics := initializer.GetMetrics()
	if metrics["components_ready"] == 0 {
		t.Error("Expected components to be ready")
	}
}

func TestMonolithicInitializer_HealthCheck(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	metrics := initializer.GetMetrics()
	if metrics["health_checks_passed"] == 0 {
		t.Error("Expected health checks to pass")
	}
}

func TestMonolithicInitializer_Shutdown(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 5 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = initializer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestMicroserviceInitializer_Initialize(t *testing.T) {
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

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	metrics := initializer.GetMetrics()
	if metrics["services_registered"] == 0 {
		t.Error("Expected services to be registered")
	}
}

func TestMicroserviceInitializer_HealthCheck(t *testing.T) {
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

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = initializer.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	metrics := initializer.GetMetrics()
	if metrics["health_checks_passed"] == 0 {
		t.Error("Expected health checks to pass")
	}
}

func TestMicroserviceInitializer_GetRegisteredServices(t *testing.T) {
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

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	services := initializer.GetRegisteredServices()
	if len(services) == 0 {
		t.Error("Expected registered services")
	}

	if _, exists := services["api-gateway"]; !exists {
		t.Error("Expected api-gateway service to be registered")
	}
}

func TestMicroserviceInitializer_Shutdown(t *testing.T) {
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

	err := initializer.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = initializer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	services := initializer.GetRegisteredServices()
	if len(services) != 0 {
		t.Error("Expected no services after shutdown")
	}
}
