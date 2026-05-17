package deployment

import "github.com/rtcdance/chainpulse/pkg/core"

// ConsulConfig holds Consul configuration
type ConsulConfig struct {
	Host               string
	Port               int
	Datacenter         string
	Token              core.SecretString
	Scheme             string
	TLS                bool
	InsecureSkipVerify bool
	CAFile             string
	CertFile           string
	KeyFile            string
	ConnectTimeout     int
	ReadTimeout        int
	WriteTimeout       int
	MaxRetries         int
	RetryWaitMin       int
	RetryWaitMax       int
}

// NewConsulConfig creates a new Consul configuration with defaults
func NewConsulConfig() *ConsulConfig {
	return &ConsulConfig{
		Host:           "localhost",
		Port:           8500,
		Datacenter:     "dc1",
		Scheme:         "http",
		TLS:            false,
		ConnectTimeout: 5,
		ReadTimeout:    10,
		WriteTimeout:   10,
		MaxRetries:     3,
		RetryWaitMin:   100,
		RetryWaitMax:   1000,
	}
}

// Validate validates the Consul configuration
func (c *ConsulConfig) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8500
	}
	if c.Datacenter == "" {
		c.Datacenter = "dc1"
	}
	if c.Scheme == "" {
		c.Scheme = "http"
	}
	return nil
}
