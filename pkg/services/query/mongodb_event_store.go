package query

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/eventsig"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/observability"
)

// MongoDBEventStore implements EventStore for MongoDB
type MongoDBEventStore struct {
	dbManager   mongoClientProvider
	logger      core.Logger
	metrics     core.MetricsCollector
	config      *EventStoreConfig
	collection  *mongo.Collection
	initialized bool
	tracer      *observability.DefaultTracer
}

// NewMongoDBEventStore creates a new MongoDB event store
func NewMongoDBEventStore(
	dbManager mongoClientProvider,
	logger core.Logger,
	metrics core.MetricsCollector,
	config *EventStoreConfig,
) *MongoDBEventStore {
	if config == nil {
		config = DefaultEventStoreConfig()
	}
	return &MongoDBEventStore{
		dbManager:   dbManager,
		logger:      logger,
		metrics:     metrics,
		config:      config,
		tracer:      observability.NewDefaultTracer(logger, metrics),
		initialized: false,
	}
}

// Initialize initializes the MongoDB event store
func (s *MongoDBEventStore) Initialize(ctx context.Context) error {
	if s.initialized {
		return nil
	}

	// Get MongoDB client
	clientInterface, err := s.dbManager.GetMongoClient(ctx)
	if err != nil {
		s.logger.Error("Failed to get MongoDB client", "error", err)
		return fmt.Errorf("failed to get MongoDB client: %w", err)
	}

	// Type assert to *mongo.Client
	client, ok := clientInterface.(*mongo.Client)
	if !ok {
		return fmt.Errorf("failed to assert MongoDB client type")
	}

	// Get database
	db := client.Database("chainpulse")
	s.collection = db.Collection(s.config.CollectionName)

	// Create indexes
	if err := s.createIndexes(ctx); err != nil {
		s.logger.Error("Failed to create indexes", "error", err)
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	s.initialized = true
	s.logger.Info("MongoDB event store initialized", "collection", s.config.CollectionName)
	return nil
}

// createIndexes creates necessary indexes for the events collection
func (s *MongoDBEventStore) createIndexes(ctx context.Context) error {
	if s.collection == nil {
		return fmt.Errorf("MongoDB collection not initialized")
	}

	indexModel := []mongo.IndexModel{
		{
			Keys: bson.D{
				bson.E{Key: "chainId", Value: 1},
				bson.E{Key: "blockNumber", Value: 1},
			},
		},
		{
			Keys: bson.D{
				bson.E{Key: "contractAddress", Value: 1},
				bson.E{Key: "eventName", Value: 1},
			},
		},
		{
			Keys: bson.D{
				bson.E{Key: "timestamp", Value: 1},
			},
		},
		{
			Keys: bson.D{
				bson.E{Key: "id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	// Add TTL index if TTL is configured
	if s.config.TTLDays > 0 {
		indexModel = append(indexModel, mongo.IndexModel{
			Keys: bson.D{
				bson.E{Key: "expiresAt", Value: 1},
			},
		})
	}

	opts := options.CreateIndexes().SetMaxTime(s.config.IndexTimeout)
	_, err := s.collection.Indexes().CreateMany(ctx, indexModel, opts)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// InsertEvent inserts a single event into the store
func (s *MongoDBEventStore) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	ctx, span := s.tracer.StartSpan(ctx, "storage.insert_event", observability.SpanKindInternal)
	defer s.tracer.EndSpan(&span)

	if !s.initialized {
		return fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return fmt.Errorf("MongoDB collection not initialized")
	}

	if event == nil {
		return fmt.Errorf("event is required")
	}

	s.tracer.SetAttribute(&span, "event_id", event.ID)
	s.tracer.SetAttribute(&span, "chain_id", event.ChainID)

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_insert_time_ms", float64(duration), map[string]string{})
	}()

	// Prepare document with TTL
	doc := bson.M{
		"id":              event.ID,
		"eventHash":       event.EventHash,
		"chainId":         normalizeChainIDForStorage(event.ChainID),
		"network":         event.Network,
		"blockNumber":     int64(event.BlockNumber),
		"blockTimestamp":  event.BlockTimestamp,
		"transactionHash": event.TransactionHash.Hex(),
		"logIndex":        int64(event.LogIndex),
		"contractAddress": event.ContractAddress.Hex(),
		"eventName":       event.EventName,
		"eventSignature":  event.EventSignature.Hex(),
		"eventData":       event.EventData,
		"decodedData":     event.DecodedData,
		"createdAt":       event.CreatedAt,
		"processedAt":     time.Now(),
		"indexedAt":       time.Now(),
	}

	// Add TTL expiration if configured
	if s.config.TTLDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, s.config.TTLDays)
		doc["expiresAt"] = expiresAt
	}

	_, err := s.collection.InsertOne(ctx, doc)
	if err != nil {
		s.logger.Error("Failed to insert event", "id", event.ID, "error", err)
		s.metrics.RecordCounter("mongodb_event_insert_error", 1, map[string]string{})
		return fmt.Errorf("failed to insert event: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_insert_success", 1, map[string]string{})
	return nil
}

// InsertEventBatch inserts multiple events in a batch operation
func (s *MongoDBEventStore) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	if !s.initialized {
		return fmt.Errorf("event store not initialized")
	}

	if len(events) == 0 {
		return nil
	}

	if s.collection == nil {
		return fmt.Errorf("MongoDB collection not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_batch_insert_time_ms", float64(duration), map[string]string{})
	}()

	// Prepare documents
	docs := make([]any, len(events))
	for i, event := range events {
		if event == nil {
			continue
		}

		doc := bson.M{
			"id":              event.ID,
			"eventHash":       event.EventHash,
			"chainId":         normalizeChainIDForStorage(event.ChainID),
			"network":         event.Network,
			"blockNumber":     int64(event.BlockNumber),
			"blockTimestamp":  event.BlockTimestamp,
			"transactionHash": event.TransactionHash.Hex(),
			"logIndex":        int64(event.LogIndex),
			"contractAddress": event.ContractAddress.Hex(),
			"eventName":       event.EventName,
			"eventSignature":  event.EventSignature.Hex(),
			"eventData":       event.EventData,
			"decodedData":     event.DecodedData,
			"createdAt":       event.CreatedAt,
			"processedAt":     time.Now(),
			"indexedAt":       time.Now(),
		}

		// Add TTL expiration if configured
		if s.config.TTLDays > 0 {
			expiresAt := time.Now().AddDate(0, 0, s.config.TTLDays)
			doc["expiresAt"] = expiresAt
		}

		docs[i] = doc
	}

	// Insert batch
	opts := options.InsertMany().SetOrdered(false)
	result, err := s.collection.InsertMany(ctx, docs, opts)
	if err != nil {
		s.logger.Error("Failed to insert event batch", "count", len(events), "error", err)
		s.metrics.RecordCounter("mongodb_event_batch_insert_error", 1, map[string]string{})
		return fmt.Errorf("failed to insert event batch: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_batch_insert_success", int64(len(result.InsertedIDs)), map[string]string{})
	return nil
}

// GetEvent retrieves a single event by ID
func (s *MongoDBEventStore) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	if eventID == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_get_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"id": eventID}
	var result bson.M
	err := s.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		s.logger.Error("Failed to get event", "id", eventID, "error", err)
		s.metrics.RecordCounter("mongodb_event_get_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Convert BSON to BlockchainEvent
	blockNumber, err := bsonNumericToUint64(result["blockNumber"])
	if err != nil {
		return nil, fmt.Errorf("invalid blockNumber: %w", err)
	}
	logIndex, err := bsonNumericToUint64(result["logIndex"])
	if err != nil {
		return nil, fmt.Errorf("invalid logIndex: %w", err)
	}
	event := decodeMongoEventDocument(result)
	event.ID = eventID
	event.BlockNumber = blockNumber
	event.LogIndex = logIndex

	s.metrics.RecordCounter("mongodb_event_get_success", 1, map[string]string{})
	return event, nil
}

func bsonNumericToUint64(value any) (uint64, error) {
	switch typed := value.(type) {
	case int32:
		if typed < 0 {
			return 0, fmt.Errorf("negative value %d", typed)
		}
		return uint64(typed), nil
	case int64:
		if typed < 0 {
			return 0, fmt.Errorf("negative value %d", typed)
		}
		return uint64(typed), nil
	case uint32:
		return uint64(typed), nil
	case uint64:
		return typed, nil
	case float64:
		if typed < 0 {
			return 0, fmt.Errorf("negative value %f", typed)
		}
		if typed > 1<<53 {
			return 0, fmt.Errorf("float64 precision loss: value %f exceeds 2^53", typed)
		}
		return uint64(typed), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

// GetEventsByChain retrieves events for a specific chain
func (s *MongoDBEventStore) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_chain_time_ms", float64(duration), map[string]string{})
	}()

	filter := buildChainLookupFilter(chainID)
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"blockNumber": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by chain", "chainId", chainID, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_chain_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by chain: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_chain_success", int64(len(events)), map[string]string{})
	return events, nil
}

func normalizeChainIDForStorage(chainID string) any {
	trimmed := strings.TrimSpace(chainID)
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func buildChainLookupFilter(chainID int) bson.M {
	if chainID == 0 {
		return bson.M{}
	}

	// Build string-only $in filter — chainId is always stored as string
	// after the ChainID string migration.
	values := []any{strconv.Itoa(chainID)}
	if name := chainid.ResolveChainName(chainID); name != strconv.Itoa(chainID) {
		values = append(values, name)
	}
	return bson.M{
		"chainId": bson.M{
			"$in": values,
		},
	}
}

// GetEventsByContract retrieves events for a specific contract
func (s *MongoDBEventStore) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_contract_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"contractAddress": contractAddress}
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"blockNumber": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by contract", "contractAddress", contractAddress, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_contract_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by contract: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_contract_success", int64(len(events)), map[string]string{})
	return events, nil
}

