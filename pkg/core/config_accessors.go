package core

import "fmt"

type configAccessor struct {
	get func(*Config) (any, error)
	set func(*Config, any) error
}

func errTypeError(key, want string) error {
	return NewSystemError(
		ErrorTypePermanent,
		ErrorCodeValidation,
		fmt.Sprintf("config %q must be %s", key, want),
		nil,
	)
}

// configFields maps config key names to get/set accessors.
// Fields not registered here are not accessible via Get/Set but can still be
// read/written directly on the Config struct.
var configFields = map[string]configAccessor{
	"data_puller_type": {
		get: func(c *Config) (any, error) { return c.DataPullerType, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("data_puller_type", "string")
			}
			c.DataPullerType = s
			return nil
		},
	},
	"blockchain_node_url": {
		get: func(c *Config) (any, error) { return c.BlockchainNodeURL, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("blockchain_node_url", "string")
			}
			c.BlockchainNodeURL = s
			return nil
		},
	},
	"mq_type": {
		get: func(c *Config) (any, error) { return c.MQType, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("mq_type", "string")
			}
			c.MQType = s
			return nil
		},
	},
	"cache_type": {
		get: func(c *Config) (any, error) { return c.CacheType, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("cache_type", "string")
			}
			c.CacheType = s
			return nil
		},
	},
	"database_type": {
		get: func(c *Config) (any, error) { return c.DatabaseType, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("database_type", "string")
			}
			c.DatabaseType = s
			return nil
		},
	},
	"api_type": {
		get: func(c *Config) (any, error) { return c.APIType, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("api_type", "string")
			}
			c.APIType = s
			return nil
		},
	},
	"api_port": {
		get: func(c *Config) (any, error) { return c.APIPort, nil },
		set: func(c *Config, v any) error {
			i, ok := v.(int)
			if !ok {
				return errTypeError("api_port", "int")
			}
			c.APIPort = i
			return nil
		},
	},
	"worker_pool_size": {
		get: func(c *Config) (any, error) { return c.WorkerPoolSize, nil },
		set: func(c *Config, v any) error {
			i, ok := v.(int)
			if !ok {
				return errTypeError("worker_pool_size", "int")
			}
			c.WorkerPoolSize = i
			return nil
		},
	},
	"batch_size": {
		get: func(c *Config) (any, error) { return c.BatchSize, nil },
		set: func(c *Config, v any) error {
			i, ok := v.(int)
			if !ok {
				return errTypeError("batch_size", "int")
			}
			c.BatchSize = i
			return nil
		},
	},
	"max_retries": {
		get: func(c *Config) (any, error) { return c.MaxRetries, nil },
		set: func(c *Config, v any) error {
			i, ok := v.(int)
			if !ok {
				return errTypeError("max_retries", "int")
			}
			c.MaxRetries = i
			return nil
		},
	},
	"deployment_mode": {
		get: func(c *Config) (any, error) { return c.DeploymentMode, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("deployment_mode", "string")
			}
			c.DeploymentMode = s
			return nil
		},
	},
	"log_level": {
		get: func(c *Config) (any, error) { return c.LogLevel, nil },
		set: func(c *Config, v any) error {
			s, ok := v.(string)
			if !ok {
				return errTypeError("log_level", "string")
			}
			c.LogLevel = s
			return nil
		},
	},
}
