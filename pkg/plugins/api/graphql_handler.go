package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

// jsonScalar is a custom GraphQL scalar that passes through arbitrary JSON values
var jsonScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:         "JSON",
	Description:  "Arbitrary JSON value",
	Serialize:    func(value any) any { return value },
	ParseValue:   func(value any) any { return value },
	ParseLiteral: func(valueAST ast.Value) any { return nil },
})

type GraphQLRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
	OperationName string         `json:"operationName"`
}

type GraphQLResponse struct {
	Data   any            `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message    string         `json:"message"`
	Locations  []Location     `json:"locations,omitempty"`
	Path       []any          `json:"path,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLHandler handles GraphQL requests
type GraphQLHandler struct {
	queryService domainquery.Service
	eventStore   domainquery.EventStore
	logger       core.Logger
	metrics      core.MetricsCollector
	schema       *graphql.Schema
	mu           sync.RWMutex
	initialized  bool
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(queryService domainquery.Service, eventStore domainquery.EventStore, logger core.Logger, metrics core.MetricsCollector) *GraphQLHandler {
	h := &GraphQLHandler{
		queryService: queryService,
		eventStore:   eventStore,
		logger:       logger,
		metrics:      metrics,
	}

	schema, err := h.buildSchema()
	if err != nil {
		logger.Warn("Failed to build GraphQL schema, using mock", "error", err.Error())
	} else {
		h.schema = schema
	}

	return h
}

func (h *GraphQLHandler) buildSchema() (*graphql.Schema, error) {
	eventType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Event",
		Fields: graphql.Fields{
			"id":               &graphql.Field{Type: graphql.String},
			"eventHash":        &graphql.Field{Type: graphql.String},
			"blockNumber":      &graphql.Field{Type: graphql.Int},
			"blockHash":        &graphql.Field{Type: graphql.String},
			"blockTimestamp":   &graphql.Field{Type: graphql.Int},
			"transactionHash":  &graphql.Field{Type: graphql.String},
			"transactionIndex": &graphql.Field{Type: graphql.Int},
			"logIndex":         &graphql.Field{Type: graphql.Int},
			"contractAddress":  &graphql.Field{Type: graphql.String},
			"eventName":        &graphql.Field{Type: graphql.String},
			"eventSignature":   &graphql.Field{Type: graphql.String},
			"eventData":        &graphql.Field{Type: jsonScalar},
			"chainId":          &graphql.Field{Type: graphql.String},
			"status":           &graphql.Field{Type: graphql.String},
			"processedAt":      &graphql.Field{Type: graphql.Int},
		},
	})

	eventEdgeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "EventEdge",
		Fields: graphql.Fields{
			"node":   &graphql.Field{Type: eventType},
			"cursor": &graphql.Field{Type: graphql.String},
		},
	})

	pageInfoType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PageInfo",
		Fields: graphql.Fields{
			"hasNextPage":     &graphql.Field{Type: graphql.Boolean},
			"hasPreviousPage": &graphql.Field{Type: graphql.Boolean},
			"startCursor":     &graphql.Field{Type: graphql.String},
			"endCursor":       &graphql.Field{Type: graphql.String},
		},
	})

	eventConnection := graphql.NewObject(graphql.ObjectConfig{
		Name: "EventConnection",
		Fields: graphql.Fields{
			"edges":    &graphql.Field{Type: graphql.NewList(eventEdgeType)},
			"pageInfo": &graphql.Field{Type: pageInfoType},
			"total":    &graphql.Field{Type: graphql.Int},
		},
	})

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"event": &graphql.Field{
				Type: eventType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					id, ok := p.Args["id"].(string)
					if !ok || id == "" {
						return nil, fmt.Errorf("id is required")
					}
					if h.eventStore == nil {
						return nil, fmt.Errorf("event store not configured")
					}
					evt, err := h.eventStore.GetEvent(p.Context, id)
					if err != nil {
						return nil, fmt.Errorf("failed to get event")
					}
					if evt == nil {
						return nil, nil
					}
					return eventToMap(evt), nil
				},
			},
			"events": &graphql.Field{
				Type: eventConnection,
				Args: graphql.FieldConfigArgument{
					"first":   &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 20},
					"after":   &graphql.ArgumentConfig{Type: graphql.String},
					"chainId": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					first := 20
					if f, ok := p.Args["first"].(int); ok && f > 0 {
						first = f
					}
					if first > 100 {
						first = 100
					}

					chainID := 0 // 0 = all chains
					if cid, ok := p.Args["chainId"].(string); ok && cid != "" {
						chainID = chainid.ResolveChainID(cid)
					}

					if h.eventStore == nil {
						return map[string]any{
							"edges": []any{},
							"pageInfo": map[string]any{
								"hasNextPage":     false,
								"hasPreviousPage": false,
							},
							"total": 0,
						}, nil
					}

					events, err := h.eventStore.GetEventsByChain(p.Context, chainID, first, 0)
					if err != nil {
						return nil, fmt.Errorf("failed to list events")
					}

					edges := make([]any, len(events))
					for i, evt := range events {
						edges[i] = map[string]any{
							"node":   eventToMap(evt),
							"cursor": fmt.Sprintf("cursor:%d", i),
						}
					}

					return map[string]any{
						"edges": edges,
						"pageInfo": map[string]any{
							"hasNextPage":     len(events) >= first,
							"hasPreviousPage": false,
							"startCursor":     "",
							"endCursor":       "",
						},
						"total": len(events),
					}, nil
				},
			},
			"block": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "Block",
					Fields: graphql.Fields{
						"number":       &graphql.Field{Type: graphql.Int},
						"hash":         &graphql.Field{Type: graphql.String},
						"parentHash":   &graphql.Field{Type: graphql.String},
						"timestamp":    &graphql.Field{Type: graphql.Int},
						"transactions": &graphql.Field{Type: graphql.NewList(graphql.String)},
					},
				}),
				Args: graphql.FieldConfigArgument{
					"number": &graphql.ArgumentConfig{Type: graphql.Int},
					"hash":   &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return nil, fmt.Errorf("block queries are not supported; use the events query instead")
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schema")
	}

	h.logger.Info("GraphQL schema built successfully")
	return &schema, nil
}

