package deployment

import (
	"testing"

	"chainpulse/pkg/core"

	"github.com/stretchr/testify/assert"
)

// TestNewConsulConfig tests creating a new Consul configuration
func TestNewConsulConfig(t *testing.T) {
	t.Parallel()
	config := NewConsulConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8500, config.Port)
	assert.Equal(t, "dc1", config.Datacenter)
	assert.Equal(t, "http", config.Scheme)
	assert.False(t, config.TLS)
}

// TestConsulConfigDefaults tests Consul configuration defaults
func TestConsulConfigDefaults(t *testing.T) {
	t.Parallel()
	config := NewConsulConfig()

	assert.Equal(t, 5, config.ConnectTimeout)
	assert.Equal(t, 10, config.ReadTimeout)
	assert.Equal(t, 10, config.WriteTimeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100, config.RetryWaitMin)
	assert.Equal(t, 1000, config.RetryWaitMax)
}

// TestConsulConfigValidate tests Consul configuration validation
func TestConsulConfigValidate(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8500, config.Port)
	assert.Equal(t, "dc1", config.Datacenter)
	assert.Equal(t, "http", config.Scheme)
}

// TestConsulConfigValidateWithExistingValues tests validation preserves existing values
func TestConsulConfigValidateWithExistingValues(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Host:       "consul.example.com",
		Port:       8501,
		Datacenter: "dc2",
		Scheme:     "https",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "consul.example.com", config.Host)
	assert.Equal(t, 8501, config.Port)
	assert.Equal(t, "dc2", config.Datacenter)
	assert.Equal(t, "https", config.Scheme)
}

// TestConsulConfigWithCustomHost tests Consul config with custom host
func TestConsulConfigWithCustomHost(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Host: "consul-primary",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "consul-primary", config.Host)
}

// TestConsulConfigWithCustomPort tests Consul config with custom port
func TestConsulConfigWithCustomPort(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Port: 8501,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 8501, config.Port)
}

// TestConsulConfigWithCustomDatacenter tests Consul config with custom datacenter
func TestConsulConfigWithCustomDatacenter(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Datacenter: "dc-us-west",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "dc-us-west", config.Datacenter)
}

// TestConsulConfigWithToken tests Consul config with token
func TestConsulConfigWithToken(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Token: core.SecretString("mytoken123"),
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, core.SecretString("mytoken123"), config.Token)
}

// TestConsulConfigWithHTTPScheme tests Consul config with HTTP scheme
func TestConsulConfigWithHTTPScheme(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Scheme: "http",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "http", config.Scheme)
}

// TestConsulConfigWithHTTPSScheme tests Consul config with HTTPS scheme
func TestConsulConfigWithHTTPSScheme(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Scheme: "https",
		TLS:    true,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "https", config.Scheme)
	assert.True(t, config.TLS)
}

// TestConsulConfigWithTLS tests Consul config with TLS enabled
func TestConsulConfigWithTLS(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		TLS:                true,
		InsecureSkipVerify: false,
		CAFile:             "/path/to/ca.pem",
		CertFile:           "/path/to/cert.pem",
		KeyFile:            "/path/to/key.pem",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.True(t, config.TLS)
	assert.False(t, config.InsecureSkipVerify)
}

// TestConsulConfigWithInsecureSkipVerify tests Consul config with insecure skip verify
func TestConsulConfigWithInsecureSkipVerify(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		TLS:                true,
		InsecureSkipVerify: true,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.True(t, config.InsecureSkipVerify)
}

// TestConsulConfigWithCustomTimeouts tests Consul config with custom timeouts
func TestConsulConfigWithCustomTimeouts(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		ConnectTimeout: 10,
		ReadTimeout:    20,
		WriteTimeout:   20,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 10, config.ConnectTimeout)
	assert.Equal(t, 20, config.ReadTimeout)
	assert.Equal(t, 20, config.WriteTimeout)
}

