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

// TestPprofGuardIsNotBypassableByPathShape is the T08 acceptance test for
// "anonymous pprof must be forbidden".
//
// Background: the guard inspects r.URL.Path with
// strings.HasPrefix(path, "/debug/pprof/"), i.e. the RAW path, while the router
// normalises paths. A guard that reads the raw path and a router that cleans it
// is the classic middleware bypass, so this test probes the shapes that carry no
// guarded prefix yet can resolve to a pprof handler.
//
// Measured behaviour with CHAINPULSE_PPROF_ENABLED unset:
//
//	/debug/pprof/heap          -> 404  (the guard matched)
//	//debug/pprof/heap         -> 307  Location: /debug/pprof/heap
//	///debug/pprof/heap        -> 307
//	/a/../debug/pprof/heap     -> 307
//	/./debug/pprof/heap        -> 307
//	/debug//pprof/heap         -> 307
//	/debug/pprof%2fheap        -> 404
//
// So the guard is NOT bypassable today, but not because of its own logic:
// http.ServeMux normalises and issues a 307 BEFORE dispatching, and that
// redirect carries no profile body. A client that follows it lands on the
// guarded 404.
//
// That means the safety is incidental to ServeMux. Replacing it with a router
// that dispatches unclean paths, or moving this guard below path normalisation,
// would silently remove the protection — which is why the property is pinned
// here. The assertion is deliberately just "never 200": a leak is a 200, and a
// redirect discloses nothing.
func TestPprofGuardIsNotBypassableByPathShape(t *testing.T) {
	t.Setenv("CHAINPULSE_PPROF_ENABLED", "")

	mux := http.NewServeMux()
	RegisterPprofEndpoints(mux)
	server := httptest.NewServer(pprofGuardMiddleware(mux))
	t.Cleanup(server.Close)

	client := &http.Client{
		// Do not follow redirects: a redirect is an acceptable outcome, a 200 is
		// the failure we are looking for.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	shapes := []string{
		"/debug/pprof/heap",
		"/debug/pprof/",
		"//debug/pprof/heap",
		"///debug/pprof/heap",
		"/a/../debug/pprof/heap",
		"/./debug/pprof/heap",
		"/debug/pprof/heap/",
		"/debug//pprof/heap",
		"/debug/pprof/heap?debug=1",
		"/debug/pprof%2fheap",
	}

	for _, shape := range shapes {
		t.Run(shape, func(t *testing.T) {
			resp, err := client.Get(server.URL + shape)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				t.Errorf("with the debug endpoints disabled, %q returned 200: the guard was bypassed", shape)
			}
		})
	}
}

// TestPprofGuardAllowsWhenExplicitlyEnabled pins that the guard is a gate and
// not a removal: with CHAINPULSE_PPROF_ENABLED=true the endpoints must serve.
func TestPprofGuardAllowsWhenExplicitlyEnabled(t *testing.T) {
	t.Setenv("CHAINPULSE_PPROF_ENABLED", "true")

	mux := http.NewServeMux()
	RegisterPprofEndpoints(mux)
	server := httptest.NewServer(pprofGuardMiddleware(mux))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/debug/pprof/heap?debug=1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("with the debug endpoints enabled, expected 200, got %d", resp.StatusCode)
	}
}