func eventToMap(evt any) map[string]any {
	if evt == nil {
		return nil
	}

	if e, ok := evt.(*core.BlockchainEvent); ok {
		return map[string]any{
			"id":               e.ID,
			"eventHash":        e.EventHash,
			"blockNumber":      e.BlockNumber,
			"blockHash":        e.BlockHash.Hex(),
			"blockTimestamp":   e.BlockTimestamp,
			"transactionHash":  e.TransactionHash.Hex(),
			"transactionIndex": e.TransactionIndex,
			"logIndex":         e.LogIndex,
			"contractAddress":  e.ContractAddress.Hex(),
			"eventName":        e.EventName,
			"eventSignature":   e.EventSignature.Hex(),
			"eventData":        eventDataMap(e.DecodedData),
			"chainId":          e.ChainID,
			"status":           string(e.Status),
			"processedAt":      e.BlockTimestamp,
		}
	}

	if e, ok := evt.(core.BlockchainEvent); ok {
		return map[string]any{
			"id":               e.ID,
			"eventHash":        e.EventHash,
			"blockNumber":      e.BlockNumber,
			"blockHash":        e.BlockHash.Hex(),
			"blockTimestamp":   e.BlockTimestamp,
			"transactionHash":  e.TransactionHash.Hex(),
			"transactionIndex": e.TransactionIndex,
			"logIndex":         e.LogIndex,
			"contractAddress":  e.ContractAddress.Hex(),
			"eventName":        e.EventName,
			"eventSignature":   e.EventSignature.Hex(),
			"eventData":        eventDataMap(e.DecodedData),
			"chainId":          e.ChainID,
			"status":           string(e.Status),
			"processedAt":      e.BlockTimestamp,
		}
	}

	return map[string]any{"id": "unknown"}
}

func eventDataMap(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	return data
}

// Initialize initializes the GraphQL handler
func (h *GraphQLHandler) Initialize(config *core.Config) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.initialized {
		return fmt.Errorf("GraphQL handler already initialized")
	}

	h.initialized = true

	h.logger.Info("GraphQL handler initialized", core.LogKeyComponent, "graphql_handler")

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
	var variables map[string]any

	if strings.Contains(contentType, "application/json") {
		var gqlReq GraphQLRequest
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid JSON in request body")
			return
		}
		query = gqlReq.Query
		variables = gqlReq.Variables
	} else {
		if err := r.ParseForm(); err != nil {
			h.writeError(w, http.StatusBadRequest, "failed to parse request")
			return
		}
		query = r.FormValue("query")
	}

	if query == "" {
		h.writeError(w, http.StatusBadRequest, "Query is required")
		return
	}

	if h.schema != nil {
		if err := validateQueryDepth(query, maxQueryDepth); err != nil {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		params := graphql.Params{
			Schema:         *h.schema,
			RequestString:  query,
			VariableValues: variables,
			OperationName:  "",
			Context:        r.Context(),
		}
		result := graphql.Do(params)

		if len(result.Errors) > 0 {
			errors := make([]GraphQLError, len(result.Errors))
			for i, e := range result.Errors {
				errors[i] = GraphQLError{
					Message: e.Message,
				}
			}
			resp := GraphQLResponse{Errors: errors}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				h.logger.Warn("failed to encode graphql error response", "error", err.Error())
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(GraphQLResponse{Data: result.Data}); err != nil {
			h.logger.Warn("failed to encode graphql success response", "error", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	if err := json.NewEncoder(w).Encode(GraphQLResponse{
		Data: map[string]string{"message": "GraphQL schema not initialized"},
	}); err != nil {
		h.logger.Warn("failed to encode graphql schema-not-initialized response", "error", err.Error())
	}
}

func (h *GraphQLHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(GraphQLResponse{
		Errors: []GraphQLError{{Message: message}},
	}); err != nil {
		h.logger.Warn("failed to encode graphql handler error response", "error", err.Error())
	}
}

// Stop stops the GraphQL handler
func (h *GraphQLHandler) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.initialized = false

	h.logger.Info("GraphQL handler stopped", core.LogKeyComponent, "graphql_handler")

	return nil
}

const maxQueryDepth = 10

// validateQueryDepth performs a basic check on query nesting depth to
// prevent malicious deep queries that could overwhelm the server.
func validateQueryDepth(query string, maxDepth int) error {
	depth := 0
	maxSeen := 0
	for _, ch := range query {
		if ch == '{' {
			depth++
			if depth > maxSeen {
				maxSeen = depth
			}
		}
		if ch == '}' {
			depth--
		}
		if maxSeen > maxDepth {
			return fmt.Errorf("query exceeds maximum depth of %d", maxDepth)
		}
	}
	return nil
}
