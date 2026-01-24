package http

import (
	"fmt"
	"net/http"
	"sync"

	"chainpulse/pkg/plugins/api/core"
	"chainpulse/pkg/plugins/api/shared"
)

// HTTPPlugin implements the HTTP protocol handler
type HTTPPlugin struct {
	name        string
	port        int
	httpsPort   int
	router      *core.APIRouter
	apiLayer    *core.APILayer
	server      *http.Server
	httpsServer *http.Server
	tlsManager  *shared.TLSManager
	processor   core.RequestProcessor
	mu          sync.RWMutex
	running     bool
	middleware  []core.Middleware
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
		name:        name,
		port:        port,
		httpsPort:   httpsPort,
		apiLayer:    apiLayer,
		router:      core.NewAPIRouter(),
		tlsManager:  tlsManager,
		processor:   processor,
		middleware:  make([]core.Middleware, 0),
	}, nil
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
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: mux,
	}

	p.running = true

	// Start HTTP server in background
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
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
		Addr:      fmt.Sprintf(":%d", p.httpsPort),
		Handler:   mux,
		TLSConfig: p.tlsManager.GetConfig(),
	}

	// Start HTTPS server in background
	go func() {
		if err := p.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTPS server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (p *HTTPPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("HTTP plugin not running")
	}

	if p.server != nil {
		if err := p.server.Close(); err != nil {
			return err
		}
	}

	if p.httpsServer != nil {
		if err := p.httpsServer.Close(); err != nil {
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

// IsRunning returns whether the plugin is running
func (p *HTTPPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// handleRequest handles incoming HTTP requests
func (p *HTTPPlugin) handleRequest(w http.ResponseWriter, r *http.Request) {
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
func (p *HTTPPlugin) GetTLSMetrics() map[string]interface{} {
	if p.tlsManager == nil {
		return nil
	}
	return p.tlsManager.GetMetrics()
}
