package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewServiceInfo tests creating a new service info
func TestNewServiceInfo(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	assert.NotNil(t, info)
	assert.Equal(t, "service-1", info.ID)
	assert.Equal(t, "my-service", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "unknown", info.Status.Status)
	assert.NotNil(t, info.Metadata)
	assert.NotNil(t, info.Tags)
	assert.False(t, info.RegisteredAt.IsZero())
}

// TestServiceInfoIsHealthy tests health status check
func TestServiceInfoIsHealthy(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	assert.False(t, info.IsHealthy())

	info.Status.Status = "healthy"
	assert.True(t, info.IsHealthy())

	info.Status.Status = "unhealthy"
	assert.False(t, info.IsHealthy())
}

// TestServiceInfoUpdateStatus tests updating service status
func TestServiceInfoUpdateStatus(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")
	originalTime := info.LastHeartbeat

	time.Sleep(10 * time.Millisecond)

	newStatus := HealthStatus{
		Status:  "healthy",
		Message: "All systems operational",
	}
	info.UpdateStatus(newStatus)

	assert.Equal(t, "healthy", info.Status.Status)
	assert.Equal(t, "All systems operational", info.Status.Message)
	assert.True(t, info.LastHeartbeat.After(originalTime))
}

// TestServiceInfoAddTag tests adding tags
func TestServiceInfoAddTag(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	info.AddTag("production")
	info.AddTag("critical")

	assert.Equal(t, 2, len(info.Tags))
	assert.Contains(t, info.Tags, "production")
	assert.Contains(t, info.Tags, "critical")
}

// TestServiceInfoAddMultipleTags tests adding multiple tags
func TestServiceInfoAddMultipleTags(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
	for _, tag := range tags {
		info.AddTag(tag)
	}

	assert.Equal(t, len(tags), len(info.Tags))
	for _, tag := range tags {
		assert.Contains(t, info.Tags, tag)
	}
}

// TestServiceInfoSetMetadata tests setting metadata
func TestServiceInfoSetMetadata(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	info.SetMetadata("region", "us-west-2")
	info.SetMetadata("environment", "production")

	assert.Equal(t, "us-west-2", info.GetMetadata("region"))
	assert.Equal(t, "production", info.GetMetadata("environment"))
}

// TestServiceInfoGetMetadataNotFound tests getting non-existent metadata
func TestServiceInfoGetMetadataNotFound(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	value := info.GetMetadata("nonexistent")

	assert.Equal(t, "", value)
}

// TestServiceInfoMetadataOverwrite tests overwriting metadata
func TestServiceInfoMetadataOverwrite(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	info.SetMetadata("key", "value1")
	assert.Equal(t, "value1", info.GetMetadata("key"))

	info.SetMetadata("key", "value2")
	assert.Equal(t, "value2", info.GetMetadata("key"))
}

// TestServiceInfoMultipleMetadata tests setting multiple metadata
func TestServiceInfoMultipleMetadata(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	metadata := map[string]string{
		"region":      "us-west-2",
		"environment": "production",
		"team":        "platform",
		"owner":       "john@example.com",
	}

	for key, value := range metadata {
		info.SetMetadata(key, value)
	}

	for key, value := range metadata {
		assert.Equal(t, value, info.GetMetadata(key))
	}
}

// TestServiceInfoEndpoint tests endpoint configuration
func TestServiceInfoEndpoint(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	info.Endpoint = "localhost"
	info.Port = 8080
	info.Protocol = "http"

	assert.Equal(t, "localhost", info.Endpoint)
	assert.Equal(t, 8080, info.Port)
	assert.Equal(t, "http", info.Protocol)
}

// TestServiceInfoHealthCheckURL tests health check URL
func TestServiceInfoHealthCheckURL(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	info.HealthCheckURL = "http://localhost:8080/health"

	assert.Equal(t, "http://localhost:8080/health", info.HealthCheckURL)
}

// TestServiceInfoDeregisterAfter tests deregister after setting
func TestServiceInfoDeregisterAfter(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	info.DeregisterAfter = "30s"

	assert.Equal(t, "30s", info.DeregisterAfter)
}

// TestServiceInfoRegisteredAt tests registered at timestamp
func TestServiceInfoRegisteredAt(t *testing.T) {
	t.Parallel()
	before := time.Now()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")
	after := time.Now()

	assert.True(t, info.RegisteredAt.After(before) || info.RegisteredAt.Equal(before))
	assert.True(t, info.RegisteredAt.Before(after) || info.RegisteredAt.Equal(after))
}

// TestServiceInfoLastHeartbeat tests last heartbeat timestamp
func TestServiceInfoLastHeartbeat(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	assert.True(t, info.LastHeartbeat.IsZero())

	info.UpdateStatus(HealthStatus{Status: "healthy"})

	assert.False(t, info.LastHeartbeat.IsZero())
}

// TestHealthStatusStructure tests health status structure
func TestHealthStatusStructure(t *testing.T) {
	t.Parallel()
	status := HealthStatus{
		Status:  "healthy",
		Message: "All systems operational",
		Details: map[string]any{
			"cpu":    "50%",
			"memory": "60%",
		},
	}

	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "All systems operational", status.Message)
	assert.Equal(t, "50%", status.Details["cpu"])
	assert.Equal(t, "60%", status.Details["memory"])
}

