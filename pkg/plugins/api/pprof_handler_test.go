package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterPprofEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPprofEndpoints(mux)

	endpoints := []string{
		"/debug/pprof/",
		"/debug/pprof/allocs",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
		"/debug/pprof/block",
		"/debug/pprof/mutex",
		"/debug/pprof/profile",
		"/debug/pprof/trace",
		"/debug/pprof/symbol",
		"/debug/pprof/cmdline",
	}

	for _, path := range endpoints {
		req := httptest.NewRequest("GET", path, nil)
		handler, pattern := mux.Handler(req)
		if handler == nil || pattern == "" {
			t.Errorf("expected handler for %s, got nil", path)
		}
	}
}

func TestRegisterPprofEndpoints_NoPanic(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPprofEndpoints(mux)
	mux2 := http.NewServeMux()
	registerPprof(mux2)
}

func TestIsPprofPath_True(t *testing.T) {
	paths := []string{
		"/debug/pprof",
		"/debug/pprof/",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine?debug=1",
	}
	for _, path := range paths {
		if !IsPprofPath(path) {
			t.Errorf("IsPprofPath(%q) should be true", path)
		}
	}
}

func TestIsPprofPath_False(t *testing.T) {
	paths := []string{
		"/api/events",
		"/health",
		"/debug",
		"/pprof/heap",
		"",
	}
	for _, path := range paths {
		if IsPprofPath(path) {
			t.Errorf("IsPprofPath(%q) should be false", path)
		}
	}
}

func TestIsPprofPath_CaseSensitive(t *testing.T) {
	if IsPprofPath(strings.ToUpper("/debug/pprof/heap")) {
		t.Fatal("IsPprofPath should be case-sensitive")
	}
}
