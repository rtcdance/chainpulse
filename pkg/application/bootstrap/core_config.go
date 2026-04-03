package bootstrap

import "chainpulse/pkg/core"

// CoreConfigOverrides defines additive deployment-mode overrides.
type CoreConfigOverrides struct {
	APIType      *string
	APIPort      *int
	FeatureFlags map[string]bool
}

// NewAPIServiceCoreConfig creates default core config for api-service mode.
func NewAPIServiceCoreConfig(port int, logLevel string) core.Config {
	return core.Config{
		APIType:      "service",
		APIPort:      port,
		LogLevel:     logLevel,
		FeatureFlags: make(map[string]bool),
	}
}

// NewMonolithicCoreConfig creates default core config for monolithic mode.
func NewMonolithicCoreConfig(logLevel, databaseType, databaseURL, cacheType string) core.Config {
	return core.Config{
		APIType:      "graphql",
		APIPort:      8080,
		LogLevel:     logLevel,
		DatabaseType: databaseType,
		DatabaseURL:  databaseURL,
		CacheType:    cacheType,
		FeatureFlags: make(map[string]bool),
	}
}

// ApplyCoreConfigOverrides applies deployment-specific overrides on top of a base config.
func ApplyCoreConfigOverrides(base core.Config, overrides CoreConfigOverrides) core.Config {
	cfg := base

	if cfg.FeatureFlags == nil {
		cfg.FeatureFlags = make(map[string]bool)
	}

	if overrides.APIType != nil {
		cfg.APIType = *overrides.APIType
	}

	if overrides.APIPort != nil {
		cfg.APIPort = *overrides.APIPort
	}

	for key, value := range overrides.FeatureFlags {
		cfg.FeatureFlags[key] = value
	}

	return cfg
}
