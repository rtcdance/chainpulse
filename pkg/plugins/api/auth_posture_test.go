package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newProductionAuthMiddleware builds the middleware exactly the way the
// production composition root does (bootstrap/security.go and
// cmd/monolithic/chainpulse/main.go): a real TokenValidator, a real RBACChecker
// with the default role table registered, and a real audit logger. Nothing about
// the policy is stubbed, so a test here exercises the shipped decision.
func newProductionAuthMiddleware(t *testing.T) *AuthMiddleware {
	t.Helper()

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewRBACChecker(logger, metrics)
	if err := rbacChecker.RegisterDefaultRoles(); err != nil {
		t.Fatalf("RegisterDefaultRoles: %v", err)
	}
	auditLogger := NewAuditLogger(logger, metrics)

	return NewAuthMiddleware(tokenValidator, rbacChecker, auditLogger, logger, metrics)
}

// mintToken issues a JWT carrying the given roles, as a login would.
func mintToken(t *testing.T, am *AuthMiddleware, roles []string) string {
	t.Helper()

	token, err := am.tokenValidator.GenerateJWT("client", "user", roles, nil, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	return token
}

// TestOperatorSurfacesRequireAuthorization is the T08 acceptance test for
// "anonymous replay/pprof/control must be forbidden" and for the role boundary
// behind it.
//
// The existing auth fuzz test seeds these same paths but only asserts that
// nothing panics, so nothing pinned the actual decision. This test does, using
// the production policy objects.
//
// The three-way assertion is deliberate:
//   - anonymous must be rejected (401), which is the literal requirement;
//   - an authenticated user with the wrong role must ALSO be rejected (403),
//     because "anonymous is blocked" alone would pass even if every logged-in
//     user could replay the DLQ;
//   - the allowed role must get through, and a read endpoint must still work for
//     a viewer, so the test cannot pass by denying everything.
func TestOperatorSurfacesRequireAuthorization(t *testing.T) {
	t.Parallel()

	am := newProductionAuthMiddleware(t)
	viewer := mintToken(t, am, []string{"viewer"})
	operator := mintToken(t, am, []string{"operator"})
	admin := mintToken(t, am, []string{"admin"})

	// "/events" is a read endpoint with no endpoint roles registered, so it is
	// the control case: the middleware must not block it.
	cases := []struct {
		path        string
		anonymous   int
		viewer      int
		operator    int
		admin       int
		description string
	}{
		{"/dlq/replay", 401, 403, 200, 200, "POST /dlq/replay"},
		{"/runtime/control", 401, 403, 200, 200, "GET /runtime/control"},
		{"/admin/keys", 401, 403, 403, 200, "GET /admin/keys is admin-only"},
		{"/events", 401, 200, 200, 200, "read endpoint stays open to any authenticated role"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			guarded := am.Handler(inner)

			attempt := func(t *testing.T, token string) int {
				t.Helper()
				req := httptest.NewRequest(http.MethodPost, tc.path, nil)
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				recorder := httptest.NewRecorder()
				guarded.ServeHTTP(recorder, req)
				return recorder.Code
			}

			if got := attempt(t, ""); got != tc.anonymous {
				t.Errorf("anonymous: got %d, want %d", got, tc.anonymous)
			}
			if got := attempt(t, viewer); got != tc.viewer {
				t.Errorf("viewer: got %d, want %d", got, tc.viewer)
			}
			if got := attempt(t, operator); got != tc.operator {
				t.Errorf("operator: got %d, want %d", got, tc.operator)
			}
			if got := attempt(t, admin); got != tc.admin {
				t.Errorf("admin: got %d, want %d", got, tc.admin)
			}
		})
	}
}

// TestAPIV1PrefixDoesNotBypassTheRoleCheck pins the normalisation the middleware
// performs before the RBAC lookup. The endpoint table is keyed without the
// prefix, so a request to /api/v1/<protected> must reach the SAME decision as
// /<protected>. Without the normalisation the lookup would miss and the endpoint
// would silently require nothing.
func TestAPIV1PrefixDoesNotBypassTheRoleCheck(t *testing.T) {
	t.Parallel()

	am := newProductionAuthMiddleware(t)
	viewer := mintToken(t, am, []string{"viewer"})

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guarded := am.Handler(inner)

	for _, path := range []string{"/dlq/replay", "/api/v1/dlq/replay", "/runtime/control", "/api/v1/runtime/control"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Authorization", "Bearer "+viewer)
		recorder := httptest.NewRecorder()
		guarded.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusForbidden {
			t.Errorf("%s: a viewer must be forbidden (403), got %d — the prefix must not change the decision",
				path, recorder.Code)
		}
	}
}

// TestOperatorSurfacesRejectMalformedCredentials guards the other direction: a
// token that cannot be validated is never treated as authenticated.
func TestOperatorSurfacesRejectMalformedCredentials(t *testing.T) {
	t.Parallel()

	am := newProductionAuthMiddleware(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guarded := am.Handler(inner)

	headers := []struct {
		name  string
		value string
	}{
		{"empty bearer", "Bearer "},
		{"garbage", "Bearer not-a-jwt"},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
	}

	for _, h := range headers {
		t.Run(h.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/dlq/replay", nil)
			req.Header.Set("Authorization", h.value)
			recorder := httptest.NewRecorder()
			guarded.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("got %d, want 401", recorder.Code)
			}
		})
	}
}

// TestRoleCheckHasTeeth is the control for TestOperatorSurfacesRequireAuthorization.
//
// RegisterDefaultRoles is documented as "must be called after NewRBACChecker to
// enable RBAC enforcement", so a composition root that forgets it leaves the
// endpoint-role table empty and every authenticated user passes the check. If a
// viewer were allowed through in BOTH configurations, the acceptance test above
// would not be measuring the role boundary at all.
//
// This asserts the difference: with the default roles registered a viewer is
// forbidden, and with them omitted the same request is allowed. The second half
// deliberately demonstrates the failure mode — it is a property of the policy
// object as shipped, not something this test introduces.
func TestRoleCheckHasTeeth(t *testing.T) {
	t.Parallel()

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	viewer, err := tokenValidator.GenerateJWT("client", "user", []string{"viewer"}, nil, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	guardedWith := func(t *testing.T, registerDefaults bool) int {
		t.Helper()

		rbacChecker := NewRBACChecker(logger, metrics)
		if registerDefaults {
			if err := rbacChecker.RegisterDefaultRoles(); err != nil {
				t.Fatalf("RegisterDefaultRoles: %v", err)
			}
		}

		am := NewAuthMiddleware(tokenValidator, rbacChecker, NewAuditLogger(logger, metrics), logger, metrics)
		inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodPost, "/dlq/replay", nil)
		req.Header.Set("Authorization", "Bearer "+viewer)
		recorder := httptest.NewRecorder()
		am.Handler(inner).ServeHTTP(recorder, req)
		return recorder.Code
	}

	withDefaults := guardedWith(t, true)
	withoutDefaults := guardedWith(t, false)

	if withDefaults != http.StatusForbidden {
		t.Errorf("with RegisterDefaultRoles a viewer must be forbidden (403), got %d", withDefaults)
	}
	if withoutDefaults == withDefaults {
		t.Fatalf("omitting RegisterDefaultRoles produced the same status (%d), so the acceptance test "+
			"does not depend on the role table and would pass with RBAC disabled", withoutDefaults)
	}
	t.Logf("control: viewer on /dlq/replay is %d with default roles and %d without them",
		withDefaults, withoutDefaults)
}