// GetEventsByEventName retrieves events by event name
func (s *MongoDBEventStore) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	if eventName == "" {
		return nil, fmt.Errorf("event name is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_name_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"eventName": eventName}
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"timestamp": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by name", "eventName", eventName, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_name_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by name: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_name_success", int64(len(events)), map[string]string{})
	return events, nil
}

// GetEventsByCorrelationID retrieves events across all chains that share a correlation ID.
func (s *MongoDBEventStore) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}
	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}
	if correlationID == "" {
		return nil, fmt.Errorf("correlation ID is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_correlation_id_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"correlationid": correlationID}
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"timestamp": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by correlation ID", "correlationId", correlationID, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_correlation_id_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by correlation ID: %w", err)
	}
	defer cursor.Close(ctx)

	events := make([]*core.BlockchainEvent, 0, limit)
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode correlated events", "error", err)
		return nil, fmt.Errorf("failed to decode correlated events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_correlation_id_success", int64(len(events)), map[string]string{})
	return events, nil
}

// DeleteExpiredEvents deletes events that have exceeded their TTL
func (s *MongoDBEventStore) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	if !s.initialized {
		return 0, fmt.Errorf("event store not initialized")
	}

	if s.config.TTLDays == 0 {
		return 0, nil // TTL not configured
	}

	if s.collection == nil {
		return 0, fmt.Errorf("MongoDB collection not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_delete_expired_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{
		"expiresAt": bson.M{
			"$lt": time.Now(),
		},
	}

	result, err := s.collection.DeleteMany(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to delete expired events", "error", err)
		s.metrics.RecordCounter("mongodb_event_delete_expired_error", 1, map[string]string{})
		return 0, fmt.Errorf("failed to delete expired events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_delete_expired_success", result.DeletedCount, map[string]string{})
	return result.DeletedCount, nil
}

// Health returns the health status of the event store
func (s *MongoDBEventStore) Health(ctx context.Context) *core.HealthStatus {
	if !s.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "event store not initialized",
		}
	}

	// Check if collection is nil
	if s.collection == nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "MongoDB collection not initialized",
		}
	}

	// Try to ping the collection
	err := s.collection.Database().Client().Ping(ctx, nil)
	if err != nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("MongoDB ping failed: %v", err),
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "event store is healthy",
	}
}

