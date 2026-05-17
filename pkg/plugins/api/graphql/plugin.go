package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"chainpulse/pkg/plugins/api/core"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

// GraphQLPlugin implements the GraphQL protocol handler
type GraphQLPlugin struct {
	name                string
	port                int
	apiLayer            *core.APILayer
	schema              graphql.Schema
	server              *http.Server
	processor           core.RequestProcessor
	mu                  sync.RWMutex
	running             bool
	middleware          []core.Middleware
	resolvers           map[string]GraphQLResolver
	router              *core.APIRouter
	schemaBuilder       *SchemaBuilder
	subscriptionManager *SubscriptionManager
	upgrader            websocket.Upgrader
	allowedOrigins      []string

	// WebSocket safety
	wsMu sync.Mutex     // protects concurrent WebSocket writes
	wsWg sync.WaitGroup // tracks subscription goroutines for graceful shutdown
}

// GraphQLResolver defines a GraphQL resolver function
type GraphQLResolver func(p graphql.ResolveParams) (any, error)

// NewGraphQLPlugin creates a new GraphQL plugin
func NewGraphQLPlugin(name string, port int, apiLayer *core.APILayer) *GraphQLPlugin {
	processor := core.NewDefaultRequestProcessor(apiLayer)
	p := &GraphQLPlugin{
		name:       name,
		port:       port,
		apiLayer:   apiLayer,
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		resolvers:  make(map[string]GraphQLResolver),
		router:     core.NewAPIRouter(),
	}
	p.upgrader = websocket.Upgrader{CheckOrigin: p.checkOrigin}
	return p
}

// SetSchemaBuilder sets the schema builder for building the GraphQL schema
func (p *GraphQLPlugin) SetSchemaBuilder(sb *SchemaBuilder) {
	p.schemaBuilder = sb
}

// SetSubscriptionManager sets the subscription manager for GraphQL subscriptions
func (p *GraphQLPlugin) SetSubscriptionManager(sm *SubscriptionManager) {
	p.subscriptionManager = sm
}

func (p *GraphQLPlugin) WithAllowedOrigins(origins []string) *GraphQLPlugin {
	p.allowedOrigins = origins
	return p
}

func (p *GraphQLPlugin) checkOrigin(r *http.Request) bool {
	if len(p.allowedOrigins) == 0 {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	for _, allowed := range p.allowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

// Start starts the GraphQL server
func (p *GraphQLPlugin) Start() error {
	p.mu.Lock()

	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("GraphQL plugin already running")
	}

	if len(p.allowedOrigins) == 0 {
		slog.Warn("GraphQL WebSocket: no origin restrictions configured — accepting all origins (development mode only)")
	}

	// Build schema
	if err := p.buildSchema(); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("failed to build GraphQL schema: %w", err)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", p.handleGraphQL)
	mux.HandleFunc("/graphql/playground", p.handlePlayground)
	mux.HandleFunc("/graphql/ws", p.handleWebSocket)

	p.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	p.running = true

	// Capture server reference before spawning goroutine to avoid race
	srv := p.server
	p.mu.Unlock()

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("GraphQL server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the GraphQL server
func (p *GraphQLPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("GraphQL plugin not running")
	}

	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown GraphQL server: %w", err)
		}
	}

	// Wait for subscription goroutines to drain
	p.wsWg.Wait()

	p.running = false
	return nil
}

// GetName returns the plugin name
func (p *GraphQLPlugin) GetName() string {
	return p.name
}

// GetProtocolName returns the protocol name (implements ProtocolHandler)
func (p *GraphQLPlugin) GetProtocolName() string {
	return p.name
}

// IsRunning returns whether the plugin is running
func (p *GraphQLPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// RegisterResolver registers a GraphQL resolver
func (p *GraphQLPlugin) RegisterResolver(name string, resolver GraphQLResolver) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resolvers[name] = resolver
}

