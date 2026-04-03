package bootstrap

import (
	"chainpulse/pkg/core"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	EnvChainPulseEnv               = "CHAINPULSE_ENV"
	EnvAppEnv                      = "APP_ENV"
	EnvEnv                         = "ENV"
	EnvOverridePolicyAllowProfiles = "CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES"
	EnvOverridePolicyPreset        = "CHAINPULSE_OVERRIDE_POLICY_PRESET"
	EnvOverridePolicyEnforcement   = "CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT"
	EnvPolicyMetricSchemaMode      = "CHAINPULSE_POLICY_METRIC_SCHEMA_MODE"

	PolicyPresetStrict   = "strict"
	PolicyPresetBalanced = "balanced"
	PolicyPresetOpen     = "open"

	PolicyEnforcementEnforce = "enforce"
	PolicyEnforcementAudit   = "audit"

	PolicyMetricSchemaV1        = "v1"
	PolicyMetricSchemaDualWrite = "dual_write"
	PolicyMetricSchemaV2        = "v2"

	PolicyMetricOverridesAppliedV1 = "core_config_overrides_applied_total"
	PolicyMetricEvaluationV1       = "core_config_overrides_policy_evaluation_total"
	PolicyMetricOverridesAppliedV2 = "chainpulse_policy_overrides_applied_total"
	PolicyMetricEvaluationV2       = "chainpulse_policy_overrides_evaluation_total"

	PolicyErrProfileNotAllowlisted = "POLICY_PROFILE_NOT_ALLOWLISTED"
	PolicyErrAPITypeDenied         = "POLICY_API_TYPE_DENIED"
	PolicyErrAPIPortOutOfRange     = "POLICY_API_PORT_OUT_OF_RANGE"
	PolicyErrFeatureFlagDenied     = "POLICY_FEATURE_FLAG_DENIED"
)

var productionDisallowedFeatureFlags = map[string]struct{}{
	"dev_mode":                    {},
	"enable_unsafe_debug":         {},
	"experimental_runtime_routes": {},
}

// CoreOverrideKeyPolicy describes per-key runtime override policy constraints.
type CoreOverrideKeyPolicy struct {
	AllowedAPITypes               map[string]struct{}
	MinAPIPort                    int
	MaxAPIPort                    int
	DisallowedEnabledFeatureFlags map[string]struct{}
}

// OverridePolicyRuntime describes the resolved override policy at startup.
type OverridePolicyRuntime struct {
	Preset        string
	PresetSource  string
	AllowProfiles map[string]struct{}
	KeyPolicy     CoreOverrideKeyPolicy
}

// OverridePolicyEvaluation captures runtime decision for enforcement/audit workflows.
type OverridePolicyEvaluation struct {
	EnforcementMode string
	ViolationCode   string
	Violation       bool
	Blocked         bool
}

// PolicyMetricSchemaPlan describes which metric schemas should be emitted.
type PolicyMetricSchemaPlan struct {
	Mode       string
	EmitV1     bool
	EmitV2     bool
	Deprecated bool
}

// OverridePolicyError is a machine-readable policy validation error.
type OverridePolicyError struct {
	Code    string
	Message string
}

