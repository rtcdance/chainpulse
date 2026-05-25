package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_NewCORSMiddleware(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	if m == nil {
		t.Fatal("NewCORSMiddleware returned nil")
	}
	if len(m.allowedOrigins) != 1 || m.allowedOrigins[0] != "*" {
		t.Error("default origin not *")
	}
	if len(m.allowedMethods) < 3 {
		t.Error("expected default methods")
	}
	if m.maxAge != 86400 {
		t.Errorf("maxAge = %d, want 86400", m.maxAge)
	}
}

func TestCORSMiddleware_SetAllowedOrigins(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedOrigins([]string{"https://example.com", "https://test.com"})
	if len(m.allowedOrigins) != 2 {
		t.Errorf("allowedOrigins len = %d", len(m.allowedOrigins))
	}
}

func TestCORSMiddleware_SetAllowedMethods(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedMethods([]string{"GET", "POST"})
	if len(m.allowedMethods) != 2 {
		t.Error("allowedMethods not set")
	}
}

func TestCORSMiddleware_SetAllowedHeaders(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedHeaders([]string{"Content-Type"})
	if len(m.allowedHeaders) != 1 {
		t.Error("allowedHeaders not set")
	}
}

func TestCORSMiddleware_SetMaxAge(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetMaxAge(3600)
	if m.maxAge != 3600 {
		t.Errorf("maxAge = %d", m.maxAge)
	}
}

func TestCORSMiddleware_isOriginAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origins []string
		test    string
		want    bool
	}{
		{"wildcard", []string{"*"}, "https://evil.com", true},
		{"exact_match", []string{"https://example.com"}, "https://example.com", true},
		{"no_match", []string{"https://example.com"}, "https://evil.com", false},
		{"multiple_match", []string{"https://a.com", "https://b.com"}, "https://b.com", true},
		{"multiple_no_match", []string{"https://a.com", "https://b.com"}, "https://c.com", false},
		{"empty_allowed", []string{}, "https://test.com", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := NewCORSMiddleware(nil)
			m.allowedOrigins = tc.origins
			if got := m.isOriginAllowed(tc.test); got != tc.want {
				t.Errorf("isOriginAllowed(%q) = %v, want %v", tc.test, got, tc.want)
			}
		})
	}
}

func TestCORSMiddleware_Integration(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedOrigins([]string{"https://app.chainpulse.io"})
	m.SetAllowedMethods([]string{"GET", "OPTIONS"})
	m.SetAllowedHeaders([]string{"X-API-Key"})
	m.SetMaxAge(7200)

	if !m.isOriginAllowed("https://app.chainpulse.io") {
		t.Error("should allow configured origin")
	}
	if m.isOriginAllowed("https://evil.com") {
		t.Error("should not allow unconfigured origin")
	}
}

func TestCORSMiddleware_Handler_NoOriginHeader(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCORSMiddleware_Handler_AllowedOrigin(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedOrigins([]string{"https://example.com"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("expected Access-Control-Allow-Origin header")
	}
}

func TestCORSMiddleware_Handler_DisallowedOrigin(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedOrigins([]string{"https://example.com"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no Access-Control-Allow-Origin header for disallowed origin")
	}
}

func TestCORSMiddleware_Handler_OptionsRequest(t *testing.T) {
	t.Parallel()

	m := NewCORSMiddleware(nil)
	m.SetAllowedOrigins([]string{"https://example.com"})
	m.SetAllowedMethods([]string{"GET", "POST"})
	m.SetAllowedHeaders([]string{"Content-Type", "Authorization"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}
