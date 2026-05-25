package shared

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/plugins/api/core"
)

func TestNewMiddlewareChain(t *testing.T) {
	t.Parallel()

	c := NewMiddlewareChain()
	if c == nil {
		t.Fatal("NewMiddlewareChain returned nil")
	}
	if len(c.middlewares) != 0 {
		t.Error("expected empty middlewares slice")
	}
}

func TestMiddlewareChain_Add(t *testing.T) {
	t.Parallel()

	c := NewMiddlewareChain()
	mw := func(next core.Handler) core.Handler {
		return next
	}
	c.Add(mw)
	if len(c.middlewares) != 1 {
		t.Errorf("middlewares len = %d, want 1", len(c.middlewares))
	}
}

func TestMiddlewareChain_Build(t *testing.T) {
	c := NewMiddlewareChain()

	orderLog := make([]string, 0)

	mw1 := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			orderLog = append(orderLog, "mw1")
			return next.Handle(req)
		})
	}
	mw2 := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			orderLog = append(orderLog, "mw2")
			return next.Handle(req)
		})
	}

	c.Add(mw1, mw2)
	handler := c.Build(core.HandlerFunc(func(req core.Request) (core.Response, error) {
		orderLog = append(orderLog, "handler")
		return core.NewBaseResponse(nil), nil
	}))

	req := core.NewBaseRequest(nil, "GET", "/test", nil, nil)
	handler.Handle(req)

	if len(orderLog) != 3 {
		t.Fatalf("expected 3 log entries, got %d: %v", len(orderLog), orderLog)
	}
	if orderLog[0] != "mw1" {
		t.Errorf("expected mw1 first, got %q", orderLog[0])
	}
	if orderLog[2] != "handler" {
		t.Errorf("expected handler last, got %q", orderLog[2])
	}
}

func TestNewSecurityMiddlewareGroup(t *testing.T) {
	t.Parallel()

	g := NewSecurityMiddlewareGroup()
	if g == nil {
		t.Fatal("NewSecurityMiddlewareGroup returned nil")
	}
}

func TestSecurityMiddlewareGroup_SetAuthMiddleware(t *testing.T) {
	t.Parallel()

	g := NewSecurityMiddlewareGroup()
	mw := func(next core.Handler) core.Handler {
		return next
	}
	g.SetAuthMiddleware(mw)
}

func TestSecurityMiddlewareGroup_SetTLSMiddleware(t *testing.T) {
	t.Parallel()

	g := NewSecurityMiddlewareGroup()
	mw := func(next core.Handler) core.Handler {
		return next
	}
	g.SetTLSMiddleware(mw)
}

func TestSecurityMiddlewareGroup_GetMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		g := NewSecurityMiddlewareGroup()
		mws := g.GetMiddlewares()
		if len(mws) != 0 {
			t.Errorf("expected 0 middlewares, got %d", len(mws))
		}
	})

	t.Run("with auth and tls", func(t *testing.T) {
		t.Parallel()
		g := NewSecurityMiddlewareGroup()
		authMw := func(next core.Handler) core.Handler { return next }
		tlsMw := func(next core.Handler) core.Handler { return next }
		g.SetAuthMiddleware(authMw)
		g.SetTLSMiddleware(tlsMw)
		mws := g.GetMiddlewares()
		if len(mws) != 2 {
			t.Errorf("expected 2 middlewares, got %d", len(mws))
		}
	})
}

func TestNewObservabilityMiddlewareGroup(t *testing.T) {
	t.Parallel()

	g := NewObservabilityMiddlewareGroup()
	if g == nil {
		t.Fatal("NewObservabilityMiddlewareGroup returned nil")
	}
}

func TestObservabilityMiddlewareGroup_SettersAndGetters(t *testing.T) {
	t.Parallel()

	g := NewObservabilityMiddlewareGroup()
	healthMw := func(next core.Handler) core.Handler { return next }
	monitoringMw := func(next core.Handler) core.Handler { return next }

	g.SetHealthMiddleware(healthMw)
	g.SetMonitoringMiddleware(monitoringMw)

	mws := g.GetMiddlewares()
	if len(mws) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(mws))
	}
}

func TestObservabilityMiddlewareGroup_GetMiddlewaresEmpty(t *testing.T) {
	t.Parallel()

	g := NewObservabilityMiddlewareGroup()
	mws := g.GetMiddlewares()
	if len(mws) != 0 {
		t.Errorf("expected 0 middlewares, got %d", len(mws))
	}
}

func TestNewPerformanceMiddlewareGroup(t *testing.T) {
	t.Parallel()

	g := NewPerformanceMiddlewareGroup()
	if g == nil {
		t.Fatal("NewPerformanceMiddlewareGroup returned nil")
	}
}

