package bootstrap

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	// EnvOverrideAPIType sets APIType override.
	EnvOverrideAPIType = "CHAINPULSE_CORE_API_TYPE"
	// EnvOverrideAPIPort sets APIPort override.
	EnvOverrideAPIPort = "CHAINPULSE_CORE_API_PORT"
	// EnvOverrideFeatureFlags sets feature flag overrides in key=value,key2=value2 format.
	EnvOverrideFeatureFlags = "CHAINPULSE_CORE_FEATURE_FLAGS"
)

// ParseCoreConfigOverridesFromEnv parses and validates shared core-config overrides from environment.
func ParseCoreConfigOverridesFromEnv() (CoreConfigOverrides, error) {
	overrides := CoreConfigOverrides{
		FeatureFlags: make(map[string]bool),
	}

	if rawType := strings.TrimSpace(os.Getenv(EnvOverrideAPIType)); rawType != "" {
		apiType := rawType
		overrides.APIType = &apiType
	}

	if rawPort := strings.TrimSpace(os.Getenv(EnvOverrideAPIPort)); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return CoreConfigOverrides{}, fmt.Errorf("%s must be an integer: %w", EnvOverrideAPIPort, err)
		}
		if port <= 0 || port > 65535 {
			return CoreConfigOverrides{}, fmt.Errorf("%s must be within 1..65535, got %d", EnvOverrideAPIPort, port)
		}
		overrides.APIPort = &port
	}

	if rawFlags := strings.TrimSpace(os.Getenv(EnvOverrideFeatureFlags)); rawFlags != "" {
		flags, err := parseFeatureFlagsOverride(rawFlags)
		if err != nil {
			return CoreConfigOverrides{}, err
		}
		overrides.FeatureFlags = flags
	}

	return overrides, nil
}

// SummarizeCoreConfigOverrides creates a deterministic audit summary for startup logs.
func SummarizeCoreConfigOverrides(overrides CoreConfigOverrides) string {
	parts := make([]string, 0, 3)

	if overrides.APIType != nil {
		parts = append(parts, fmt.Sprintf("api_type=%s", *overrides.APIType))
	}

	if overrides.APIPort != nil {
		parts = append(parts, fmt.Sprintf("api_port=%d", *overrides.APIPort))
	}

	if len(overrides.FeatureFlags) > 0 {
		keys := make([]string, 0, len(overrides.FeatureFlags))
		for key := range overrides.FeatureFlags {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		flagPairs := make([]string, 0, len(keys))
		for _, key := range keys {
			flagPairs = append(flagPairs, fmt.Sprintf("%s=%t", key, overrides.FeatureFlags[key]))
		}
		parts = append(parts, fmt.Sprintf("feature_flags=[%s]", strings.Join(flagPairs, ",")))
	}

	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, " ")
}

func parseFeatureFlagsOverride(raw string) (map[string]bool, error) {
	flags := make(map[string]bool)
	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		pair := strings.TrimSpace(entry)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s expects key=value pairs, got %q", EnvOverrideFeatureFlags, pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("%s has empty feature key in %q", EnvOverrideFeatureFlags, pair)
		}

		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("%s has invalid bool for %s: %w", EnvOverrideFeatureFlags, key, err)
		}
		flags[key] = enabled
	}

	return flags, nil
}
