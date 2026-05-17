package api

import (
	"net/http"
	"net/http/pprof"
	"strings"
)

// RegisterPprofEndpoints registers the standard pprof debug endpoints on the given mux.
// Mounted at /debug/pprof/ for live profiling of a running service.
//
// Endpoints:
//
//	GET /debug/pprof/           — pprof index
//	GET /debug/pprof/allocs     — cumulative allocs since start
//	GET /debug/pprof/heap       — live heap profile
//	GET /debug/pprof/goroutine  — stack traces of all goroutines
//	GET /debug/pprof/block      — stack traces leading to blocking sync primitives
//	GET /debug/pprof/mutex      — stack traces of holders of contended mutexes
//	GET /debug/pprof/profile    — CPU profile (30s default)
//	GET /debug/pprof/trace      — execution trace (5s default)
//
// Security: only enable in dev/staging, or protect behind auth middleware.
func RegisterPprofEndpoints(mux *http.ServeMux) {
	prefix := "/debug/pprof"

	mux.HandleFunc(prefix+"/", pprof.Index)
	mux.HandleFunc(prefix+"/allocs", pprof.Handler("allocs").ServeHTTP)
	mux.HandleFunc(prefix+"/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc(prefix+"/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc(prefix+"/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc(prefix+"/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc(prefix+"/profile", pprof.Profile)
	mux.HandleFunc(prefix+"/trace", pprof.Trace)
	mux.HandleFunc(prefix+"/symbol", pprof.Symbol)
	mux.HandleFunc(prefix+"/cmdline", pprof.Cmdline)
}

// IsPprofPath checks if the request path is a pprof debug endpoint.
func IsPprofPath(path string) bool {
	return strings.HasPrefix(path, "/debug/pprof")
}

func registerPprof(mux *http.ServeMux) {
	RegisterPprofEndpoints(mux)
}
