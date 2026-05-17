package core

import "fmt"

func checkRequired(val string, name string) error {
	if val == "" {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError, name+" is required", nil)
	}
	return nil
}

func checkOneOf(val string, allowed []string, name string) error {
	for _, a := range allowed {
		if val == a {
			return nil
		}
	}
	return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
		fmt.Sprintf("invalid %s: %s (allowed: %v)", name, val, allowed), nil)
}

func checkPositive(val int, name string) error {
	if val <= 0 {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			name+" must be greater than 0", nil)
	}
	return nil
}

func checkPort(val int, name string) error {
	if val <= 0 || val > 65535 {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("invalid %s: %d (must be 1-65535)", name, val), nil)
	}
	return nil
}

func (cm *DefaultConfigManager) Validate(config Config) error {
	checks := []func() error{
		func() error { return checkRequired(config.DataPullerType, "DataPullerType") },
		func() error {
			return checkOneOf(config.DataPullerType, []string{"https-jsonrpc", "websocket", "grpc"}, "DataPullerType")
		},
		func() error { return checkRequired(config.BlockchainNodeURL, "BlockchainNodeURL") },

		func() error { return checkRequired(config.MQType, "MQType") },
		func() error { return checkOneOf(config.MQType, []string{"kafka", "redis", "zeromq"}, "MQType") },
		func() error { return checkRequired(config.MQConnectionURL, "MQConnectionURL") },

		func() error { return checkRequired(config.CacheType, "CacheType") },
		func() error { return checkOneOf(config.CacheType, []string{"redis", "memory"}, "CacheType") },
		func() error { return checkPositive(config.CacheTTL, "CacheTTL") },

		func() error { return checkRequired(config.DatabaseType, "DatabaseType") },
		func() error { return checkOneOf(config.DatabaseType, []string{"postgres", "mongodb"}, "DatabaseType") },
		func() error { return checkRequired(config.DatabaseURL, "DatabaseURL") },

		func() error { return checkRequired(config.APIType, "APIType") },
		func() error { return checkOneOf(config.APIType, []string{"rest", "grpc", "websocket"}, "APIType") },
		func() error { return checkPort(config.APIPort, "APIPort") },

		func() error { return checkPositive(config.WorkerPoolSize, "WorkerPoolSize") },
		func() error { return checkPositive(config.BatchSize, "BatchSize") },
		func() error {
			if config.MaxRetries < 0 {
				return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError, "MaxRetries must be non-negative", nil)
			}
			return nil
		},
		func() error { return checkPositive(config.RetryBackoff, "RetryBackoff") },

		func() error { return checkRequired(config.DeploymentMode, "DeploymentMode") },
		func() error {
			return checkOneOf(config.DeploymentMode, []string{"monolithic", "microservice"}, "DeploymentMode")
		},
		func() error { return checkRequired(config.ServiceName, "ServiceName") },

		func() error { return checkRequired(config.LogLevel, "LogLevel") },
		func() error {
			return checkOneOf(config.LogLevel, []string{"debug", "info", "warn", "error", "fatal"}, "LogLevel")
		},
	}

	for _, c := range checks {
		if err := c(); err != nil {
			return err
		}
	}
	return nil
}
