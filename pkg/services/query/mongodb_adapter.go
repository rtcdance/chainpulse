package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DefaultMongoDBAdapter provides MongoDB query operations
type DefaultMongoDBAdapter struct {
	mu              sync.RWMutex
	dbManager       database.DatabaseManager
	mongoClient     *mongo.Client
	logger          core.Logger
	metricsCollector core.MetricsCollector
	initialized     bool
}

// NewMongoDBAdapter creates a new MongoDB adapter
func NewMongoDBAdapter(
	dbManager database.DatabaseManager,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) MongoDBAdapter {
	return &DefaultMongoDBAdapter{
		dbManager:        dbManager,
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the MongoDB adapter
func (ma *DefaultMongoDBAdapter) Initialize(ctx context.Context) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if ma.initialized {
		return fmt.Errorf("MongoDB adapter already initialized")
	}

	// Get MongoDB client from database manager
	clientInterface, err := ma.dbManager.GetMongoClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get MongoDB client: %w", err)
	}

	// Type assert to *mongo.Client
	client, ok := clientInterface.(*mongo.Client)
	if !ok {
		return fmt.Errorf("failed to assert MongoDB client type")
	}

	ma.mongoClient = client
	ma.initialized = true

	ma.logger.Info("MongoDB adapter initialized", map[string]interface{}{
		"component": "mongodb-adapter",
	})

	return nil
}

// Query executes a query against MongoDB
func (ma *DefaultMongoDBAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if !ma.initialized {
		return nil, fmt.Errorf("MongoDB adapter not initialized")
	}

	if req == nil {
		return nil, fmt.Errorf("query request is required")
	}

	if req.Collection == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	start := time.Now()

	// Get collection
	collection := ma.mongoClient.Database("chainpulse").Collection(req.Collection)

	// Build filter
	filter := bson.M{}
	if req.Filter != nil {
		for k, v := range req.Filter {
			filter[k] = v
		}
	}

	// Build options
	opts := options.Find()

	// Set limit
	if req.Limit > 0 {
		opts.SetLimit(req.Limit)
	}

	// Set skip (offset)
	if req.Offset > 0 {
		opts.SetSkip(req.Offset)
	}

	// Set sort
	if req.Sort != nil {
		sortBson := bson.M{}
		for k, v := range req.Sort {
			sortBson[k] = v
		}
		opts.SetSort(sortBson)
	}

	// Execute query
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		duration := time.Since(start).Milliseconds()
		ma.metricsCollector.RecordCounter("mongodb_query_error", 1, map[string]string{})
		ma.logger.Error("MongoDB query failed", map[string]interface{}{
			"collection": req.Collection,
			"error":      err.Error(),
			"duration":   duration,
		})
		return nil, fmt.Errorf("MongoDB query failed: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	// Decode results
	var events []core.BlockchainEvent
	if err := cursor.All(ctx, &events); err != nil {
		duration := time.Since(start).Milliseconds()
		ma.metricsCollector.RecordCounter("mongodb_decode_error", 1, map[string]string{})
		ma.logger.Error("Failed to decode MongoDB results", map[string]interface{}{
			"collection": req.Collection,
			"error":      err.Error(),
			"duration":   duration,
		})
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		ma.logger.Error("Failed to count MongoDB documents", map[string]interface{}{
			"collection": req.Collection,
			"error":      err.Error(),
		})
		total = int64(len(events))
	}

	duration := time.Since(start).Milliseconds()
	ma.metricsCollector.RecordHistogram("mongodb_query_time_ms", float64(duration), map[string]string{})
	ma.metricsCollector.RecordCounter("mongodb_query_success", 1, map[string]string{})

	ma.logger.Info("MongoDB query successful", map[string]interface{}{
		"collection": req.Collection,
		"count":      len(events),
		"total":      total,
		"duration":   duration,
	})

	return &QueryResult{
		Events:       events,
		Total:        total,
		ResponseTime: duration,
		Source:       "mongodb",
	}, nil
}

// QueryByHash retrieves a single item by hash
func (ma *DefaultMongoDBAdapter) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if !ma.initialized {
		return nil, fmt.Errorf("MongoDB adapter not initialized")
	}

	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	// Get collection
	collection := ma.mongoClient.Database("chainpulse").Collection("events")

	// Build filter
	filter := bson.M{"hash": hash}

	// Execute query
	var event core.BlockchainEvent
	err := collection.FindOne(ctx, filter).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			duration := time.Since(start).Milliseconds()
			ma.logger.Debug("Event not found in MongoDB", map[string]interface{}{
				"hash":     hash,
				"duration": duration,
			})
			return nil, nil
		}

		duration := time.Since(start).Milliseconds()
		ma.metricsCollector.RecordCounter("mongodb_query_by_hash_error", 1, map[string]string{})
		ma.logger.Error("MongoDB query by hash failed", map[string]interface{}{
			"hash":     hash,
			"error":    err.Error(),
			"duration": duration,
		})
		return nil, fmt.Errorf("MongoDB query failed: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	ma.metricsCollector.RecordHistogram("mongodb_query_by_hash_time_ms", float64(duration), map[string]string{})
	ma.metricsCollector.RecordCounter("mongodb_query_by_hash_success", 1, map[string]string{})

	ma.logger.Info("MongoDB query by hash successful", map[string]interface{}{
		"hash":     hash,
		"duration": duration,
	})

	return &event, nil
}

// Health returns the health status
func (ma *DefaultMongoDBAdapter) Health(ctx context.Context) *core.HealthStatus {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if !ma.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "MongoDB adapter not initialized",
		}
	}

	// Ping MongoDB
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := ma.mongoClient.Ping(pingCtx, nil); err != nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("MongoDB ping failed: %v", err),
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "MongoDB adapter healthy",
	}
}
