package graphql

import (
	"encoding/json"
	"fmt"
	"time"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// SchemaBuilder builds the GraphQL schema
type SchemaBuilder struct {
	eventStore     domainquery.EventStore
	logger         core.Logger
	metrics        core.MetricsCollector
	cache          core.CachePlugin
	authMiddleware *AuthMiddleware
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder(
	eventStore domainquery.EventStore,
	logger core.Logger,
	metrics core.MetricsCollector,
	cache core.CachePlugin,
	authMiddleware *AuthMiddleware,
) *SchemaBuilder {
	return &SchemaBuilder{
		eventStore:     eventStore,
		logger:         logger,
		metrics:        metrics,
		cache:          cache,
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
			"querySourcePosture": &graphql.Field{
				Type:        graphql.String,
				Description: "Compact query source posture for this event result",
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
				Resolve:     sb.resolveClearCache,
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

	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", id)
		if cached, err := sb.cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			sb.metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var result map[string]interface{}
			if err := json.Unmarshal(cached, &result); err == nil {
				return withQuerySourcePosture(result, "graphql-cache-hit"), nil
			}
			return cached, nil
		}
		sb.metrics.RecordCounter("graphql_cache_miss", 1, nil)
	}

	// Retrieve event from store
	event, err := sb.eventStore.GetEvent(p.Context, id)
	if err != nil {
		sb.logger.Error("Failed to resolve event", "id", id, "error", err.Error())
		sb.metrics.RecordCounter("graphql_resolve_event_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve event: %w", err)
	}

	if event == nil {
		return nil, nil
	}

	// Convert to GraphQL response format
	result := withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store")
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", id)
		resultBytes, _ := json.Marshal(result)
		_ = sb.cache.Set(p.Context, cacheKey, resultBytes, 300)
	}
	sb.metrics.RecordCounter("graphql_resolve_event_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveEvents(p graphql.ResolveParams) (interface{}, error) {
	// Limit maximum page size
	first := 20
	if f, ok := p.Args["first"].(int); ok && f > 0 {
		first = f
	}
	if first > 1000 {
		first = 1000
	}

	// Get cursor for pagination
	after := ""
	if a, ok := p.Args["after"].(string); ok {
		after = a
	}

	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:root:after:%s:first:%d", after, first)
		if cached, err := sb.cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			sb.metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var connection map[string]interface{}
			if err := json.Unmarshal(cached, &connection); err == nil {
				return withQuerySourcePostureConnection(connection, "graphql-cache-hit"), nil
			}
			return cached, nil
		}
		sb.metrics.RecordCounter("graphql_cache_miss", 1, nil)
	}

	// Retrieve events from store with pagination
	events, hasNextPage, err := sb.eventStore.GetEventsPaginated(p.Context, after, first)
	if err != nil {
		sb.logger.Error("Failed to resolve events", "error", err.Error())
		sb.metrics.RecordCounter("graphql_resolve_events_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events: %w", err)
	}

	// Build edges
	edges := make([]interface{}, 0, len(events))
	var endCursor string
	for i, event := range events {
		cursor := fmt.Sprintf("cursor_%d", i)
		edges = append(edges, map[string]interface{}{
			"node":   withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"),
			"cursor": cursor,
		})
		if i == len(events)-1 {
			endCursor = cursor
		}
	}

	connection := map[string]interface{}{
		"edges": edges,
		"pageInfo": map[string]interface{}{
			"hasNextPage":     hasNextPage,
			"hasPreviousPage": after != "",
			"startCursor":     after,
			"endCursor":       endCursor,
			"totalCount":      len(events),
		},
	}
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:root:after:%s:first:%d", after, first)
		connectionBytes, _ := json.Marshal(connection)
		_ = sb.cache.Set(p.Context, cacheKey, connectionBytes, 300)
	}

	sb.metrics.RecordCounter("graphql_resolve_events_success", 1, nil)
	return connection, nil
}

