package shared

import (
	"testing"
	"time"
)

func TestAuthenticationRuntimeMetricsUnconfigured(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	metrics := auth.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "auth-unconfigured" {
		t.Fatalf("expected auth-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-unobserved" {
		t.Fatalf("expected auth-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestAuthenticationMetricsIncludesPostureFields(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()
	err := auth.RegisterToken("token-1", "user-1", time.Now().Add(time.Hour), []string{"read"})
	if err != nil {
		t.Fatalf("failed to register token: %v", err)
	}

	metrics := auth.GetMetrics()
	if metrics["coverage_posture"] != "auth-active-only" {
		t.Fatalf("expected auth-active-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-ready" {
		t.Fatalf("expected auth-ready, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "authentication runtime has active tokens available and no expired-token drift" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestAuthenticationRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()
	err := auth.RegisterToken("token-1", "user-1", time.Now().Add(time.Hour), []string{"read"})
	if err != nil {
		t.Fatalf("failed to register token: %v", err)
	}

	metrics := auth.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "auth-active-only" {
		t.Fatalf("expected auth-active-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-ready" {
		t.Fatalf("expected auth-ready, got %v", metrics["runtime_posture"])
	}
}

func TestAuthenticationRuntimeMetricsDegraded(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()
	err := auth.RegisterToken("token-1", "user-1", time.Now().Add(-time.Hour), []string{"read"})
	if err != nil {
		t.Fatalf("failed to register token: %v", err)
	}

	metrics := auth.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "auth-expired-only" {
		t.Fatalf("expected auth-expired-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-degraded" {
		t.Fatalf("expected auth-degraded, got %v", metrics["runtime_posture"])
	}
}

func TestAuthentication_ValidateToken(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()
		valid, err := auth.ValidateToken("")
		if err == nil {
			t.Fatal("expected error for empty token")
		}
		if valid {
			t.Error("expected false for empty token")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		valid, err := auth.ValidateToken("unknown")
		if err == nil {
			t.Fatal("expected error for unknown token")
		}
		if valid {
			t.Error("expected false for unknown token")
		}
	})

	t.Run("valid token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("valid-token", "user-1", time.Now().Add(time.Hour), []string{"read"})
		valid, err := auth.ValidateToken("valid-token")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !valid {
			t.Error("expected true for valid token")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("expired-token", "user-1", time.Now().Add(-time.Hour), []string{"read"})
		valid, err := auth.ValidateToken("expired-token")
		if err == nil {
			t.Fatal("expected error for expired token")
		}
		if valid {
			t.Error("expected false for expired token")
		}
	})
}

func TestAuthentication_GetUserID(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()
		_, err := auth.GetUserID("")
		if err == nil {
			t.Fatal("expected error for empty token")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, err := auth.GetUserID("unknown")
		if err == nil {
			t.Fatal("expected error for unknown token")
		}
	})

	t.Run("valid token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("uid-token", "user-42", time.Now().Add(time.Hour), []string{"read"})
		userID, err := auth.GetUserID("uid-token")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if userID != "user-42" {
			t.Errorf("expected user-42, got %s", userID)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("uid-expired", "user-42", time.Now().Add(-time.Hour), []string{"read"})
		_, err := auth.GetUserID("uid-expired")
		if err == nil {
			t.Fatal("expected error for expired token")
		}
	})
}

func TestAuthentication_CheckPermission(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()
		_, err := auth.CheckPermission("", "read")
		if err == nil {
			t.Fatal("expected error for empty token")
		}
	})

	t.Run("empty permission", func(t *testing.T) {
		t.Parallel()
		_, err := auth.CheckPermission("token", "")
		if err == nil {
			t.Fatal("expected error for empty permission")
		}
	})

	t.Run("token not found", func(t *testing.T) {
		t.Parallel()
		_, err := auth.CheckPermission("unknown", "read")
		if err == nil {
			t.Fatal("expected error for unknown token")
		}
	})

	t.Run("has permission", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("perm-token", "user-1", time.Now().Add(time.Hour), []string{"read", "write"})
		has, err := auth.CheckPermission("perm-token", "read")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !has {
			t.Error("expected true for read permission")
		}
	})

	t.Run("wildcard permission", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("wildcard-token", "user-1", time.Now().Add(time.Hour), []string{"*"})
		has, err := auth.CheckPermission("wildcard-token", "admin")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !has {
			t.Error("expected true with wildcard permission")
		}
	})

	t.Run("no permission", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("noperm-token", "user-1", time.Now().Add(time.Hour), []string{"read"})
		has, err := auth.CheckPermission("noperm-token", "admin")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if has {
			t.Error("expected false for missing permission")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("exp-perm-token", "user-1", time.Now().Add(-time.Hour), []string{"read"})
		_, err := auth.CheckPermission("exp-perm-token", "read")
		if err == nil {
			t.Fatal("expected error for expired token")
		}
	})
}

func TestAuthentication_GetTokenPermissions(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()
		_, err := auth.GetTokenPermissions("")
		if err == nil {
			t.Fatal("expected error for empty token")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, err := auth.GetTokenPermissions("unknown")
		if err == nil {
			t.Fatal("expected error for unknown token")
		}
	})

	t.Run("valid token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("perms-token", "user-1", time.Now().Add(time.Hour), []string{"read", "write"})
		perms, err := auth.GetTokenPermissions("perms-token")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(perms) != 2 {
			t.Errorf("expected 2 permissions, got %d", len(perms))
		}
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("exp-perms-token", "user-1", time.Now().Add(-time.Hour), []string{"read"})
		_, err := auth.GetTokenPermissions("exp-perms-token")
		if err == nil {
			t.Fatal("expected error for expired token")
		}
	})
}

func TestAuthentication_RevokeToken(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()
		err := auth.RevokeToken("")
		if err == nil {
			t.Fatal("expected error for empty token")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		err := auth.RevokeToken("unknown")
		if err == nil {
			t.Fatal("expected error for unknown token")
		}
	})

	t.Run("valid revoke", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("revoke-token", "user-1", time.Now().Add(time.Hour), []string{"read"})
		err := auth.RevokeToken("revoke-token")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		_, err = auth.ValidateToken("revoke-token")
		if err == nil {
			t.Fatal("expected error for revoked token")
		}
	})
}

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()

	t.Run("empty header", func(t *testing.T) {
		_, err := ExtractBearerToken("")
		if err == nil {
			t.Fatal("expected error for empty header")
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := ExtractBearerToken("Bearer")
		if err == nil {
			t.Fatal("expected error for invalid format")
		}
	})

	t.Run("wrong scheme", func(t *testing.T) {
		_, err := ExtractBearerToken("Basic abc123")
		if err == nil {
			t.Fatal("expected error for wrong scheme")
		}
	})

	t.Run("valid bearer", func(t *testing.T) {
		token, err := ExtractBearerToken("Bearer abc123xyz")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "abc123xyz" {
			t.Errorf("expected abc123xyz, got %s", token)
		}
	})
}

func TestAuthentication_GetTokenInfo(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		info := auth.GetTokenInfo("unknown")
		if info["error"] != "token not found" {
			t.Errorf("expected token not found error, got %v", info["error"])
		}
	})

	t.Run("valid token", func(t *testing.T) {
		t.Parallel()
		_ = auth.RegisterToken("info-token", "user-1", time.Now().Add(time.Hour), []string{"read"})
		info := auth.GetTokenInfo("info-token")
		if info["user_id"] != "user-1" {
			t.Errorf("expected user-1, got %v", info["user_id"])
		}
		if info["is_expired"].(bool) {
			t.Error("expected not expired")
		}
	})
}

