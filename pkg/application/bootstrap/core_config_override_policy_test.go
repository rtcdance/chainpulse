package bootstrap

import (
	"chainpulse/pkg/core"
	"os"
	"reflect"
	"testing"
)

func TestValidateCoreConfigOverridesForProfileProductionDenylist(t *testing.T) {
	apiType := "gateway"
	err := ValidateCoreConfigOverridesForProfileWithAllowlist(CoreConfigOverrides{
		APIType: &apiType,
	}, "production", nil)
	if err == nil {
		t.Fatal("expected production denylist error for APIType gateway")
	}
	if PolicyErrorCode(err) != PolicyErrAPITypeDenied {
		t.Fatalf("expected code %s, got %s", PolicyErrAPITypeDenied, PolicyErrorCode(err))
	}
}

func TestValidateCoreConfigOverridesForProfileProductionFeatureFlagDenylist(t *testing.T) {
	err := ValidateCoreConfigOverridesForProfileWithAllowlist(CoreConfigOverrides{
		FeatureFlags: map[string]bool{"dev_mode": true},
	}, "prod", nil)
	if err == nil {
		t.Fatal("expected production denylist error for dev_mode=true")
	}
	if PolicyErrorCode(err) != PolicyErrFeatureFlagDenied {
		t.Fatalf("expected code %s, got %s", PolicyErrFeatureFlagDenied, PolicyErrorCode(err))
	}
}

func TestValidateCoreConfigOverridesForProfileProductionPortRange(t *testing.T) {
	port := 80
	err := ValidateCoreConfigOverridesForProfileWithAllowlist(CoreConfigOverrides{
		APIPort: &port,
	}, "production", nil)
	if err == nil {
		t.Fatal("expected production port range error")
	}
	if PolicyErrorCode(err) != PolicyErrAPIPortOutOfRange {
		t.Fatalf("expected code %s, got %s", PolicyErrAPIPortOutOfRange, PolicyErrorCode(err))
	}
}

func TestValidateCoreConfigOverridesForProfileNonProductionPassesWithoutAllowlist(t *testing.T) {
	apiType := "gateway"
	err := ValidateCoreConfigOverridesForProfileWithAllowlist(CoreConfigOverrides{
		APIType: &apiType,
		FeatureFlags: map[string]bool{
			"dev_mode": true,
		},
	}, "development", nil)
	if err != nil {
		t.Fatalf("expected no error in non-production profile, got: %v", err)
	}
}

