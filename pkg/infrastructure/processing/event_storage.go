package processing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventStore defines the interface for event storage
type EventStore interface {
	StoreEvent(ctx context.Context, event *Event) error
	StoreBatch(ctx context.Context, events []*Event) error
	GetEvent(ctx context.Context, eventID string) (*Event, error)
	QueryEvents(ctx context.Context, filter *EventFilter) ([]*Event, error)
	DeleteEvent(ctx context.Context, eventID string) error
	GetMetrics() StorageMetrics
}

// EventFilter defines event query filters
type EventFilter struct {
	ChainID         string
	ContractAddress string
	EventName       string
	FromBlock       uint64
	ToBlock         uint64
	FromTime        time.Time
	ToTime          time.Time
	Status          string
	Limit           int
	Offset          int
}

// StorageMetrics tracks storage metrics
type StorageMetrics struct {
	mu                  sync.RWMutex
	EventsStored        int64
	EventsRetrieved     int64
	EventsDeleted       int64
	BatchesProcessed    int64
	AverageLatency      time.Duration
	TotalStorageTime    time.Duration
	LastStorageTime     time.Time
	StorageErrors       int64
	TransactionCommits  int64
	TransactionRollbacks int64
}

// InMemoryEventStore is an in-memory implementation of EventStore
type InMemoryEventStore struct {
	mu       sync.RWMutex
	events   map[string]*Event
	metrics  *StorageMetrics
	maxSize  int
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(maxSize int) *InMemoryEventStore {
	return &InMemoryEventStore{
		events:  make(map[string]*Event),
		maxSize: maxSize,
		metrics: &StorageMetrics{
			LastStorageTime: time.Now(),
		},
	}
}

// StoreEvent stores a single event
func (ies *InMemoryEventStore) StoreEvent(ctx context.Context, event *Event) error {
	start := time.Now()
	defer func() {
		ies.recordLatency(time.Since(start))
	}()

	if event == nil {
		ies.metrics.mu.Lock()
		ies.metrics.StorageErrors++
		ies.metrics.mu.Unlock()
		return fmt.Errorf("event is nil")
	}

	if event.ID == "" {
		ies.metrics.mu.Lock()
		ies.metrics.StorageErrors++
		ies.metrics.mu.Unlock()
		return fmt.Errorf("event ID is empty")
	}

	ies.mu.Lock()
	defer ies.mu.Unlock()

	// Check size limit
	if len(ies.events) >= ies.maxSize {
		ies.metrics.mu.Lock()
		ies.metrics.StorageErrors++
		ies.metrics.mu.Unlock()
		return fmt.Errorf("event store is full")
	}

	ies.events[event.ID] = event

	ies.metrics.mu.Lock()
	ies.metrics.EventsStored++
	ies.metrics.mu.Unlock()

	return nil
}

// StoreBatch stores a batch of events
func (ies *InMemoryEventStore) StoreBatch(ctx context.Context, events []*Event) error {
	start := time.Now()
	defer func() {
		ies.recordBatchLatency(time.Since(start))
	}()

	if len(events) == 0 {
		return nil
	}

	ies.mu.Lock()
	defer ies.mu.Unlock()

	// Check if all events can fit
	if len(ies.events)+len(events) > ies.maxSize {
		ies.metrics.mu.Lock()
		ies.metrics.StorageErrors++
		ies.metrics.mu.Unlock()
		return fmt.Errorf("batch exceeds store capacity")
	}

	// Store all events
	for _, event := range events {
		if event != nil && event.ID != "" {
			ies.events[event.ID] = event
		}
	}

	ies.metrics.mu.Lock()
	ies.metrics.EventsStored += int64(len(events))
	ies.metrics.BatchesProcessed++
	ies.metrics.mu.Unlock()

	return nil
}

// GetEvent retrieves a single event
func (ies *InMemoryEventStore) GetEvent(ctx context.Context, eventID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID is empty")
	}

	ies.mu.RLock()
	defer ies.mu.RUnlock()

	event, exists := ies.events[eventID]
	if !exists {
		return nil, fmt.Errorf("event not found")
	}

	ies.metrics.mu.Lock()
	ies.metrics.EventsRetrieved++
	ies.metrics.mu.Unlock()

	return event, nil
}

// QueryEvents queries events with filters
func (ies *InMemoryEventStore) QueryEvents(ctx context.Context, filter *EventFilter) ([]*Event, error) {
	if filter == nil {
		filter = &EventFilter{}
	}

	ies.mu.RLock()
	defer ies.mu.RUnlock()

	var results []*Event

	for _, event := range ies.events {
		if ies.matchesFilter(event, filter) {
			results = append(results, event)
		}
	}

	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}

	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	ies.metrics.mu.Lock()
	ies.metrics.EventsRetrieved += int64(len(results))
	ies.metrics.mu.Unlock()

	return results, nil
}

