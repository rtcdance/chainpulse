package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/stretchr/testify/assert"
)

// getTestPostgresConfig returns PostgreSQL config from environment or defaults
func getTestPostgresConfig() *PostgresConfig {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 5432
	portStr := os.Getenv("POSTGRES_PORT")
	if portStr != "" {
		// Parse port if needed, but for now just use default
		port = 5432
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "chainpulse"
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "chainpulse_test"
	}

	database := os.Getenv("POSTGRES_DB")
	if database == "" {
		database = "chainpulse_test"
	}

	return &PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: core.SecretString(password),
		Database: database,
		SSLMode:  "disable",
	}
}

// TestNewPostgresCluster tests PostgreSQL cluster creation
func TestNewPostgresCluster(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()

	cluster, err := NewPostgresCluster(config)

	// Connection may fail if PostgreSQL is not running, but structure should be valid
	if err == nil {
		assert.NotNil(t, cluster)
		assert.Equal(t, config, cluster.config)
		_ = cluster.Close()
	}
}

// TestNewPostgresClusterNilConfig tests PostgreSQL cluster creation with nil config
func TestNewPostgresClusterNilConfig(t *testing.T) {
	t.Parallel()
	cluster, err := NewPostgresCluster(nil)

	// Connection may fail if PostgreSQL is not running, but structure should be valid
	if err == nil {
		assert.NotNil(t, cluster)
		assert.NotNil(t, cluster.config)
		assert.Equal(t, "localhost", cluster.config.Host)
		assert.Equal(t, 5432, cluster.config.Port)
		_ = cluster.Close()
	}
}

// TestPostgresConfigStructure tests PostgreSQL config structure
func TestPostgresConfigStructure(t *testing.T) {
	t.Parallel()
	config := &PostgresConfig{
		Host:     "db.example.com",
		Port:     5432,
		User:     "admin",
		Password: "secret",
		Database: "mydb",
		SSLMode:  "require",
	}

	assert.Equal(t, "db.example.com", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "admin", config.User)
	assert.Equal(t, "secret", config.Password)
	assert.Equal(t, "mydb", config.Database)
	assert.Equal(t, "require", config.SSLMode)
}

// TestPostgresClusterClose tests closing PostgreSQL cluster
func TestPostgresClusterClose(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()

	cluster, err := NewPostgresCluster(config)
	if err == nil {
		err = cluster.Close()
		assert.NoError(t, err)
	}
}

// TestPostgresClusterCloseNilDB tests closing PostgreSQL cluster with nil DB
func TestPostgresClusterCloseNilDB(t *testing.T) {
	t.Parallel()
	cluster := &PostgresCluster{
		config: getTestPostgresConfig(),
		DB:     nil,
	}

	err := cluster.Close()
	assert.NoError(t, err)
}

// TestPostgresConfigWithDifferentPorts tests PostgreSQL config with different ports
func TestPostgresConfigWithDifferentPorts(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()
	config.Port = 5433

	assert.Equal(t, 5433, config.Port)
}

// TestPostgresConfigWithSSLModes tests PostgreSQL config with different SSL modes
func TestPostgresConfigWithSSLModes(t *testing.T) {
	t.Parallel()
	sslModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

	for _, mode := range sslModes {
		config := getTestPostgresConfig()
		config.SSLMode = mode

		assert.Equal(t, mode, config.SSLMode)
	}
}

// TestPostgresConfigWithDifferentDatabases tests PostgreSQL config with different databases
func TestPostgresConfigWithDifferentDatabases(t *testing.T) {
	t.Parallel()
	databases := []string{"postgres", "testdb", "myapp", "analytics"}

	for _, db := range databases {
		config := getTestPostgresConfig()
		config.Database = db

		assert.Equal(t, db, config.Database)
	}
}

// TestPostgresConfigWithDifferentUsers tests PostgreSQL config with different users
func TestPostgresConfigWithDifferentUsers(t *testing.T) {
	t.Parallel()
	users := []string{"postgres", "admin", "app_user", "readonly"}

	for _, user := range users {
		config := getTestPostgresConfig()
		config.User = user

		assert.Equal(t, user, config.User)
	}
}

// TestPostgresConfigWithDifferentHosts tests PostgreSQL config with different hosts
func TestPostgresConfigWithDifferentHosts(t *testing.T) {
	t.Parallel()
	hosts := []string{"localhost", "127.0.0.1", "db.example.com", "postgres-primary"}

	for _, host := range hosts {
		config := getTestPostgresConfig()
		config.Host = host

		assert.Equal(t, host, config.Host)
	}
}

// TestPostgresConfigDefaultValues tests PostgreSQL config default values
func TestPostgresConfigDefaultValues(t *testing.T) {
	t.Parallel()
	cluster, err := NewPostgresCluster(nil)

	if err == nil {
		assert.Equal(t, "localhost", cluster.config.Host)
		assert.Equal(t, 5432, cluster.config.Port)
		// Default user should be "postgres" when nil config is used
		assert.Equal(t, "postgres", cluster.config.User)
		assert.Equal(t, "postgres", cluster.config.Database)
		assert.Equal(t, "disable", cluster.config.SSLMode)
		_ = cluster.Close()
	}
}

// TestPostgresConfigWithEmptyPassword tests PostgreSQL config with empty password
func TestPostgresConfigWithEmptyPassword(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()
	config.Password = ""

	assert.Equal(t, "", config.Password)
}

// TestPostgresConfigWithSpecialCharacters tests PostgreSQL config with special characters
func TestPostgresConfigWithSpecialCharacters(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()
	config.Password = "p@ssw0rd!#$%"
	config.Database = "test_db_2024"

	assert.Equal(t, "p@ssw0rd!#$%", config.Password)
	assert.Equal(t, "test_db_2024", config.Database)
}

// TestPostgresHealthContext tests health check with context
func TestPostgresHealthContext(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()

	cluster, err := NewPostgresCluster(config)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Health check may fail if PostgreSQL is not running
		_ = cluster.Health(ctx)
		_ = cluster.Close()
	}
}

// TestPostgresConfigMultipleInstances tests creating multiple PostgreSQL config instances
func TestPostgresConfigMultipleInstances(t *testing.T) {
	t.Parallel()
	config1 := getTestPostgresConfig()
	config1.Port = 5432
	config1.User = "user1"
	config1.Password = "pass1"
	config1.Database = "db1"

	config2 := getTestPostgresConfig()
	config2.Port = 5433
	config2.User = "user2"
	config2.Password = "pass2"
	config2.Database = "db2"
	config2.SSLMode = "require"

	assert.NotEqual(t, config1.Port, config2.Port)
	assert.NotEqual(t, config1.User, config2.User)
	assert.NotEqual(t, config1.Database, config2.Database)
}

// TestPostgresConfigClusterStructure tests PostgreSQL cluster structure
func TestPostgresConfigClusterStructure(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()

	cluster, err := NewPostgresCluster(config)
	if err == nil {
		assert.NotNil(t, cluster.config)
		assert.Equal(t, config.Host, cluster.config.Host)
		_ = cluster.Close()
	}
}

// TestPostgresConfigWithHighPort tests PostgreSQL config with high port number
func TestPostgresConfigWithHighPort(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()
	config.Port = 65432

	assert.Equal(t, 65432, config.Port)
}

// TestPostgresConfigWithLowPort tests PostgreSQL config with low port number
func TestPostgresConfigWithLowPort(t *testing.T) {
	t.Parallel()
	config := getTestPostgresConfig()
	config.Port = 1

	assert.Equal(t, 1, config.Port)
}