// Close closes the event store
func (s *MongoDBEventStore) Close(ctx context.Context) error {
	if !s.initialized {
		return nil
	}

	s.initialized = false
	return nil
}

// GetEventsByBlock retrieves events by block number
func (s *MongoDBEventStore) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_block_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"blockNumber": blockNumber}
	opts := options.Find().SetSort(bson.M{"logIndex": 1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by block", "blockNumber", blockNumber, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_block_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by block: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_block_success", int64(len(events)), map[string]string{})
	return events, nil
}

// GetEventsByBlockRange retrieves events from fromBlock to toBlock (inclusive)
func (s *MongoDBEventStore) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	filter := bson.M{
		"blockNumber": bson.M{
			"$gte": fromBlock,
			"$lte": toBlock,
		},
	}
	opts := options.Find().SetSort(bson.M{"blockNumber": 1, "logIndex": 1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by block range: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	return decodeMongoEventCursor(ctx, cursor)
}

// DeleteEventsByBlockRange deletes events from fromBlock to toBlock (inclusive)
func (s *MongoDBEventStore) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	if !s.initialized {
		return 0, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return 0, fmt.Errorf("MongoDB collection not initialized")
	}

	filter := bson.M{
		"blockNumber": bson.M{
			"$gte": fromBlock,
			"$lte": toBlock,
		},
	}
	result, err := s.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete events by block range: %w", err)
	}

	s.logger.Info("Deleted events by block range", core.LogKeyFromBlock, fromBlock, core.LogKeyToBlock, toBlock, core.LogKeyCount, result.DeletedCount)

	return result.DeletedCount, nil
}