func TestAuthentication_GetAllTokens(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	tokens := auth.GetAllTokens()
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}

	_ = auth.RegisterToken("t1", "u1", time.Now().Add(time.Hour), []string{"read"})
	_ = auth.RegisterToken("t2", "u2", time.Now().Add(time.Hour), []string{"write"})

	tokens = auth.GetAllTokens()
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}
}

func TestAuthentication_CleanupExpiredTokens(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	_ = auth.RegisterToken("active", "u1", time.Now().Add(time.Hour), []string{"read"})
	_ = auth.RegisterToken("expired", "u2", time.Now().Add(-time.Hour), []string{"read"})

	count := auth.CleanupExpiredTokens()
	if count != 1 {
		t.Errorf("expected 1 cleaned up, got %d", count)
	}

	tokens := auth.GetAllTokens()
	if len(tokens) != 1 {
		t.Errorf("expected 1 remaining token, got %d", len(tokens))
	}
}

func TestAuthentication_GetTokenCount(t *testing.T) {
	t.Parallel()
	auth := NewAuthentication()

	if auth.GetTokenCount() != 0 {
		t.Errorf("expected 0, got %d", auth.GetTokenCount())
	}

	_ = auth.RegisterToken("t1", "u1", time.Now().Add(time.Hour), []string{"read"})
	if auth.GetTokenCount() != 1 {
		t.Errorf("expected 1, got %d", auth.GetTokenCount())
	}
}

func TestClassifyAuthenticationCoveragePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		total    int
		active   int
		expired  int
		expected string
	}{
		{"unconfigured", 0, 0, 0, "auth-unconfigured"},
		{"expired only", 2, 0, 2, "auth-expired-only"},
		{"active only", 2, 2, 0, "auth-active-only"},
		{"mixed", 3, 2, 1, "auth-mixed"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAuthenticationCoveragePosture(tt.total, tt.active, tt.expired)
			if got != tt.expected {
				t.Errorf("classifyAuthenticationCoveragePosture(%d, %d, %d) = %q, want %q",
					tt.total, tt.active, tt.expired, got, tt.expected)
			}
		})
	}
}

func TestClassifyAuthenticationRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		total    int
		active   int
		expired  int
		expected string
	}{
		{"unobserved", 0, 0, 0, "auth-unobserved"},
		{"degraded", 1, 0, 1, "auth-degraded"},
		{"aging", 3, 1, 2, "auth-aging"},
		{"ready", 2, 2, 0, "auth-ready"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAuthenticationRuntimePosture(tt.total, tt.active, tt.expired)
			if got != tt.expected {
				t.Errorf("classifyAuthenticationRuntimePosture(%d, %d, %d) = %q, want %q",
					tt.total, tt.active, tt.expired, got, tt.expected)
			}
		})
	}
}

func TestBuildAuthenticationReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coveragePosture string
		runtimePosture  string
		expected        string
	}{
		{
			name:            "degraded",
			coveragePosture: "auth-expired-only",
			runtimePosture:  "auth-degraded",
			expected:        "authentication runtime has no active tokens; refresh credentials before treating the surface as available",
		},
		{
			name:            "aging",
			coveragePosture: "auth-mixed",
			runtimePosture:  "auth-aging",
			expected:        "authentication runtime has more expired than active tokens; verify token rotation and cleanup cadence",
		},
		{
			name:            "mixed",
			coveragePosture: "auth-mixed",
			runtimePosture:  "auth-ready",
			expected:        "authentication runtime has mixed active and expired tokens; continue observing token freshness",
		},
		{
			name:            "active only",
			coveragePosture: "auth-active-only",
			runtimePosture:  "auth-ready",
			expected:        "authentication runtime has active tokens available and no expired-token drift",
		},
		{
			name:            "default",
			coveragePosture: "auth-unconfigured",
			runtimePosture:  "auth-unobserved",
			expected:        "authentication runtime has not been configured with active tokens yet",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildAuthenticationReliabilityHint(tt.coveragePosture, tt.runtimePosture)
			if got != tt.expected {
				t.Errorf("buildAuthenticationReliabilityHint(%q, %q) = %q, want %q",
					tt.coveragePosture, tt.runtimePosture, got, tt.expected)
			}
		})
	}
}
