package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api/shared"
)

// HTTPPlugin implements the HTTP protocol handler
type HTTPPlugin struct {
	name          string
	port          int
	httpsPort     int
	router        *core.APIRouter
	apiLayer      *core.APILayer
	server        *http.Server
	httpsServer   *http.Server
	tlsManager    *shared.TLSManager
	processor     core.RequestProcessor
	mu            sync.RWMutex
	running       bool
	middleware    []core.Middleware
	nativeHandler http.HandlerFunc
	tracer        *observability.DefaultTracer
}

// NewHTTPPlugin creates a new HTTP plugin
func NewHTTPPlugin(name string, port int, apiLayer *core.APILayer) *HTTPPlugin {
	processor := core.NewDefaultRequestProcessor(apiLayer)
	return &HTTPPlugin{
		name:       name,
		port:       port,
		httpsPort:  port + 443, // Default HTTPS port offset
		apiLayer:   apiLayer,
		router:     core.NewAPIRouter(),
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		tracer:     observability.NewDefaultTracer(nil, nil), // will be overridden via SetTracer if provider is available
	}
}

// NewHTTPPluginWithTLS creates a new HTTP plugin with TLS support
func NewHTTPPluginWithTLS(name string, port int, httpsPort int, certFile, keyFile string, apiLayer *core.APILayer) (*HTTPPlugin, error) {
	tlsManager, err := shared.NewTLSManager(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS manager: %w", err)
	}

	processor := core.NewDefaultRequestProcessor(apiLayer)
	return &HTTPPlugin{
		name:       name,
		port:       port,
		httpsPort:  httpsPort,
		apiLayer:   apiLayer,
		router:     core.NewAPIRouter(),
		tlsManager: tlsManager,
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		tracer:     observability.NewDefaultTracer(nil, nil), // will be overridden via SetTracer if provider is available
	}, nil
}

// SetTracer sets the tracer for the HTTP plugin, replacing the default nil-logger tracer.
func (p *HTTPPlugin) SetTracer(tracer *observability.DefaultTracer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tracer = tracer
}

// Start starts the HTTP server
func (p *HTTPPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("HTTP plugin already running")
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRequest)

	p.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	p.running = true

	// Start HTTP server in background
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Start HTTPS server if TLS manager is configured
	if p.tlsManager != nil {
		if err := p.startHTTPS(mux); err != nil {
			return err
		}
	}

	return nil
}

// startHTTPS starts the HTTPS server
func (p *HTTPPlugin) startHTTPS(mux *http.ServeMux) error {
	p.httpsServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.httpsPort),
		Handler:           mux,
		TLSConfig:         p.tlsManager.GetConfig(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start HTTPS server in background
	go func() {
		if err := p.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTPS server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server with a 30-second deadline for graceful connection draining.
func (p *HTTPPlugin) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return p.ShutdownWithContext(ctx)
}

// ShutdownWithContext stops the HTTP server with a context deadline,
// allowing in-flight requests to complete before closing connections.
func (p *HTTPPlugin) ShutdownWithContext(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("HTTP plugin not running")
	}

	if p.server != nil {
		if err := p.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	if p.httpsServer != nil {
		if err := p.httpsServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	p.running = false
	return nil
}

// GetName returns the plugin name
func (p *HTTPPlugin) GetName() string {
	return p.name
}

// GetProtocolName returns the protocol name (implements ProtocolHandler)
func (p *HTTPPlugin) GetProtocolName() string {
	return p.name
}

// Handler returns the native HTTP request handler for integration and testing.
func (p *HTTPPlugin) Handler() http.HandlerFunc {
	return p.handleRequest
}

// IsRunning returns whether the plugin is running
func (p *HTTPPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// handleRequest handles incoming HTTP requests
func (p *HTTPPlugin) handleRequest(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		p.handleRequestCore(w, r)
		return
	}

	if p.tracer != nil {
		p.tracer.WrapHTTPHandler(http.HandlerFunc(p.handleRequestCore), fmt.Sprintf("%s.request", p.name)).ServeHTTP(w, r)

		return
	}

	p.handleRequestCore(w, r)
}

func (p *HTTPPlugin) handleRequestCore(w http.ResponseWriter, r *http.Request) {
	// Recover from panics in any registered handler to avoid
	// crashing the HTTP server goroutine.
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("HTTP handler panicked, returning 500",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Any("panic", rec),
				)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()

	p.mu.RLock()
	nativeHandler := p.nativeHandler
	p.mu.RUnlock()
	if nativeHandler != nil {
		nativeHandler(w, r)

		return
	}

	// Create request adapter
	req := NewHTTPRequest(r)

	// Process through request processor
	ctx := r.Context()
	result, err := p.processor.ProcessRequest(ctx, req)
	if err != nil {
		result = p.processor.HandleError(ctx, err)
	}

	// Write response
	for key, value := range result.Headers() {
		w.Header().Set(key, value)
	}

	w.WriteHeader(result.Status())
	_, _ = w.Write(result.Body())
}

// SetNativeHandler sets an optional native HTTP request handler override.
func (p *HTTPPlugin) SetNativeHandler(handler http.HandlerFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nativeHandler = handler
}

// RegisterRoute registers a route handler (implements ProtocolHandler)
func (p *HTTPPlugin) RegisterRoute(path string, handler core.Handler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot register route while handler is running")
	}

	p.router.Register(path, handler)
	return nil
}

// Use adds middleware (implements ProtocolHandler)
func (p *HTTPPlugin) Use(middleware ...core.Middleware) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot add middleware while handler is running")
	}

	p.middleware = append(p.middleware, middleware...)
	p.router.Use(middleware...)
	return nil
}

