package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockCacheService is a mock implementation of a cache service for testing
type MockCacheService struct {
	mu       sync.RWMutex
	data     map[string][]byte
	ttl      map[string]time.Time
	calls    map[string]int
	errors   map[string]error
	failNext map[string]bool
	hits     int64
	misses   int64
}

// NewMockCacheService creates a new mock cache service
func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data:     make(map[string][]byte),
		ttl:      make(map[string]time.Time),
		calls:    make(map[string]int),
		errors:   make(map[string]error),
		failNext: make(map[string]bool),
	}
}

// Get retrieves a value from the cache
func (mcs *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	mcs.calls["Get"]++

	if mcs.failNext["Get"] {
		mcs.failNext["Get"] = false
		return nil, fmt.Errorf("get failed")
	}

	if err, exists := mcs.errors["Get"]; exists {
		return nil, err
	}

	// Check if key exists and is not expired
	if expiry, exists := mcs.ttl[key]; exists && time.Now().After(expiry) {
		delete(mcs.data, key)
		delete(mcs.ttl, key)
		mcs.misses++
		return nil, nil
	}

	value, exists := mcs.data[key]
	if !exists {
		mcs.misses++
		return nil, nil
	}

	mcs.hits++
	return value, nil
}

// Set stores a value in the cache
func (mcs *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	mcs.calls["Set"]++

	if mcs.failNext["Set"] {
		mcs.failNext["Set"] = false
		return fmt.Errorf("set failed")
	}

	if err, exists := mcs.errors["Set"]; exists {
		return err
	}

	mcs.data[key] = value
	if ttl > 0 {
		mcs.ttl[key] = time.Now().Add(ttl)
	}

	return nil
}

// Delete removes a value from the cache
func (mcs *MockCacheService) Delete(ctx context.Context, key string) error {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	mcs.calls["Delete"]++

	if mcs.failNext["Delete"] {
		mcs.failNext["Delete"] = false
		return fmt.Errorf("delete failed")
	}

	if err, exists := mcs.errors["Delete"]; exists {
		return err
	}

	delete(mcs.data, key)
	delete(mcs.ttl, key)

	return nil
}

// Clear clears all values from the cache
func (mcs *MockCacheService) Clear(ctx context.Context) error {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	mcs.calls["Clear"]++

	if mcs.failNext["Clear"] {
		mcs.failNext["Clear"] = false
		return fmt.Errorf("clear failed")
	}

	if err, exists := mcs.errors["Clear"]; exists {
		return err
	}

	mcs.data = make(map[string][]byte)
	mcs.ttl = make(map[string]time.Time)

	return nil
}

// Exists checks if a key exists in the cache
func (mcs *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()

	mcs.calls["Exists"]++

	if mcs.failNext["Exists"] {
		mcs.failNext["Exists"] = false
		return false, fmt.Errorf("exists failed")
	}

	if err, exists := mcs.errors["Exists"]; exists {
		return false, err
	}

	// Check if key exists and is not expired
	if expiry, exists := mcs.ttl[key]; exists && time.Now().After(expiry) {
		return false, nil
	}

	_, exists := mcs.data[key]
	return exists, nil
}

// GetCallCount returns the number of times a method was called
func (mcs *MockCacheService) GetCallCount(method string) int {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()
	return mcs.calls[method]
}

// SetError sets an error to be returned by a method
func (mcs *MockCacheService) SetError(method string, err error) {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()
	mcs.errors[method] = err
}

// FailNext causes the next call to a method to fail
func (mcs *MockCacheService) FailNext(method string) {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()
	mcs.failNext[method] = true
}

// GetHitRate returns the cache hit rate
func (mcs *MockCacheService) GetHitRate() float64 {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()

	total := mcs.hits + mcs.misses
	if total == 0 {
		return 0
	}

	return float64(mcs.hits) / float64(total)
}

// GetHits returns the number of cache hits
func (mcs *MockCacheService) GetHits() int64 {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()
	return mcs.hits
}

// GetMisses returns the number of cache misses
func (mcs *MockCacheService) GetMisses() int64 {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()
	return mcs.misses
}

// ResetStats resets hit/miss statistics
func (mcs *MockCacheService) ResetStats() {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()
	mcs.hits = 0
	mcs.misses = 0
}

// GetSize returns the number of items in the cache
func (mcs *MockCacheService) GetSize() int {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()
	return len(mcs.data)
}

// GetAllKeys returns all keys in the cache
func (mcs *MockCacheService) GetAllKeys() []string {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()

	keys := make([]string, 0, len(mcs.data))
	for key := range mcs.data {
		keys = append(keys, key)
	}

	return keys
}

// Reset clears all data and statistics
func (mcs *MockCacheService) Reset() {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	mcs.data = make(map[string][]byte)
	mcs.ttl = make(map[string]time.Time)
	mcs.calls = make(map[string]int)
	mcs.hits = 0
	mcs.misses = 0
}
