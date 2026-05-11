package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"chainpulse/pkg/core"
)

// MongoDBDatabase implements DatabasePlugin for MongoDB
type MongoDBDatabase struct {
	*BaseDatabasePlugin
	connectionString string
	databaseName     string
	collectionName   string
	maxConnections   int
	queryTimeout     time.Duration
	mu               sync.RWMutex
	client           *mongo.Client // MongoDB client for real connection management
	events           map[string]*core.BlockchainEvent // in-memory cache for testing
	eventsMu         sync.RWMutex
}

// NewMongoDBDatabase creates a new MongoDB database plugin
func NewMongoDBDatabase(logger core.Logger, metricsCollector core.MetricsCollector) *MongoDBDatabase {
	return &MongoDBDatabase{
		BaseDatabasePlugin: NewBaseDatabasePlugin(logger, metricsCollector),
		databaseName:       "chainpulse",
		collectionName:     "events",
		maxConnections:     100,
		queryTimeout:       30 * time.Second,
		events:             make(map[string]*core.BlockchainEvent),
	}
}

// Initialize initializes the MongoDB database plugin
func (m *MongoDBDatabase) Initialize(config *core.Config) error {
	if err := m.BaseDatabasePlugin.Initialize(config); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract MongoDB connection string from config
	connStr := config.GetString("MONGODB_CONNECTION_STRING", "")
	if connStr == "" {
		// Build connection string from individual components
		host := config.GetString("MONGODB_HOST", "localhost")
		port := config.GetString("MONGODB_PORT", "27017")
		user := config.GetString("MONGODB_USER", "")
		password := config.GetString("MONGODB_PASSWORD", "")

		if user != "" && password != "" {
			connStr = fmt.Sprintf("mongodb://%s:%s@%s:%s", user, password, host, port)
		} else {
			connStr = fmt.Sprintf("mongodb://%s:%s", host, port)
		}
	}

	m.connectionString = connStr

	// Extract database and collection names
	m.databaseName = config.GetString("MONGODB_DATABASE", "chainpulse")
	m.collectionName = config.GetString("MONGODB_COLLECTION", "events")

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), m.queryTimeout)
	defer cancel()

	clientOpts := options.Client().ApplyURI(m.connectionString).
		SetMaxPoolSize(uint64(m.maxConnections))

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		m.RecordError()
		m.logger.Error("Failed to connect to MongoDB", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		m.RecordError()
		m.logger.Error("Failed to ping MongoDB", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client

	m.logger.Info("MongoDB database initialized", map[string]interface{}{
		"component":  "mongodb_database",
		"database":   m.databaseName,
		"collection": m.collectionName,
	})

	return nil
}

// Start starts the MongoDB database plugin
func (m *MongoDBDatabase) Start() error {
	if err := m.BaseDatabasePlugin.Start(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// In a real implementation, we would establish connection pool here
	m.logger.Info("MongoDB database started", map[string]interface{}{
		"component": "mongodb_database",
	})

	return nil
}

// Stop stops the MongoDB database plugin
func (m *MongoDBDatabase) Stop() error {
	if err := m.BaseDatabasePlugin.Stop(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect MongoDB client
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.client.Disconnect(ctx); err != nil {
			m.logger.Error("Failed to disconnect MongoDB client", map[string]interface{}{"error": err.Error()})
		}
		m.client = nil
	}

	m.logger.Info("MongoDB database stopped", map[string]interface{}{
		"component": "mongodb_database",
	})

	return nil
}

// WriteEvent writes a blockchain event to the database
func (m *MongoDBDatabase) WriteEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}

	start := time.Now()

	m.mu.RLock()
	connStr := m.connectionString
	m.mu.RUnlock()

	if connStr == "" {
		m.RecordError()
		return fmt.Errorf("database not initialized")
	}

	// In a real implementation, we would insert into MongoDB here
	// For now, we just update the in-memory cache

	duration := time.Since(start).Milliseconds()
	m.RecordWrite(duration)

	// Update in-memory cache
	m.eventsMu.Lock()
	m.events[event.EventHash] = event
	m.eventsMu.Unlock()

	// Update event count
	m.updateEventCount()

	return nil
}

// WriteEvents writes multiple blockchain events to the database (batch)
func (m *MongoDBDatabase) WriteEvents(ctx context.Context, events []core.BlockchainEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()

	m.mu.RLock()
	connStr := m.connectionString
	m.mu.RUnlock()

	if connStr == "" {
		m.RecordError()
		return fmt.Errorf("database not initialized")
	}

	// In a real implementation, we would batch insert into MongoDB here
	// For now, we just update the in-memory cache

	for _, event := range events {
		e := event
		m.eventsMu.Lock()
		m.events[event.EventHash] = &e
		m.eventsMu.Unlock()
	}

	duration := time.Since(start).Milliseconds()
	m.RecordWrite(duration)

	// Update event count
	m.updateEventCount()

	return nil
}

