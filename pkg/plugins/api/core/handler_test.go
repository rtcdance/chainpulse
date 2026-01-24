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

// testErrorMapper is a test helper
type testErrorMapper struct{}

func (m *testErrorMapper) MapError(err error) (int, map[string]string, []byte) {
	return 418, map[string]string{}, []byte("I'm a teapot")
}
