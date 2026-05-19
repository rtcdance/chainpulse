package core

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

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

func checkURLScheme(val string, expectedScheme string, name string) error {
	if val == "" {
		return nil // checkRequired handles empty
	}
	parsed, err := url.Parse(val)
	if err != nil {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("invalid %s: %s is not a valid URL (%v)", name, val, err), nil)
	}
	if expectedScheme != "" && parsed.Scheme != expectedScheme {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("invalid %s scheme: got %s, expected %s", name, parsed.Scheme, expectedScheme), nil)
	}
	if parsed.Host == "" {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("invalid %s: %s has no host", name, val), nil)
	}
	return nil
}

func checkFileExists(path string, name string) error {
	if path == "" {
		return nil // checkRequired handles empty
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
				fmt.Sprintf("%s file not found: %s", name, path), nil)
		}
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("%s stat error: %s (%v)", name, path, err), nil)
	}
	if info.IsDir() {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("%s is a directory, not a file: %s", name, path), nil)
	}
	return nil
}

func checkMinLength(val string, minLen int, name string) error {
	if len(val) < minLen {
		return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError,
			fmt.Sprintf("%s must be at least %d characters (got %d)", name, minLen, len(val)), nil)
	}
	return nil
}

func (cm *DefaultConfigManager) Validate(config Config) error {
	checks := []func() error{
		// Data puller
		func() error { return checkRequired(config.DataPullerType, "DataPullerType") },
		func() error {
			return checkOneOf(config.DataPullerType, []string{"https-jsonrpc", "websocket", "grpc"}, "DataPullerType")
		},
		func() error { return checkRequired(config.BlockchainNodeURL, "BlockchainNodeURL") },
		func() error { return checkURLScheme(config.BlockchainNodeURL, "", "BlockchainNodeURL") },

		// MQ
		func() error { return checkRequired(config.MQType, "MQType") },
		func() error { return checkOneOf(config.MQType, []string{"kafka"}, "MQType") },
		func() error { return checkRequired(config.MQConnectionURL, "MQConnectionURL") },
		func() error { return checkURLScheme(config.MQConnectionURL, "", "MQConnectionURL") },

		// Cache
		func() error { return checkRequired(config.CacheType, "CacheType") },
		func() error { return checkOneOf(config.CacheType, []string{"redis", "memory"}, "CacheType") },
		func() error { return checkPositive(config.CacheTTL, "CacheTTL") },

		// Database
		func() error { return checkRequired(config.DatabaseType, "DatabaseType") },
		func() error { return checkOneOf(config.DatabaseType, []string{"postgres", "mongodb"}, "DatabaseType") },
		func() error { return checkRequired(config.DatabaseURL, "DatabaseURL") },
		func() error { return checkURLScheme(config.DatabaseURL, "", "DatabaseURL") },

		// API
		func() error { return checkRequired(config.APIType, "APIType") },
		func() error { return checkOneOf(config.APIType, []string{"rest", "grpc", "websocket"}, "APIType") },
		func() error { return checkPort(config.APIPort, "APIPort") },

		// Processing
		func() error { return checkPositive(config.WorkerPoolSize, "WorkerPoolSize") },
		func() error { return checkPositive(config.BatchSize, "BatchSize") },
		func() error {
			if config.MaxRetries < 0 {
				return NewSystemError(ErrorTypePermanent, ErrorCodeConfigError, "MaxRetries must be non-negative", nil)
			}
			return nil
		},
		func() error { return checkPositive(config.RetryBackoff, "RetryBackoff") },

		// Deployment
		func() error { return checkRequired(config.DeploymentMode, "DeploymentMode") },
		func() error {
			return checkOneOf(config.DeploymentMode, []string{"monolithic", "microservice"}, "DeploymentMode")
		},
		func() error { return checkRequired(config.ServiceName, "ServiceName") },

		// Logging
		func() error { return checkRequired(config.LogLevel, "LogLevel") },
		func() error {
			return checkOneOf(config.LogLevel, []string{"debug", "info", "warn", "error", "fatal"}, "LogLevel")
		},

		// Cross-field: JWT secrets must be 32+ chars
		func() error {
			secret := strings.TrimSpace(config.PostgresPassword.Value())
			if secret != "" {
				return checkMinLength(secret, 32, "PostgresPassword")
			}
			return nil
		},

		// Cross-field: TLS cert+key pairs
		func() error {
			if config.TLSCertPath != "" || config.TLSKeyPath != "" {
				if err := checkRequired(config.TLSCertPath, "TLSCertPath"); err != nil {
					return err
				}
				if err := checkRequired(config.TLSKeyPath, "TLSKeyPath"); err != nil {
					return err
				}
				if err := checkFileExists(config.TLSCertPath, "TLSCertPath"); err != nil {
					return err
				}
				if err := checkFileExists(config.TLSKeyPath, "TLSKeyPath"); err != nil {
					return err
				}
			}
			return nil
		},
	}

	for _, c := range checks {
		if err := c(); err != nil {
			return err
		}
	}
	return nil
}
