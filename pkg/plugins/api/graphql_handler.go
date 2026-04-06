package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
)

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// GraphQLHandler handles GraphQL requests
type GraphQLHandler struct {
	queryService domainquery.Service
	logger       core.Logger
	metrics      core.MetricsCollector
	mu           sync.RWMutex
	initialized  bool
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(queryService domainquery.Service, logger core.Logger, metrics core.MetricsCollector) *GraphQLHandler {
	return &GraphQLHandler{
		queryService: queryService,
		logger:       logger,
		metrics:      metrics,
	}
}

// Initialize initializes the GraphQL handler
func (h *GraphQLHandler) Initialize(config *core.Config) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.initialized {
		return fmt.Errorf("GraphQL handler already initialized")
	}

	h.initialized = true

	h.logger.Info("GraphQL handler initialized", map[string]interface{}{
		"component": "graphql_handler",
	})

	return nil
}

// Handle handles a GraphQL request
func (h *GraphQLHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		http.Error(w, "GraphQL handler not initialized", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet handles GET requests (GraphQL playground)
func (h *GraphQLHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	// Return GraphQL playground HTML
	playground := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>GraphQL Playground</title>
		<style>
			body { margin: 0; padding: 0; font-family: sans-serif; }
			#playground { width: 100%; height: 100vh; }
		</style>
	</head>
	<body>
		<div id="playground">
			<p>GraphQL Playground - POST your queries to this endpoint</p>
		</div>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(playground)); err != nil {
		// Log write error but don't fail the request
		_ = err
	}
}

// handlePost handles POST requests (GraphQL queries)
func (h *GraphQLHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	var query string

	// Parse based on Content-Type
	if strings.Contains(contentType, "application/json") {
		// Parse JSON body
		var gqlReq GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}
		query = gqlReq.Query
	} else {
		// Parse form-encoded body
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
			return
		}
		query = r.FormValue("query")
	}

	if query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprintf(w, `{"data":{"message":"GraphQL query received","query":"%s"}}`, query); err != nil {
		_ = err
	}
}

// Stop stops the GraphQL handler
func (h *GraphQLHandler) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.initialized = false

	h.logger.Info("GraphQL handler stopped", map[string]interface{}{
		"component": "graphql_handler",
	})

	return nil
}