func (e *OverridePolicyError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// RuntimeProfileFromEnv returns normalized runtime profile.
func RuntimeProfileFromEnv() string {
	profile := strings.TrimSpace(os.Getenv(EnvChainPulseEnv))
	if profile == "" {
		profile = strings.TrimSpace(os.Getenv(EnvAppEnv))
	}
	if profile == "" {
		profile = strings.TrimSpace(os.Getenv(EnvEnv))
	}
	if profile == "" {
		return "development"
	}
	return normalizeRuntimeProfile(profile)
}

// ValidateCoreConfigOverridesForProfile validates overrides against profile-level policy.
func ValidateCoreConfigOverridesForProfile(overrides CoreConfigOverrides, profile string) error {
	policy := ResolveOverridePolicyRuntime(profile)
	return ValidateCoreConfigOverridesWithPolicy(overrides, profile, policy)
}

// ValidateCoreConfigOverridesWithPolicy validates overrides against full runtime policy.
func ValidateCoreConfigOverridesWithPolicy(
	overrides CoreConfigOverrides,
	profile string,
	policy OverridePolicyRuntime,
) error {
	return validateCoreConfigOverridesWithConstraints(overrides, profile, policy.AllowProfiles, policy.KeyPolicy)
}

// ValidateCoreConfigOverridesWithMode applies enforcement/audit behavior on top of policy validation.
func ValidateCoreConfigOverridesWithMode(
	overrides CoreConfigOverrides,
	profile string,
	policy OverridePolicyRuntime,
	mode string,
) (OverridePolicyEvaluation, error) {
	enforcementMode := normalizePolicyEnforcementMode(mode)
	evaluation := OverridePolicyEvaluation{
		EnforcementMode: enforcementMode,
	}

	err := ValidateCoreConfigOverridesWithPolicy(overrides, profile, policy)
	if err == nil {
		return evaluation, nil
	}

	evaluation.Violation = true
	evaluation.ViolationCode = PolicyErrorCode(err)
	if enforcementMode == PolicyEnforcementAudit {
		return evaluation, nil
	}

	evaluation.Blocked = true
	return evaluation, err
}

// ValidateCoreConfigOverridesForProfileWithAllowlist validates overrides against profile-level policy and optional profile allowlist.
func ValidateCoreConfigOverridesForProfileWithAllowlist(
	overrides CoreConfigOverrides,
	profile string,
	allowProfiles map[string]struct{},
) error {
	policy := ResolveOverridePolicyRuntime(profile)
	return validateCoreConfigOverridesWithConstraints(overrides, profile, allowProfiles, policy.KeyPolicy)
}

func validateCoreConfigOverridesWithConstraints(
	overrides CoreConfigOverrides,
	profile string,
	allowProfiles map[string]struct{},
	keyPolicy CoreOverrideKeyPolicy,
) error {
	profile = normalizeRuntimeProfile(profile)
	if !isProductionProfile(profile) {
		if hasAnyOverride(overrides) && len(allowProfiles) > 0 {
			if _, ok := allowProfiles[profile]; !ok {
				return &OverridePolicyError{
					Code:    PolicyErrProfileNotAllowlisted,
					Message: fmt.Sprintf("overrides are not allowlisted for runtime profile %q", profile),
				}
			}
		}
	}

	if overrides.APIType != nil {
		normalized := strings.ToLower(strings.TrimSpace(*overrides.APIType))
		if _, allowed := keyPolicy.AllowedAPITypes[normalized]; !allowed {
			return &OverridePolicyError{
				Code:    PolicyErrAPITypeDenied,
				Message: fmt.Sprintf("override APIType=%q is not allowed for runtime profile %q", normalized, profile),
			}
		}
	}

	if overrides.APIPort != nil {
		if *overrides.APIPort < keyPolicy.MinAPIPort || *overrides.APIPort > keyPolicy.MaxAPIPort {
			return &OverridePolicyError{
				Code:    PolicyErrAPIPortOutOfRange,
				Message: fmt.Sprintf("override APIPort=%d is outside allowed range %d..%d", *overrides.APIPort, keyPolicy.MinAPIPort, keyPolicy.MaxAPIPort),
			}
		}
	}

	for key, value := range overrides.FeatureFlags {
		if !value {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(key))
		if _, blocked := keyPolicy.DisallowedEnabledFeatureFlags[normalized]; blocked {
			return &OverridePolicyError{
				Code:    PolicyErrFeatureFlagDenied,
				Message: fmt.Sprintf("feature flag override %q=true is not allowed for runtime profile %q", normalized, profile),
			}
		}
	}

	return nil
}

// ParseOverridePolicyAllowProfilesFromEnv parses optional runtime profiles that are allowed to apply non-production overrides.
func ParseOverridePolicyAllowProfilesFromEnv() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv(EnvOverridePolicyAllowProfiles))
	return ParseOverridePolicyAllowProfiles(raw)
}

