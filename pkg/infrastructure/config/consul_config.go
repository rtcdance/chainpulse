package config

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/hashicorp/consul/api"
)

// ConsulConfig holds Consul configuration
type ConsulConfig struct {
	Address string
	Port    int
	Scheme  string
	Token   core.SecretString
}

// ConsulClient wraps Consul API client
type ConsulClient struct {
	client *api.Client
	config *ConsulConfig
	wg     sync.WaitGroup
}

// NewConsulClient creates a new Consul client
func NewConsulClient(cfg *ConsulConfig) (*ConsulClient, error) {
	if cfg == nil {
		cfg = &ConsulConfig{
			Address: "localhost",
			Port:    8500,
			Scheme:  "http",
		}
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	consulConfig.Scheme = cfg.Scheme
	tokenValue := cfg.Token.Value()
	if tokenValue != "" {
		consulConfig.Token = tokenValue
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	return &ConsulClient{
		client: client,
		config: cfg,
	}, nil
}

// RegisterService registers a service with Consul
func (c *ConsulClient) RegisterService(ctx context.Context, serviceID, serviceName string, port int, tags []string) error {
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Port:    port,
		Tags:    tags,
		Address: "localhost",
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://localhost:%d/health", port),
			Interval: "10s",
			Timeout:  "5s",
		},
	}

	if err := c.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	return nil
}

// DeregisterService deregisters a service from Consul
func (c *ConsulClient) DeregisterService(ctx context.Context, serviceID string) error {
	if err := c.client.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}
	return nil
}

// DiscoverService discovers a service from Consul
func (c *ConsulClient) DiscoverService(ctx context.Context, serviceName string) ([]*api.ServiceEntry, error) {
	entries, _, err := c.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}
	return entries, nil
}

// GetConfig retrieves a configuration value from Consul KV store
func (c *ConsulClient) GetConfig(ctx context.Context, key string) (string, error) {
	pair, _, err := c.client.KV().Get(key, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	if pair == nil {
		return "", fmt.Errorf("config key not found: %s", key)
	}
	return string(pair.Value), nil
}

// SetConfig sets a configuration value in Consul KV store
func (c *ConsulClient) SetConfig(ctx context.Context, key, value string) error {
	_, err := c.client.KV().Put(&api.KVPair{
		Key:   key,
		Value: []byte(value),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// WatchConfig watches for configuration changes
func (c *ConsulClient) WatchConfig(ctx context.Context, key string, handler func(string)) error {
	// Use a goroutine to poll for changes instead of deprecated WatchPlan
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		var lastIndex uint64
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pair, meta, err := c.client.KV().Get(key, &api.QueryOptions{
					WaitIndex: lastIndex,
					WaitTime:  30 * time.Second,
				})
				if err != nil {
					slog.Warn("consul watch error", "error", err)
					continue
				}

				if meta.LastIndex > lastIndex {
					lastIndex = meta.LastIndex
					if pair != nil {
						handler(string(pair.Value))
					}
				}
			}
		}
	}()

	return nil
}

// Health checks Consul health
func (c *ConsulClient) Health(ctx context.Context) error {
	_, err := c.client.Status().Leader()
	if err != nil {
		return fmt.Errorf("consul health check failed: %w", err)
	}
	return nil
}

// Close closes the Consul client connection and waits for watch goroutines to exit
func (c *ConsulClient) Close() error {
	c.wg.Wait()
	return nil
}

// WaitForConsul waits for Consul to be available
func WaitForConsul(ctx context.Context, cfg *ConsulConfig, timeout time.Duration) error {
	client, err := NewConsulClient(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			// Ignore close errors in defer
			_ = err
		}
	}()

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for Consul")
		}

		healthCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := client.Health(healthCtx)
		cancel()

		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}