func TestValidateCoreConfigOverridesForProfileAllowlistMatrix(t *testing.T) {
	apiType := "service"
	base := CoreConfigOverrides{APIType: &apiType}
	allowlist := ParseOverridePolicyAllowProfiles("staging,canary")

	testCases := []struct {
		name      string
		profile   string
		overrides CoreConfigOverrides
		wantErr   bool
	}{
		{
			name:      "staging allowlisted",
			profile:   "staging",
			overrides: base,
			wantErr:   false,
		},
		{
			name:      "canary allowlisted",
			profile:   "canary",
			overrides: base,
			wantErr:   false,
		},
		{
			name:      "unknown profile rejected when allowlist enabled and overrides present",
			profile:   "qa",
			overrides: base,
			wantErr:   true,
		},
		{
			name:      "unknown profile allowed when no overrides",
			profile:   "qa",
			overrides: CoreConfigOverrides{},
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCoreConfigOverridesForProfileWithAllowlist(tc.overrides, tc.profile, allowlist)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if tc.wantErr && PolicyErrorCode(err) != PolicyErrProfileNotAllowlisted {
				t.Fatalf("expected code %s, got %s", PolicyErrProfileNotAllowlisted, PolicyErrorCode(err))
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestParseOverridePolicyAllowProfilesFromEnv(t *testing.T) {
	t.Setenv(EnvOverridePolicyAllowProfiles, "stage,canary, dev ")

	got := ParseOverridePolicyAllowProfilesFromEnv()
	if _, ok := got["staging"]; !ok {
		t.Fatal("expected stage alias normalized to staging")
	}
	if _, ok := got["canary"]; !ok {
		t.Fatal("expected canary in allowlist")
	}
	if _, ok := got["development"]; !ok {
		t.Fatal("expected dev alias normalized to development")
	}
}

func TestValidateCoreConfigOverridesForProfileReadsEnvAllowlist(t *testing.T) {
	// Use os.Setenv here to verify the convenience API path that reads process env.
	//nolint:tenv // intentional process-wide env validation for wrapper behavior.
	if err := os.Setenv(EnvOverridePolicyAllowProfiles, "staging"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer func() {
		_ = os.Unsetenv(EnvOverridePolicyAllowProfiles)
	}()

	apiType := "service"
	err := ValidateCoreConfigOverridesForProfile(CoreConfigOverrides{APIType: &apiType}, "qa")
	if err == nil {
		t.Fatal("expected allowlist rejection for qa profile when env allowlist enabled")
	}
	if PolicyErrorCode(err) != PolicyErrProfileNotAllowlisted {
		t.Fatalf("expected code %s, got %s", PolicyErrProfileNotAllowlisted, PolicyErrorCode(err))
	}
}

func TestDefaultPolicyPresetForProfile(t *testing.T) {
	testCases := []struct {
		name    string
		profile string
		want    string
	}{
		{name: "production strict", profile: "production", want: PolicyPresetStrict},
		{name: "prod alias strict", profile: "prod", want: PolicyPresetStrict},
		{name: "staging balanced", profile: "staging", want: PolicyPresetBalanced},
		{name: "qa balanced", profile: "qa", want: PolicyPresetBalanced},
		{name: "dev open", profile: "development", want: PolicyPresetOpen},
		{name: "unknown open", profile: "local", want: PolicyPresetOpen},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := DefaultPolicyPresetForProfile(tc.profile)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestResolveOverridePolicyRuntimePrecedence(t *testing.T) {
	t.Setenv(EnvOverridePolicyAllowProfiles, "staging")
	t.Setenv(EnvOverridePolicyPreset, PolicyPresetOpen)

	got := ResolveOverridePolicyRuntime("qa")
	if got.PresetSource != "env_allow_profiles" {
		t.Fatalf("expected env_allow_profiles, got %s", got.PresetSource)
	}
	if _, ok := got.AllowProfiles["staging"]; !ok {
		t.Fatal("expected staging from explicit allowlist")
	}
	if _, ok := got.AllowProfiles["qa"]; ok {
		t.Fatal("did not expect qa in explicit allowlist")
	}
}

func TestResolveOverridePolicyRuntimePresetFromEnv(t *testing.T) {
	t.Setenv(EnvOverridePolicyAllowProfiles, "")
	t.Setenv(EnvOverridePolicyPreset, PolicyPresetBalanced)

	got := ResolveOverridePolicyRuntime("development")
	if got.Preset != PolicyPresetBalanced {
		t.Fatalf("expected balanced preset, got %s", got.Preset)
	}
	if got.PresetSource != "env_preset" {
		t.Fatalf("expected env_preset source, got %s", got.PresetSource)
	}
	if _, ok := got.AllowProfiles["qa"]; !ok {
		t.Fatal("expected qa in balanced allowlist")
	}
}

func TestResolveOverridePolicyRuntimeDefaultFallback(t *testing.T) {
	t.Setenv(EnvOverridePolicyAllowProfiles, "")
	t.Setenv(EnvOverridePolicyPreset, "invalid")

	got := ResolveOverridePolicyRuntime("development")
	if got.Preset != PolicyPresetOpen {
		t.Fatalf("expected default open preset for development, got %s", got.Preset)
	}
	if got.PresetSource != "default" {
		t.Fatalf("expected default source, got %s", got.PresetSource)
	}
	if len(got.AllowProfiles) != 0 {
		t.Fatalf("expected open preset to produce empty allowlist, got %v", got.AllowProfiles)
	}
}

func TestResolvePolicyEnforcementModeFromEnv(t *testing.T) {
	t.Setenv(EnvOverridePolicyEnforcement, PolicyEnforcementAudit)
	if got := ResolvePolicyEnforcementModeFromEnv(); got != PolicyEnforcementAudit {
		t.Fatalf("expected audit mode, got %s", got)
	}

	t.Setenv(EnvOverridePolicyEnforcement, "invalid")
	if got := ResolvePolicyEnforcementModeFromEnv(); got != PolicyEnforcementEnforce {
		t.Fatalf("expected enforce fallback, got %s", got)
	}
}

func TestKeyPolicyForProfileAndPresetMatrix(t *testing.T) {
	prod := KeyPolicyForProfileAndPreset("production", PolicyPresetOpen)
	if prod.MinAPIPort != 1024 || prod.MaxAPIPort != 65535 {
		t.Fatalf("unexpected production port policy: %d..%d", prod.MinAPIPort, prod.MaxAPIPort)
	}
	if _, ok := prod.AllowedAPITypes["gateway"]; ok {
		t.Fatal("production policy should not allow gateway API type")
	}
	if _, ok := prod.DisallowedEnabledFeatureFlags["dev_mode"]; !ok {
		t.Fatal("production policy should deny dev_mode=true")
	}

	balanced := KeyPolicyForProfileAndPreset("staging", PolicyPresetBalanced)
	if _, ok := balanced.AllowedAPITypes["websocket"]; !ok {
		t.Fatal("balanced policy should allow websocket API type")
	}
	if _, ok := balanced.DisallowedEnabledFeatureFlags["enable_unsafe_debug"]; !ok {
		t.Fatal("balanced policy should deny enable_unsafe_debug=true")
	}

	open := KeyPolicyForProfileAndPreset("development", PolicyPresetOpen)
	if _, ok := open.AllowedAPITypes["gateway"]; !ok {
		t.Fatal("open policy should allow gateway API type")
	}
	if len(open.DisallowedEnabledFeatureFlags) != 0 {
		t.Fatal("open policy should not deny feature flags")
	}
}

func TestValidateCoreConfigOverridesWithPolicyPresetBehavior(t *testing.T) {
	apiType := "websocket"
	balancedPolicy := OverridePolicyRuntime{
		Preset:       PolicyPresetBalanced,
		PresetSource: "test",
		KeyPolicy:    KeyPolicyForProfileAndPreset("staging", PolicyPresetBalanced),
	}
	err := ValidateCoreConfigOverridesWithPolicy(CoreConfigOverrides{APIType: &apiType}, "staging", balancedPolicy)
	if err != nil {
		t.Fatalf("expected websocket allowed under balanced preset, got %v", err)
	}

	strictPolicy := OverridePolicyRuntime{
		Preset:       PolicyPresetStrict,
		PresetSource: "test",
		KeyPolicy:    KeyPolicyForProfileAndPreset("staging", PolicyPresetStrict),
	}
	err = ValidateCoreConfigOverridesWithPolicy(CoreConfigOverrides{APIType: &apiType}, "staging", strictPolicy)
	if err == nil {
		t.Fatal("expected websocket denied under strict preset")
	}
	if PolicyErrorCode(err) != PolicyErrAPITypeDenied {
		t.Fatalf("expected code %s, got %s", PolicyErrAPITypeDenied, PolicyErrorCode(err))
	}
}

func TestValidateCoreConfigOverridesWithModeAuditVsEnforce(t *testing.T) {
	apiType := "gateway"
	policy := OverridePolicyRuntime{
		Preset:        PolicyPresetStrict,
		PresetSource:  "test",
		AllowProfiles: map[string]struct{}{},
		KeyPolicy:     KeyPolicyForProfileAndPreset("production", PolicyPresetStrict),
	}

	enforceEval, enforceErr := ValidateCoreConfigOverridesWithMode(
		CoreConfigOverrides{APIType: &apiType},
		"production",
		policy,
		PolicyEnforcementEnforce,
	)
	if enforceErr == nil {
		t.Fatal("expected error in enforce mode")
	}
	if !enforceEval.Blocked {
		t.Fatal("expected blocked=true in enforce mode")
	}
	if enforceEval.ViolationCode != PolicyErrAPITypeDenied {
		t.Fatalf("expected code %s, got %s", PolicyErrAPITypeDenied, enforceEval.ViolationCode)
	}

	auditEval, auditErr := ValidateCoreConfigOverridesWithMode(
		CoreConfigOverrides{APIType: &apiType},
		"production",
		policy,
		PolicyEnforcementAudit,
	)
	if auditErr != nil {
		t.Fatalf("expected no error in audit mode, got %v", auditErr)
	}
	if !auditEval.Violation {
		t.Fatal("expected violation=true in audit mode")
	}
	if auditEval.Blocked {
		t.Fatal("expected blocked=false in audit mode")
	}
	if auditEval.ViolationCode != PolicyErrAPITypeDenied {
		t.Fatalf("expected code %s, got %s", PolicyErrAPITypeDenied, auditEval.ViolationCode)
	}
}

func TestBuildOverrideAuditTags(t *testing.T) {
	apiType := "service"
	apiPort := 8081
	policy := OverridePolicyRuntime{
		Preset:        PolicyPresetBalanced,
		PresetSource:  "env_preset",
		AllowProfiles: ParseOverridePolicyAllowProfiles("staging,canary"),
		KeyPolicy:     KeyPolicyForProfileAndPreset("staging", PolicyPresetBalanced),
	}
	evaluation := OverridePolicyEvaluation{
		EnforcementMode: PolicyEnforcementAudit,
		ViolationCode:   PolicyErrAPITypeDenied,
		Violation:       true,
		Blocked:         false,
	}
	tags := BuildOverrideAuditTags(
		"production",
		CoreConfigOverrides{APIType: &apiType},
		CoreConfigOverrides{APIPort: &apiPort},
		CoreConfigOverrides{
			APIType:      &apiType,
			APIPort:      &apiPort,
			FeatureFlags: map[string]bool{"x": true},
		},
		policy,
		evaluation,
	)

	if tags["profile"] != "production" {
		t.Fatalf("expected profile tag production, got %s", tags["profile"])
	}
	if tags["env_overrides_present"] != "true" {
		t.Fatalf("expected env_overrides_present=true, got %s", tags["env_overrides_present"])
	}
	if tags["cli_overrides_present"] != "true" {
		t.Fatalf("expected cli_overrides_present=true, got %s", tags["cli_overrides_present"])
	}
	if tags["feature_flag_count"] != "1" {
		t.Fatalf("expected feature_flag_count=1, got %s", tags["feature_flag_count"])
	}
	if tags["allowlist_enabled"] != "true" {
		t.Fatalf("expected allowlist_enabled=true, got %s", tags["allowlist_enabled"])
	}
	if tags["profile_allowlisted"] != "false" {
		t.Fatalf("expected profile_allowlisted=false for production not in allowlist, got %s", tags["profile_allowlisted"])
	}
	if tags["is_production_profile"] != "true" {
		t.Fatalf("expected is_production_profile=true, got %s", tags["is_production_profile"])
	}
	if tags["policy_preset"] != PolicyPresetBalanced {
		t.Fatalf("expected policy_preset=balanced, got %s", tags["policy_preset"])
	}
	if tags["policy_preset_source"] != "env_preset" {
		t.Fatalf("expected policy_preset_source=env_preset, got %s", tags["policy_preset_source"])
	}
	if tags["policy_api_port_min"] != "1024" {
		t.Fatalf("expected policy_api_port_min=1024, got %s", tags["policy_api_port_min"])
	}
	if tags["policy_api_port_max"] != "65535" {
		t.Fatalf("expected policy_api_port_max=65535, got %s", tags["policy_api_port_max"])
	}
	if tags["policy_enforcement"] != PolicyEnforcementAudit {
		t.Fatalf("expected policy_enforcement=audit, got %s", tags["policy_enforcement"])
	}
	if tags["policy_violation"] != "true" {
		t.Fatalf("expected policy_violation=true, got %s", tags["policy_violation"])
	}
	if tags["policy_violation_code"] != PolicyErrAPITypeDenied {
		t.Fatalf("expected policy_violation_code=%s, got %s", PolicyErrAPITypeDenied, tags["policy_violation_code"])
	}
}

func TestBuildPolicyEvaluationTags(t *testing.T) {
	tags := BuildPolicyEvaluationTags("prod", OverridePolicyEvaluation{
		EnforcementMode: PolicyEnforcementAudit,
		ViolationCode:   PolicyErrAPITypeDenied,
		Violation:       true,
		Blocked:         false,
	})

	if tags["profile"] != "production" {
		t.Fatalf("expected normalized profile production, got %s", tags["profile"])
	}
	if tags["policy_enforcement"] != PolicyEnforcementAudit {
		t.Fatalf("expected policy_enforcement=%s, got %s", PolicyEnforcementAudit, tags["policy_enforcement"])
	}
	if tags["policy_code"] != PolicyErrAPITypeDenied {
		t.Fatalf("expected policy_code=%s, got %s", PolicyErrAPITypeDenied, tags["policy_code"])
	}
}

func TestPolicyMetricTagContractStability(t *testing.T) {
	policy := OverridePolicyRuntime{
		Preset:        PolicyPresetBalanced,
		PresetSource:  "test",
		AllowProfiles: ParseOverridePolicyAllowProfiles("staging"),
		KeyPolicy:     KeyPolicyForProfileAndPreset("staging", PolicyPresetBalanced),
	}
	auditTags := BuildOverrideAuditTags(
		"staging",
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		policy,
		OverridePolicyEvaluation{EnforcementMode: PolicyEnforcementEnforce},
	)
	expectedAuditKeys := []string{
		"allowlist_enabled",
		"any_overrides_applied",
		"api_port_overridden",
		"api_type_overridden",
		"cli_overrides_present",
		"env_overrides_present",
		"feature_flag_count",
		"is_production_profile",
		"policy_api_port_max",
		"policy_api_port_min",
		"policy_api_type_count",
		"policy_blocked",
		"policy_enforcement",
		"policy_ff_deny_count",
		"policy_preset",
		"policy_preset_source",
		"policy_violation",
		"policy_violation_code",
		"profile",
		"profile_allowlisted",
	}

	gotAuditKeys := SortedKeys(mapKeysAsSet(auditTags))
	if !reflect.DeepEqual(gotAuditKeys, expectedAuditKeys) {
		t.Fatalf("unexpected audit tag contract keys:\nwant=%v\ngot=%v", expectedAuditKeys, gotAuditKeys)
	}

	evaluationTags := BuildPolicyEvaluationTags("staging", OverridePolicyEvaluation{
		EnforcementMode: PolicyEnforcementAudit,
		ViolationCode:   PolicyErrAPITypeDenied,
		Violation:       true,
		Blocked:         false,
	})
	expectedEvalKeys := []string{
		"policy_blocked",
		"policy_code",
		"policy_enforcement",
		"policy_violation",
		"profile",
	}
	gotEvalKeys := SortedKeys(mapKeysAsSet(evaluationTags))
	if !reflect.DeepEqual(gotEvalKeys, expectedEvalKeys) {
		t.Fatalf("unexpected evaluation tag contract keys:\nwant=%v\ngot=%v", expectedEvalKeys, gotEvalKeys)
	}
}

func TestResolvePolicyMetricSchemaModeFromEnv(t *testing.T) {
	t.Setenv(EnvPolicyMetricSchemaMode, PolicyMetricSchemaDualWrite)
	if got := ResolvePolicyMetricSchemaModeFromEnv(); got != PolicyMetricSchemaDualWrite {
		t.Fatalf("expected %s, got %s", PolicyMetricSchemaDualWrite, got)
	}

	t.Setenv(EnvPolicyMetricSchemaMode, "invalid")
	if got := ResolvePolicyMetricSchemaModeFromEnv(); got != PolicyMetricSchemaV1 {
		t.Fatalf("expected fallback %s, got %s", PolicyMetricSchemaV1, got)
	}
}

func TestPolicyMetricSchemaPlanForMode(t *testing.T) {
	dual := PolicyMetricSchemaPlanForMode(PolicyMetricSchemaDualWrite)
	if !dual.EmitV1 || !dual.EmitV2 || !dual.Deprecated {
		t.Fatalf("unexpected dual_write plan: %+v", dual)
	}

	v2 := PolicyMetricSchemaPlanForMode(PolicyMetricSchemaV2)
	if v2.EmitV1 || !v2.EmitV2 || v2.Deprecated {
		t.Fatalf("unexpected v2 plan: %+v", v2)
	}
}

func TestEmitPolicyOverrideMetricsBySchemaMode(t *testing.T) {
	metrics := core.NewTestMetricsCollectorWithCapture()
	policy := OverridePolicyRuntime{
		Preset:        PolicyPresetBalanced,
		PresetSource:  "test",
		AllowProfiles: ParseOverridePolicyAllowProfiles("staging"),
		KeyPolicy:     KeyPolicyForProfileAndPreset("staging", PolicyPresetBalanced),
	}
	evaluation := OverridePolicyEvaluation{
		EnforcementMode: PolicyEnforcementEnforce,
		Violation:       false,
		Blocked:         false,
	}

	EmitPolicyOverrideMetrics(
		metrics,
		"staging",
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		policy,
		evaluation,
		PolicyMetricSchemaV1,
	)
	if got := metrics.GetCounter(PolicyMetricOverridesAppliedV1); got != 1 {
		t.Fatalf("expected v1 applied metric count 1, got %d", got)
	}
	if got := metrics.GetCounter(PolicyMetricOverridesAppliedV2); got != 0 {
		t.Fatalf("expected v2 applied metric count 0 in v1 mode, got %d", got)
	}

	metrics.Reset()
	EmitPolicyOverrideMetrics(
		metrics,
		"staging",
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		policy,
		evaluation,
		PolicyMetricSchemaDualWrite,
	)
	if got := metrics.GetCounter(PolicyMetricOverridesAppliedV1); got != 1 {
		t.Fatalf("expected v1 applied metric count 1 in dual_write mode, got %d", got)
	}
	if got := metrics.GetCounter(PolicyMetricOverridesAppliedV2); got != 1 {
		t.Fatalf("expected v2 applied metric count 1 in dual_write mode, got %d", got)
	}

	metrics.Reset()
	EmitPolicyOverrideMetrics(
		metrics,
		"staging",
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		CoreConfigOverrides{},
		policy,
		evaluation,
		PolicyMetricSchemaV2,
	)
	if got := metrics.GetCounter(PolicyMetricOverridesAppliedV1); got != 0 {
		t.Fatalf("expected v1 applied metric count 0 in v2 mode, got %d", got)
	}
	if got := metrics.GetCounter(PolicyMetricOverridesAppliedV2); got != 1 {
		t.Fatalf("expected v2 applied metric count 1 in v2 mode, got %d", got)
	}
}

func mapKeysAsSet(in map[string]string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}
