package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewAPIGatewayClusterDeployment tests creating a new cluster deployment
func TestNewAPIGatewayClusterDeployment(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances:      3,
		Port:           8080,
		Protocols:      []string{"http", "https"},
		HealthCheckURL: "/health",
		HealthCheckTTL: 30 * time.Second,
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	assert.NotNil(t, deployment)
	assert.Equal(t, config, deployment.config)
	assert.NotNil(t, deployment.gateways)
	assert.False(t, deployment.running)
}

// TestAPIGatewayClusterConfigStructure tests cluster config structure
func TestAPIGatewayClusterConfigStructure(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances:       5,
		Port:            9000,
		Protocols:       []string{"http", "https", "grpc"},
		HealthCheckURL:  "/api/health",
		HealthCheckTTL:  60 * time.Second,
		LoadBalancerURL: "lb.example.com",
	}

	assert.Equal(t, 5, config.Instances)
	assert.Equal(t, 9000, config.Port)
	assert.Equal(t, 3, len(config.Protocols))
	assert.Equal(t, "/api/health", config.HealthCheckURL)
	assert.Equal(t, 60*time.Second, config.HealthCheckTTL)
	assert.Equal(t, "lb.example.com", config.LoadBalancerURL)
}

// TestAPIGatewayClusterGetInstance tests getting a cluster instance
func TestAPIGatewayClusterGetInstance(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	// Add a gateway manually for testing
	deployment.gateways["api-gateway-0"] = &APIGateway{}

	gateway, err := deployment.GetInstance("api-gateway-0")

	assert.NoError(t, err)
	assert.NotNil(t, gateway)
}

// TestAPIGatewayClusterGetInstanceNotFound tests getting non-existent instance
func TestAPIGatewayClusterGetInstanceNotFound(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	_, err := deployment.GetInstance("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestAPIGatewayClusterListInstances tests listing instances
func TestAPIGatewayClusterListInstances(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 3,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	// Add gateways manually
	for i := 0; i < 3; i++ {
		deployment.gateways[string(rune(i))] = &APIGateway{}
	}

	instances := deployment.ListInstances()

	assert.Equal(t, 3, len(instances))
}

// TestAPIGatewayClusterListInstancesEmpty tests listing instances when empty
func TestAPIGatewayClusterListInstancesEmpty(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 0,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	instances := deployment.ListInstances()

	assert.Equal(t, 0, len(instances))
}

// TestAPIGatewayClusterGetClusterHealthNotDeployed tests health check when not deployed
func TestAPIGatewayClusterGetClusterHealthNotDeployed(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)
	ctx := context.Background()

	_, err := deployment.GetClusterHealth(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not deployed")
}

// TestAPIGatewayClusterGetClusterHealthDeployed tests health check when deployed
func TestAPIGatewayClusterGetClusterHealthDeployed(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 2,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	// Mark as running and add gateways
	deployment.running = true
	deployment.gateways["api-gateway-0"] = &APIGateway{}
	deployment.gateways["api-gateway-1"] = &APIGateway{}

	ctx := context.Background()
	health, err := deployment.GetClusterHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "ok", health["status"])
	assert.Equal(t, 2, health["healthy_instances"])
	assert.Equal(t, 2, health["total_instances"])
}

// TestAPIGatewayClusterGetMetrics tests getting cluster metrics
func TestAPIGatewayClusterGetMetrics(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	// Add gateway with metrics
	gateway := &APIGateway{
		metrics: NewAPIMetrics(),
	}
	deployment.gateways["api-gateway-0"] = gateway

	metrics := deployment.GetMetrics()

	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics["total_requests"], int64(0))
	assert.GreaterOrEqual(t, metrics["total_errors"], int64(0))
	assert.Equal(t, 1, metrics["instance_count"])
}

// TestAPIGatewayClusterDeployNotDeployed tests deploying when not already deployed
func TestAPIGatewayClusterDeployNotDeployed(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 0,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	assert.False(t, deployment.running)
}