// QueryEvents queries events from the database
func (m *MongoDBDatabase) QueryEvents(filter *core.EventFilter) (*core.QueryResult, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}

	start := time.Now()

	m.mu.RLock()
	connStr := m.connectionString
	m.mu.RUnlock()

	if connStr == "" {
		m.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	// In a real implementation, we would query MongoDB here
	// For now, we filter from the in-memory cache

	events := make([]core.BlockchainEvent, 0, len(m.events))

	m.eventsMu.RLock()
	for _, event := range m.events {
		// Apply filters
		if filter.FromBlock > 0 && event.BlockNumber < filter.FromBlock {
			continue
		}

		if filter.ToBlock > 0 && event.BlockNumber > filter.ToBlock {
			continue
		}

		// Check contract address filter
		if len(filter.ContractAddress) > 0 {
			found := false
			for _, addr := range filter.ContractAddress {
				if event.ContractAddress == addr {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filter.FromTimestamp > 0 && event.BlockTimestamp < filter.FromTimestamp {
			continue
		}

		if filter.ToTimestamp > 0 && event.BlockTimestamp > filter.ToTimestamp {
			continue
		}

		events = append(events, *event)
	}
	m.eventsMu.RUnlock()

	// Apply pagination
	total := int64(len(events))
	if filter.Offset > 0 && filter.Offset < int(total) {
		events = events[filter.Offset:]
	}

	if filter.Limit > 0 && int64(len(events)) > int64(filter.Limit) {
		events = events[:filter.Limit]
	}

	duration := time.Since(start).Milliseconds()
	m.RecordRead(duration)

	result := &core.QueryResult{
		Events: events,
		Total:  total,
	}

	return result, nil
}

// GetEventByHash retrieves an event by its hash
func (m *MongoDBDatabase) GetEventByHash(hash string) (*core.BlockchainEvent, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	// Check in-memory cache first
	m.eventsMu.RLock()
	if event, exists := m.events[hash]; exists {
		m.eventsMu.RUnlock()
		duration := time.Since(start).Milliseconds()
		m.RecordRead(duration)
		return event, nil
	}
	m.eventsMu.RUnlock()

	m.mu.RLock()
	connStr := m.connectionString
	m.mu.RUnlock()

	if connStr == "" {
		m.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	// In a real implementation, we would query MongoDB here
	// For now, we just return nil if not in cache

	duration := time.Since(start).Milliseconds()
	m.RecordRead(duration)

	return nil, nil
}

// DeleteEvent deletes an event from the database by ID
func (m *MongoDBDatabase) DeleteEvent(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event ID is required")
	}

	m.mu.RLock()
	connStr := m.connectionString
	m.mu.RUnlock()

	if connStr == "" {
		m.RecordError()
		return fmt.Errorf("database not initialized")
	}

	// In a real implementation, we would delete from MongoDB here
	// For now, we just remove from in-memory cache

	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	for key, event := range m.events {
		if event.ID == eventID {
			delete(m.events, key)
			m.RecordDelete()
			m.updateEventCount()
			return nil
		}
	}

	return nil
}

// GetStats returns database statistics
func (m *MongoDBDatabase) GetStats() *DatabaseStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgWriteTime := 0.0
	if m.writeCount > 0 {
		avgWriteTime = float64(m.totalWriteTime) / float64(m.writeCount)
	}

	avgReadTime := 0.0
	if m.readCount > 0 {
		avgReadTime = float64(m.totalReadTime) / float64(m.readCount)
	}

	return &DatabaseStats{
		WriteCount:     m.writeCount,
		ReadCount:      m.readCount,
		DeleteCount:    m.deleteCount,
		ErrorCount:     m.errorCount,
		TotalSize:      m.totalSize,
		EventCount:     m.eventCount,
		AvgWriteTimeMs: avgWriteTime,
		AvgReadTimeMs:  avgReadTime,
		LastWriteTime:  m.lastWriteTime,
		LastReadTime:   m.lastReadTime,
	}
}

// updateEventCount updates the event count
func (m *MongoDBDatabase) updateEventCount() {
	m.eventsMu.RLock()
	count := int64(len(m.events))
	m.eventsMu.RUnlock()

	m.UpdateEventCount(count)
}

// GetAllEvents retrieves all events from the database
func (m *MongoDBDatabase) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	m.eventsMu.RLock()
	defer m.eventsMu.RUnlock()

	events := make([]*core.BlockchainEvent, 0, len(m.events))
	for _, event := range m.events {
		events = append(events, event)
	}

	m.RecordRead(0)
	return events, nil
}

// GetAllBlocks retrieves all blocks from the database
func (m *MongoDBDatabase) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	// In a real MongoDB implementation, this would query the blocks collection
	// For now, return empty slice
	m.RecordRead(0)
	return []*core.Block{}, nil
}

// GetEventsByBlockRange retrieves events within a block range
func (m *MongoDBDatabase) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	m.eventsMu.RLock()
	defer m.eventsMu.RUnlock()

	var events []*core.BlockchainEvent
	for _, event := range m.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			events = append(events, event)
		}
	}

	m.RecordRead(0)
	return events, nil
}

// GetBlock retrieves a specific block by number
func (m *MongoDBDatabase) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	// In a real MongoDB implementation, this would query the blocks collection
	// For now, return nil
	m.RecordRead(0)
	return nil, nil
}

// GetLatestBlock retrieves the latest block number
func (m *MongoDBDatabase) GetLatestBlock(ctx context.Context) (uint64, error) {
	// In a real MongoDB implementation, this would query the blocks collection
	// For now, return 0
	m.RecordRead(0)
	return 0, nil
}

// DeleteEventsByBlockRange deletes events within a block range
func (m *MongoDBDatabase) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	count := int64(0)
	for key, event := range m.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			delete(m.events, key)
			count++
		}
	}

	if count > 0 {
		m.RecordDelete()
		m.updateEventCount()
	}

	return count, nil
}

// GetReorgStats retrieves reorg statistics
func (m *MongoDBDatabase) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	// In a real MongoDB implementation, this would query the reorg_stats collection
	// For now, return empty stats
	m.RecordRead(0)
	return &core.ReorgStats{}, nil
}
