package graphql

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// authTokenContextKey is the context key for the JWT token extracted from
// the HTTP Authorization header. Set by the GraphQL HTTP handler before
// invoking the schema.
type authTokenKeyType struct{}

var authTokenContextKey = authTokenKeyType{}

// JSONScalar is a custom GraphQL scalar that passes through arbitrary JSON values
// (maps, slices, strings, numbers, etc.) without stringifying them.
var JSONScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:         "JSON",
	Description:  "Arbitrary JSON value",
	Serialize:    func(value any) any { return value },
	ParseValue:   func(value any) any { return value },
	ParseLiteral: func(valueAST ast.Value) any { return nil },
})

// SchemaBuilder builds the GraphQL schema
type SchemaBuilder struct {
	eventStore          domainquery.EventReader
	logger              core.Logger
	metrics             core.MetricsCollector
	cache               core.CachePlugin
	authMiddleware      *AuthMiddleware
	subscriptionManager *SubscriptionManager
}

// SetSubscriptionManager sets the subscription manager for GraphQL subscriptions
func (sb *SchemaBuilder) SetSubscriptionManager(sm *SubscriptionManager) {
	sb.subscriptionManager = sm
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder(
	eventStore domainquery.EventReader,
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

// Subscription resolver methods

// subscribeEventCreated returns a channel that emits newly created events
func (sb *SchemaBuilder) subscribeEventCreated(p graphql.ResolveParams) (any, error) {
	if sb.subscriptionManager == nil {
		return nil, fmt.Errorf("subscriptions are not available")
	}

	chainFilter, _ := p.Args["chainId"].(string)
	contractFilter, _ := p.Args["contractAddress"].(string)

	sub, err := sb.subscriptionManager.Subscribe("event:created")
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to event:created: %w", err)
	}

	filtered := make(chan any, 100)
	go func() {
		defer close(filtered)
		for {
			select {
			case <-p.Context.Done():
				sb.subscriptionManager.Unsubscribe(sub) //nolint:errcheck // best-effort unsubscribe on context cancel
				return
			case data, ok := <-sub.Channel:
				if !ok {
					return
				}
				payload, ok := data.(*EventSubscriptionPayload)
				if !ok {
					continue
				}
				if chainFilter != "" && payload.ChainID != chainFilter {
					continue
				}
				if contractFilter != "" && payload.ContractAddress != contractFilter {
					continue
				}
				select {
				case filtered <- mapToSubscriptionPayload(payload):
				default:
				}
			}
		}
	}()

	return filtered, nil
}

// subscribeEventConfirmed returns a channel that emits confirmed events
func (sb *SchemaBuilder) subscribeEventConfirmed(p graphql.ResolveParams) (any, error) {
	if sb.subscriptionManager == nil {
		return nil, fmt.Errorf("subscriptions are not available")
	}

	sub, err := sb.subscriptionManager.Subscribe("event:confirmed")
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to event:confirmed: %w", err)
	}

	ch := make(chan any, 100)
	go func() {
		defer close(ch)
		for {
			select {
			case <-p.Context.Done():
				sb.subscriptionManager.Unsubscribe(sub) //nolint:errcheck // best-effort unsubscribe on context cancel
				return
			case data, ok := <-sub.Channel:
				if !ok {
					return
				}
				payload, ok := data.(*EventSubscriptionPayload)
				if !ok {
					continue
				}
				select {
				case ch <- mapToSubscriptionPayload(payload):
				default:
				}
			}
		}
	}()

	return ch, nil
}

// subscribeEventFailed returns a channel that emits failed event notifications
func (sb *SchemaBuilder) subscribeEventFailed(p graphql.ResolveParams) (any, error) {
	if sb.subscriptionManager == nil {
		return nil, fmt.Errorf("subscriptions are not available")
	}

	sub, err := sb.subscriptionManager.Subscribe("event:failed")
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to event:failed: %w", err)
	}

	ch := make(chan any, 100)
	go func() {
		defer close(ch)
		for {
			select {
			case <-p.Context.Done():
				sb.subscriptionManager.Unsubscribe(sub) //nolint:errcheck // best-effort unsubscribe on context cancel
				return
			case data, ok := <-sub.Channel:
				if !ok {
					return
				}
				payload, ok := data.(*EventSubscriptionPayload)
				if !ok {
					continue
				}
				select {
				case ch <- mapToSubscriptionPayload(payload):
				default:
				}
			}
		}
	}()

	return ch, nil
}