// RegisterRoute registers a route handler (implements ProtocolHandler)
func (p *GraphQLPlugin) RegisterRoute(path string, handler core.Handler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot register route while handler is running")
	}

	p.router.Register(path, handler)
	return nil
}

// Use adds middleware (implements ProtocolHandler)
func (p *GraphQLPlugin) Use(middleware ...core.Middleware) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot add middleware while handler is running")
	}

	p.middleware = append(p.middleware, middleware...)
	p.router.Use(middleware...)
	return nil
}

// buildSchema builds the GraphQL schema using SchemaBuilder
func (p *GraphQLPlugin) buildSchema() error {
	if p.schemaBuilder != nil {
		schema, err := p.schemaBuilder.BuildSchema()
		if err != nil {
			return err
		}
		p.schema = schema
		return nil
	}

	// Fallback: minimal schema when no SchemaBuilder is configured
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"health": &graphql.Field{
				Type:    graphql.String,
				Resolve: func(p graphql.ResolveParams) (any, error) { return "healthy", nil },
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{Query: queryType})
	if err != nil {
		return err
	}
	p.schema = schema
	return nil
}

// handleGraphQL handles GraphQL requests
func (p *GraphQLPlugin) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	// Create request adapter
	req := NewGraphQLRequest(r)

	// Process through API layer
	result := p.apiLayer.Handle(req)

	// Write response
	for key, value := range result.Headers() {
		w.Header().Set(key, value)
	}

	w.WriteHeader(result.Status())
	_, _ = w.Write(result.Body())
}

// handlePlayground serves GraphQL Playground
func (p *GraphQLPlugin) handlePlayground(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(playgroundHTML))
}

// handleWebSocket handles WebSocket connections for GraphQL subscriptions
func (p *GraphQLPlugin) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close() //nolint:errcheck // deferred close

	// Thread-safe write helper — gorilla/websocket requires serialized writes
	writeMsg := func(v any) {
		p.wsMu.Lock()
		defer p.wsMu.Unlock()
		_ = conn.WriteJSON(v)
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var req struct {
			ID      string          `json:"id"`
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			writeMsg(map[string]any{
				"type":    "error",
				"payload": map[string]string{"message": "invalid message format"},
			})
			continue
		}

		switch req.Type {
		case "connection_init":
			writeMsg(map[string]any{
				"type": "connection_ack",
			})

		case "start":
			var payload struct {
				Query         string         `json:"query"`
				OperationName string         `json:"operationName"`
				Variables     map[string]any `json:"variables"`
			}
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				writeMsg(map[string]any{
					"id":   req.ID,
					"type": "error",
					"payload": map[string]string{
						"message": "invalid payload",
					},
				})
				continue
			}

			// Execute subscription query
			result := graphql.Subscribe(graphql.Params{
				Schema:         p.schema,
				RequestString:  payload.Query,
				OperationName:  payload.OperationName,
				VariableValues: payload.Variables,
				Context:        r.Context(),
			})

			p.wsWg.Add(1)
			go func(id string, ch chan *graphql.Result) {
				defer p.wsWg.Done()
				for {
					select {
					case data, ok := <-ch:
						if !ok {
							writeMsg(map[string]any{
								"id":   id,
								"type": "complete",
							})
							return
						}
						writeMsg(map[string]any{
							"id":      id,
							"type":    "data",
							"payload": data,
						})
					case <-r.Context().Done():
						return
					}
				}
			}(req.ID, result)

		case "stop":
			// Client requests stop — the subscription goroutine will
			// terminate when its context is cancelled on disconnect.

		case "connection_terminate":
			return
		}
	}
}

// GetSchema returns the GraphQL schema
func (p *GraphQLPlugin) GetSchema() graphql.Schema {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.schema
}

// playgroundHTML is the GraphQL Playground HTML
const playgroundHTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css"/>
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png"/>
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root"></div>
  <script>
    window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: '/graphql',
        subscriptionEndpoint: '/graphql',
      })
    })
  </script>
</body>
</html>
`