// MarkEventsAsReorged marks events in the given block range as reorged (soft-delete)
// instead of permanently deleting them. Returns the count of affected events.
func (s *MongoDBEventStore) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	if !s.initialized {
		return 0, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return 0, fmt.Errorf("MongoDB collection not initialized")
	}

	filter := bson.M{
		"blockNumber": bson.M{
			"$gte": fromBlock,
			"$lte": toBlock,
		},
	}
	update := bson.M{
		"$set": bson.M{
			"status":    "reorged",
			"reorgedAt": time.Now(),
		},
	}
	result, err := s.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to mark events as reorged: %w", err)
	}
	return result.ModifiedCount, nil
}

// GetEventsByAddress retrieves events by contract address with limit
func (s *MongoDBEventStore) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_address_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"contractAddress": address}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.M{"blockNumber": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by address", "address", address, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_address_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by address: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_address_success", int64(len(events)), map[string]string{})
	return events, nil
}

// GetEventsByName retrieves events by event name with limit
func (s *MongoDBEventStore) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, fmt.Errorf("MongoDB collection not initialized")
	}

	if eventName == "" {
		return nil, fmt.Errorf("event name is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_eventname_time_ms", float64(duration), map[string]string{})
	}()

	filter := bson.M{"eventName": eventName}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.M{"blockNumber": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by name", "eventName", eventName, "error", err)
		s.metrics.RecordCounter("mongodb_event_query_eventname_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by name: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_eventname_success", int64(len(events)), map[string]string{})
	return events, nil
}

// GetEventsPaginated retrieves events with cursor-based pagination
func (s *MongoDBEventStore) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	if !s.initialized {
		return nil, false, fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return nil, false, fmt.Errorf("MongoDB collection not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_query_paginated_time_ms", float64(duration), map[string]string{})
	}()

	// Build filter based on cursor
	filter := bson.M{}
	if cursor != "" {
		// Try decoding as opaque PageCursor first
		if pc, ok := domainquery.DecodePageCursor(cursor); ok {
			filter = bson.M{
				"$or": []bson.M{
					{"blockNumber": bson.M{"$lt": pc.BlockNumber}},
					{"blockNumber": pc.BlockNumber, "logIndex": bson.M{"$lt": pc.LogIndex}},
				},
			}
		} else {
			// Legacy cursor: treat as block number string
			filter = bson.M{
				"blockNumber": bson.M{
					"$lt": cursor,
				},
			}
		}
	}

	// Query with limit + 1 to determine if there are more results
	opts := options.Find().
		SetLimit(int64(limit + 1)).
		SetSort(bson.M{"blockNumber": -1})

	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cursorResult, err := s.collection.Find(queryCtx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query paginated events", "error", err)
		s.metrics.RecordCounter("mongodb_event_query_paginated_error", 1, map[string]string{})
		return nil, false, fmt.Errorf("failed to query paginated events: %w", err)
	}
	defer func() { _ = cursorResult.Close(ctx) }()

	events, err := decodeMongoEventCursor(ctx, cursorResult)
	if err != nil {
		s.logger.Error("Failed to decode events", "error", err)
		return nil, false, fmt.Errorf("failed to decode events: %w", err)
	}

	// Check if there are more results
	hasNextPage := len(events) > limit
	if hasNextPage {
		events = events[:limit]
	}

	s.metrics.RecordCounter("mongodb_event_query_paginated_success", int64(len(events)), map[string]string{})
	return events, hasNextPage, nil
}

