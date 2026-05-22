package bootstrap

import (
	"os"
	"testing"
)

func TestNewAPIServiceCoreConfigDefaults(t *testing.T) {
	t.Parallel()
	cfg := NewAPIServiceCoreConfig(18081, "info")
	if cfg.APIType != "service" {
		t.Fatalf("expected APIType service, got %s", cfg.APIType)
	}
	if cfg.APIPort != 18081 {
		t.Fatalf("expected APIPort 18081, got %d", cfg.APIPort)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected LogLevel info, got %s", cfg.LogLevel)
	}
}

func TestNewMonolithicCoreConfigDefaults(t *testing.T) {
	t.Parallel()
	cfg := NewMonolithicCoreConfig("debug", "postgres", "postgres://x", "redis", "https-jsonrpc", "http://localhost:8545")
	if cfg.APIType != "rest" {
		t.Errorf("expected APIType rest, got %s", cfg.APIType)
	}
	if cfg.APIPort != 8080 {
		t.Fatalf("expected APIPort 8080, got %d", cfg.APIPort)
	}
	if cfg.DatabaseType != "postgres" {
		t.Fatalf("expected DatabaseType postgres, got %s", cfg.DatabaseType)
	}
}

func TestApplyCoreConfigOverridesTableDriven(t *testing.T) {
	t.Parallel()
	apiType := "rest"
	apiPort := 9099

	tests := []struct {
		name          string
		baseType      string
		basePort      int
		baseFlags     map[string]bool
		override      CoreConfigOverrides
		wantType      string
		wantPort      int
		wantFlagValue bool
	}{
		{
			name:      "no override keeps base",
			baseType:  "service",
			basePort:  8081,
			baseFlags: map[string]bool{"a": true},
			override:  CoreConfigOverrides{},
			wantType:  "service",
			wantPort:  8081,
		},
		{
			name:      "override api type and port",
			baseType:  "service",
			basePort:  8081,
			baseFlags: map[string]bool{},
			override: CoreConfigOverrides{
				APIType: &apiType,
				APIPort: &apiPort,
			},
			wantType: "rest",
			wantPort: 9099,
		},
		{
			name:      "feature flag merge overrides existing key",
			baseType:  "graphql",
			basePort:  8080,
			baseFlags: map[string]bool{"feature-x": false},
			override: CoreConfigOverrides{
				FeatureFlags: map[string]bool{"feature-x": true},
			},
			wantType:      "graphql",
			wantPort:      8080,
			wantFlagValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewAPIServiceCoreConfig(tt.basePort, "info")
			base.APIType = tt.baseType
			base.FeatureFlags = tt.baseFlags

			got := ApplyCoreConfigOverrides(base, tt.override)

			if got.APIType != tt.wantType {
				t.Fatalf("want APIType=%s, got %s", tt.wantType, got.APIType)
			}
			if got.APIPort != tt.wantPort {
				t.Fatalf("want APIPort=%d, got %d", tt.wantPort, got.APIPort)
			}

			if _, hasFlag := tt.override.FeatureFlags["feature-x"]; hasFlag {
				if got.FeatureFlags["feature-x"] != tt.wantFlagValue {
					t.Fatalf("want feature-x=%v, got %v", tt.wantFlagValue, got.FeatureFlags["feature-x"])
				}
			}
		})
	}
}

func TestApplyConfigOverridesFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		apiType     string
		apiPort     string
		featureFlag string
		wantType    string
		wantPort    int
		wantFlagVal bool
		wantFlagKey string
	}{
		{
			name:     "empty env no change",
			wantType: "rest",
			wantPort: 8080,
		},
		{
			name:     "override api type",
			apiType:  "graphql",
			wantType: "graphql",
			wantPort: 8080,
		},
		{
			name:     "override api port",
			apiPort:  "9099",
			wantType: "rest",
			wantPort: 9099,
		},
		{
			name:        "override feature flags",
			featureFlag: "beta=true,debug=false",
			wantType:    "rest",
			wantPort:    8080,
			wantFlagKey: "beta",
			wantFlagVal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CHAINPULSE_CORE_API_TYPE", tt.apiType)
			os.Setenv("CHAINPULSE_CORE_API_PORT", tt.apiPort)
			os.Setenv("CHAINPULSE_CORE_FEATURE_FLAGS", tt.featureFlag)
			defer func() {
				os.Unsetenv("CHAINPULSE_CORE_API_TYPE")
				os.Unsetenv("CHAINPULSE_CORE_API_PORT")
				os.Unsetenv("CHAINPULSE_CORE_FEATURE_FLAGS")
			}()

			cfg := NewAPIServiceCoreConfig(8080, "info")
			ApplyConfigOverrides(&cfg)

			if cfg.APIType != tt.wantType {
				t.Fatalf("want APIType=%s, got %s", tt.wantType, cfg.APIType)
			}
			if cfg.APIPort != tt.wantPort {
				t.Fatalf("want APIPort=%d, got %d", tt.wantPort, cfg.APIPort)
			}
			if tt.wantFlagKey != "" {
				if v, ok := cfg.FeatureFlags[tt.wantFlagKey]; !ok || v != tt.wantFlagVal {
					t.Fatalf("want flag %s=%v, got %v", tt.wantFlagKey, tt.wantFlagVal, cfg.FeatureFlags)
				}
			}
		})
	}
}

