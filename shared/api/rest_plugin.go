package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"chainpulse/services/api/handlers"
	"chainpulse/shared/database"

	"github.com/gorilla/mux"
)

// RESTPluginImpl implements the REST API plugin
type RESTPluginImpl struct {
	server           *http.Server
	router           *mux.Router
	db               *database.DB
	port             string
	metricsCollector *MetricsCollector
	config           map[string]interface{}
	mutex            sync.RWMutex
	name             string
}

// NewRESTPlugin creates a new REST API plugin instance
func NewRESTPlugin() *RESTPluginImpl {
	router := mux.NewRouter()
	return &RESTPluginImpl{
		router: router,
		name:   "rest-api",
	}
}

// GetName returns the name of the plugin
func (r *RESTPluginImpl) GetName() string {
	return r.name
}

// GetType returns the type of the plugin
func (r *RESTPluginImpl) GetType() string {
	return "rest"
}

// Initialize initializes the REST plugin with configuration
func (r *RESTPluginImpl) Initialize(config map[string]interface{}) error {
	r.config = config

	// Extract port from config or use default
	portInterface, exists := config["port"]
	if !exists {
		r.port = "8080" // default port
	} else {
		if portStr, ok := portInterface.(string); ok {
			r.port = portStr
		} else {
			r.port = "8080" // default if not a string
		}
	}

	// Set up routes
	r.setupRoutes()

	// Create HTTP server
	r.server = &http.Server{
		Addr:    ":" + r.port,
		Handler: r.router,
	}

	return nil
}

// setupRoutes configures the API routes
func (r *RESTPluginImpl) setupRoutes() {
	// Initialize handlers
	eventHandler := handlers.NewEventHandler(r.db)
	contractHandler := handlers.NewContractHandler(r.db)
	statsHandler := handlers.NewStatsHandler(r.db)

	// Health check endpoint
	r.router.HandleFunc("/health", r.healthCheck).Methods("GET")

	// Event endpoints
	r.router.HandleFunc("/api/v1/events", eventHandler.GetEvents).Methods("GET")
	r.router.HandleFunc("/api/v1/events/{txHash}", eventHandler.GetEventByTxHash).Methods("GET")
	r.router.HandleFunc("/api/v1/events/block/{blockNumber}", eventHandler.GetEventsByBlockNumber).Methods("GET")

	// Contract endpoints
	r.router.HandleFunc("/api/v1/contracts", contractHandler.GetContracts).Methods("GET")
	r.router.HandleFunc("/api/v1/contracts/{address}", contractHandler.GetContractByAddress).Methods("GET")

	// Stats endpoints
	r.router.HandleFunc("/api/v1/stats", statsHandler.GetStats).Methods("GET")
	
	// Metrics endpoint
	r.router.HandleFunc("/api/v1/metrics", r.metricsHandler).Methods("GET")
}

// healthCheck returns the health status of the service
func (r *RESTPluginImpl) healthCheck(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "api-gateway",
		"time":    time.Now().Format(time.RFC3339),
	})

	// Record metrics
	if r.metricsCollector != nil {
		r.metricsCollector.RecordRequest("rest", time.Since(startTime), err)
	}
}

// metricsHandler returns the metrics of the service
func (r *RESTPluginImpl) metricsHandler(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	var err error
	defer func() {
		// Record metrics
		if r.metricsCollector != nil {
			r.metricsCollector.RecordRequest("rest", time.Since(startTime), err)
		}
	}()

	if r.metricsCollector == nil {
		http.Error(w, "Metrics collector not available", http.StatusInternalServerError)
		err = fmt.Errorf("metrics collector not available")
		return
	}

	// Get all plugin metrics
	pluginMetrics := r.metricsCollector.GetAllMetrics()
	
	// Get global metrics
	totalRequests, totalErrors, totalSuccess, avgResponseTime := r.metricsCollector.GetGlobalMetrics()
	
	// Create response
	response := map[string]interface{}{
		"global": map[string]interface{}{
			"total_requests":     totalRequests,
			"total_errors":       totalErrors,
			"total_success":      totalSuccess,
			"avg_response_time":  avgResponseTime.String(),
		},
		"plugins": map[string]interface{}{},
	}

	// Add plugin-specific metrics
	for name, metrics := range pluginMetrics {
		response["plugins"].(map[string]interface{})[name] = map[string]interface{}{
			"name":               metrics.Name,
			"total_requests":     metrics.TotalRequests,
			"total_errors":       metrics.TotalErrors,
			"total_success":      metrics.TotalSuccess,
			"avg_response_time":  metrics.AvgResponseTime.String(),
			"last_request_time":  metrics.LastRequestTime.Format(time.RFC3339),
			"last_error_time":    metrics.LastErrorTime.Format(time.RFC3339),
			"last_error":         metrics.LastError,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
}

// Start starts the REST API service
func (r *RESTPluginImpl) Start(ctx context.Context) error {
	log.Printf("Starting REST API service on port %s", r.port)
	
	// Run server in a goroutine
	go func() {
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting REST API server: %v", err)
		}
	}()

	// Wait for context cancellation to shutdown server
	<-ctx.Done()
	
	return r.Stop(context.Background())
}

// Stop stops the REST API service
func (r *RESTPluginImpl) Stop(ctx context.Context) error {
	if r.server != nil {
		return r.server.Shutdown(ctx)
	}
	return nil
}

// RegisterRoute registers a new route with the REST plugin
func (r *RESTPluginImpl) RegisterRoute(path string, handler HTTPHandler, methods ...string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(methods) == 0 {
		r.router.HandleFunc(path, handler)
	} else {
		r.router.HandleFunc(path, handler).Methods(methods...)
	}

	return nil
}

// RegisterRoutes registers multiple routes at once
func (r *RESTPluginImpl) RegisterRoutes(routes []Route) error {
	for _, route := range routes {
		if err := r.RegisterRoute(route.Path, route.Handler, route.Methods...); err != nil {
			return err
		}
	}
	return nil
}

// SetDatabase sets the database for the REST plugin
func (r *RESTPluginImpl) SetDatabase(db interface{}) {
	if databaseDB, ok := db.(*database.DB); ok {
		r.db = databaseDB
		
		// Re-setup routes with new database
		r.setupRoutes()
	}
}

// SetMetricsCollector sets the metrics collector for the REST plugin
func (r *RESTPluginImpl) SetMetricsCollector(collector *MetricsCollector) {
	r.metricsCollector = collector
}