// CountEvents returns the total number of events in the store
func (s *MongoDBEventStore) CountEvents(ctx context.Context) (int64, error) {
	if !s.initialized {
		return 0, fmt.Errorf("event store not initialized")
	}
	if s.collection == nil {
		return 0, fmt.Errorf("MongoDB collection not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_count_time_ms", float64(duration), map[string]string{})
	}()

	count, err := s.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		s.logger.Error("Failed to count events", "error", err)
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

func (s *MongoDBEventStore) GetEventStats(ctx context.Context) (map[string]int64, map[string]int64, int64, error) {
	if !s.initialized || s.collection == nil {
		return nil, nil, 0, fmt.Errorf("event store not initialized")
	}

	byChain := make(map[string]int64)
	byEventName := make(map[string]int64)
	var reorged int64

	// Aggregate by chainId
	chainPipeline := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$chainId"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	chainCursor, err := s.collection.Aggregate(ctx, chainPipeline)
	if err == nil {
		var chainResults []struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := chainCursor.All(ctx, &chainResults); err == nil {
			for _, r := range chainResults {
				if r.ID == "" {
					continue
				}
				// Resolve numeric chain IDs to names
				if name := chainid.ResolveChainName(func() int {
					if id := chainid.ResolveChainID(r.ID); id != 0 {
						return id
					}
					return 0
				}()); name != "" && name != r.ID {
					byChain[name] = r.Count
				} else {
					byChain[r.ID] = r.Count
				}
			}
		}
		chainCursor.Close(ctx)
	}

	// Aggregate by eventName
	eventPipeline := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$eventName"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	eventCursor, err := s.collection.Aggregate(ctx, eventPipeline)
	if err == nil {
		var eventResults []struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := eventCursor.All(ctx, &eventResults); err == nil {
			for _, r := range eventResults {
				name := r.ID
				// Resolve hex topic0 to human-readable name
				if strings.HasPrefix(name, "0x") {
					if resolved := eventsig.ResolveEventNameFromTopic(name); resolved != name {
						name = resolved
					}
				}
				byEventName[name] += r.Count
			}
		}
		eventCursor.Close(ctx)
	}

	// Count reorged events
	reorgedCount, _ := s.collection.CountDocuments(ctx, bson.M{"status": "reorged"})
	if reorgedCount > 0 {
		reorged = reorgedCount
	}

	return byChain, byEventName, reorged, nil
}

func decodeMongoEventCursor(ctx context.Context, cursor *mongo.Cursor) ([]*core.BlockchainEvent, error) {
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode mongo cursor: %w", err)
	}

	events := make([]*core.BlockchainEvent, 0, len(docs))
	for _, doc := range docs {
		events = append(events, decodeMongoEventDocument(doc))
	}
	return events, nil
}

func decodeMongoEventDocument(doc bson.M) *core.BlockchainEvent {
	eventSig := common.Hash{}
	if sigStr := bsonString(doc["eventSignature"]); sigStr != "" {
		eventSig = common.HexToHash(sigStr)
	}
	return &core.BlockchainEvent{
		ID:              bsonString(doc["id"]),
		EventHash:       bsonString(doc["eventHash"]),
		ChainID:         bsonString(doc["chainId"]),
		Network:         bsonString(doc["network"]),
		BlockNumber:     bsonUint64(doc["blockNumber"]),
		BlockTimestamp:  bsonInt64(doc["blockTimestamp"]),
		TransactionHash: common.HexToHash(bsonString(doc["transactionHash"])),
		LogIndex:        uint64(bsonInt64(doc["logIndex"])),
		ContractAddress: common.HexToAddress(bsonString(doc["contractAddress"])),
		EventName:       bsonString(doc["eventName"]),
		EventSignature:  eventSig,
		EventData:       bsonBytes(doc["eventData"]),
		DecodedData:     bsonMap(doc["decodedData"]),
		CreatedAt:       bsonTime(doc["createdAt"]),
		ProcessedAt:     bsonTime(doc["processedAt"]),
		IndexedAt:       bsonTime(doc["indexedAt"]),
	}
}

func bsonString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func bsonUint64(value any) uint64 {
	switch typed := value.(type) {
	case int32:
		return uint64(typed)
	case int64:
		return uint64(typed)
	case uint32:
		return uint64(typed)
	case uint64:
		return typed
	case float64:
		if typed < 0 {
			return 0
		}
		if typed > 1<<53 {
			return 0
		}
		return uint64(typed)
	case primitive.DateTime:
		return uint64(typed.Time().Unix())
	default:
		return 0
	}
}

func bsonInt64(value any) int64 {
	switch typed := value.(type) {
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	case float64:
		return int64(typed)
	case primitive.DateTime:
		return typed.Time().Unix()
	default:
		return 0
	}
}

func bsonBytes(value any) []byte {
	switch typed := value.(type) {
	case []byte:
		return typed
	case primitive.Binary:
		return typed.Data
	default:
		return nil
	}
}

func bsonMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case bson.M:
		return map[string]any(typed)
	default:
		return nil
	}
}

func bsonTime(value any) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case primitive.DateTime:
		return typed.Time()
	default:
		return time.Time{}
	}
}
