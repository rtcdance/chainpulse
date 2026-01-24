package graphql

import (
	"fmt"
	"net/http"
	"sync"

	"chainpulse/pkg/plugins/api/core"
	"github.com/graphql-go/graphql"
)

// GraphQLPlugin implements the GraphQL protocol handler
type GraphQLPlugin struct {
	name       string
	port       int
	apiLayer   *core.APILayer
	schema     graphql.Schema
	server     *http.Server
	processor  core.RequestProcessor
	mu         sync.RWMutex
	running    bool
	middleware []core.Middleware
	resolvers  map[string]GraphQLResolver
	router     *core.APIRouter
}

// GraphQLResolver defines a GraphQL resolver function
type GraphQLResolver func(p graphql.ResolveParams) (interface{}, error)

// NewGraphQLPlugin creates a new GraphQL plugin
func NewGraphQLPlugin(name string, port int, apiLayer *core.APILayer) *GraphQLPlugin {
	processor := core.NewDefaultRequestProcessor(apiLayer)
	return &GraphQLPlugin{
		name:       name,
		port:       port,
		apiLayer:   apiLayer,
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		resolvers:  make(map[string]GraphQLResolver),
		router:     core.NewAPIRouter(),
	}
}

// Start starts the GraphQL server
func (p *GraphQLPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("GraphQL plugin already running")
	}

	// Build schema
	if err := p.buildSchema(); err != nil {
		return fmt.Errorf("failed to build GraphQL schema: %w", err)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", p.handleGraphQL)
	mux.HandleFunc("/graphql/playground", p.handlePlayground)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: mux,
	}

	p.running = true

	// Start server in background
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("GraphQL server error: %v\n", err)
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
		if err := p.server.Close(); err != nil {
			return err
		}
	}

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

// buildSchema builds the GraphQL schema
func (p *GraphQLPlugin) buildSchema() error {
	// Define query type
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"event": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: p.resolveEvent,
			},
			"events": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: p.resolveEvents,
			},
			"token": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"address": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: p.resolveToken,
			},
			"pool": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"address": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: p.resolvePool,
			},
			"health": &graphql.Field{
				Type: graphql.String,
				Resolve: p.resolveHealth,
			},
		},
	})

	// Define mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"executeQuery": &graphql.Field{
				Type: graphql.String,
				Args: graphql.FieldConfigArgument{
					"query": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: p.resolveExecuteQuery,
			},
		},
	})

	// Create schema
	schemaConfig := graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	}

	schema, err := graphql.NewSchema(schemaConfig)
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

// Resolver functions
func (p *GraphQLPlugin) resolveEvent(params graphql.ResolveParams) (interface{}, error) {
	id, ok := params.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid id parameter")
	}
	return fmt.Sprintf(`{"id":"%s","type":"event"}`, id), nil
}

func (p *GraphQLPlugin) resolveEvents(params graphql.ResolveParams) (interface{}, error) {
	limit := 20
	offset := 0

	if l, ok := params.Args["limit"].(int); ok {
		limit = l
	}
	if o, ok := params.Args["offset"].(int); ok {
		offset = o
	}

	return fmt.Sprintf(`{"limit":%d,"offset":%d,"events":[]}`, limit, offset), nil
}

func (p *GraphQLPlugin) resolveToken(params graphql.ResolveParams) (interface{}, error) {
	address, ok := params.Args["address"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid address parameter")
	}
	return fmt.Sprintf(`{"address":"%s","type":"token"}`, address), nil
}

func (p *GraphQLPlugin) resolvePool(params graphql.ResolveParams) (interface{}, error) {
	address, ok := params.Args["address"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid address parameter")
	}
	return fmt.Sprintf(`{"address":"%s","type":"pool"}`, address), nil
}

func (p *GraphQLPlugin) resolveHealth(params graphql.ResolveParams) (interface{}, error) {
	return `{"status":"healthy"}`, nil
}

func (p *GraphQLPlugin) resolveExecuteQuery(params graphql.ResolveParams) (interface{}, error) {
	query, ok := params.Args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid query parameter")
	}
	return fmt.Sprintf(`{"query":"%s","result":"executed"}`, query), nil
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
