package cache

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

func TestRedisCacheInitialize(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	err := cache.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if cache.GetConnectionURL() != "redis://localhost:6379" {
		t.Fatalf("Expected connection URL redis://localhost:6379, got %s", cache.GetConnectionURL())
	}

	// Test double initialization
	err = cache.Initialize(config)
	if err == nil {
		t.Fatal("Expected error on double initialization")
	}
}

func TestRedisCacheLifecycle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)

	// Start
	err := cache.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check health
	health := cache.Health()
	if health.Status != "healthy" {
		t.Fatalf("Expected healthy status, got %s", health.Status)
	}

	// Stop
	err = cache.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Check health after stop
	health = cache.Health()
	if health.Status != "unhealthy" {
		t.Fatalf("Expected unhealthy status after stop, got %s", health.Status)
	}
}

func TestRedisCacheSetAndGet(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set a value
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	err := cache.Set(entry)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value
	retrieved, err := cache.Get("test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve value")
	}

	if retrieved.Key != "test_key" {
		t.Fatalf("Expected key test_key, got %s", retrieved.Key)
	}

	if retrieved.Value != "test_value" {
		t.Fatalf("Expected value test_value, got %v", retrieved.Value)
	}
}

func TestRedisCacheDelete(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set a value
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	// Verify it exists
	retrieved, _ := cache.Get("test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist")
	}

	// Delete it
	err := cache.Delete("test_key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	retrieved, _ = cache.Get("test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be deleted")
	}
}

func TestRedisCacheExists(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set a value
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	// Check exists
	exists, err := cache.Exists("test_key")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Fatal("Expected key to exist")
	}

	// Check non-existent
	exists, err = cache.Exists("non_existent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Fatal("Expected key to not exist")
	}
}

func TestRedisCacheExpire(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set a value
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	// Set new expiration
	err := cache.Expire("test_key", 1)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Check if expired
	exists, _ := cache.Exists("test_key")
	if exists {
		t.Fatal("Expected key to be expired")
	}
}

func TestRedisCacheTTL(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set a value
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   10,
	}

	_ = cache.Set(entry)

	// Get TTL
	ttl, err := cache.TTL("test_key")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 || ttl > 10 {
		t.Fatalf("Expected TTL between 0 and 10, got %d", ttl)
	}

	// Get TTL for non-existent key
	ttl, err = cache.TTL("non_existent")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl != -2 {
		t.Fatalf("Expected TTL -2 for non-existent key, got %d", ttl)
	}
}

func TestRedisCacheIncrement(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Increment non-existent key
	val, err := cache.Increment("counter", 5)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}

	if val != 5 {
		t.Fatalf("Expected 5, got %d", val)
	}

	// Increment again
	val, err = cache.Increment("counter", 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}

	if val != 8 {
		t.Fatalf("Expected 8, got %d", val)
	}
}

func TestRedisCacheDecrement(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set initial value
	entry := &CacheEntry{
		Key:   "counter",
		Value: int64(10),
		TTL:   3600,
	}

	_ = cache.Set(entry)

	// Decrement
	val, err := cache.Decrement("counter", 3)
	if err != nil {
		t.Fatalf("Decrement failed: %v", err)
	}

	if val != 7 {
		t.Fatalf("Expected 7, got %d", val)
	}
}

func TestRedisCacheGetAllKeys(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set multiple values
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   "key_" + string(rune(i)),
			Value: "value_" + string(rune(i)),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Get all keys
	keys, err := cache.GetAllKeys()
	if err != nil {
		t.Fatalf("GetAllKeys failed: %v", err)
	}

	if len(keys) != 5 {
		t.Fatalf("Expected 5 keys, got %d", len(keys))
	}
}

func TestRedisCachePing(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Ping
	err := cache.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestRedisCacheFlushDB(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set multiple values
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   "key_" + string(rune(i)),
			Value: "value_" + string(rune(i)),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Flush
	err := cache.FlushDB(context.Background())
	if err != nil {
		t.Fatalf("FlushDB failed: %v", err)
	}

	// Verify all keys are gone
	keys, _ := cache.GetAllKeys()
	if len(keys) != 0 {
		t.Fatalf("Expected 0 keys after flush, got %d", len(keys))
	}
}

func TestRedisCacheGetKeyCount(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set multiple values
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   "key_" + string(rune(i)),
			Value: "value_" + string(rune(i)),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Get key count
	count := cache.GetKeyCount()
	if count != 5 {
		t.Fatalf("Expected 5 keys, got %d", count)
	}
}

func TestRedisCacheStats(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Set values
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			Key:   "key_" + string(rune(i)),
			Value: "value_" + string(rune(i)),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Get values (hits)
	for i := 0; i < 3; i++ {
		_, _ = cache.Get("key_" + string(rune(i)))
	}

	// Get non-existent (misses)
	for i := 0; i < 2; i++ {
		_, _ = cache.Get("non_existent_" + string(rune(i)))
	}

	// Check stats
	stats := cache.GetStats()
	if stats.HitCount != 3 {
		t.Fatalf("Expected 3 hits, got %d", stats.HitCount)
	}

	if stats.MissCount != 2 {
		t.Fatalf("Expected 2 misses, got %d", stats.MissCount)
	}

	if stats.EntryCount != 3 {
		t.Fatalf("Expected 3 entries, got %d", stats.EntryCount)
	}
}

func TestRedisCacheConcurrentOperations(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)
	defer func() { _ = cache.Stop() }()

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start()

	// Concurrent set operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				entry := &CacheEntry{
					Key:   "key_" + string(rune(id*10+j)),
					Value: "value_" + string(rune(id*10+j)),
					TTL:   3600,
				}
				_ = cache.Set(entry)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all entries
	count := cache.GetKeyCount()
	if count != 100 {
		t.Fatalf("Expected 100 entries, got %d", count)
	}
}

func TestRedisCacheDefaultConnectionURL(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
		// No CacheConnectionURL specified
	}

	_ = cache.Initialize(config)

	// Should use default
	if cache.GetConnectionURL() != "redis://localhost:6379" {
		t.Fatalf("Expected default connection URL redis://localhost:6379, got %s", cache.GetConnectionURL())
	}
}
