package query

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
)

// MongoDBEventStore implements EventStore for MongoDB
type MongoDBEventStore struct {
	dbManager   database.DatabaseManager
	logger      core.Logger
	metrics     core.MetricsCollector
	config      *EventStoreConfig
	collection  *mongo.Collection
	initialized bool
}

// NewMongoDBEventStore creates a new MongoDB event store
func NewMongoDBEventStore(
	dbManager database.DatabaseManager,
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
		s.logger.Error("Failed to get MongoDB client", "error", err.Error())
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
		s.logger.Error("Failed to create indexes", "error", err.Error())
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
	if !s.initialized {
		return fmt.Errorf("event store not initialized")
	}

	if s.collection == nil {
		return fmt.Errorf("MongoDB collection not initialized")
	}

	if event == nil {
		return fmt.Errorf("event is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("mongodb_event_insert_time_ms", float64(duration), map[string]string{})
	}()

	// Prepare document with TTL
	doc := bson.M{
		"id":              event.ID,
		"chainId":         event.ChainID,
		"blockNumber":     event.BlockNumber,
		"transactionHash": event.TransactionHash.Hex(),
		"logIndex":        event.LogIndex,
		"contractAddress": event.ContractAddress.Hex(),
		"eventName":       event.EventName,
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
		s.logger.Error("Failed to insert event", "id", event.ID, "error", err.Error())
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
	docs := make([]interface{}, len(events))
	for i, event := range events {
		if event == nil {
			continue
		}

		doc := bson.M{
			"id":              event.ID,
			"chainId":         event.ChainID,
			"blockNumber":     event.BlockNumber,
			"transactionHash": event.TransactionHash.Hex(),
			"logIndex":        event.LogIndex,
			"contractAddress": event.ContractAddress.Hex(),
			"eventName":       event.EventName,
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
		s.logger.Error("Failed to insert event batch", "count", len(events), "error", err.Error())
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

	filter := bson.M{"eventId": eventID}
	var result bson.M
	err := s.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		s.logger.Error("Failed to get event", "id", eventID, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_get_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Convert BSON to BlockchainEvent
	blockNumber, err := bsonNumericToUint64(result["blockNumber"])
	if err != nil {
		return nil, fmt.Errorf("invalid blockNumber: %w", err)
	}
	logIndex, err := bsonNumericToUint(result["logIndex"])
	if err != nil {
		return nil, fmt.Errorf("invalid logIndex: %w", err)
	}
	event := &core.BlockchainEvent{
		ID:              eventID,
		ChainID:         result["chainId"].(string),
		BlockNumber:     blockNumber,
		TransactionHash: common.HexToHash(result["transactionHash"].(string)),
		LogIndex:        logIndex,
		ContractAddress: common.HexToAddress(result["contractAddress"].(string)),
		EventName:       result["eventName"].(string),
		EventData:       result["eventData"].([]byte),
		DecodedData:     result["decodedData"].(map[string]interface{}),
		CreatedAt:       result["createdAt"].(time.Time),
	}

	s.metrics.RecordCounter("mongodb_event_get_success", 1, map[string]string{})
	return event, nil
}

func bsonNumericToUint64(value interface{}) (uint64, error) {
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
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func bsonNumericToUint(value interface{}) (uint, error) {
	number, err := bsonNumericToUint64(value)
	if err != nil {
		return 0, err
	}
	if number > uint64(math.MaxUint) {
		return 0, fmt.Errorf("value %d exceeds uint range", number)
	}
	return uint(number), nil
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

	filter := bson.M{"chainId": chainID}
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"blockNumber": -1})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to query events by chain", "chainId", chainID, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_chain_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by chain: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_chain_success", int64(len(events)), map[string]string{})
	return events, nil
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
		s.logger.Error("Failed to query events by contract", "contractAddress", contractAddress, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_contract_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by contract: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
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
		s.logger.Error("Failed to query events by name", "eventName", eventName, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_name_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by name: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_name_success", int64(len(events)), map[string]string{})
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
		s.logger.Error("Failed to delete expired events", "error", err.Error())
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
		s.logger.Error("Failed to query events by block", "blockNumber", blockNumber, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_block_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by block: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	s.metrics.RecordCounter("mongodb_event_query_block_success", int64(len(events)), map[string]string{})
	return events, nil
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
		s.logger.Error("Failed to query events by address", "address", address, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_address_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by address: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
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
		s.logger.Error("Failed to query events by name", "eventName", eventName, "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_eventname_error", 1, map[string]string{})
		return nil, fmt.Errorf("failed to query events by name: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
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
		// Cursor is a block number - get events after this block
		filter = bson.M{
			"blockNumber": bson.M{
				"$lt": cursor,
			},
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
		s.logger.Error("Failed to query paginated events", "error", err.Error())
		s.metrics.RecordCounter("mongodb_event_query_paginated_error", 1, map[string]string{})
		return nil, false, fmt.Errorf("failed to query paginated events: %w", err)
	}
	defer func() { _ = cursorResult.Close(ctx) }()

	var events []*core.BlockchainEvent
	if err := cursorResult.All(ctx, &events); err != nil {
		s.logger.Error("Failed to decode events", "error", err.Error())
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
