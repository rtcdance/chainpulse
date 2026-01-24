package graphql

import (
	"fmt"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// SchemaBuilder builds the GraphQL schema
type SchemaBuilder struct {
	eventStore    query.EventStore
	logger        core.Logger
	metrics       core.MetricsCollector
	authMiddleware *AuthMiddleware
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder(
	eventStore query.EventStore,
	logger core.Logger,
	metrics core.MetricsCollector,
	authMiddleware *AuthMiddleware,
) *SchemaBuilder {
	return &SchemaBuilder{
		eventStore:    eventStore,
		logger:        logger,
		metrics:       metrics,
		authMiddleware: authMiddleware,
	}
}

// BuildSchema builds the complete GraphQL schema
func (sb *SchemaBuilder) BuildSchema() (graphql.Schema, error) {
	// Define scalar types
	bigIntType := graphql.NewScalar(graphql.ScalarConfig{
		Name:        "BigInt",
		Description: "Big integer type for large numbers",
		Serialize: func(value interface{}) interface{} {
			return fmt.Sprintf("%v", value)
		},
		ParseValue: func(value interface{}) interface{} {
			return value
		},
		ParseLiteral: func(valueAST ast.Value) interface{} {
			switch v := valueAST.(type) {
			case *ast.StringValue:
				return v.Value
			case *ast.IntValue:
				return v.Value
			default:
				return nil
			}
		},
	})

	// Define Event type
	eventType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Event",
		Description: "Blockchain event",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Event ID",
			},
			"eventHash": &graphql.Field{
				Type:        graphql.String,
				Description: "Event hash",
			},
			"blockNumber": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Block number",
			},
			"blockHash": &graphql.Field{
				Type:        graphql.String,
				Description: "Block hash",
			},
			"blockTimestamp": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Block timestamp",
			},
			"transactionHash": &graphql.Field{
				Type:        graphql.String,
				Description: "Transaction hash",
			},
			"transactionIndex": &graphql.Field{
				Type:        graphql.Int,
				Description: "Transaction index",
			},
			"logIndex": &graphql.Field{
				Type:        graphql.Int,
				Description: "Log index",
			},
			"contractAddress": &graphql.Field{
				Type:        graphql.String,
				Description: "Contract address",
			},
			"eventName": &graphql.Field{
				Type:        graphql.String,
				Description: "Event name",
			},
			"chainId": &graphql.Field{
				Type:        graphql.String,
				Description: "Chain ID",
			},
			"network": &graphql.Field{
				Type:        graphql.String,
				Description: "Network name",
			},
			"status": &graphql.Field{
				Type:        graphql.String,
				Description: "Event status (pending, confirmed, failed, reorged)",
			},
			"removed": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Whether the event was removed",
			},
			"gasUsed": &graphql.Field{
				Type:        bigIntType,
				Description: "Gas used",
			},
			"gasPrice": &graphql.Field{
				Type:        bigIntType,
				Description: "Gas price",
			},
			"decodedData": &graphql.Field{
				Type:        graphql.String,
				Description: "Decoded event data as JSON",
			},
			"createdAt": &graphql.Field{
				Type:        graphql.String,
				Description: "Creation timestamp",
			},
			"processedAt": &graphql.Field{
				Type:        graphql.String,
				Description: "Processing timestamp",
			},
			"indexedAt": &graphql.Field{
				Type:        graphql.String,
				Description: "Indexing timestamp",
			},
		},
	})

	// Define EventConnection for pagination
	eventConnectionType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "EventConnection",
		Description: "Connection for paginated events",
		Fields: graphql.Fields{
			"edges": &graphql.Field{
				Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
					Name: "EventEdge",
					Fields: graphql.Fields{
						"node": &graphql.Field{
							Type: eventType,
						},
						"cursor": &graphql.Field{
							Type: graphql.String,
						},
					},
				})),
				Description: "Event edges",
			},
			"pageInfo": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "PageInfo",
					Fields: graphql.Fields{
						"hasNextPage": &graphql.Field{
							Type: graphql.Boolean,
						},
						"hasPreviousPage": &graphql.Field{
							Type: graphql.Boolean,
						},
						"startCursor": &graphql.Field{
							Type: graphql.String,
						},
						"endCursor": &graphql.Field{
							Type: graphql.String,
						},
						"totalCount": &graphql.Field{
							Type: graphql.Int,
						},
					},
				}),
				Description: "Pagination info",
			},
		},
	})

	// Define Query type
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Query",
		Description: "Root query type",
		Fields: graphql.Fields{
			"event": &graphql.Field{
				Type:        eventType,
				Description: "Get a single event by ID",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Event ID",
					},
				},
				Resolve: sb.resolveEvent,
			},
			"events": &graphql.Field{
				Type:        eventConnectionType,
				Description: "Get events with pagination",
				Args: graphql.FieldConfigArgument{
					"first": &graphql.ArgumentConfig{
						Type:        graphql.Int,
						Description: "Number of events to return",
					},
					"after": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Cursor for pagination",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Filter criteria as JSON",
					},
					"sort": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Sort criteria as JSON",
					},
				},
				Resolve: sb.resolveEvents,
			},
			"eventsByBlock": &graphql.Field{
				Type:        graphql.NewList(eventType),
				Description: "Get events by block number",
				Args: graphql.FieldConfigArgument{
					"blockNumber": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.Int),
						Description: "Block number",
					},
				},
				Resolve: sb.resolveEventsByBlock,
			},
			"eventsByAddress": &graphql.Field{
				Type:        graphql.NewList(eventType),
				Description: "Get events by contract address",
				Args: graphql.FieldConfigArgument{
					"address": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Contract address",
					},
					"limit": &graphql.ArgumentConfig{
						Type:        graphql.Int,
						Description: "Maximum number of events",
					},
				},
				Resolve: sb.resolveEventsByAddress,
			},
			"eventsByName": &graphql.Field{
				Type:        graphql.NewList(eventType),
				Description: "Get events by event name",
				Args: graphql.FieldConfigArgument{
					"eventName": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Event name",
					},
					"limit": &graphql.ArgumentConfig{
						Type:        graphql.Int,
						Description: "Maximum number of events",
					},
				},
				Resolve: sb.resolveEventsByName,
			},
			"health": &graphql.Field{
				Type:        graphql.String,
				Description: "Health check",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return "healthy", nil
				},
			},
		},
	})

	// Define Mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Mutation",
		Description: "Root mutation type",
		Fields: graphql.Fields{
			"invalidateCache": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Invalidate cache for an event",
				Args: graphql.FieldConfigArgument{
					"eventId": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Event ID",
					},
				},
				Resolve: sb.resolveInvalidateCache,
			},
			"clearCache": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Clear all cache",
				Resolve: sb.resolveClearCache,
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
		return graphql.Schema{}, fmt.Errorf("failed to create GraphQL schema: %w", err)
	}

	return schema, nil
}