// ResolvePolicyEnforcementModeFromEnv resolves policy enforcement mode with safe default.
func ResolvePolicyEnforcementModeFromEnv() string {
	return normalizePolicyEnforcementMode(os.Getenv(EnvOverridePolicyEnforcement))
}

// ResolvePolicyMetricSchemaModeFromEnv resolves policy metric schema mode.
func ResolvePolicyMetricSchemaModeFromEnv() string {
	return normalizePolicyMetricSchemaMode(os.Getenv(EnvPolicyMetricSchemaMode))
}

// PolicyMetricSchemaPlanForMode returns the metric schema emission plan.
func PolicyMetricSchemaPlanForMode(mode string) PolicyMetricSchemaPlan {
	mode = normalizePolicyMetricSchemaMode(mode)
	switch mode {
	case PolicyMetricSchemaV2:
		return PolicyMetricSchemaPlan{
			Mode:       mode,
			EmitV1:     false,
			EmitV2:     true,
			Deprecated: false,
		}
	case PolicyMetricSchemaDualWrite:
		return PolicyMetricSchemaPlan{
			Mode:       mode,
			EmitV1:     true,
			EmitV2:     true,
			Deprecated: true,
		}
	default:
		return PolicyMetricSchemaPlan{
			Mode:       PolicyMetricSchemaV1,
			EmitV1:     true,
			EmitV2:     false,
			Deprecated: true,
		}
	}
}

// ParseOverridePolicyAllowProfiles parses a comma-separated allowlist of runtime profiles.
func ParseOverridePolicyAllowProfiles(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		profile := normalizeRuntimeProfile(trimmed)
		out[profile] = struct{}{}
	}
	return out
}

// ResolveOverridePolicyRuntime resolves effective policy preset and allowlist with precedence:
// explicit allowlist env > preset env > profile-safe defaults.
func ResolveOverridePolicyRuntime(profile string) OverridePolicyRuntime {
	explicitAllowProfiles := ParseOverridePolicyAllowProfilesFromEnv()
	normalizedProfile := normalizeRuntimeProfile(profile)
	if len(explicitAllowProfiles) > 0 {
		return OverridePolicyRuntime{
			Preset:        PolicyPresetStrict,
			PresetSource:  "env_allow_profiles",
			AllowProfiles: explicitAllowProfiles,
			KeyPolicy:     KeyPolicyForProfileAndPreset(normalizedProfile, PolicyPresetStrict),
		}
	}

	rawPreset := strings.TrimSpace(os.Getenv(EnvOverridePolicyPreset))
	if preset, ok := normalizePolicyPreset(rawPreset); ok {
		return OverridePolicyRuntime{
			Preset:        preset,
			PresetSource:  "env_preset",
			AllowProfiles: AllowProfilesForPolicyPreset(preset),
			KeyPolicy:     KeyPolicyForProfileAndPreset(normalizedProfile, preset),
		}
	}

	defaultPreset := DefaultPolicyPresetForProfile(normalizedProfile)
	return OverridePolicyRuntime{
		Preset:        defaultPreset,
		PresetSource:  "default",
		AllowProfiles: AllowProfilesForPolicyPreset(defaultPreset),
		KeyPolicy:     KeyPolicyForProfileAndPreset(normalizedProfile, defaultPreset),
	}
}

// DefaultPolicyPresetForProfile returns startup-safe default by environment tier.
func DefaultPolicyPresetForProfile(profile string) string {
	normalized := normalizeRuntimeProfile(profile)
	if isProductionProfile(normalized) {
		return PolicyPresetStrict
	}

	switch normalized {
	case "staging", "canary", "preprod", "qa", "test":
		return PolicyPresetBalanced
	default:
		return PolicyPresetOpen
	}
}

// AllowProfilesForPolicyPreset returns the non-production allowlist for a preset.
func AllowProfilesForPolicyPreset(preset string) map[string]struct{} {
	switch preset {
	case PolicyPresetStrict:
		return ParseOverridePolicyAllowProfiles("staging,canary")
	case PolicyPresetBalanced:
		return ParseOverridePolicyAllowProfiles("development,staging,canary,preprod,qa,test")
	default:
		return map[string]struct{}{}
	}
}

