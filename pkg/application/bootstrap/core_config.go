package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
)

// CoreConfigOverrides defines additive deployment-mode overrides.
// Deprecated: Use ApplyConfigOverrides instead.
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

// NewMonolithicCoreConfig creates default core config for monolithic mode,
// reading all required fields from environment variables with fallback defaults.
func NewMonolithicCoreConfig(logLevel, databaseType, databaseURL, cacheType, dataPullerType, blockchainNodeURL string) core.Config {
	if dataPullerType == "" {
		dataPullerType = env.Get("DATA_PULLER_TYPE", "https-jsonrpc")
	}
	if blockchainNodeURL == "" {
		blockchainNodeURL = env.Get("BLOCKCHAIN_NODE_URL", "http://localhost:8545")
	}

	cfg := core.Config{
		APIType:           env.Get("API_TYPE", "rest"),
		APIPort:           env.GetInt("API_PORT", 8080),
		LogLevel:          logLevel,
		DatabaseType:      databaseType,
		DatabaseURL:       databaseURL,
		CacheType:         cacheType,
		DataPullerType:    dataPullerType,
		BlockchainNodeURL: blockchainNodeURL,
		FeatureFlags:      make(map[string]bool),
	}

	// Populate remaining required fields from env vars
	cfg.MQType = env.Get("MQ_TYPE", "redis")
	cfg.MQConnectionURL = env.Get("MQ_CONNECTION_URL", "localhost:6379")
	cfg.CacheConnectionURL = env.Get("CACHE_CONNECTION_URL", "localhost:6379")
	cfg.DeploymentMode = env.Get("DEPLOYMENT_MODE", "monolithic")
	cfg.ServiceName = env.Get("SERVICE_NAME", "chainpulse")
	cfg.ChainID = env.Get("CHAIN_ID", "1")
	cfg.CacheTTL = env.GetInt("CACHE_TTL", 3600)
	cfg.WorkerPoolSize = env.GetInt("WORKER_POOL_SIZE", 8)
	cfg.BatchSize = env.GetInt("BATCH_SIZE", 100)
	cfg.MaxRetries = env.GetInt("MAX_RETRIES", 3)
	cfg.RetryBackoff = env.GetInt("RETRY_BACKOFF", 100)
	cfg.IdempotencyRecordTTL = env.GetInt("IDEMPOTENCY_RECORD_TTL", 86400)
	cfg.IdempotencyCleanupInterval = env.GetInt("IDEMPOTENCY_CLEANUP_INTERVAL", 600)
	cfg.StartBlock = uint64(env.GetInt("START_BLOCK", 0))

	return cfg
}

// =============================================================================
// Simplified config overrides — replaces the previous ~1000-line policy system.
// Just reads CHAINPULSE_CORE_* env vars and applies them directly.
// =============================================================================

// CoreOverrideKeyPolicy describes per-key runtime override policy constraints.
// Deprecated: Retained for backward compatibility with old policy system.
type CoreOverrideKeyPolicy struct {
	AllowedAPITypes               map[string]struct{}
	MinAPIPort                    int
	MaxAPIPort                    int
	DisallowedEnabledFeatureFlags map[string]struct{}
}

// OverridePolicyRuntime describes the resolved override policy at startup.
// Deprecated: Retained for backward compatibility with old policy system.
type OverridePolicyRuntime struct {
	Preset        string
	PresetSource  string
	AllowProfiles map[string]struct{}
	KeyPolicy     CoreOverrideKeyPolicy
}

// OverridePolicyEvaluation captures runtime decision for enforcement/audit workflows.
// Deprecated: Retained for backward compatibility with old policy system.
type OverridePolicyEvaluation struct {
	EnforcementMode string
	ViolationCode   string
	Violation       bool
	Blocked         bool
}