// Resolver functions
func (sb *SchemaBuilder) resolveEvent(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid id parameter")
	}

	// TODO: Implement actual event retrieval
	return map[string]interface{}{
		"id":    id,
		"status": "confirmed",
	}, nil
}

func (sb *SchemaBuilder) resolveEvents(p graphql.ResolveParams) (interface{}, error) {
	// TODO: Implement actual events retrieval with pagination
	return map[string]interface{}{
		"edges":    []interface{}{},
		"pageInfo": map[string]interface{}{
			"hasNextPage":     false,
			"hasPreviousPage": false,
			"totalCount":      0,
		},
	}, nil
}

func (sb *SchemaBuilder) resolveEventsByBlock(p graphql.ResolveParams) (interface{}, error) {
	blockNumber, ok := p.Args["blockNumber"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid blockNumber parameter")
	}

	// TODO: Implement actual events retrieval by block
	_ = blockNumber
	return []interface{}{}, nil
}

func (sb *SchemaBuilder) resolveEventsByAddress(p graphql.ResolveParams) (interface{}, error) {
	address, ok := p.Args["address"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid address parameter")
	}

	limit := 100
	if l, ok := p.Args["limit"].(int); ok {
		limit = l
	}

	// TODO: Implement actual events retrieval by address
	_ = address
	_ = limit
	return []interface{}{}, nil
}

func (sb *SchemaBuilder) resolveEventsByName(p graphql.ResolveParams) (interface{}, error) {
	eventName, ok := p.Args["eventName"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid eventName parameter")
	}

	limit := 100
	if l, ok := p.Args["limit"].(int); ok {
		limit = l
	}

	// TODO: Implement actual events retrieval by name
	_ = eventName
	_ = limit
	return []interface{}{}, nil
}

func (sb *SchemaBuilder) resolveInvalidateCache(p graphql.ResolveParams) (interface{}, error) {
	eventID, ok := p.Args["eventId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid eventId parameter")
	}

	// TODO: Implement cache invalidation
	_ = eventID
	return true, nil
}

func (sb *SchemaBuilder) resolveClearCache(p graphql.ResolveParams) (interface{}, error) {
	// TODO: Implement cache clearing
	return true, nil
}