// DeleteEvent deletes an event
func (ies *InMemoryEventStore) DeleteEvent(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event ID is empty")
	}

	ies.mu.Lock()
	defer ies.mu.Unlock()

	if _, exists := ies.events[eventID]; !exists {
		return fmt.Errorf("event not found")
	}

	delete(ies.events, eventID)

	ies.metrics.mu.Lock()
	ies.metrics.EventsDeleted++
	ies.metrics.mu.Unlock()

	return nil
}

// matchesFilter checks if an event matches the filter
func (ies *InMemoryEventStore) matchesFilter(event *Event, filter *EventFilter) bool {
	if filter.ChainID != "" && event.ChainID != filter.ChainID {
		return false
	}

	if filter.ContractAddress != "" && event.ContractAddress != filter.ContractAddress {
		return false
	}

	if filter.EventName != "" && event.EventName != filter.EventName {
		return false
	}

	if filter.FromBlock > 0 && event.BlockNumber < filter.FromBlock {
		return false
	}

	if filter.ToBlock > 0 && event.BlockNumber > filter.ToBlock {
		return false
	}

	if !filter.FromTime.IsZero() && event.Timestamp.Before(filter.FromTime) {
		return false
	}

	if !filter.ToTime.IsZero() && event.Timestamp.After(filter.ToTime) {
		return false
	}

	if filter.Status != "" && event.Status != filter.Status {
		return false
	}

	return true
}

// recordLatency records storage latency
func (ies *InMemoryEventStore) recordLatency(latency time.Duration) {
	ies.metrics.mu.Lock()
	defer ies.metrics.mu.Unlock()

	ies.metrics.TotalStorageTime += latency
	if ies.metrics.EventsStored > 0 {
		ies.metrics.AverageLatency = ies.metrics.TotalStorageTime / time.Duration(ies.metrics.EventsStored)
	}
	ies.metrics.LastStorageTime = time.Now()
}

// recordBatchLatency records batch storage latency
func (ies *InMemoryEventStore) recordBatchLatency(latency time.Duration) {
	ies.metrics.mu.Lock()
	defer ies.metrics.mu.Unlock()

	ies.metrics.TotalStorageTime += latency
}

// GetMetrics returns storage metrics
func (ies *InMemoryEventStore) GetMetrics() StorageMetrics {
	ies.metrics.mu.RLock()
	defer ies.metrics.mu.RUnlock()

	return StorageMetrics{
		EventsStored:         ies.metrics.EventsStored,
		EventsRetrieved:      ies.metrics.EventsRetrieved,
		EventsDeleted:        ies.metrics.EventsDeleted,
		BatchesProcessed:     ies.metrics.BatchesProcessed,
		AverageLatency:       ies.metrics.AverageLatency,
		TotalStorageTime:     ies.metrics.TotalStorageTime,
		LastStorageTime:      ies.metrics.LastStorageTime,
		StorageErrors:        ies.metrics.StorageErrors,
		TransactionCommits:   ies.metrics.TransactionCommits,
		TransactionRollbacks: ies.metrics.TransactionRollbacks,
	}
}

// GetSize returns the current number of events in the store
func (ies *InMemoryEventStore) GetSize() int {
	ies.mu.RLock()
	defer ies.mu.RUnlock()
	return len(ies.events)
}

// Clear clears all events from the store
func (ies *InMemoryEventStore) Clear() {
	ies.mu.Lock()
	defer ies.mu.Unlock()
	ies.events = make(map[string]*Event)
}

// TransactionManager manages database transactions
type TransactionManager struct {
	mu              sync.RWMutex
	activeTransactions map[string]*Transaction
	commitCount     int64
	rollbackCount   int64
}

// Transaction represents a database transaction
type Transaction struct {
	ID        string
	StartTime time.Time
	Status    string // "active", "committed", "rolled_back"
	Events    []*Event
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		activeTransactions: make(map[string]*Transaction),
	}
}

// BeginTransaction starts a new transaction
func (tm *TransactionManager) BeginTransaction(txID string) *Transaction {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx := &Transaction{
		ID:        txID,
		StartTime: time.Now(),
		Status:    "active",
		Events:    make([]*Event, 0),
	}

	tm.activeTransactions[txID] = tx
	return tx
}

// CommitTransaction commits a transaction
func (tm *TransactionManager) CommitTransaction(txID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx, exists := tm.activeTransactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status != "active" {
		return fmt.Errorf("transaction is not active")
	}

	tx.Status = "committed"
	tm.commitCount++

	delete(tm.activeTransactions, txID)
	return nil
}

// RollbackTransaction rolls back a transaction
func (tm *TransactionManager) RollbackTransaction(txID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx, exists := tm.activeTransactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status != "active" {
		return fmt.Errorf("transaction is not active")
	}

	tx.Status = "rolled_back"
	tm.rollbackCount++

	delete(tm.activeTransactions, txID)
	return nil
}

// GetMetrics returns transaction manager metrics
func (tm *TransactionManager) GetMetrics() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return map[string]interface{}{
		"active_transactions": len(tm.activeTransactions),
		"commit_count":        tm.commitCount,
		"rollback_count":      tm.rollbackCount,
	}
}