func TestApplyConfigOverridesInvalidPortIgnored(t *testing.T) {
	os.Setenv("CHAINPULSE_CORE_API_PORT", "not-a-number")
	defer os.Unsetenv("CHAINPULSE_CORE_API_PORT")

	cfg := NewAPIServiceCoreConfig(8080, "info")
	ApplyConfigOverrides(&cfg)

	if cfg.APIPort != 8080 {
		t.Fatalf("expected port to remain 8080, got %d", cfg.APIPort)
	}
}

func TestApplyConfigOverridesInvalidPortRangeIgnored(t *testing.T) {
	os.Setenv("CHAINPULSE_CORE_API_PORT", "99999")
	defer os.Unsetenv("CHAINPULSE_CORE_API_PORT")

	cfg := NewAPIServiceCoreConfig(8080, "info")
	ApplyConfigOverrides(&cfg)

	if cfg.APIPort != 8080 {
		t.Fatalf("expected port to remain 8080, got %d", cfg.APIPort)
	}
}

func TestApplyConfigOverridesInvalidFlagsIgnored(t *testing.T) {
	os.Setenv("CHAINPULSE_CORE_FEATURE_FLAGS", "a:true")
	defer os.Unsetenv("CHAINPULSE_CORE_FEATURE_FLAGS")

	cfg := NewAPIServiceCoreConfig(8080, "info")
	ApplyConfigOverrides(&cfg)

	if len(cfg.FeatureFlags) != 0 {
		t.Fatalf("expected no feature flags set, got %v", cfg.FeatureFlags)
	}
}

func TestApplyConfigOverridesNilConfig(t *testing.T) {
	ApplyConfigOverrides(nil) // should not panic
}

func TestParseCoreConfigOverridesFromEnvTable(t *testing.T) {
	tests := []struct {
		name         string
		apiType      string
		apiPort      string
		featureFlags string
		wantErr      bool
		wantAPIType  string
		wantAPIPort  int
		wantHasPort  bool
		wantFlagA    bool
		wantHasFlagA bool
	}{
		{
			name:        "empty env returns empty overrides",
			wantErr:     false,
			wantHasPort: false,
		},
		{
			name:        "valid api type and port",
			apiType:     "rest",
			apiPort:     "9090",
			wantErr:     false,
			wantAPIType: "rest",
			wantAPIPort: 9090,
			wantHasPort: true,
		},
		{
			name:         "valid feature flags",
			featureFlags: "a=true,b=false",
			wantErr:      false,
			wantHasFlagA: true,
			wantFlagA:    true,
		},
		{
			name:    "invalid api port",
			apiPort: "not-a-number",
			wantErr: true,
		},
		{
			name:    "api port out of range",
			apiPort: "70000",
			wantErr: true,
		},
		{
			name:         "invalid feature flags format",
			featureFlags: "a:true",
			wantErr:      true,
		},
		{
			name:         "invalid feature flags bool",
			featureFlags: "a=maybe",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CHAINPULSE_CORE_API_TYPE", tt.apiType)
			t.Setenv("CHAINPULSE_CORE_API_PORT", tt.apiPort)
			t.Setenv("CHAINPULSE_CORE_FEATURE_FLAGS", tt.featureFlags)

			got, err := ParseCoreConfigOverridesFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantAPIType != "" {
				if got.APIType == nil || *got.APIType != tt.wantAPIType {
					t.Fatalf("expected APIType %q, got %+v", tt.wantAPIType, got.APIType)
				}
			}

			if tt.wantHasPort {
				if got.APIPort == nil || *got.APIPort != tt.wantAPIPort {
					t.Fatalf("expected APIPort %d, got %+v", tt.wantAPIPort, got.APIPort)
				}
			}

			if tt.wantHasFlagA {
				gotA, ok := got.FeatureFlags["a"]
				if !ok || gotA != tt.wantFlagA {
					t.Fatalf("expected feature flag a=%v, got map=%v", tt.wantFlagA, got.FeatureFlags)
				}
			}
		})
	}
}

func TestSummarizeCoreConfigOverrides(t *testing.T) {
	typ := "graphql"
	port := 8080
	summary := SummarizeCoreConfigOverrides(CoreConfigOverrides{
		APIType:      &typ,
		APIPort:      &port,
		FeatureFlags: map[string]bool{"beta": true},
	})

	if summary == "none" {
		t.Fatal("expected non-empty summary")
	}

	noneSummary := SummarizeCoreConfigOverrides(CoreConfigOverrides{})
	if noneSummary != "none" {
		t.Fatalf("expected 'none', got %q", noneSummary)
	}
}