// mapToSubscriptionPayload converts an EventSubscriptionPayload to a map for GraphQL resolution
func mapToSubscriptionPayload(payload *EventSubscriptionPayload) map[string]any {
	return map[string]any{
		"type":            payload.Type,
		"eventId":         payload.EventID,
		"chainId":         payload.ChainID,
		"contractAddress": payload.ContractAddress,
		"eventName":       payload.EventName,
		"blockNumber":     payload.BlockNumber,
		"timestamp":       payload.Timestamp,
	}
}

// BuildSchema builds the complete GraphQL schema
func (sb *SchemaBuilder) BuildSchema() (graphql.Schema, error) {
	// Define scalar types
	bigIntType := graphql.NewScalar(graphql.ScalarConfig{
		Name:        "BigInt",
		Description: "Big integer type for large numbers",
		Serialize: func(value any) any {
			return fmt.Sprintf("%v", value)
		},
		ParseValue: func(value any) any {
			return value
		},
		ParseLiteral: func(valueAST ast.Value) any {
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
				Type:        JSONScalar,
				Description: "Decoded event data as JSON object",
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

	// Define EventSubscriptionPayload type for subscriptions
	eventSubscriptionPayloadType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "EventSubscriptionPayload",
		Description: "Payload for event subscriptions",
		Fields: graphql.Fields{
			"type": &graphql.Field{
				Type:        graphql.String,
				Description: "Event type (created, confirmed, failed)",
			},
			"eventId": &graphql.Field{
				Type:        graphql.String,
				Description: "Event ID",
			},
			"chainId": &graphql.Field{
				Type:        graphql.String,
				Description: "Chain ID",
			},
			"contractAddress": &graphql.Field{
				Type:        graphql.String,
				Description: "Contract address",
			},
			"eventName": &graphql.Field{
				Type:        graphql.String,
				Description: "Event name",
			},
			"blockNumber": &graphql.Field{
				Type:        graphql.Int,
				Description: "Block number",
			},
			"timestamp": &graphql.Field{
				Type:        graphql.Int,
				Description: "Unix timestamp",
			},
		},
	})

	// EventFilterInput defines structured filter criteria for the events query.
	eventFilterInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "EventFilterInput",
		Description: "Filter criteria for blockchain events",
		Fields: graphql.InputObjectConfigFieldMap{
			"chainId": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Filter by chain ID (e.g. '1', '137')",
			},
			"contractAddress": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Filter by contract address (0x-prefixed hex)",
			},
			"eventName": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Filter by event name (e.g. 'Transfer', 'Swap')",
			},
			"status": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Filter by status (pending, confirmed, failed, reorged)",
			},
			"blockNumberGte": &graphql.InputObjectFieldConfig{
				Type:        graphql.Int,
				Description: "Minimum block number (inclusive)",
			},
			"blockNumberLte": &graphql.InputObjectFieldConfig{
				Type:        graphql.Int,
				Description: "Maximum block number (inclusive)",
			},
			"removed": &graphql.InputObjectFieldConfig{
				Type:        graphql.Boolean,
				Description: "Filter by removed flag",
			},
		},
	})

	// EventSortInput defines structured sort criteria for the events query.
	eventSortInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "EventSortInput",
		Description: "Sort criteria for blockchain events",
		Fields: graphql.InputObjectConfigFieldMap{
			"field": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Field to sort by (blockNumber, blockTimestamp, logIndex, eventName)",
			},
			"order": &graphql.InputObjectFieldConfig{
				Type:        graphql.String,
				Description: "Sort order: ASC or DESC (default: DESC)",
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
						Type:        eventFilterInput,
						Description: "Structured filter criteria for events",
					},
					"sort": &graphql.ArgumentConfig{
						Type:        eventSortInput,
						Description: "Sort criteria for events",
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
				Resolve: func(p graphql.ResolveParams) (any, error) {
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

	// Define Subscription type
	subscriptionType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Subscription",
		Description: "Real-time event subscriptions",
		Fields: graphql.Fields{
			"eventCreated": &graphql.Field{
				Type:        eventSubscriptionPayloadType,
				Description: "Subscribe to newly created events",
				Args: graphql.FieldConfigArgument{
					"chainId": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Optional chain ID filter",
					},
					"contractAddress": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Optional contract address filter",
					},
				},
				Subscribe: sb.subscribeEventCreated,
			},
			"eventConfirmed": &graphql.Field{
				Type:        eventSubscriptionPayloadType,
				Description: "Subscribe to confirmed events",
				Subscribe:   sb.subscribeEventConfirmed,
			},
			"eventFailed": &graphql.Field{
				Type:        eventSubscriptionPayloadType,
				Description: "Subscribe to failed event notifications",
				Subscribe:   sb.subscribeEventFailed,
			},
		},
	})

	// Create schema
	schemaConfig := graphql.SchemaConfig{
		Query:        queryType,
		Mutation:     mutationType,
		Subscription: subscriptionType,
	}

	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return graphql.Schema{}, fmt.Errorf("failed to create GraphQL schema: %w", err)
	}

	return schema, nil
}

