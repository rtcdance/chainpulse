package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// MockDatabase is a mock implementation of a database for testing
type MockDatabase struct {
	mu       sync.RWMutex
	data     map[string]any
	calls    map[string]int
	errors   map[string]error
	failNext map[string]bool
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		data:     make(map[string]any),
		calls:    make(map[string]int),
		errors:   make(map[string]error),
		failNext: make(map[string]bool),
	}
}

// Insert inserts a record into the database
func (md *MockDatabase) Insert(ctx context.Context, collection string, data any) (string, error) {
	md.mu.Lock()
	defer md.mu.Unlock()

	md.calls["Insert"]++

	if md.failNext["Insert"] {
		md.failNext["Insert"] = false
		return "", fmt.Errorf("insert failed")
	}

	if err, exists := md.errors["Insert"]; exists {
		return "", err
	}

	key := fmt.Sprintf("%s:%d", collection, len(md.data))
	md.data[key] = data

	return key, nil
}

// Find retrieves records from the database
func (md *MockDatabase) Find(ctx context.Context, collection string, query map[string]any) ([]any, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	md.calls["Find"]++

	if md.failNext["Find"] {
		md.failNext["Find"] = false
		return nil, fmt.Errorf("find failed")
	}

	if err, exists := md.errors["Find"]; exists {
		return nil, err
	}

	var results []any
	for key, value := range md.data {
		if len(key) > len(collection) && key[:len(collection)] == collection {
			results = append(results, value)
		}
	}

	return results, nil
}

// FindOne retrieves a single record from the database
func (md *MockDatabase) FindOne(ctx context.Context, collection string, query map[string]any) (any, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	md.calls["FindOne"]++

	if md.failNext["FindOne"] {
		md.failNext["FindOne"] = false
		return nil, fmt.Errorf("find one failed")
	}

	if err, exists := md.errors["FindOne"]; exists {
		return nil, err
	}

	for key, value := range md.data {
		if len(key) > len(collection) && key[:len(collection)] == collection {
			return value, nil
		}
	}

	return nil, nil
}

// Update updates a record in the database
func (md *MockDatabase) Update(ctx context.Context, collection string, query map[string]any, update any) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	md.calls["Update"]++

	if md.failNext["Update"] {
		md.failNext["Update"] = false
		return fmt.Errorf("update failed")
	}

	if err, exists := md.errors["Update"]; exists {
		return err
	}

	for key := range md.data {
		if len(key) > len(collection) && key[:len(collection)] == collection {
			md.data[key] = update
			return nil
		}
	}

	return nil
}

// Delete deletes a record from the database
func (md *MockDatabase) Delete(ctx context.Context, collection string, query map[string]any) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	md.calls["Delete"]++

	if md.failNext["Delete"] {
		md.failNext["Delete"] = false
		return fmt.Errorf("delete failed")
	}

	if err, exists := md.errors["Delete"]; exists {
		return err
	}

	for key := range md.data {
		if len(key) > len(collection) && key[:len(collection)] == collection {
			delete(md.data, key)
			return nil
		}
	}

	return nil
}

// Count counts records in the database
func (md *MockDatabase) Count(ctx context.Context, collection string, query map[string]any) (int64, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	md.calls["Count"]++

	if md.failNext["Count"] {
		md.failNext["Count"] = false
		return 0, fmt.Errorf("count failed")
	}

	if err, exists := md.errors["Count"]; exists {
		return 0, err
	}

	count := int64(0)
	for key := range md.data {
		if len(key) > len(collection) && key[:len(collection)] == collection {
			count++
		}
	}

	return count, nil
}

// GetCallCount returns the number of times a method was called
func (md *MockDatabase) GetCallCount(method string) int {
	md.mu.RLock()
	defer md.mu.RUnlock()
	return md.calls[method]
}

// SetError sets an error to be returned by a method
func (md *MockDatabase) SetError(method string, err error) {
	md.mu.Lock()
	defer md.mu.Unlock()
	md.errors[method] = err
}

// FailNext causes the next call to a method to fail
func (md *MockDatabase) FailNext(method string) {
	md.mu.Lock()
	defer md.mu.Unlock()
	md.failNext[method] = true
}

