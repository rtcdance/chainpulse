package bootstrap

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	FlagOverrideAPIType      = "--core-api-type"
	FlagOverrideAPIPort      = "--core-api-port"
	FlagOverrideFeatureFlags = "--core-feature-flags"
)

var validCoreAPITypes = map[string]struct{}{
	"rest":      {},
	"grpc":      {},
	"websocket": {},
	"service":   {},
	"graphql":   {},
	"gateway":   {},
}

// ParseCoreConfigOverridesFromCLI parses shared core-config overrides from CLI args.
func ParseCoreConfigOverridesFromCLI(args []string) (CoreConfigOverrides, error) {
	overrides := CoreConfigOverrides{
		FeatureFlags: make(map[string]bool),
	}

	apiTypeRaw, foundType, err := parseStringFlagArg(args, FlagOverrideAPIType)
	if err != nil {
		return CoreConfigOverrides{}, err
	}
	if foundType {
		apiTypeRaw = strings.TrimSpace(apiTypeRaw)
		if apiTypeRaw == "" {
			return CoreConfigOverrides{}, fmt.Errorf("%s cannot be empty", FlagOverrideAPIType)
		}
		if _, ok := validCoreAPITypes[apiTypeRaw]; !ok {
			return CoreConfigOverrides{}, fmt.Errorf("%s has unsupported value %q", FlagOverrideAPIType, apiTypeRaw)
		}
		overrides.APIType = &apiTypeRaw
	}

	apiPortRaw, foundPort, err := parseStringFlagArg(args, FlagOverrideAPIPort)
	if err != nil {
		return CoreConfigOverrides{}, err
	}
	if foundPort {
		port, convErr := strconv.Atoi(strings.TrimSpace(apiPortRaw))
		if convErr != nil {
			return CoreConfigOverrides{}, fmt.Errorf("%s must be an integer: %w", FlagOverrideAPIPort, convErr)
		}
		if port <= 0 || port > 65535 {
			return CoreConfigOverrides{}, fmt.Errorf("%s must be within 1..65535, got %d", FlagOverrideAPIPort, port)
		}
		overrides.APIPort = &port
	}

	featureRaw, foundFlags, err := parseStringFlagArg(args, FlagOverrideFeatureFlags)
	if err != nil {
		return CoreConfigOverrides{}, err
	}
	if foundFlags {
		flags, parseErr := parseFeatureFlagsOverride(featureRaw)
		if parseErr != nil {
			return CoreConfigOverrides{}, fmt.Errorf("%s parse error: %w", FlagOverrideFeatureFlags, parseErr)
		}
		overrides.FeatureFlags = flags
	}

	return overrides, nil
}

// MergeCoreConfigOverrides merges low and high precedence overrides.
func MergeCoreConfigOverrides(low, high CoreConfigOverrides) CoreConfigOverrides {
	out := CoreConfigOverrides{
		FeatureFlags: make(map[string]bool),
	}

	if low.APIType != nil {
		value := *low.APIType
		out.APIType = &value
	}
	if high.APIType != nil {
		value := *high.APIType
		out.APIType = &value
	}

	if low.APIPort != nil {
		value := *low.APIPort
		out.APIPort = &value
	}
	if high.APIPort != nil {
		value := *high.APIPort
		out.APIPort = &value
	}

	for key, value := range low.FeatureFlags {
		out.FeatureFlags[key] = value
	}
	for key, value := range high.FeatureFlags {
		out.FeatureFlags[key] = value
	}

	return out
}

func parseStringFlagArg(args []string, flagName string) (string, bool, error) {
	for i := 0; i < len(args); i++ {
		current := args[i]
		if current == flagName {
			if i+1 >= len(args) {
				return "", false, fmt.Errorf("%s requires a value", flagName)
			}
			return args[i+1], true, nil
		}

		prefix := flagName + "="
		if strings.HasPrefix(current, prefix) {
			return strings.TrimPrefix(current, prefix), true, nil
		}
	}

	return "", false, nil
}
