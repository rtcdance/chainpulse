package core

import (
	"context"
	"errors"
	"testing"
)

func TestHandlerFunc(t *testing.T) {
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	req := NewBaseRequest("GET", "/test", nil, []byte(""), context.Background())
	resp, err := handler.Handle(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestNewAPIRouter(t *testing.T) {
	router := NewAPIRouter()

	if router == nil {
		t.Fatal("expected router, got nil")
	}

	if len(router.handlers) != 0 {
		t.Error("expected empty handlers map")
	}
}

func TestAPIRouterRegister(t *testing.T) {
	router := NewAPIRouter()

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	router.Register("/api/users", handler)

	if len(router.handlers) != 1 {
		t.Error("expected 1 handler registered")
	}
}

func TestAPIRouterRoute(t *testing.T) {
	router := NewAPIRouter()

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	router.Register("/api/users", handler)

	retrieved := router.Route("/api/users")
	if retrieved == nil {
		t.Error("expected handler, got nil")
	}
}

func TestAPIRouterRouteCount(t *testing.T) {
	router := NewAPIRouter()

	if router.RouteCount() != 0 {
		t.Errorf("expected empty route count, got %d", router.RouteCount())
	}

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	router.Register("/api/users", handler)
	router.Register("/health", handler)

	if router.RouteCount() != 2 {
		t.Errorf("expected 2 routes, got %d", router.RouteCount())
	}
}

func TestAPIRouterRuntimeMetricsUnobserved(t *testing.T) {
	router := NewAPIRouter()

	metrics := router.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "router-empty" {
		t.Fatalf("expected router-empty, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "router-unobserved" {
		t.Fatalf("expected router-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestAPIRouterRuntimeMetricsWatch(t *testing.T) {
	router := NewAPIRouter()
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	router.Register("/api/users", handler)

	metrics := router.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "router-routes-only" {
		t.Fatalf("expected router-routes-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "router-watch" {
		t.Fatalf("expected router-watch, got %v", metrics["runtime_posture"])
	}
}

func TestAPIRouterRuntimeMetricsReady(t *testing.T) {
	router := NewAPIRouter()
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})
	middleware := func(next Handler) Handler {
		return HandlerFunc(func(req Request) (Response, error) {
			return next.Handle(req)
		})
	}

	router.Use(middleware)
	router.Register("/api/users", handler)

	metrics := router.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "router-guarded" {
		t.Fatalf("expected router-guarded, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "router-ready" {
		t.Fatalf("expected router-ready, got %v", metrics["runtime_posture"])
	}
}

func TestAPIRouterRouteNotFound(t *testing.T) {
	router := NewAPIRouter()

	retrieved := router.Route("/api/nonexistent")
	if retrieved != nil {
		t.Error("expected nil for non-existent route")
	}
}

func TestAPIRouterHandle(t *testing.T) {
	router := NewAPIRouter()

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("Success"))
		return resp, nil
	})

	router.Register("/api/users", handler)

	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp, err := router.Handle(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestAPIRouterHandleNotFound(t *testing.T) {
	router := NewAPIRouter()

	req := NewBaseRequest("GET", "/api/nonexistent", nil, []byte(""), context.Background())
	resp, err := router.Handle(req)

	if err == nil {
		t.Error("expected error for non-existent route")
	}

	if resp.Status() != 404 {
		t.Errorf("expected status 404, got %d", resp.Status())
	}
}

func TestAPIRouterMiddleware(t *testing.T) {
	router := NewAPIRouter()

	// Create middleware that adds a header
	middleware := func(next Handler) Handler {
		return HandlerFunc(func(req Request) (Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware", "applied")
			}
			return resp, err
		})
	}

	router.Use(middleware)

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	router.Register("/api/users", handler)

	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp, err := router.Handle(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.Header("X-Middleware") != "applied" {
		t.Error("expected middleware header")
	}
}

func TestDefaultErrorMapper(t *testing.T) {
	mapper := NewDefaultErrorMapper()

	// Test nil error
	status, _, _ := mapper.MapError(nil)
	if status != 200 {
		t.Errorf("expected status 200 for nil error, got %d", status)
	}

	// Test error
	err := errors.New("test error")
	status, headers, body := mapper.MapError(err)

	if status != 500 {
		t.Errorf("expected status 500, got %d", status)
	}

	if headers["Content-Type"] != "application/json" {
		t.Error("expected Content-Type header")
	}

	if len(body) == 0 {
		t.Error("expected error body")
	}
}

func TestNewAPILayer(t *testing.T) {
	layer := NewAPILayer()

	if layer == nil {
		t.Fatal("expected API layer, got nil")
	}

	if layer.router == nil {
		t.Error("expected router")
	}

	if layer.errorMapper == nil {
		t.Error("expected error mapper")
	}
}

func TestAPILayerRegisterHandler(t *testing.T) {
	layer := NewAPILayer()

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	layer.RegisterHandler("/api/users", handler)

	// Verify handler is registered
	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp := layer.Handle(req)

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestAPILayerHandle(t *testing.T) {
	layer := NewAPILayer()

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("Success"))
		return resp, nil
	})

	layer.RegisterHandler("/api/users", handler)

	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp := layer.Handle(req)

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestAPILayerHandleError(t *testing.T) {
	layer := NewAPILayer()

	handler := HandlerFunc(func(req Request) (Response, error) {
		return nil, errors.New("handler error")
	})

	layer.RegisterHandler("/api/users", handler)

	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp := layer.Handle(req)

	if resp.Status() != 500 {
		t.Errorf("expected status 500, got %d", resp.Status())
	}
}

func TestAPILayerSetErrorMapper(t *testing.T) {
	layer := NewAPILayer()

	customMapper := &testErrorMapper{}
	layer.SetErrorMapper(customMapper)

	handler := HandlerFunc(func(req Request) (Response, error) {
		return nil, errors.New("test error")
	})

	layer.RegisterHandler("/api/users", handler)

	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp := layer.Handle(req)

	if resp.Status() != 418 {
		t.Errorf("expected status 418, got %d", resp.Status())
	}
}

func TestAPILayerMiddleware(t *testing.T) {
	layer := NewAPILayer()

	middleware := func(next Handler) Handler {
		return HandlerFunc(func(req Request) (Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware", "applied")
			}
			return resp, err
		})
	}

	layer.Use(middleware)

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	layer.RegisterHandler("/api/users", handler)

	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())
	resp := layer.Handle(req)

	if resp.Header("X-Middleware") != "applied" {
		t.Error("expected middleware header")
	}
}

func TestAPILayerRuntimeMetricsUnobserved(t *testing.T) {
	layer := NewAPILayer()
	layer.router = NewAPIRouter()
	layer.errorMapper = nil

	metrics := layer.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "layer-empty" {
		t.Fatalf("expected layer-empty, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "layer-unobserved" {
		t.Fatalf("expected layer-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestAPILayerRuntimeMetricsWatch(t *testing.T) {
	layer := NewAPILayer()
	layer.errorMapper = nil
	layer.RegisterHandler("/api/users", HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	}))

	metrics := layer.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "layer-routes-only" {
		t.Fatalf("expected layer-routes-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "layer-watch" {
		t.Fatalf("expected layer-watch, got %v", metrics["runtime_posture"])
	}
}

func TestAPILayerRuntimeMetricsHardened(t *testing.T) {
	layer := NewAPILayer()
	layer.Use(func(next Handler) Handler {
		return HandlerFunc(func(req Request) (Response, error) {
			return next.Handle(req)
		})
	})
	layer.RegisterHandler("/api/users", HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	}))

	metrics := layer.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "layer-guarded" {
		t.Fatalf("expected layer-guarded, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "layer-hardened" {
		t.Fatalf("expected layer-hardened, got %v", metrics["runtime_posture"])
	}
}

// testErrorMapper is a test helper
type testErrorMapper struct{}

func (m *testErrorMapper) MapError(err error) (int, map[string]string, []byte) {
	return 418, map[string]string{}, []byte("I'm a teapot")
}
