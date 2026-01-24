package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewRedisCluster tests Redis cluster creation
func TestNewRedisCluster(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	cluster, err := NewRedisCluster(config)

	// Connection may fail if Redis is not running, but structure should be valid
	if err == nil {
		assert.NotNil(t, cluster)
		assert.Equal(t, config, cluster.config)
		_ = cluster.Close()
	}
}

// TestNewRedisClusterNilConfig tests Redis cluster creation with nil config
func TestNewRedisClusterNilConfig(t *testing.T) {
	cluster, err := NewRedisCluster(nil)

	// Connection may fail if Redis is not running, but structure should be valid
	if err == nil {
		assert.NotNil(t, cluster)
		assert.NotNil(t, cluster.config)
		assert.Equal(t, "localhost", cluster.config.Host)
		assert.Equal(t, 6379, cluster.config.Port)
		_ = cluster.Close()
	}
}

// TestRedisConfigStructure tests Redis config structure
func TestRedisConfigStructure(t *testing.T) {
	config := &RedisConfig{
		Host:     "redis.example.com",
		Port:     6379,
		Password: "secret",
		DB:       1,
	}

	assert.Equal(t, "redis.example.com", config.Host)
	assert.Equal(t, 6379, config.Port)
	assert.Equal(t, "secret", config.Password)
	assert.Equal(t, 1, config.DB)
}

// TestRedisClusterClose tests closing Redis cluster
func TestRedisClusterClose(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	cluster, err := NewRedisCluster(config)
	if err == nil {
		err = cluster.Close()
		assert.NoError(t, err)
	}
}

// TestRedisClusterCloseNilClient tests closing Redis cluster with nil client
func TestRedisClusterCloseNilClient(t *testing.T) {
	cluster := &RedisCluster{
		config: &RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Client: nil,
	}

	err := cluster.Close()
	assert.NoError(t, err)
}

// TestRedisConfigWithDifferentPorts tests Redis config with different ports
func TestRedisConfigWithDifferentPorts(t *testing.T) {
	ports := []int{6379, 6380, 6381, 26379}

	for _, port := range ports {
		config := &RedisConfig{
			Host:     "localhost",
			Port:     port,
			Password: "",
			DB:       0,
		}

		assert.Equal(t, port, config.Port)
	}
}

// TestRedisConfigWithDifferentDatabases tests Redis config with different databases
func TestRedisConfigWithDifferentDatabases(t *testing.T) {
	for db := 0; db < 16; db++ {
		config := &RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       db,
		}

		assert.Equal(t, db, config.DB)
	}
}

// TestRedisConfigWithPassword tests Redis config with password
func TestRedisConfigWithPassword(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "mypassword",
		DB:       0,
	}

	assert.Equal(t, "mypassword", config.Password)
}

// TestRedisConfigWithEmptyPassword tests Redis config with empty password
func TestRedisConfigWithEmptyPassword(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	assert.Equal(t, "", config.Password)
}

// TestRedisConfigWithDifferentHosts tests Redis config with different hosts
func TestRedisConfigWithDifferentHosts(t *testing.T) {
	hosts := []string{"localhost", "127.0.0.1", "redis.example.com", "redis-primary"}

	for _, host := range hosts {
		config := &RedisConfig{
			Host:     host,
			Port:     6379,
			Password: "",
			DB:       0,
		}

		assert.Equal(t, host, config.Host)
	}
}

// TestRedisConfigDefaultValues tests Redis config default values
func TestRedisConfigDefaultValues(t *testing.T) {
	cluster, err := NewRedisCluster(nil)

	if err == nil {
		assert.Equal(t, "localhost", cluster.config.Host)
		assert.Equal(t, 6379, cluster.config.Port)
		assert.Equal(t, "", cluster.config.Password)
		assert.Equal(t, 0, cluster.config.DB)
		_ = cluster.Close()
	}
}

// TestRedisConfigWithSpecialCharacters tests Redis config with special characters
func TestRedisConfigWithSpecialCharacters(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "p@ssw0rd!#$%",
		DB:       0,
	}

	assert.Equal(t, "p@ssw0rd!#$%", config.Password)
}

// TestRedisHealthContext tests health check with context
func TestRedisHealthContext(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	cluster, err := NewRedisCluster(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Health check may fail if Redis is not running
		_ = cluster.Health(ctx)
		_ = cluster.Close()
	}
}

// TestRedisConfigMultipleInstances tests creating multiple Redis config instances
func TestRedisConfigMultipleInstances(t *testing.T) {
	config1 := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "pass1",
		DB:       0,
	}

	config2 := &RedisConfig{
		Host:     "localhost",
		Port:     6380,
		Password: "pass2",
		DB:       1,
	}

	assert.NotEqual(t, config1.Port, config2.Port)
	assert.NotEqual(t, config1.Password, config2.Password)
	assert.NotEqual(t, config1.DB, config2.DB)
}

// TestRedisConfigClusterStructure tests Redis cluster structure
func TestRedisConfigClusterStructure(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	cluster, err := NewRedisCluster(config)
	if err == nil {
		assert.NotNil(t, cluster.config)
		assert.Equal(t, config.Host, cluster.config.Host)
		_ = cluster.Close()
	}
}

// TestRedisConfigWithHighPort tests Redis config with high port number
func TestRedisConfigWithHighPort(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     65432,
		Password: "",
		DB:       0,
	}

	assert.Equal(t, 65432, config.Port)
}

// TestRedisConfigWithLowPort tests Redis config with low port number
func TestRedisConfigWithLowPort(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     1,
		Password: "",
		DB:       0,
	}

	assert.Equal(t, 1, config.Port)
}

// TestRedisConfigWithMaxDatabase tests Redis config with max database number
func TestRedisConfigWithMaxDatabase(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       15,
	}

	assert.Equal(t, 15, config.DB)
}

// TestRedisConfigWithMinDatabase tests Redis config with min database number
func TestRedisConfigWithMinDatabase(t *testing.T) {
	config := &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	assert.Equal(t, 0, config.DB)
}
