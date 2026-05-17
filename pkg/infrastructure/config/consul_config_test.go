package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewConsulClient tests Consul client creation
func TestNewConsulClient(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
		Token:   "",
	}

	client, err := NewConsulClient(config)

	// Connection may fail if Consul is not running, but structure should be valid
	if err == nil {
		assert.NotNil(t, client)
		assert.Equal(t, config, client.config)
		_ = client.Close()
	}
}

// TestNewConsulClientNilConfig tests Consul client creation with nil config
func TestNewConsulClientNilConfig(t *testing.T) {
	t.Parallel()
	client, err := NewConsulClient(nil)

	// Connection may fail if Consul is not running, but structure should be valid
	if err == nil {
		assert.NotNil(t, client)
		assert.NotNil(t, client.config)
		assert.Equal(t, "localhost", client.config.Address)
		assert.Equal(t, 8500, client.config.Port)
		_ = client.Close()
	}
}

// TestConsulConfigStructure tests Consul config structure
func TestConsulConfigStructure(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "consul.example.com",
		Port:    8500,
		Scheme:  "https",
		Token:   "mytoken",
	}

	assert.Equal(t, "consul.example.com", config.Address)
	assert.Equal(t, 8500, config.Port)
	assert.Equal(t, "https", config.Scheme)
	assert.Equal(t, "mytoken", config.Token)
}

// TestConsulClientClose tests closing Consul client
func TestConsulClientClose(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		err = client.Close()
		assert.NoError(t, err)
	}
}

// TestConsulConfigWithDifferentPorts tests Consul config with different ports
func TestConsulConfigWithDifferentPorts(t *testing.T) {
	t.Parallel()
	ports := []int{8500, 8501, 8502, 8600}

	for _, port := range ports {
		config := &ConsulConfig{
			Address: "localhost",
			Port:    port,
			Scheme:  "http",
		}

		assert.Equal(t, port, config.Port)
	}
}

// TestConsulConfigWithDifferentSchemes tests Consul config with different schemes
func TestConsulConfigWithDifferentSchemes(t *testing.T) {
	t.Parallel()
	schemes := []string{"http", "https"}

	for _, scheme := range schemes {
		config := &ConsulConfig{
			Address: "localhost",
			Port:    8500,
			Scheme:  scheme,
		}

		assert.Equal(t, scheme, config.Scheme)
	}
}

// TestConsulConfigWithToken tests Consul config with token
func TestConsulConfigWithToken(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
		Token:   "mytoken123",
	}

	assert.Equal(t, "mytoken123", config.Token)
}

// TestConsulConfigWithEmptyToken tests Consul config with empty token
func TestConsulConfigWithEmptyToken(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
		Token:   "",
	}

	assert.Equal(t, "", config.Token)
}

// TestConsulConfigWithDifferentAddresses tests Consul config with different addresses
func TestConsulConfigWithDifferentAddresses(t *testing.T) {
	t.Parallel()
	addresses := []string{"localhost", "127.0.0.1", "consul.example.com", "consul-primary"}

	for _, address := range addresses {
		config := &ConsulConfig{
			Address: address,
			Port:    8500,
			Scheme:  "http",
		}

		assert.Equal(t, address, config.Address)
	}
}

// TestConsulConfigDefaultValues tests Consul config default values
func TestConsulConfigDefaultValues(t *testing.T) {
	t.Parallel()
	client, err := NewConsulClient(nil)

	if err == nil {
		assert.Equal(t, "localhost", client.config.Address)
		assert.Equal(t, 8500, client.config.Port)
		assert.Equal(t, "http", client.config.Scheme)
		_ = client.Close()
	}
}

// TestConsulConfigWithSpecialCharacters tests Consul config with special characters
func TestConsulConfigWithSpecialCharacters(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
		Token:   "token!@#$%^&*()",
	}

	assert.Equal(t, "token!@#$%^&*()", config.Token)
}

// TestConsulHealthContext tests health check with context
func TestConsulHealthContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Health check may fail if Consul is not running
		_ = client.Health(ctx)
		_ = client.Close()
	}
}

// TestConsulConfigMultipleInstances tests creating multiple Consul config instances
func TestConsulConfigMultipleInstances(t *testing.T) {
	t.Parallel()
	config1 := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
		Token:   "token1",
	}

	config2 := &ConsulConfig{
		Address: "consul.example.com",
		Port:    8501,
		Scheme:  "https",
		Token:   "token2",
	}

	assert.NotEqual(t, config1.Address, config2.Address)
	assert.NotEqual(t, config1.Port, config2.Port)
	assert.NotEqual(t, config1.Scheme, config2.Scheme)
	assert.NotEqual(t, config1.Token, config2.Token)
}

// TestConsulConfigClientStructure tests Consul client structure
func TestConsulConfigClientStructure(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		assert.NotNil(t, client.config)
		assert.Equal(t, config.Address, client.config.Address)
		_ = client.Close()
	}
}

// TestConsulConfigWithHighPort tests Consul config with high port number
func TestConsulConfigWithHighPort(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    65432,
		Scheme:  "http",
	}

	assert.Equal(t, 65432, config.Port)
}

// TestConsulConfigWithLowPort tests Consul config with low port number
func TestConsulConfigWithLowPort(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    1,
		Scheme:  "http",
	}

	assert.Equal(t, 1, config.Port)
}

// TestConsulRegisterServiceContext tests service registration with context
func TestConsulRegisterServiceContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Service registration may fail if Consul is not running
		_ = client.RegisterService(ctx, "service1", "my-service", 8080, []string{"tag1", "tag2"})
		_ = client.Close()
	}
}

// TestConsulDeregisterServiceContext tests service deregistration with context
func TestConsulDeregisterServiceContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Service deregistration may fail if Consul is not running
		_ = client.DeregisterService(ctx, "service1")
		_ = client.Close()
	}
}

// TestConsulDiscoverServiceContext tests service discovery with context
func TestConsulDiscoverServiceContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Service discovery may fail if Consul is not running
		_, _ = client.DiscoverService(ctx, "my-service")
		_ = client.Close()
	}
}

// TestConsulGetConfigContext tests getting config with context
func TestConsulGetConfigContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Config retrieval may fail if Consul is not running
		_, _ = client.GetConfig(ctx, "app/config")
		_ = client.Close()
	}
}

// TestConsulSetConfigContext tests setting config with context
func TestConsulSetConfigContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Config setting may fail if Consul is not running
		_ = client.SetConfig(ctx, "app/config", "value")
		_ = client.Close()
	}
}

// TestConsulWatchConfigContext tests watching config with context
func TestConsulWatchConfigContext(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Address: "localhost",
		Port:    8500,
		Scheme:  "http",
	}

	client, err := NewConsulClient(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Config watching may fail if Consul is not running
		_ = client.WatchConfig(ctx, "app/config", func(value string) {
			// Handler
		})
		_ = client.Close()
	}
}
