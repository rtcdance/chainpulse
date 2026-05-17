package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/rtcdance/chainpulse/pkg/plugins/api/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api/shared"
)

// Plugin implements the WebSocket protocol handler.
type Plugin struct {
	name           string
	port           int
	wssPort        int
	apiLayer       *core.APILayer
	upgrader       websocket.Upgrader
	server         *http.Server
	wssServer      *http.Server
	tlsManager     *shared.TLSManager
	processor      core.RequestProcessor
	mu             sync.RWMutex
	running        bool
	middleware     []core.Middleware
	clients        map[*websocket.Conn]bool
	clientsMu      sync.RWMutex
	router         *core.APIRouter
	allowedOrigins []string
}

// NewWebSocketPlugin creates a new WebSocket plugin
func NewWebSocketPlugin(name string, port int, apiLayer *core.APILayer) *Plugin {
	processor := core.NewDefaultRequestProcessor(apiLayer)
	p := &Plugin{
		name:       name,
		port:       port,
		wssPort:    port + 443, // Default WSS port offset
		apiLayer:   apiLayer,
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		clients:    make(map[*websocket.Conn]bool),
		router:     core.NewAPIRouter(),
	}
	p.upgrader = websocket.Upgrader{CheckOrigin: p.checkOrigin}
	return p
}

// NewWebSocketPluginWithTLS creates a new WebSocket plugin with TLS support
func NewWebSocketPluginWithTLS(name string, port int, wssPort int, certFile, keyFile string, apiLayer *core.APILayer) (*Plugin, error) {
	tlsManager, err := shared.NewTLSManager(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS manager: %w", err)
	}

	processor := core.NewDefaultRequestProcessor(apiLayer)
	p := &Plugin{
		name:       name,
		port:       port,
		wssPort:    wssPort,
		apiLayer:   apiLayer,
		tlsManager: tlsManager,
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		clients:    make(map[*websocket.Conn]bool),
		router:     core.NewAPIRouter(),
	}
	p.upgrader = websocket.Upgrader{CheckOrigin: p.checkOrigin}
	return p, nil
}

// WithAllowedOrigins sets the allowed origins for WebSocket connections.
// If no origins are set, all origins are allowed (useful for development only).
// In production, specify explicit origins e.g. ["https://app.example.com"].
func (p *Plugin) WithAllowedOrigins(origins []string) *Plugin {
	p.allowedOrigins = origins
	return p
}

// checkOrigin validates the Origin header against the allowed origins list.
func (p *Plugin) checkOrigin(r *http.Request) bool {
	if len(p.allowedOrigins) == 0 {
		return true // No restriction configured — development mode
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Non-browser clients don't send Origin
	}

	for _, allowed := range p.allowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

// Start starts the WebSocket server
func (p *Plugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("WebSocket plugin already running")
	}

	if len(p.allowedOrigins) == 0 {
		slog.Warn("WebSocket plugin: no origin restrictions configured — accepting all origins (development mode only)")
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", p.handleWebSocket)

	p.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	p.running = true

	// Start WebSocket server in background
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("WebSocket server error", "error", err)
		}
	}()

	// Start WSS server if TLS manager is configured
	if p.tlsManager != nil {
		if err := p.startWSS(mux); err != nil {
			return fmt.Errorf("failed to start WSS server: %w", err)
		}
	}

	return nil
}

// startWSS starts the WSS (WebSocket Secure) server
func (p *Plugin) startWSS(mux *http.ServeMux) error {
	p.wssServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.wssPort),
		Handler:           mux,
		TLSConfig:         p.tlsManager.GetConfig(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start WSS server in background
	go func() {
		if err := p.wssServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			slog.Error("WSS server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the WebSocket server
func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("WebSocket plugin not running")
	}

	// Close all client connections
	p.clientsMu.Lock()
	for conn := range p.clients {
		_ = conn.Close()
	}
	p.clients = make(map[*websocket.Conn]bool)
	p.clientsMu.Unlock()

	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown WebSocket server: %w", err)
		}
	}

	if p.wssServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.wssServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown WSS server: %w", err)
		}
	}

	p.running = false
	return nil
}

// GetName returns the plugin name
func (p *Plugin) GetName() string {
	return p.name
}

// GetProtocolName returns the protocol name (implements ProtocolHandler)
func (p *Plugin) GetProtocolName() string {
	return p.name
}

