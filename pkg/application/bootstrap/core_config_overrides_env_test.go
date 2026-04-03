package bootstrap

import "testing"

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
			t.Setenv(EnvOverrideAPIType, tt.apiType)
			t.Setenv(EnvOverrideAPIPort, tt.apiPort)
			t.Setenv(EnvOverrideFeatureFlags, tt.featureFlags)

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
