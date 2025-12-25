package handlers

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"chainpulse/shared/datapuller"
	"chainpulse/shared/logger"
	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
)

// IndexerService interface defines the methods that the indexer service should implement
type IndexerService interface {
	StartIndexing(ctx context.Context, contractAddresses []common.Address) error
	ProcessHistoricalEvents(ctx context.Context, contractAddresses []common.Address, fromBlock, toBlock *big.Int) error
	GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error)
	GetEventByID(id uint) (*types.IndexedEvent, error)
	GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error)
	GetLastProcessedBlock() (*big.Int, error)
	ResumeEvents(ctx context.Context, fromBlock, toBlock *big.Int) error
}

// Server represents the API server
type Server struct {
	router         *mux.Router
	indexerService IndexerService
	jwtSecret      string
	logger         logger.Logger
	metricsCollector *datapuller.MetricsCollector
}

// NewServer creates a new API server instance
func NewServer(indexerService IndexerService, jwtSecret string, metricsCollector *datapuller.MetricsCollector) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		indexerService: indexerService,
		jwtSecret:      jwtSecret,
		logger:         logger.NewLogger(),
		metricsCollector: metricsCollector,
	}

	// Register routes
	s.registerRoutes()

	return s
}

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	s.router.HandleFunc("/events", s.GetEventsHandler).Methods("GET")
	s.router.HandleFunc("/events/{id}", s.GetEventByIDHandler).Methods("GET")
	s.router.HandleFunc("/health", s.HealthHandler).Methods("GET")
	s.router.HandleFunc("/metrics", s.MetricsHandler).Methods("GET")
}

// GetRouter returns the router instance
func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// GetEventsHandler handles GET /events requests
func (s *Server) GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	var filter types.EventFilter

	// Parse query parameters
	fromBlockStr := r.URL.Query().Get("from_block")
	if fromBlockStr != "" {
		if fromBlock, err := strconv.ParseInt(fromBlockStr, 10, 64); err == nil {
			filter.FromBlock = big.NewInt(fromBlock)
		}
	}

	toBlockStr := r.URL.Query().Get("to_block")
	if toBlockStr != "" {
		if toBlock, err := strconv.ParseInt(toBlockStr, 10, 64); err == nil {
			filter.ToBlock = big.NewInt(toBlock)
		}
	}

	eventName := r.URL.Query().Get("event_name")
	if eventName != "" {
		filter.EventName = eventName
	}

	contract := r.URL.Query().Get("contract")
	if contract != "" {
		filter.Contract = contract
	}

	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			filter.Page = page
		}
	}

	pageSizeStr := r.URL.Query().Get("page_size")
	if pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			filter.PageSize = pageSize
		}
	}

	events, err := s.indexerService.GetEvents(&filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// GetEventByIDHandler handles GET /events/{id} requests
func (s *Server) GetEventByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	event, err := s.indexerService.GetEventByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if event == nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// HealthHandler handles GET /health requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// MetricsHandler handles GET /metrics requests
func (s *Server) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if s.metricsCollector == nil {
		http.Error(w, "Metrics collector not available", http.StatusInternalServerError)
		return
	}

	// Get all plugin metrics
	pluginMetrics := s.metricsCollector.GetAllMetrics()
	
	// Get global metrics
	totalRequests, totalErrors, totalSuccess, avgResponseTime := s.metricsCollector.GetGlobalMetrics()
	
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
	json.NewEncoder(w).Encode(response)
}