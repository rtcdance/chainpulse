package api

import (
	"fmt"
	"net/http"
	"sync"

	corelib "chainpulse/pkg/core"
	apicore "chainpulse/pkg/plugins/api/core"
	httpapi "chainpulse/pkg/plugins/api/http"
)

// APIGatewayPlugin is the main API gateway that routes requests to appropriate handlers
type APIGatewayPlugin struct {
	logger           corelib.Logger
	metrics          corelib.MetricsCollector
	httpPlugin       *httpapi.HTTPPlugin
	handlers         map[string]Handler
	mu               sync.RWMutex
	initialized      bool
	running          bool
}

// Handler defines the interface for API handlers
type Handler interface {
	Initialize(config *corelib.Config) error
	Handle(w http.ResponseWriter, r *http.Request)
	Stop() error
}

// NewAPIGatewayPlugin creates a new API gateway plugin
func NewAPIGatewayPlugin(logger corelib.Logger, metrics corelib.MetricsCollector) *APIGatewayPlugin {
	return &APIGatewayPlugin{
		logger:      logger,
		metrics:     metrics,
		handlers:    make(map[string]Handler),
	}
}

// Name returns the plugin name
func (g *APIGatewayPlugin) Name() string {
	return "api-gateway"
}

// Version returns the plugin version
func (g *APIGatewayPlugin) Version() string {
	return "1.0.0"
}

// Health checks the plugin health
func (g *APIGatewayPlugin) Health() error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if !g.running {
		return fmt.Errorf("API gateway not running")
	}
	
	return nil
}

// Initialize initializes the API gateway
func (g *APIGatewayPlugin) Initialize(config corelib.Config) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.initialized {
		return fmt.Errorf("API gateway already initialized")
	}

	// Create HTTP plugin
	apiLayer := apicore.NewAPILayer()

	g.httpPlugin = httpapi.NewHTTPPlugin("api-gateway", 8080, apiLayer)
	g.initialized = true

	g.logger.Info("API gateway initialized", map[string]interface{}{
		"component": "api_gateway",
	})

	return nil
}

// Start starts the API gateway
func (g *APIGatewayPlugin) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.initialized {
		return fmt.Errorf("API gateway not initialized")
	}

	if g.running {
		return fmt.Errorf("API gateway already running")
	}

	if err := g.httpPlugin.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP plugin: %w", err)
	}

	g.running = true

	g.logger.Info("API gateway started", map[string]interface{}{
		"component": "api_gateway",
		"port":      8080,
	})

	return nil
}

// Stop stops the API gateway
func (g *APIGatewayPlugin) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.running {
		return fmt.Errorf("API gateway not running")
	}

	if err := g.httpPlugin.Stop(); err != nil {
		return fmt.Errorf("failed to stop HTTP plugin: %w", err)
	}

	g.running = false

	g.logger.Info("API gateway stopped", map[string]interface{}{
		"component": "api_gateway",
	})

	return nil
}

// RegisterHandler registers a handler for a path
func (g *APIGatewayPlugin) RegisterHandler(path string, handler Handler) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.initialized {
		return fmt.Errorf("API gateway not initialized")
	}

	g.handlers[path] = handler
	return nil
}