// ApplyConfigOverrides applies CHAINPULSE_CORE_* environment variable overrides
// to the given config in-place. This replaces the previous multi-file override
// + policy + metrics system with a single function.
//
// Supported env vars:
//
//	CHAINPULSE_CORE_API_TYPE       — override API type (rest, grpc, graphql, etc.)
//	CHAINPULSE_CORE_API_PORT       — override API port (1..65535)
//	CHAINPULSE_CORE_FEATURE_FLAGS  — comma-separated key=bool pairs
func ApplyConfigOverrides(cfg *core.Config) {
	if cfg == nil {
		return
	}
	if cfg.FeatureFlags == nil {
		cfg.FeatureFlags = make(map[string]bool)
	}

	if rawType := strings.TrimSpace(os.Getenv("CHAINPULSE_CORE_API_TYPE")); rawType != "" {
		cfg.APIType = rawType
	}

	if rawPort := strings.TrimSpace(os.Getenv("CHAINPULSE_CORE_API_PORT")); rawPort != "" {
		if port, err := strconv.Atoi(rawPort); err == nil && port > 0 && port <= 65535 {
			cfg.APIPort = port
		}
	}

	if rawFlags := strings.TrimSpace(os.Getenv("CHAINPULSE_CORE_FEATURE_FLAGS")); rawFlags != "" {
		for _, pair := range strings.Split(rawFlags, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				if v, err := strconv.ParseBool(strings.TrimSpace(parts[1])); err == nil {
					cfg.FeatureFlags[strings.TrimSpace(parts[0])] = v
				}
			}
		}
	}
}

// =============================================================================
// Backward-compatible wrappers — delegate to ApplyConfigOverrides.
// These are kept so existing callers (main.go, microservices) continue to compile.
// New code should call ApplyConfigOverrides directly.
// =============================================================================

// ParseCoreConfigOverridesFromEnv reads override env vars into a CoreConfigOverrides struct.
// Deprecated: Call ApplyConfigOverrides directly instead.
func ParseCoreConfigOverridesFromEnv() (CoreConfigOverrides, error) {
	overrides := CoreConfigOverrides{
		FeatureFlags: make(map[string]bool),
	}

	if rawType := strings.TrimSpace(os.Getenv("CHAINPULSE_CORE_API_TYPE")); rawType != "" {
		overrides.APIType = &rawType
	}

	if rawPort := strings.TrimSpace(os.Getenv("CHAINPULSE_CORE_API_PORT")); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return CoreConfigOverrides{}, fmt.Errorf("CHAINPULSE_CORE_API_PORT must be an integer: %w", err)
		}
		if port <= 0 || port > 65535 {
			return CoreConfigOverrides{}, fmt.Errorf("CHAINPULSE_CORE_API_PORT must be 1..65535, got %d", port)
		}
		overrides.APIPort = &port
	}

	if rawFlags := strings.TrimSpace(os.Getenv("CHAINPULSE_CORE_FEATURE_FLAGS")); rawFlags != "" {
		flags, err := parseFeatureFlags(rawFlags)
		if err != nil {
			return CoreConfigOverrides{}, err
		}
		overrides.FeatureFlags = flags
	}

	return overrides, nil
}

// ParseCoreConfigOverridesFromCLI reads override CLI flags into a CoreConfigOverrides struct.
// Deprecated: Call ApplyConfigOverrides directly instead.
func ParseCoreConfigOverridesFromCLI(args []string) (CoreConfigOverrides, error) {
	overrides := CoreConfigOverrides{
		FeatureFlags: make(map[string]bool),
	}

	apiTypeRaw, foundType, err := parseStringFlagArg(args, "--core-api-type")
	if err != nil {
		return CoreConfigOverrides{}, err
	}
	if foundType {
		apiTypeRaw = strings.TrimSpace(apiTypeRaw)
		if apiTypeRaw == "" {
			return CoreConfigOverrides{}, fmt.Errorf("--core-api-type cannot be empty")
		}
		overrides.APIType = &apiTypeRaw
	}

	apiPortRaw, foundPort, err := parseStringFlagArg(args, "--core-api-port")
	if err != nil {
		return CoreConfigOverrides{}, err
	}
	if foundPort {
		port, convErr := strconv.Atoi(strings.TrimSpace(apiPortRaw))
		if convErr != nil {
			return CoreConfigOverrides{}, fmt.Errorf("--core-api-port must be an integer: %w", convErr)
		}
		if port <= 0 || port > 65535 {
			return CoreConfigOverrides{}, fmt.Errorf("--core-api-port must be 1..65535, got %d", port)
		}
		overrides.APIPort = &port
	}

	featureRaw, foundFlags, err := parseStringFlagArg(args, "--core-feature-flags")
	if err != nil {
		return CoreConfigOverrides{}, err
	}
	if foundFlags {
		flags, flagErr := parseFeatureFlags(featureRaw)
		if flagErr != nil {
			return CoreConfigOverrides{}, flagErr
		}
		overrides.FeatureFlags = flags
	}

	return overrides, nil
}