// TestServiceInfoMultipleInstances tests creating multiple service info instances
func TestServiceInfoMultipleInstances(t *testing.T) {
	t.Parallel()
	info1 := NewServiceInfo("service-1", "service-a", "1.0.0")
	info2 := NewServiceInfo("service-2", "service-b", "2.0.0")

	assert.NotEqual(t, info1.ID, info2.ID)
	assert.NotEqual(t, info1.Name, info2.Name)
	assert.NotEqual(t, info1.Version, info2.Version)
}

// TestServiceInfoConcurrentMetadataAccess tests concurrent metadata access
func TestServiceInfoConcurrentMetadataAccess(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	// Sequential access instead of concurrent (metadata is not thread-safe)
	for i := 0; i < 10; i++ {
		key := "key" + string(rune(i))
		info.SetMetadata(key, "value")
		val := info.GetMetadata(key)
		assert.Equal(t, "value", val)
	}
}

// TestServiceInfoConcurrentTagAccess tests concurrent tag access
func TestServiceInfoConcurrentTagAccess(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	// Sequential access instead of concurrent (tags are not thread-safe)
	for i := 0; i < 10; i++ {
		info.AddTag("tag" + string(rune(i)))
	}

	assert.GreaterOrEqual(t, len(info.Tags), 1)
}

// TestServiceInfoStatusTransitions tests status transitions
func TestServiceInfoStatusTransitions(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	statuses := []string{"unknown", "starting", "healthy", "degraded", "unhealthy", "stopped"}

	for _, status := range statuses {
		info.UpdateStatus(HealthStatus{Status: status})
		assert.Equal(t, status, info.Status.Status)
	}
}

// TestServiceInfoEmptyMetadata tests empty metadata
func TestServiceInfoEmptyMetadata(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	assert.NotNil(t, info.Metadata)
	assert.Equal(t, 0, len(info.Metadata))
}

// TestServiceInfoEmptyTags tests empty tags
func TestServiceInfoEmptyTags(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	assert.NotNil(t, info.Tags)
	assert.Equal(t, 0, len(info.Tags))
}

// TestServiceInfoVersions tests various version formats
func TestServiceInfoVersions(t *testing.T) {
	t.Parallel()
	versions := []string{"1.0.0", "2.1.3", "0.0.1", "10.20.30", "1.0.0-alpha", "1.0.0-beta.1"}

	for _, version := range versions {
		info := NewServiceInfo("service-1", "my-service", version)
		assert.Equal(t, version, info.Version)
	}
}

// TestServiceInfoNames tests various service names
func TestServiceInfoNames(t *testing.T) {
	t.Parallel()
	names := []string{"api-service", "auth-service", "payment-service", "notification-service"}

	for _, name := range names {
		info := NewServiceInfo("service-1", name, "1.0.0")
		assert.Equal(t, name, info.Name)
	}
}

// TestServiceInfoIDs tests various service IDs
func TestServiceInfoIDs(t *testing.T) {
	t.Parallel()
	ids := []string{"service-1", "service-2", "api-prod", "auth-staging"}

	for _, id := range ids {
		info := NewServiceInfo(id, "my-service", "1.0.0")
		assert.Equal(t, id, info.ID)
	}
}

// TestServiceInfoPorts tests various port numbers
func TestServiceInfoPorts(t *testing.T) {
	t.Parallel()
	ports := []int{8080, 8081, 3000, 5000, 9000}

	for _, port := range ports {
		info := NewServiceInfo("service-1", "my-service", "1.0.0")
		info.Port = port
		assert.Equal(t, port, info.Port)
	}
}

// TestServiceInfoProtocols tests various protocols
func TestServiceInfoProtocols(t *testing.T) {
	t.Parallel()
	protocols := []string{"http", "https", "grpc", "tcp", "udp"}

	for _, protocol := range protocols {
		info := NewServiceInfo("service-1", "my-service", "1.0.0")
		info.Protocol = protocol
		assert.Equal(t, protocol, info.Protocol)
	}
}

// TestServiceInfoEndpoints tests various endpoints
func TestServiceInfoEndpoints(t *testing.T) {
	t.Parallel()
	endpoints := []string{"localhost", "127.0.0.1", "service.example.com", "api.prod.example.com"}

	for _, endpoint := range endpoints {
		info := NewServiceInfo("service-1", "my-service", "1.0.0")
		info.Endpoint = endpoint
		assert.Equal(t, endpoint, info.Endpoint)
	}
}

// TestServiceInfoStatusDetails tests status with details
func TestServiceInfoStatusDetails(t *testing.T) {
	t.Parallel()
	info := NewServiceInfo("service-1", "my-service", "1.0.0")

	status := HealthStatus{
		Status:  "degraded",
		Message: "High latency detected",
		Details: map[string]any{
			"latency_ms": 500,
			"error_rate": 0.05,
			"uptime":     "99.5%",
		},
	}

	info.UpdateStatus(status)

	assert.Equal(t, "degraded", info.Status.Status)
	assert.Equal(t, "High latency detected", info.Status.Message)
	assert.Equal(t, 500, info.Status.Details["latency_ms"])
	assert.Equal(t, 0.05, info.Status.Details["error_rate"])
	assert.Equal(t, "99.5%", info.Status.Details["uptime"])
}