// TestAPIGatewayClusterUndeployNotDeployed tests undeploying when not deployed
func TestAPIGatewayClusterUndeployNotDeployed(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)
	ctx := context.Background()

	err := deployment.Undeploy(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not deployed")
}

// TestAPIGatewayClusterScaleUpZero tests scaling up by zero
func TestAPIGatewayClusterScaleUpZero(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)
	ctx := context.Background()

	// Should not error when scaling up by 0
	err := deployment.ScaleUp(ctx, 0)

	assert.NoError(t, err)
}

// TestAPIGatewayClusterScaleDownMoreThanAvailable tests scaling down more than available
func TestAPIGatewayClusterScaleDownMoreThanAvailable(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)
	ctx := context.Background()

	err := deployment.ScaleDown(ctx, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot scale down")
}

// TestAPIGatewayClusterConcurrentAccess tests concurrent access to cluster
func TestAPIGatewayClusterConcurrentAccess(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)
	deployment.gateways["api-gateway-0"] = &APIGateway{}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, _ = deployment.GetInstance("api-gateway-0")
			_ = deployment.ListInstances()
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestAPIGatewayClusterMultipleProtocols tests cluster with multiple protocols
func TestAPIGatewayClusterMultipleProtocols(t *testing.T) {
	t.Parallel()
	config := APIGatewayClusterConfig{
		Instances: 1,
		Port:      8080,
		Protocols: []string{"http", "https", "grpc", "websocket"},
	}

	deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

	assert.Equal(t, 4, len(deployment.config.Protocols))
	assert.Contains(t, deployment.config.Protocols, "http")
	assert.Contains(t, deployment.config.Protocols, "https")
	assert.Contains(t, deployment.config.Protocols, "grpc")
	assert.Contains(t, deployment.config.Protocols, "websocket")
}

// TestAPIGatewayClusterPortConfiguration tests cluster port configuration
func TestAPIGatewayClusterPortConfiguration(t *testing.T) {
	t.Parallel()
	ports := []int{8080, 8081, 9000, 3000}

	for _, port := range ports {
		config := APIGatewayClusterConfig{
			Instances: 1,
			Port:      port,
			Protocols: []string{"http"},
		}

		deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

		assert.Equal(t, port, deployment.config.Port)
	}
}

// TestAPIGatewayClusterInstanceCount tests various instance counts
func TestAPIGatewayClusterInstanceCount(t *testing.T) {
	t.Parallel()
	counts := []int{1, 3, 5, 10}

	for _, count := range counts {
		config := APIGatewayClusterConfig{
			Instances: count,
			Port:      8080,
			Protocols: []string{"http"},
		}

		deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

		assert.Equal(t, count, deployment.config.Instances)
	}
}

// TestAPIGatewayClusterHealthCheckURL tests health check URL configuration
func TestAPIGatewayClusterHealthCheckURL(t *testing.T) {
	t.Parallel()
	urls := []string{"/health", "/api/health", "/status", "/ping"}

	for _, url := range urls {
		config := APIGatewayClusterConfig{
			Instances:      1,
			Port:           8080,
			Protocols:      []string{"http"},
			HealthCheckURL: url,
		}

		deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

		assert.Equal(t, url, deployment.config.HealthCheckURL)
	}
}

// TestAPIGatewayClusterHealthCheckTTL tests health check TTL configuration
func TestAPIGatewayClusterHealthCheckTTL(t *testing.T) {
	t.Parallel()
	ttls := []time.Duration{10 * time.Second, 30 * time.Second, 60 * time.Second}

	for _, ttl := range ttls {
		config := APIGatewayClusterConfig{
			Instances:      1,
			Port:           8080,
			Protocols:      []string{"http"},
			HealthCheckTTL: ttl,
		}

		deployment := NewAPIGatewayClusterDeployment(config, nil, nil, nil, nil)

		assert.Equal(t, ttl, deployment.config.HealthCheckTTL)
	}
}