// Clear clears all data from the database
func (md *MockDatabase) Clear() {
	md.mu.Lock()
	defer md.mu.Unlock()
	md.data = make(map[string]any)
	md.calls = make(map[string]int)
}

// GetData returns all data in the database
func (md *MockDatabase) GetData() map[string]any {
	md.mu.RLock()
	defer md.mu.RUnlock()

	data := make(map[string]any)
	for k, v := range md.data {
		data[k] = v
	}

	return data
}

// MockEventStore is a mock implementation of an event store for testing
type MockEventStore struct {
	mu       sync.RWMutex
	events   map[string]*blockchain.BlockchainEvent
	calls    map[string]int
	errors   map[string]error
	failNext map[string]bool
}

// NewMockEventStore creates a new mock event store
func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events:   make(map[string]*blockchain.BlockchainEvent),
		calls:    make(map[string]int),
		errors:   make(map[string]error),
		failNext: make(map[string]bool),
	}
}

// StoreEvent stores an event
func (mes *MockEventStore) StoreEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	mes.calls["StoreEvent"]++

	if mes.failNext["StoreEvent"] {
		mes.failNext["StoreEvent"] = false
		return fmt.Errorf("store event failed")
	}

	if err, exists := mes.errors["StoreEvent"]; exists {
		return err
	}

	mes.events[event.ID] = event
	return nil
}

// GetEvent retrieves an event
func (mes *MockEventStore) GetEvent(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	mes.calls["GetEvent"]++

	if mes.failNext["GetEvent"] {
		mes.failNext["GetEvent"] = false
		return nil, fmt.Errorf("get event failed")
	}

	if err, exists := mes.errors["GetEvent"]; exists {
		return nil, err
	}

	event, exists := mes.events[eventID]
	if !exists {
		return nil, nil
	}

	return event, nil
}

// GetEventsByChain retrieves events by chain
func (mes *MockEventStore) GetEventsByChain(ctx context.Context, chainID string) ([]*blockchain.BlockchainEvent, error) {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	mes.calls["GetEventsByChain"]++

	if mes.failNext["GetEventsByChain"] {
		mes.failNext["GetEventsByChain"] = false
		return nil, fmt.Errorf("get events by chain failed")
	}

	if err, exists := mes.errors["GetEventsByChain"]; exists {
		return nil, err
	}

	var events []*blockchain.BlockchainEvent
	for _, event := range mes.events {
		if event.ChainID == chainID {
			events = append(events, event)
		}
	}

	return events, nil
}

// DeleteEvent deletes an event
func (mes *MockEventStore) DeleteEvent(ctx context.Context, eventID string) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	mes.calls["DeleteEvent"]++

	if mes.failNext["DeleteEvent"] {
		mes.failNext["DeleteEvent"] = false
		return fmt.Errorf("delete event failed")
	}

	if err, exists := mes.errors["DeleteEvent"]; exists {
		return err
	}

	delete(mes.events, eventID)
	return nil
}

// GetCallCount returns the number of times a method was called
func (mes *MockEventStore) GetCallCount(method string) int {
	mes.mu.RLock()
	defer mes.mu.RUnlock()
	return mes.calls[method]
}

// SetError sets an error to be returned by a method
func (mes *MockEventStore) SetError(method string, err error) {
	mes.mu.Lock()
	defer mes.mu.Unlock()
	mes.errors[method] = err
}

// FailNext causes the next call to a method to fail
func (mes *MockEventStore) FailNext(method string) {
	mes.mu.Lock()
	defer mes.mu.Unlock()
	mes.failNext[method] = true
}

// Clear clears all events from the store
func (mes *MockEventStore) Clear() {
	mes.mu.Lock()
	defer mes.mu.Unlock()
	mes.events = make(map[string]*blockchain.BlockchainEvent)
	mes.calls = make(map[string]int)
}

// GetAllEvents returns all events in the store
func (mes *MockEventStore) GetAllEvents() []*blockchain.BlockchainEvent {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	var events []*blockchain.BlockchainEvent
	for _, event := range mes.events {
		events = append(events, event)
	}

	return events
}
