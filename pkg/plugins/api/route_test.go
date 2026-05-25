package api

import (
	"net/http"
	"testing"
)

func TestNewRoute(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	if r.ID != "test-route" {
		t.Errorf("expected ID test-route, got %s", r.ID)
	}
	if r.Pattern != "/api/v1/users" {
		t.Errorf("expected Pattern /api/v1/users, got %s", r.Pattern)
	}
	if r.Method != "GET" {
		t.Errorf("expected Method GET, got %s", r.Method)
	}
	if len(r.Handlers) != 0 {
		t.Errorf("expected 0 handlers, got %d", len(r.Handlers))
	}
}

func TestRoute_AddHandler(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	h := &RequestHandler{ID: "handler-1", Name: "test handler"}
	if err := r.AddHandler(h); err != nil {
		t.Fatal(err)
	}
	if len(r.Handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(r.Handlers))
	}
}

func TestRoute_AddHandler_Nil(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	err := r.AddHandler(nil)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestRoute_AddHandler_Duplicate(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	h := &RequestHandler{ID: "handler-1"}
	_ = r.AddHandler(h)
	err := r.AddHandler(h)
	if err == nil {
		t.Error("expected error for duplicate handler")
	}
}

func TestRoute_RemoveHandler(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	r.AddHandler(&RequestHandler{ID: "handler-1"})
	r.AddHandler(&RequestHandler{ID: "handler-2"})

	if err := r.RemoveHandler("handler-1"); err != nil {
		t.Fatal(err)
	}
	if len(r.Handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(r.Handlers))
	}
	if r.Handlers[0].ID != "handler-2" {
		t.Errorf("expected handler-2, got %s", r.Handlers[0].ID)
	}
}

func TestRoute_RemoveHandler_NotFound(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	err := r.RemoveHandler("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent handler")
	}
}

func TestRoute_GetHandlers(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	r.AddHandler(&RequestHandler{ID: "handler-1"})
	handlers := r.GetHandlers()
	if len(handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(handlers))
	}
	handlers[0] = nil
	if r.Handlers[0] == nil {
		t.Error("GetHandlers should return a copy, not reference")
	}
}

func TestRoute_SetPriority(t *testing.T) {
	t.Parallel()
	r := NewRoute("test-route", "/api/v1/users", "GET")
	r.SetPriority(10)
	if r.Priority != 10 {
		t.Errorf("expected Priority 10, got %d", r.Priority)
	}
}

func TestRoute_Match_Static(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api/v1/users", "GET")
	params, ok := r.Match("/api/v1/users")
	if !ok {
		t.Error("expected match")
	}
	if len(params) != 0 {
		t.Errorf("expected 0 params, got %d", len(params))
	}
}

func TestRoute_Match_WithParams(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api/v1/users/:id", "GET")
	params, ok := r.Match("/api/v1/users/42")
	if !ok {
		t.Error("expected match")
	}
	if params["id"] != "42" {
		t.Errorf("expected id=42, got %s", params["id"])
	}
}

func TestRoute_Match_NoMatch(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api/v1/users", "GET")
	_, ok := r.Match("/api/v1/admin")
	if ok {
		t.Error("expected no match")
	}
}

func TestRoute_AddMiddleware(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api", "GET")
	mw := func(next http.Handler) http.Handler { return next }
	r.AddMiddleware(mw)
	if len(r.Middleware) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(r.Middleware))
	}
}

func TestRoute_GetMiddleware(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api", "GET")
	mw := func(next http.Handler) http.Handler { return next }
	r.AddMiddleware(mw)
	mwList := r.GetMiddleware()
	if len(mwList) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(mwList))
	}
	mwList[0] = nil
	if r.Middleware[0] == nil {
		t.Error("GetMiddleware should return a copy")
	}
}

func TestRoute_GetMiddleware_Empty(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api", "GET")
	mwList := r.GetMiddleware()
	if len(mwList) != 0 {
		t.Errorf("expected 0 middleware, got %d", len(mwList))
	}
}

func TestRoute_String(t *testing.T) {
	t.Parallel()
	r := NewRoute("test", "/api", "GET")
	r.AddHandler(&RequestHandler{ID: "h1"})
	s := r.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}