// KeyPolicyForProfileAndPreset returns key-level constraints by environment tier and policy preset.
func KeyPolicyForProfileAndPreset(profile, preset string) CoreOverrideKeyPolicy {
	profile = normalizeRuntimeProfile(profile)
	if isProductionProfile(profile) {
		return CoreOverrideKeyPolicy{
			AllowedAPITypes:               cloneStringSet("rest", "grpc", "service", "graphql"),
			MinAPIPort:                    1024,
			MaxAPIPort:                    65535,
			DisallowedEnabledFeatureFlags: cloneMapSet(productionDisallowedFeatureFlags),
		}
	}

	switch preset {
	case PolicyPresetStrict:
		return CoreOverrideKeyPolicy{
			AllowedAPITypes:               cloneStringSet("rest", "grpc", "service", "graphql"),
			MinAPIPort:                    1024,
			MaxAPIPort:                    65535,
			DisallowedEnabledFeatureFlags: cloneMapSet(productionDisallowedFeatureFlags),
		}
	case PolicyPresetBalanced:
		return CoreOverrideKeyPolicy{
			AllowedAPITypes:               cloneStringSet("rest", "grpc", "service", "graphql", "websocket"),
			MinAPIPort:                    1024,
			MaxAPIPort:                    65535,
			DisallowedEnabledFeatureFlags: cloneStringSet("enable_unsafe_debug"),
		}
	default:
		return CoreOverrideKeyPolicy{
			AllowedAPITypes:               cloneMapSet(validCoreAPITypes),
			MinAPIPort:                    1,
			MaxAPIPort:                    65535,
			DisallowedEnabledFeatureFlags: map[string]struct{}{},
		}
	}
}

// BuildOverrideAuditTags builds structured tags for override audit metrics.
func BuildOverrideAuditTags(
	profile string,
	envOverrides CoreConfigOverrides,
	cliOverrides CoreConfigOverrides,
	merged CoreConfigOverrides,
	policy OverridePolicyRuntime,
	evaluation OverridePolicyEvaluation,
) map[string]string {
	normalizedProfile := normalizeRuntimeProfile(profile)
	_, allowlisted := policy.AllowProfiles[normalizedProfile]
	tags := map[string]string{
		"profile":               normalizedProfile,
		"env_overrides_present": strconv.FormatBool(hasAnyOverride(envOverrides)),
		"cli_overrides_present": strconv.FormatBool(hasAnyOverride(cliOverrides)),
		"api_type_overridden":   strconv.FormatBool(merged.APIType != nil),
		"api_port_overridden":   strconv.FormatBool(merged.APIPort != nil),
		"feature_flag_count":    strconv.Itoa(len(merged.FeatureFlags)),
		"allowlist_enabled":     strconv.FormatBool(len(policy.AllowProfiles) > 0),
		"profile_allowlisted":   strconv.FormatBool(allowlisted),
		"is_production_profile": strconv.FormatBool(isProductionProfile(normalizedProfile)),
		"any_overrides_applied": strconv.FormatBool(hasAnyOverride(merged)),
		"policy_preset":         policy.Preset,
		"policy_preset_source":  policy.PresetSource,
		"policy_api_port_min":   strconv.Itoa(policy.KeyPolicy.MinAPIPort),
		"policy_api_port_max":   strconv.Itoa(policy.KeyPolicy.MaxAPIPort),
		"policy_api_type_count": strconv.Itoa(len(policy.KeyPolicy.AllowedAPITypes)),
		"policy_ff_deny_count":  strconv.Itoa(len(policy.KeyPolicy.DisallowedEnabledFeatureFlags)),
		"policy_enforcement":    evaluation.EnforcementMode,
		"policy_violation":      strconv.FormatBool(evaluation.Violation),
		"policy_blocked":        strconv.FormatBool(evaluation.Blocked),
		"policy_violation_code": evaluation.ViolationCode,
	}

	return tags
}