// SetHTTPSPort sets the HTTPS port
func (p *HTTPPlugin) SetHTTPSPort(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.httpsPort = port
}

// GetHTTPSPort returns the HTTPS port
func (p *HTTPPlugin) GetHTTPSPort() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.httpsPort
}

// GetTLSMetrics returns TLS metrics
func (p *HTTPPlugin) GetTLSMetrics() map[string]any {
	if p.tlsManager == nil {
		return nil
	}
	return p.tlsManager.GetMetrics()
}

// GetRuntimeMetrics returns compact runtime metrics for the HTTP transport
// surface.
func (p *HTTPPlugin) GetRuntimeMetrics() map[string]any {
	running := p.IsRunning()
	routeCount := p.router.RouteCount()
	transportPosture := classifyHTTPTransportPosture(p.tlsManager != nil)
	routePosture := classifyHTTPRoutePosture(routeCount)
	runtimePosture := classifyHTTPRuntimePosture(running, routeCount)

	return map[string]any{
		"running":           running,
		"route_count":       routeCount,
		"transport_posture": transportPosture,
		"route_posture":     routePosture,
		"runtime_posture":   runtimePosture,
		"reliability_hint":  buildHTTPReliabilityHint(transportPosture, runtimePosture),
	}
}

func classifyHTTPTransportPosture(tlsEnabled bool) string {
	if tlsEnabled {
		return "http-tls-enabled"
	}
	return "http-plaintext-only"
}

func classifyHTTPRoutePosture(routeCount int) string {
	if routeCount > 0 {
		return "http-routes-present"
	}
	return "http-routes-empty"
}

func classifyHTTPRuntimePosture(running bool, routeCount int) string {
	if !running {
		return "http-stopped"
	}
	if routeCount > 0 {
		return "http-serving"
	}
	return "http-running-unrouted"
}

func buildHTTPReliabilityHint(transportPosture string, runtimePosture string) string {
	switch runtimePosture {
	case "http-serving":
		if transportPosture == "http-tls-enabled" {
			return "http runtime is serving registered routes with a TLS-capable transport"
		}
		return "http runtime is serving registered routes on a plaintext transport"
	case "http-running-unrouted":
		return "http runtime is running but no routes are registered yet"
	default:
		if transportPosture == "http-tls-enabled" {
			return "http runtime is stopped; restart before relying on TLS-capable route serving"
		}
		return "http runtime is stopped; restart before relying on route serving"
	}
}