// IsRunning returns whether the plugin is running
func (p *Plugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// handleWebSocket handles incoming WebSocket connections
func (p *Plugin) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("WebSocket upgrade error", "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// Register client
	p.clientsMu.Lock()
	p.clients[conn] = true
	p.clientsMu.Unlock()

	// Unregister client on disconnect
	defer func() {
		p.clientsMu.Lock()
		delete(p.clients, conn)
		p.clientsMu.Unlock()
	}()

	// Handle messages
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket error", "error", err)
			}
			break
		}

		// Only handle text messages
		if messageType != websocket.TextMessage {
			continue
		}

		// Create request adapter
		req := NewWebSocketRequest(r, data)

		// Process through the shared request processor so route middleware and
		// error handling are consistent with the rest of the API layer.
		result, err := p.processor.ProcessRequest(r.Context(), req)
		if err != nil {
			result = p.processor.HandleError(r.Context(), err)
		}

		// Send response back through WebSocket
		if err := conn.WriteMessage(websocket.TextMessage, result.Body()); err != nil {
			slog.Warn("WebSocket write error", "error", err)
			break
		}
	}
}

// Use adds middleware (implements ProtocolHandler)
func (p *Plugin) Use(middleware ...core.Middleware) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot add middleware while handler is running")
	}

	p.middleware = append(p.middleware, middleware...)
	p.router.Use(middleware...)

	if p.apiLayer != nil {
		p.apiLayer.Use(middleware...)
	}

	return nil
}

// RegisterRoute registers a route handler (implements ProtocolHandler)
func (p *Plugin) RegisterRoute(path string, handler core.Handler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot register route while handler is running")
	}

	p.router.Register(path, handler)

	if p.apiLayer != nil {
		p.apiLayer.RegisterHandler(path, handler)
	}

	return nil
}

// ProcessRequest executes an adapter-backed WebSocket message through the
// shared request processor so middleware and routing can be tested directly.
func (p *Plugin) ProcessRequest(ctx context.Context, req *Request) (core.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	result, err := p.processor.ProcessRequest(ctx, req)
	if err != nil {
		return p.processor.HandleError(ctx, err), nil
	}

	return result, nil
}

// GetClientCount returns the number of connected clients
func (p *Plugin) GetClientCount() int {
	p.clientsMu.RLock()
	defer p.clientsMu.RUnlock()
	return len(p.clients)
}

// SetWSSPort sets the WSS port
func (p *Plugin) SetWSSPort(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wssPort = port
}

// GetWSSPort returns the WSS port
func (p *Plugin) GetWSSPort() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.wssPort
}

// GetTLSMetrics returns TLS metrics
func (p *Plugin) GetTLSMetrics() map[string]any {
	if p.tlsManager == nil {
		return nil
	}
	return p.tlsManager.GetMetrics()
}

// GetConnectionMetrics returns compact runtime connection metrics for the
// websocket transport surface.
func (p *Plugin) GetConnectionMetrics() map[string]any {
	running := p.IsRunning()
	clientCount := p.GetClientCount()
	transportPosture := classifyWebSocketTransportPosture(p.tlsManager != nil)
	connectionPosture := classifyWebSocketConnectionPosture(running, clientCount)

	return map[string]any{
		"running":            running,
		"client_count":       clientCount,
		"transport_posture":  transportPosture,
		"connection_posture": connectionPosture,
		"reliability_hint":   buildWebSocketReliabilityHint(transportPosture, connectionPosture),
	}
}

func classifyWebSocketTransportPosture(tlsEnabled bool) string {
	if tlsEnabled {
		return "ws-tls-enabled"
	}
	return "ws-plaintext-only"
}

func classifyWebSocketConnectionPosture(running bool, clientCount int) string {
	if !running {
		return "ws-stopped"
	}
	if clientCount > 0 {
		return "ws-active"
	}
	return "ws-idle"
}

func buildWebSocketReliabilityHint(transportPosture string, connectionPosture string) string {
	switch connectionPosture {
	case "ws-active":
		if transportPosture == "ws-tls-enabled" {
			return "websocket transport is active with connected clients over a TLS-capable runtime"
		}
		return "websocket transport is active with connected clients on a plaintext runtime"
	case "ws-idle":
		if transportPosture == "ws-tls-enabled" {
			return "websocket runtime is up with TLS enabled but no connected clients are currently observed"
		}
		return "websocket runtime is up without TLS and no connected clients are currently observed"
	default:
		if transportPosture == "ws-tls-enabled" {
			return "websocket runtime is currently stopped; restart before relying on TLS websocket delivery"
		}
		return "websocket runtime is currently stopped; restart before relying on websocket delivery"
	}
}