// BuildPolicyEvaluationTags builds tags for policy evaluation counter.
func BuildPolicyEvaluationTags(profile string, evaluation OverridePolicyEvaluation) map[string]string {
	return map[string]string{
		"profile":            normalizeRuntimeProfile(profile),
		"policy_enforcement": evaluation.EnforcementMode,
		"policy_violation":   strconv.FormatBool(evaluation.Violation),
		"policy_blocked":     strconv.FormatBool(evaluation.Blocked),
		"policy_code":        evaluation.ViolationCode,
	}
}

// EmitPolicyOverrideMetrics emits override metrics following the selected schema plan.
func EmitPolicyOverrideMetrics(
	metrics core.MetricsCollector,
	profile string,
	envOverrides CoreConfigOverrides,
	cliOverrides CoreConfigOverrides,
	merged CoreConfigOverrides,
	policy OverridePolicyRuntime,
	evaluation OverridePolicyEvaluation,
	mode string,
) {
	if metrics == nil {
		return
	}

	plan := PolicyMetricSchemaPlanForMode(mode)
	auditTags := BuildOverrideAuditTags(profile, envOverrides, cliOverrides, merged, policy, evaluation)
	evalTags := BuildPolicyEvaluationTags(profile, evaluation)

	if plan.EmitV1 {
		metrics.RecordCounter(PolicyMetricOverridesAppliedV1, 1, withMetricSchemaTags(auditTags, PolicyMetricSchemaV1, plan.Deprecated))
		metrics.RecordCounter(PolicyMetricEvaluationV1, 1, withMetricSchemaTags(evalTags, PolicyMetricSchemaV1, plan.Deprecated))
	}
	if plan.EmitV2 {
		metrics.RecordCounter(PolicyMetricOverridesAppliedV2, 1, withMetricSchemaTags(auditTags, PolicyMetricSchemaV2, false))
		metrics.RecordCounter(PolicyMetricEvaluationV2, 1, withMetricSchemaTags(evalTags, PolicyMetricSchemaV2, false))
	}
}

func normalizePolicyPreset(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case PolicyPresetStrict, PolicyPresetBalanced, PolicyPresetOpen:
		return normalized, true
	default:
		return "", false
	}
}

func normalizePolicyEnforcementMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case PolicyEnforcementAudit:
		return PolicyEnforcementAudit
	default:
		return PolicyEnforcementEnforce
	}
}

func normalizePolicyMetricSchemaMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case PolicyMetricSchemaV2:
		return PolicyMetricSchemaV2
	case PolicyMetricSchemaDualWrite:
		return PolicyMetricSchemaDualWrite
	default:
		return PolicyMetricSchemaV1
	}
}

func normalizeRuntimeProfile(profile string) string {
	normalized := strings.ToLower(strings.TrimSpace(profile))
	switch normalized {
	case "":
		return "development"
	case "prod":
		return "production"
	case "dev":
		return "development"
	case "stage":
		return "staging"
	default:
		return normalized
	}
}

func isProductionProfile(profile string) bool {
	return normalizeRuntimeProfile(profile) == "production"
}

func hasAnyOverride(overrides CoreConfigOverrides) bool {
	if overrides.APIType != nil {
		return true
	}
	if overrides.APIPort != nil {
		return true
	}
	return len(overrides.FeatureFlags) > 0
}

func cloneStringSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func cloneMapSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

// PolicyErrorCode returns the structured code for policy errors.
func PolicyErrorCode(err error) string {
	if err == nil {
		return ""
	}
	policyErr, ok := err.(*OverridePolicyError)
	if !ok {
		return ""
	}
	return policyErr.Code
}

// SortedKeys returns deterministic sorted keys for diagnostics/tests.
func SortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func withMetricSchemaTags(in map[string]string, schema string, deprecated bool) map[string]string {
	out := make(map[string]string, len(in)+2)
	for key, value := range in {
		out[key] = value
	}
	out["metric_schema_version"] = schema
	out["metric_schema_deprecated"] = strconv.FormatBool(deprecated)
	return out
}