// Resolver functions
func (sb *SchemaBuilder) resolveEvent(p graphql.ResolveParams) (any, error) {
	id, ok := p.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid id parameter")
	}

	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", id)
		if cached, err := sb.cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			sb.metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var result map[string]any
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
		return nil, fmt.Errorf("failed to resolve event")
	}

	if event == nil {
		return nil, nil
	}

	// Convert to GraphQL response format
	result := withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store")
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", id)
		resultBytes, _ := json.Marshal(result)
		if err := sb.cache.Set(p.Context, cacheKey, resultBytes, 300); err != nil {
			sb.logger.Error("cache set error", "key", cacheKey, "error", err)
		}
	}
	sb.metrics.RecordCounter("graphql_resolve_event_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveEvents(p graphql.ResolveParams) (any, error) {
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
			var connection map[string]any
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
		return nil, fmt.Errorf("failed to resolve events")
	}

	// Apply structured filter if provided
	if filterRaw, ok := p.Args["filter"]; ok && filterRaw != nil {
		events = applyEventFilter(events, filterRaw)
	}

	// Apply structured sort if provided
	if sortRaw, ok := p.Args["sort"]; ok && sortRaw != nil {
		events = applyEventSort(events, sortRaw)
	}

	// Get total count for pagination info (use cached count to avoid N+1)
	totalCount := int64(len(events))
	if sb.cache != nil {
		if cached, err := sb.cache.Get(p.Context, "graphql:events:count"); err == nil && cached != nil {
			if parsed, parseErr := strconv.ParseInt(string(cached), 10, 64); parseErr == nil {
				totalCount = parsed
			}
		}
	}
	if totalCount == int64(len(events)) {
		// Cache miss or stale — query the store and cache the result
		if count, err := sb.eventStore.CountEvents(p.Context); err == nil {
			totalCount = count
			if sb.cache != nil {
				if err := sb.cache.Set(p.Context, "graphql:events:count", []byte(strconv.FormatInt(count, 10)), 30); err != nil {
					sb.logger.Error("cache set error", "key", "graphql:events:count", "error", err)
				}
			}
		}
	}

	// Build edges with opaque cursors based on stable sort keys
	edges := make([]any, 0, len(events))
	var endCursor string
	for i, event := range events {
		cursor := domainquery.EncodePageCursor(event.BlockNumber, event.LogIndex, event.ID)
		edges = append(edges, map[string]any{
			"node":   withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"),
			"cursor": cursor,
		})
		if i == len(events)-1 {
			endCursor = cursor
		}
	}

	connection := map[string]any{
		"edges": edges,
		"pageInfo": map[string]any{
			"hasNextPage":     hasNextPage,
			"hasPreviousPage": after != "",
			"startCursor":     after,
			"endCursor":       endCursor,
			"totalCount":      totalCount,
		},
	}
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:root:after:%s:first:%d", after, first)
		connectionBytes, _ := json.Marshal(connection)
		if err := sb.cache.Set(p.Context, cacheKey, connectionBytes, 300); err != nil {
			sb.logger.Error("cache set error", "key", cacheKey, "error", err)
		}
	}

	sb.metrics.RecordCounter("graphql_resolve_events_success", 1, nil)
	return connection, nil
}

