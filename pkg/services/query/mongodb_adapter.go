package query

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoClientProvider interface {
	GetMongoClient(ctx context.Context) (any, error)
}

// DefaultMongoDBAdapter provides MongoDB query operations
type DefaultMongoDBAdapter struct {
	initMu           sync.Mutex
	initialized      atomic.Bool
	dbManager        mongoClientProvider
	mongoClient      *mongo.Client
	logger           core.Logger
	metricsCollector core.MetricsCollector
}

// NewMongoDBAdapter creates a new MongoDB adapter
func NewMongoDBAdapter(
	dbManager mongoClientProvider,
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
	ma.initMu.Lock()
	defer ma.initMu.Unlock()

	if ma.initialized.Load() {
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
	ma.initialized.Store(true)

	ma.logger.Info("MongoDB adapter initialized", core.LogKeyComponent, "mongodb-adapter")

	return nil
}

// Query executes a query against MongoDB
func (ma *DefaultMongoDBAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	if !ma.initialized.Load() {
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
		// Reject dangerous MongoDB operators that enable NoSQL injection
		dangerousOps := map[string]bool{
			"$where": true, "$accumulator": true, "$function": true,
		}
		for k, v := range req.Filter {
			if dangerousOps[k] {
				return nil, fmt.Errorf("rejected dangerous filter operator %q", k)
			}
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
		ma.logger.Error("MongoDB query failed", "collection", req.Collection, core.LogKeyError, err, core.LogKeyDuration, duration)
		return nil, fmt.Errorf("MongoDB query failed: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	// Decode results using manual bson.M decoding (same approach as MongoDBEventStore)
	// This is required because BlockchainEvent has no bson tags and contains types like
	// common.Hash ([32]byte) that can't directly decode from BSON hex strings.
	decodedEvents, err := decodeMongoEventCursor(ctx, cursor)
	if err != nil {
		duration := time.Since(start).Milliseconds()
		ma.metricsCollector.RecordCounter("mongodb_decode_error", 1, map[string]string{})
		ma.logger.Error("Failed to decode MongoDB results", "collection", req.Collection, core.LogKeyError, err, core.LogKeyDuration, duration)
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	events := make([]core.BlockchainEvent, len(decodedEvents))
	for i, e := range decodedEvents {
		if e != nil {
			events[i] = *e
		}
	}

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		ma.logger.Error("Failed to count MongoDB documents", "collection", req.Collection, core.LogKeyError, err)
		total = int64(len(events))
	}

	duration := time.Since(start).Milliseconds()
	ma.metricsCollector.RecordHistogram("mongodb_query_time_ms", float64(duration), map[string]string{})
	ma.metricsCollector.RecordCounter("mongodb_query_success", 1, map[string]string{})

	ma.logger.Info("MongoDB query successful", "collection", req.Collection, core.LogKeyCount, len(events), "total", total, core.LogKeyDuration, duration)

	return &QueryResult{
		Events:       events,
		Total:        total,
		ResponseTime: duration,
		Source:       "mongodb",
	}, nil
}

// QueryByHash retrieves a single item by hash
func (ma *DefaultMongoDBAdapter) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	if !ma.initialized.Load() {
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

	// Execute query — decode to bson.M first, then convert manually (same reason as Query)
	var result bson.M
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			duration := time.Since(start).Milliseconds()
			ma.logger.Debug("Event not found in MongoDB", core.LogKeyHash, hash, core.LogKeyDuration, duration)
			return nil, nil
		}

		duration := time.Since(start).Milliseconds()
		ma.metricsCollector.RecordCounter("mongodb_query_by_hash_error", 1, map[string]string{})
		ma.logger.Error("MongoDB query by hash failed", core.LogKeyHash, hash, core.LogKeyError, err, core.LogKeyDuration, duration)
		return nil, fmt.Errorf("MongoDB query failed: %w", err)
	}

	event := decodeMongoEventDocument(result)

	duration := time.Since(start).Milliseconds()
	ma.metricsCollector.RecordHistogram("mongodb_query_by_hash_time_ms", float64(duration), map[string]string{})
	ma.metricsCollector.RecordCounter("mongodb_query_by_hash_success", 1, map[string]string{})

	ma.logger.Info("MongoDB query by hash successful", core.LogKeyHash, hash, core.LogKeyDuration, duration)

	return event, nil
}

// Health returns the health status
func (ma *DefaultMongoDBAdapter) Health(ctx context.Context) *core.HealthStatus {
	if !ma.initialized.Load() {
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