func TestPerformanceMiddlewareGroup_SettersAndGetters(t *testing.T) {
	t.Parallel()

	g := NewPerformanceMiddlewareGroup()
	compMw := func(next core.Handler) core.Handler { return next }
	batchMw := func(next core.Handler) core.Handler { return next }
	poolMw := func(next core.Handler) core.Handler { return next }

	g.SetCompressionMiddleware(compMw)
	g.SetBatchingMiddleware(batchMw)
	g.SetPoolMiddleware(poolMw)

	mws := g.GetMiddlewares()
	if len(mws) != 3 {
		t.Errorf("expected 3 middlewares, got %d", len(mws))
	}
}

func TestPerformanceMiddlewareGroup_GetMiddlewaresEmpty(t *testing.T) {
	t.Parallel()

	g := NewPerformanceMiddlewareGroup()
	mws := g.GetMiddlewares()
	if len(mws) != 0 {
		t.Errorf("expected 0 middlewares, got %d", len(mws))
	}
}

func TestNewErrorHandlingMiddleware(t *testing.T) {
	t.Parallel()

	m := NewErrorHandlingMiddleware(nil)
	if m == nil {
		t.Fatal("NewErrorHandlingMiddleware returned nil")
	}
}

func TestErrorHandlingMiddleware_Middleware(t *testing.T) {
	t.Parallel()

	t.Run("no error mapper", func(t *testing.T) {
		t.Parallel()
		m := NewErrorHandlingMiddleware(nil)
		mw := m.Middleware()

		handler := core.HandlerFunc(func(req core.Request) (core.Response, error) {
			return core.NewBaseResponse(nil), nil
		})

		req := core.NewBaseRequest(nil, "GET", "/test", nil, nil)
		resp, err := mw(handler).Handle(req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if resp == nil {
			t.Error("expected non-nil response")
		}
	})
}

func TestNewMiddlewareBuilder(t *testing.T) {
	t.Parallel()

	b := NewMiddlewareBuilder()
	if b == nil {
		t.Fatal("NewMiddlewareBuilder returned nil")
	}
	if b.chain == nil {
		t.Fatal("builder chain is nil")
	}
}

func TestMiddlewareBuilder_Fluent(t *testing.T) {
	t.Parallel()

	b := NewMiddlewareBuilder()
	security := NewSecurityMiddlewareGroup()
	observability := NewObservabilityMiddlewareGroup()
	performance := NewPerformanceMiddlewareGroup()
	errMw := NewErrorHandlingMiddleware(nil)

	handler := b.
		WithSecurity(security).
		WithObservability(observability).
		WithPerformance(performance).
		WithErrorHandling(errMw.Middleware()).
		Build(core.HandlerFunc(func(req core.Request) (core.Response, error) {
			return core.NewBaseResponse(nil), nil
		}))

	if handler == nil {
		t.Fatal("Build returned nil handler")
	}

	req := core.NewBaseRequest(nil, "GET", "/test", nil, nil)
	resp, err := handler.Handle(req)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Error("expected non-nil response")
	}
}

func TestNewMiddlewareRegistry(t *testing.T) {
	t.Parallel()

	r := NewMiddlewareRegistry()
	if r == nil {
		t.Fatal("NewMiddlewareRegistry returned nil")
	}
	if r.security == nil {
		t.Fatal("registry security group is nil")
	}
	if r.observability == nil {
		t.Fatal("registry observability group is nil")
	}
	if r.performance == nil {
		t.Fatal("registry performance group is nil")
	}
}

func TestMiddlewareRegistry_Getters(t *testing.T) {
	t.Parallel()

	r := NewMiddlewareRegistry()

	sg := r.GetSecurityGroup()
	if sg == nil {
		t.Fatal("GetSecurityGroup returned nil")
	}

	og := r.GetObservabilityGroup()
	if og == nil {
		t.Fatal("GetObservabilityGroup returned nil")
	}

	pg := r.GetPerformanceGroup()
	if pg == nil {
		t.Fatal("GetPerformanceGroup returned nil")
	}
}

func TestMiddlewareRegistry_SetErrorHandling(t *testing.T) {
	t.Parallel()

	r := NewMiddlewareRegistry()
	em := NewErrorHandlingMiddleware(nil)
	r.SetErrorHandling(em)
}

func TestMiddlewareRegistry_BuildChain(t *testing.T) {
	t.Parallel()

	r := NewMiddlewareRegistry()
	em := NewErrorHandlingMiddleware(nil)
	r.SetErrorHandling(em)

	chain := r.BuildChain()
	if chain == nil {
		t.Fatal("BuildChain returned nil")
	}
}

