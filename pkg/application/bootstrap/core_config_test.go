package bootstrap

import "testing"

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