func (sb *SchemaBuilder) resolveEventsByBlock(p graphql.ResolveParams) (any, error) {
	blockNumber, ok := p.Args["blockNumber"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid blockNumber parameter")
	}

	// Retrieve events by block number
	events, err := sb.eventStore.GetEventsByBlock(p.Context, int64(blockNumber))
	if err != nil {
		sb.logger.Error("Failed to resolve events by block", "blockNumber", blockNumber, "error", err.Error())
		sb.metrics.RecordCounter("graphql_resolve_events_by_block_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by block")
	}

	// Convert to GraphQL response format
	result := make([]any, 0, len(events))
	for _, event := range events {
		result = append(result, withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"))
	}

	sb.metrics.RecordCounter("graphql_resolve_events_by_block_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveEventsByAddress(p graphql.ResolveParams) (any, error) {
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
			var result []any
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
		return nil, fmt.Errorf("failed to resolve events by address")
	}

	// Convert to GraphQL response format
	result := make([]any, 0, len(events))
	for _, event := range events {
		result = append(result, withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"))
	}
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:address:%s:limit:%d", address, limit)
		resultBytes, _ := json.Marshal(result)
		if err := sb.cache.Set(p.Context, cacheKey, resultBytes, 600); err != nil {
			sb.logger.Error("cache set error", "key", cacheKey, "error", err)
		}
	}

	sb.metrics.RecordCounter("graphql_resolve_events_by_address_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveEventsByName(p graphql.ResolveParams) (any, error) {
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
			var result []any
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
		return nil, fmt.Errorf("failed to resolve events by name")
	}

	// Convert to GraphQL response format
	result := make([]any, 0, len(events))
	for _, event := range events {
		result = append(result, withQuerySourcePosture(eventToGraphQLResponse(event), "graphql-event-store"))
	}
	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:name:%s:limit:%d", eventName, limit)
		resultBytes, _ := json.Marshal(result)
		if err := sb.cache.Set(p.Context, cacheKey, resultBytes, 600); err != nil {
			sb.logger.Error("cache set error", "key", cacheKey, "error", err)
		}
	}

	sb.metrics.RecordCounter("graphql_resolve_events_by_name_success", 1, nil)
	return result, nil
}

func (sb *SchemaBuilder) resolveInvalidateCache(p graphql.ResolveParams) (any, error) {
	if err := sb.requireMutationAuth(p); err != nil {
		return nil, fmt.Errorf("mutation auth failed: %w", err)
	}

	eventID, ok := p.Args["eventId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid eventId parameter")
	}

	if sb.cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", eventID)
		if err := sb.cache.Delete(p.Context, cacheKey); err != nil {
			sb.logger.Error("Failed to invalidate cache entry", "eventId", eventID, "error", err.Error())
		}
	}

	sb.logger.Info("Cache invalidation completed", "eventId", eventID)
	sb.metrics.RecordCounter("graphql_invalidate_cache_success", 1, nil)
	return true, nil
}

func (sb *SchemaBuilder) resolveClearCache(p graphql.ResolveParams) (any, error) {
	if err := sb.requireMutationAuth(p); err != nil {
		return nil, fmt.Errorf("mutation auth failed: %w", err)
	}

	if sb.cache != nil {
		// Delete known GraphQL cache key patterns
		patterns := []string{
			"graphql:event:",
			"graphql:events:root:",
			"graphql:events:address:",
			"graphql:events:name:",
			"graphql:events:block:",
		}
		for _, pattern := range patterns {
			if err := sb.cache.Delete(p.Context, pattern); err != nil {
				sb.logger.Error("Failed to clear cache pattern", "pattern", pattern, "error", err.Error())
			}
		}
	}

	sb.logger.Info("Cache clearing completed")
	sb.metrics.RecordCounter("graphql_clear_cache_success", 1, nil)
	return true, nil
}

