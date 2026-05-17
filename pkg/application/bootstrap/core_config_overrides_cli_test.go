package bootstrap

import "testing"

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
			args:        []string{FlagOverrideAPIType, "graphql", FlagOverrideAPIPort, "8088"},
			wantErr:     false,
			wantType:    "graphql",
			wantPort:    8088,
			wantPortSet: true,
		},
		{
			name:        "equal args",
			args:        []string{FlagOverrideAPIType + "=service", FlagOverrideAPIPort + "=9090"},
			wantErr:     false,
			wantType:    "service",
			wantPort:    9090,
			wantPortSet: true,
		},
		{
			name:         "feature flags parse",
			args:         []string{FlagOverrideFeatureFlags + "=a=true,b=false"},
			wantErr:      false,
			wantHasA:     true,
			wantFeatureA: true,
		},
		{
			name:    "unsupported api type",
			args:    []string{FlagOverrideAPIType + "=invalid"},
			wantErr: true,
		},
		{
			name:    "missing value for key value form",
			args:    []string{FlagOverrideAPIPort},
			wantErr: true,
		},
		{
			name:    "invalid port",
			args:    []string{FlagOverrideAPIPort + "=abc"},
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