// TestConsulConfigWithCustomRetries tests Consul config with custom retry settings
func TestConsulConfigWithCustomRetries(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		MaxRetries:   5,
		RetryWaitMin: 200,
		RetryWaitMax: 2000,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 200, config.RetryWaitMin)
	assert.Equal(t, 2000, config.RetryWaitMax)
}

// TestConsulConfigMultipleInstances tests creating multiple Consul config instances
func TestConsulConfigMultipleInstances(t *testing.T) {
	t.Parallel()
	config1 := NewConsulConfig()
	config2 := NewConsulConfig()

	assert.Equal(t, config1.Host, config2.Host)
	assert.Equal(t, config1.Port, config2.Port)

	config2.Host = "different-host"
	assert.NotEqual(t, config1.Host, config2.Host)
}

// TestConsulConfigValidateEmptyHost tests validation with empty host
func TestConsulConfigValidateEmptyHost(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Host: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "localhost", config.Host)
}

// TestConsulConfigValidateZeroPort tests validation with zero port
func TestConsulConfigValidateZeroPort(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Port: 0,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 8500, config.Port)
}

// TestConsulConfigValidateEmptyDatacenter tests validation with empty datacenter
func TestConsulConfigValidateEmptyDatacenter(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Datacenter: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "dc1", config.Datacenter)
}

// TestConsulConfigValidateEmptyScheme tests validation with empty scheme
func TestConsulConfigValidateEmptyScheme(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Scheme: "",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "http", config.Scheme)
}

// TestConsulConfigWithCAFile tests Consul config with CA file
func TestConsulConfigWithCAFile(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		CAFile: "/etc/consul/ca.pem",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "/etc/consul/ca.pem", config.CAFile)
}

// TestConsulConfigWithCertFile tests Consul config with cert file
func TestConsulConfigWithCertFile(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		CertFile: "/etc/consul/cert.pem",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "/etc/consul/cert.pem", config.CertFile)
}

// TestConsulConfigWithKeyFile tests Consul config with key file
func TestConsulConfigWithKeyFile(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		KeyFile: "/etc/consul/key.pem",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, "/etc/consul/key.pem", config.KeyFile)
}

// TestConsulConfigWithAllTLSFiles tests Consul config with all TLS files
func TestConsulConfigWithAllTLSFiles(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		TLS:      true,
		CAFile:   "/etc/consul/ca.pem",
		CertFile: "/etc/consul/cert.pem",
		KeyFile:  "/etc/consul/key.pem",
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.True(t, config.TLS)
	assert.Equal(t, "/etc/consul/ca.pem", config.CAFile)
	assert.Equal(t, "/etc/consul/cert.pem", config.CertFile)
	assert.Equal(t, "/etc/consul/key.pem", config.KeyFile)
}

// TestConsulConfigWithHighPort tests Consul config with high port number
func TestConsulConfigWithHighPort(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Port: 65432,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 65432, config.Port)
}

// TestConsulConfigWithLowPort tests Consul config with low port number
func TestConsulConfigWithLowPort(t *testing.T) {
	t.Parallel()
	config := &ConsulConfig{
		Port: 1,
	}

	err := config.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 1, config.Port)
}

// TestConsulConfigWithMultipleDatacenters tests Consul config with different datacenters
func TestConsulConfigWithMultipleDatacenters(t *testing.T) {
	t.Parallel()
	datacenters := []string{"dc1", "dc2", "dc-us-west", "dc-eu-central"}

	for _, dc := range datacenters {
		config := &ConsulConfig{
			Datacenter: dc,
		}

		err := config.Validate()

		assert.NoError(t, err)
		assert.Equal(t, dc, config.Datacenter)
	}
}

// TestConsulConfigStructure tests Consul config structure
func TestConsulConfigStructure(t *testing.T) {
	t.Parallel()
	config := NewConsulConfig()

	assert.NotNil(t, config.Host)
	assert.NotZero(t, config.Port)
	assert.NotNil(t, config.Datacenter)
	assert.NotNil(t, config.Scheme)
}