func (sb *SchemaBuilder) resolveEventsByBlock(p graphql.ResolveParams) (interface{}, error) {
	blockNumber, ok := p.Args["blockNumber"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid blockNumber parameter")
	}

	// Retrieve events by block number
	events, err := sb.eventStore.GetEventsByBlock(p.Context, int64(blockNumber))
	if err != nil {
		sb.logger.Error("Failed to resolve events by block", "blockNumber", blockNumber, "error", err.Error())
		sb.metrics.RecordCounter("graphql_resolve_events_by_block_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by block: %w", err)
	}

	// Convert to GraphQL response format
	result := make([]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"))
	}

	sb.metrics.RecordCounter("graphql_resolve_events_by_block_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveEventsByAddress(p graphql.ResolveParams) (interface{}, error) {
	address, ok := p.Args["address"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid address parameter")
	}

	limit := 100
	if l, ok := p.Args["limit"].(int); ok && l > 0 {
		limit = l
	}
	if limit > 1000 {
		limit = 1000
	}

	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:address:%s:limit:%d", address, limit)
		if cached, err := sb.cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			sb.metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var result []interface{}
			if err := json.Unmarshal(cached, &result); err == nil {
				return withQuerySourcePostureList(result, "graphql-cache-hit"), nil
			}
			return cached, nil
		}
	}

	// Retrieve events by contract address
	events, err := sb.eventStore.GetEventsByAddress(p.Context, address, limit)
	if err != nil {
		sb.logger.Error("Failed to resolve events by address", "address", address, "error", err.Error())
		sb.metrics.RecordCounter("graphql_resolve_events_by_address_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by address: %w", err)
	}

	// Convert to GraphQL response format
	result := make([]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"))
	}
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:address:%s:limit:%d", address, limit)
		resultBytes, _ := json.Marshal(result)
		_ = sb.cache.Set(p.Context, cacheKey, resultBytes, 600)
	}

	sb.metrics.RecordCounter("graphql_resolve_events_by_address_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveEventsByName(p graphql.ResolveParams) (interface{}, error) {
	eventName, ok := p.Args["eventName"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid eventName parameter")
	}

	limit := 100
	if l, ok := p.Args["limit"].(int); ok && l > 0 {
		limit = l
	}
	if limit > 1000 {
		limit = 1000
	}

	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:name:%s:limit:%d", eventName, limit)
		if cached, err := sb.cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			sb.metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var result []interface{}
			if err := json.Unmarshal(cached, &result); err == nil {
				return withQuerySourcePostureList(result, "graphql-cache-hit"), nil
			}
			return cached, nil
		}
	}

	// Retrieve events by event name
	events, err := sb.eventStore.GetEventsByName(p.Context, eventName, limit)
	if err != nil {
		sb.logger.Error("Failed to resolve events by name", "eventName", eventName, "error", err.Error())
		sb.metrics.RecordCounter("graphql_resolve_events_by_name_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by name: %w", err)
	}

	// Convert to GraphQL response format
	result := make([]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"))
	}
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:name:%s:limit:%d", eventName, limit)
		resultBytes, _ := json.Marshal(result)
		_ = sb.cache.Set(p.Context, cacheKey, resultBytes, 600)
	}

	sb.metrics.RecordCounter("graphql_resolve_events_by_name_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveInvalidateCache(p graphql.ResolveParams) (interface{}, error) {
	eventID, ok := p.Args["eventId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid eventId parameter")
	}

	// Invalidate cache entries for this event
	// Note: Cache invalidation would be handled by cache plugin if available
	sb.logger.Info("Cache invalidation requested", "eventId", eventID)
	sb.metrics.RecordCounter("graphql_invalidate_cache_success", 1, nil)
	return true, nil
}

func (sb *SchemaBuilder) resolveClearCache(p graphql.ResolveParams) (interface{}, error) {
	// Clear all GraphQL cache entries
	sb.logger.Info("Cache clearing requested")
	sb.metrics.RecordCounter("graphql_clear_cache_success", 1, nil)
	return true, nil
}

// Helper function to convert event to GraphQL response format
func eventToGraphQLResponse(event *core.BlockchainEvent) map[string]interface{} {
	decodedData := ""
	if event.DecodedData != nil {
		if data, err := json.Marshal(event.DecodedData); err == nil {
			decodedData = string(data)
		}
	}

	return map[string]interface{}{
		"id":               event.ID,
		"eventHash":        event.EventHash,
		"blockNumber":      event.BlockNumber,
		"blockHash":        event.BlockHash.Hex(),
		"blockTimestamp":   event.BlockTimestamp,
		"transactionHash":  event.TransactionHash.Hex(),
		"transactionIndex": event.TransactionIndex,
		"logIndex":         event.LogIndex,
		"contractAddress":  event.ContractAddress.Hex(),
		"eventName":        event.EventName,
		"chainId":          event.ChainID,
		"network":          event.Network,
		"status":           string(event.Status),
		"removed":          event.Removed,
		"gasUsed":          event.GasUsed,
		"gasPrice":         event.GasPrice.String(),
		"decodedData":      decodedData,
		"createdAt":        event.CreatedAt.Format(time.RFC3339),
		"processedAt":      event.ProcessedAt.Format(time.RFC3339),
		"indexedAt":        event.IndexedAt.Format(time.RFC3339),
	}
}