// requireMutationAuth validates authentication for mutation operations.
// If authMiddleware is configured, requires a valid token with write scope.
// If authMiddleware is nil, mutations are allowed (development mode).
func (sb *SchemaBuilder) requireMutationAuth(p graphql.ResolveParams) error {
	if sb.authMiddleware == nil {
		return nil // No auth configured — development mode
	}

	// Extract token from context (set by HTTP handler from Authorization header)
	token, ok := p.Context.Value(authTokenContextKey).(string)
	if !ok || token == "" {
		sb.metrics.RecordCounter("graphql_mutation_auth_missing", 1, nil)
		return fmt.Errorf("authentication required for mutations")
	}

	authCtx, err := sb.authMiddleware.Authenticate(p.Context, token)
	if err != nil {
		sb.metrics.RecordCounter("graphql_mutation_auth_failed", 1, nil)
		return fmt.Errorf("authentication failed")
	}

	// Verify the user has write scope
	hasWriteScope := false
	for _, scope := range authCtx.Scopes {
		if scope == "write:cache" || scope == "admin" {
			hasWriteScope = true
			break
		}
	}
	if !hasWriteScope {
		sb.metrics.RecordCounter("graphql_mutation_auth_forbidden", 1, nil)
		return fmt.Errorf("insufficient permissions for mutation")
	}

	return nil
}

// Helper function to convert event to GraphQL response format
func eventToGraphQLResponse(event *blockchain.BlockchainEvent) map[string]any {
	decodedData := event.DecodedData
	if decodedData == nil {
		decodedData = map[string]any{}
	}

	return map[string]any{
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

// applyEventFilter applies structured filter criteria to a list of events.
func applyEventFilter(events []*blockchain.BlockchainEvent, filterRaw any) []*blockchain.BlockchainEvent {
	filter, ok := filterRaw.(map[string]any)
	if !ok {
		return events
	}

	var result []*blockchain.BlockchainEvent
	for _, event := range events {
		if matchesFilter(event, filter) {
			result = append(result, event)
		}
	}
	return result
}

// matchesFilter checks if a single event matches all provided filter criteria.
func matchesFilter(event *blockchain.BlockchainEvent, filter map[string]any) bool {
	if v, ok := filter["chainId"].(string); ok && v != "" {
		if event.ChainID != v {
			return false
		}
	}
	if v, ok := filter["contractAddress"].(string); ok && v != "" {
		if event.ContractAddress.Hex() != v {
			return false
		}
	}
	if v, ok := filter["eventName"].(string); ok && v != "" {
		if event.EventName != v {
			return false
		}
	}
	if v, ok := filter["status"].(string); ok && v != "" {
		if string(event.Status) != v {
			return false
		}
	}
	if v, ok := filter["blockNumberGte"].(int); ok {
		if event.BlockNumber < uint64(v) {
			return false
		}
	}
	if v, ok := filter["blockNumberLte"].(int); ok {
		if event.BlockNumber > uint64(v) {
			return false
		}
	}
	if v, ok := filter["removed"].(bool); ok {
		if event.Removed != v {
			return false
		}
	}
	return true
}

// applyEventSort sorts events based on the structured sort input.
func applyEventSort(events []*blockchain.BlockchainEvent, sortRaw any) []*blockchain.BlockchainEvent {
	sortInput, ok := sortRaw.(map[string]any)
	if !ok {
		return events
	}

	field, _ := sortInput["field"].(string)
	order, _ := sortInput["order"].(string)
	if field == "" {
		return events
	}

	ascending := strings.EqualFold(order, "ASC")

	sorted := make([]*blockchain.BlockchainEvent, len(events))
	copy(sorted, events)

	sort.Slice(sorted, func(i, j int) bool {
		var less bool
		switch field {
		case "blockNumber":
			less = sorted[i].BlockNumber < sorted[j].BlockNumber
		case "blockTimestamp":
			less = sorted[i].BlockTimestamp < sorted[j].BlockTimestamp
		case "logIndex":
			less = sorted[i].LogIndex < sorted[j].LogIndex
		case "eventName":
			less = sorted[i].EventName < sorted[j].EventName
		default:
			less = sorted[i].BlockNumber < sorted[j].BlockNumber
		}
		if ascending {
			return less
		}
		return !less
	})

	return sorted
}