// MergeCoreConfigOverrides merges env and CLI overrides, with CLI taking precedence.
// Deprecated: Call ApplyConfigOverrides directly instead.
func MergeCoreConfigOverrides(envOverrides, cliOverrides CoreConfigOverrides) CoreConfigOverrides {
	merged := CoreConfigOverrides{
		FeatureFlags: make(map[string]bool),
	}

	if envOverrides.APIType != nil {
		merged.APIType = envOverrides.APIType
	}
	if cliOverrides.APIType != nil {
		merged.APIType = cliOverrides.APIType
	}

	if envOverrides.APIPort != nil {
		merged.APIPort = envOverrides.APIPort
	}
	if cliOverrides.APIPort != nil {
		merged.APIPort = cliOverrides.APIPort
	}

	for k, v := range envOverrides.FeatureFlags {
		merged.FeatureFlags[k] = v
	}
	for k, v := range cliOverrides.FeatureFlags {
		merged.FeatureFlags[k] = v
	}

	return merged
}

// ApplyCoreConfigOverrides applies deployment-specific overrides on top of a base config.
// Deprecated: Call ApplyConfigOverrides directly instead.
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

// SummarizeCoreConfigOverrides returns a human-readable summary of overrides.
// Deprecated: Used only for startup log messages.
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

// RuntimeProfileFromEnv returns a simplified runtime profile.
// Deprecated: Part of the old policy system. Kept for backward compatibility.
func RuntimeProfileFromEnv() string {
	profile := strings.TrimSpace(os.Getenv("CHAINPULSE_ENV"))
	if profile == "" {
		profile = strings.TrimSpace(os.Getenv("APP_ENV"))
	}
	if profile == "" {
		profile = strings.TrimSpace(os.Getenv("ENV"))
	}
	if profile == "" {
		return "development"
	}
	profile = strings.ToLower(strings.TrimSpace(profile))
	switch profile {
	case "prod", "production":
		return "production"
	case "staging", "stage":
		return "staging"
	case "test", "testing":
		return "testing"
	default:
		return "development"
	}
}

// ResolveOverridePolicyRuntime returns a stub policy that allows everything.
// Deprecated: Part of the old policy system. Policies are no longer enforced.
func ResolveOverridePolicyRuntime(string) OverridePolicyRuntime {
	return OverridePolicyRuntime{
		Preset: "open",
		KeyPolicy: CoreOverrideKeyPolicy{
			AllowedAPITypes: map[string]struct{}{
				"rest": {}, "grpc": {}, "websocket": {},
				"service": {}, "graphql": {}, "gateway": {},
			},
			MinAPIPort: 1024,
			MaxAPIPort: 65535,
		},
	}
}

// ResolvePolicyEnforcementModeFromEnv returns "audit" mode (no enforcement).
// Deprecated: Part of the old policy system.
func ResolvePolicyEnforcementModeFromEnv() string {
	return "audit"
}

// ValidateCoreConfigOverridesWithMode always passes validation.
// Deprecated: Part of the old policy system.
func ValidateCoreConfigOverridesWithMode(CoreConfigOverrides, string, OverridePolicyRuntime, string) (OverridePolicyEvaluation, error) {
	return OverridePolicyEvaluation{EnforcementMode: "audit"}, nil
}

// PolicyErrorCode always returns "" as policies are no longer enforced.
// Deprecated: Part of the old policy system.
func PolicyErrorCode(error) string {
	return ""
}

// ResolvePolicyMetricSchemaModeFromEnv always returns "v1".
// Deprecated: Part of the old policy system.
func ResolvePolicyMetricSchemaModeFromEnv() string {
	return "v1"
}

// EmitPolicyOverrideMetrics is a no-op.
// Deprecated: Part of the old policy system.
func EmitPolicyOverrideMetrics(core.MetricsCollector, string, CoreConfigOverrides, CoreConfigOverrides, CoreConfigOverrides, OverridePolicyRuntime, OverridePolicyEvaluation, string) {
}

// =============================================================================
// Internal helpers
// =============================================================================

func parseFeatureFlags(raw string) (map[string]bool, error) {
	flags := make(map[string]bool)
	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		pair := strings.TrimSpace(entry)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid feature flag pair: %q", pair)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("empty feature key in %q", pair)
		}
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid bool for %s: %w", key, err)
		}
		flags[key] = enabled
	}
	return flags, nil
}

func parseStringFlagArg(args []string, flag string) (string, bool, error) {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1], true, nil
		}
	}
	return "", false, nil
}