func TestMergeCoreConfigOverridesPrecedence(t *testing.T) {
	t.Parallel()
	lowType := "service"
	highType := "graphql"
	lowPort := 8080
	highPort := 9999

	merged := MergeCoreConfigOverrides(
		CoreConfigOverrides{
			APIType:      &lowType,
			APIPort:      &lowPort,
			FeatureFlags: map[string]bool{"x": false, "shared": false},
		},
		CoreConfigOverrides{
			APIType:      &highType,
			APIPort:      &highPort,
			FeatureFlags: map[string]bool{"y": true, "shared": true},
		},
	)

	if merged.APIType == nil || *merged.APIType != "graphql" {
		t.Fatalf("expected APIType graphql, got %+v", merged.APIType)
	}
	if merged.APIPort == nil || *merged.APIPort != 9999 {
		t.Fatalf("expected APIPort 9999, got %+v", merged.APIPort)
	}
	if merged.FeatureFlags["x"] {
		t.Fatalf("expected x to remain false")
	}
	if !merged.FeatureFlags["y"] {
		t.Fatalf("expected y to be true")
	}
	if !merged.FeatureFlags["shared"] {
		t.Fatalf("expected shared to use high precedence override")
	}
}

func TestParseCoreConfigOverridesFromCLITable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		args         []string
		wantErr      bool
		wantType     string
		wantPort     int
		wantPortSet  bool
		wantFeatureA bool
		wantHasA     bool
	}{
		{
			name:    "empty args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:        "key value args",
			args:        []string{"--core-api-type", "graphql", "--core-api-port", "8088"},
			wantErr:     false,
			wantType:    "graphql",
			wantPort:    8088,
			wantPortSet: true,
		},
		{
			name:         "feature flags parse",
			args:         []string{"--core-feature-flags=a=true,b=false"},
			wantErr:      false,
			wantHasA:     true,
			wantFeatureA: true,
		},
		{
			name:    "unsupported api type",
			args:    []string{"--core-api-type=invalid"},
			wantErr: true,
		},
		{
			name:    "missing value for key value form",
			args:    []string{"--core-api-port"},
			wantErr: true,
		},
		{
			name:    "invalid port",
			args:    []string{"--core-api-port=abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCoreConfigOverridesFromCLI(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantType != "" {
				if got.APIType == nil || *got.APIType != tt.wantType {
					t.Fatalf("expected APIType %q, got %+v", tt.wantType, got.APIType)
				}
			}

			if tt.wantPortSet {
				if got.APIPort == nil || *got.APIPort != tt.wantPort {
					t.Fatalf("expected APIPort %d, got %+v", tt.wantPort, got.APIPort)
				}
			}

			if tt.wantHasA {
				value, ok := got.FeatureFlags["a"]
				if !ok || value != tt.wantFeatureA {
					t.Fatalf("expected feature flag a=%v, got map=%v", tt.wantFeatureA, got.FeatureFlags)
				}
			}
		})
	}
}

func TestRuntimeProfileFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   string
	}{
		{name: "empty defaults to development", envVal: "", want: "development"},
		{name: "production", envVal: "production", want: "production"},
		{name: "prod alias", envVal: "prod", want: "production"},
		{name: "staging", envVal: "staging", want: "staging"},
		{name: "testing", envVal: "testing", want: "testing"},
		{name: "development", envVal: "development", want: "development"},
		{name: "unknown defaults to development", envVal: "foo", want: "development"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CHAINPULSE_ENV", tt.envVal)
			defer os.Unsetenv("CHAINPULSE_ENV")
			got := RuntimeProfileFromEnv()
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestResolveOverridePolicyRuntime(t *testing.T) {
	policy := ResolveOverridePolicyRuntime("production")
	if policy.Preset != "open" {
		t.Fatalf("expected open policy, got %s", policy.Preset)
	}
}

func TestValidateCoreConfigOverridesWithMode(t *testing.T) {
	eval, err := ValidateCoreConfigOverridesWithMode(CoreConfigOverrides{}, "production", OverridePolicyRuntime{}, "enforce")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eval.EnforcementMode != "audit" {
		t.Fatalf("expected audit mode, got %s", eval.EnforcementMode)
	}
}

func TestPolicyErrorCode(t *testing.T) {
	code := PolicyErrorCode(nil)
	if code != "" {
		t.Fatalf("expected empty code, got %s", code)
	}
}

func TestResolvePolicyMetricSchemaModeFromEnv(t *testing.T) {
	mode := ResolvePolicyMetricSchemaModeFromEnv()
	if mode != "v1" {
		t.Fatalf("expected v1, got %s", mode)
	}
}

func TestEmitPolicyOverrideMetricsNoPanic(t *testing.T) {
	EmitPolicyOverrideMetrics(nil, "", CoreConfigOverrides{}, CoreConfigOverrides{}, CoreConfigOverrides{}, OverridePolicyRuntime{}, OverridePolicyEvaluation{}, "")
}