func TestMiddlewareRegistry_GetAllMiddleware(t *testing.T) {
	t.Parallel()

	r := NewMiddlewareRegistry()
	em := NewErrorHandlingMiddleware(nil)
	r.SetErrorHandling(em)

	mws := r.GetAllMiddleware()
	if len(mws) != 1 {
		t.Errorf("expected 1 middleware (error handling), got %d", len(mws))
	}
}

func TestMiddlewareRegistry_GetRuntimeMetrics(t *testing.T) {
	t.Parallel()

	r := NewMiddlewareRegistry()
	em := NewErrorHandlingMiddleware(nil)
	r.SetErrorHandling(em)

	metrics := r.GetRuntimeMetrics()
	if metrics == nil {
		t.Fatal("GetRuntimeMetrics returned nil")
	}
	if metrics["total_middleware"].(int) != 1 {
		t.Errorf("expected 1 total middleware, got %v", metrics["total_middleware"])
	}
	if metrics["error_handling_enabled"].(bool) != true {
		t.Errorf("expected error handling enabled, got %v", metrics["error_handling_enabled"])
	}
}

func TestClassifyMiddlewareCoveragePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		total         int
		security      int
		observability int
		performance   int
		errorHandling bool
		expected      string
	}{
		{"unconfigured", 0, 0, 0, 0, false, "middleware-unconfigured"},
		{"full stack", 4, 1, 1, 1, true, "middleware-full-stack"},
		{"error only", 1, 0, 0, 0, true, "middleware-error-only"},
		{"partial", 2, 1, 0, 0, true, "middleware-partial"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyMiddlewareCoveragePosture(tt.total, tt.security, tt.observability, tt.performance, tt.errorHandling)
			if got != tt.expected {
				t.Errorf("classifyMiddlewareCoveragePosture(%d, %d, %d, %d, %v) = %q, want %q",
					tt.total, tt.security, tt.observability, tt.performance, tt.errorHandling, got, tt.expected)
			}
		})
	}
}

func TestClassifyMiddlewareRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		total         int
		security      int
		observability int
		performance   int
		errorHandling bool
		expected      string
	}{
		{"unobserved", 0, 0, 0, 0, false, "middleware-unobserved"},
		{"degraded no security", 2, 0, 1, 0, true, "middleware-degraded"},
		{"degraded no observability", 2, 1, 0, 0, true, "middleware-degraded"},
		{"watch no error handling", 2, 1, 1, 0, false, "middleware-watch"},
		{"balanced no performance", 2, 1, 1, 0, true, "middleware-balanced"},
		{"ready", 3, 1, 1, 1, true, "middleware-ready"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyMiddlewareRuntimePosture(tt.total, tt.security, tt.observability, tt.performance, tt.errorHandling)
			if got != tt.expected {
				t.Errorf("classifyMiddlewareRuntimePosture(%d, %d, %d, %d, %v) = %q, want %q",
					tt.total, tt.security, tt.observability, tt.performance, tt.errorHandling, got, tt.expected)
			}
		})
	}
}

func TestBuildMiddlewareReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coveragePosture string
		runtimePosture  string
		expected        string
	}{
		{
			name:            "degraded",
			coveragePosture: "middleware-full-stack",
			runtimePosture:  "middleware-degraded",
			expected:        "middleware registry is missing core security or observability coverage; verify baseline protection before relying on the stack",
		},
		{
			name:            "watch",
			coveragePosture: "middleware-full-stack",
			runtimePosture:  "middleware-watch",
			expected:        "middleware registry has core groups but lacks error-handling wiring; verify mapped error behavior before treating the stack as complete",
		},
		{
			name:            "error only",
			coveragePosture: "middleware-error-only",
			runtimePosture:  "middleware-ready",
			expected:        "middleware registry is only exposing error handling; add baseline security and observability coverage before relying on the chain",
		},
		{
			name:            "partial",
			coveragePosture: "middleware-partial",
			runtimePosture:  "middleware-ready",
			expected:        "middleware registry has partial group coverage; continue observing whether the configured stack matches route expectations",
		},
		{
			name:            "full stack",
			coveragePosture: "middleware-full-stack",
			runtimePosture:  "middleware-ready",
			expected:        "middleware registry has a balanced stack with security, observability, performance, and error handling coverage",
		},
		{
			name:            "default",
			coveragePosture: "unknown",
			runtimePosture:  "unknown",
			expected:        "middleware registry has not been configured yet",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildMiddlewareReliabilityHint(tt.coveragePosture, tt.runtimePosture)
			if got != tt.expected {
				t.Errorf("buildMiddlewareReliabilityHint(%q, %q) = %q, want %q",
					tt.coveragePosture, tt.runtimePosture, got, tt.expected)
			}
		})
	}
}
