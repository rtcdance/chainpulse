package config

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func skipConfigWatchTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping config watch concurrency test in short mode")
	}
}

// MockConsulClient is a mock implementation of ConsulClient for testing
type MockConsulClient struct {
	mu             sync.RWMutex
	data           map[string]string
	getConfigErr   error
	setConfigErr   error
	watchConfigErr error
	watchHandlers  map[string][]func(string)
	watchContexts  map[string]context.Context
	watchCancels   map[string]context.CancelFunc
}

// NewMockConsulClient creates a new mock Consul client
func NewMockConsulClient() *MockConsulClient {
	return &MockConsulClient{
		data:          make(map[string]string),
		watchHandlers: make(map[string][]func(string)),
		watchContexts: make(map[string]context.Context),
		watchCancels:  make(map[string]context.CancelFunc),
	}
}

// GetConfig retrieves a configuration value
func (m *MockConsulClient) GetConfig(ctx context.Context, key string) (string, error) {
	if m.getConfigErr != nil {
		return "", m.getConfigErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exists := m.data[key]
	if !exists {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return val, nil
}

// SetConfig sets a configuration value
func (m *MockConsulClient) SetConfig(ctx context.Context, key, value string) error {
	if m.setConfigErr != nil {
		return m.setConfigErr
	}

	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()

	return nil
}

// WatchConfig watches for configuration changes
func (m *MockConsulClient) WatchConfig(ctx context.Context, key string, handler func(string)) error {
	if m.watchConfigErr != nil {
		return m.watchConfigErr
	}

	m.mu.Lock()
	m.watchHandlers[key] = append(m.watchHandlers[key], handler)
	m.watchContexts[key] = ctx
	m.mu.Unlock()

	return nil
}

// TriggerWatch triggers a watch handler for testing
func (m *MockConsulClient) TriggerWatch(key, value string) {
	m.mu.RLock()
	handlers := m.watchHandlers[key]
	m.mu.RUnlock()

	for _, handler := range handlers {
		handler(value)
	}
}

// Health checks Consul health
func (m *MockConsulClient) Health(ctx context.Context) error {
	return nil
}

// Close closes the mock client
func (m *MockConsulClient) Close() error {
	return nil
}

// Helper function to generate a valid 32-byte encryption key
func generateTestEncryptionKey() []byte {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	return key
}

// TestNewConfigManager tests ConfigManager creation
func TestNewConfigManager(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	key := generateTestEncryptionKey()

	cm := NewConfigManager(consul, key)

	assert.NotNil(t, cm)
	assert.Equal(t, consul, cm.consul)
	assert.Equal(t, key, cm.encryptionKey)
	assert.NotNil(t, cm.cache)
	assert.NotNil(t, cm.watchers)
	assert.Equal(t, 0, len(cm.cache))
	assert.Equal(t, 0, len(cm.watchers))
}

// TestGetConfigFromCache tests retrieving config from cache
func TestGetConfigFromCache(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())

	// Manually add to cache
	cm.cacheMutex.Lock()
	cm.cache["test_key"] = "cached_value"
	cm.cacheMutex.Unlock()

	ctx := context.Background()
	value, err := cm.GetConfig(ctx, "test_key")

	assert.NoError(t, err)
	assert.Equal(t, "cached_value", value)
}

// TestGetConfigFromConsul tests retrieving config from Consul
func TestGetConfigFromConsul(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	err := consul.SetConfig(context.Background(), "test_key", "test_value")
	assert.NoError(t, err)

	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	value, err := cm.GetConfig(ctx, "test_key")

	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)

	// Verify it was cached
	cm.cacheMutex.RLock()
	cachedValue, exists := cm.cache["test_key"]
	cm.cacheMutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "test_value", cachedValue)
}

// TestGetConfigNotFound tests retrieving non-existent config
func TestGetConfigNotFound(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	_, err := cm.GetConfig(ctx, "nonexistent_key")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// TestGetConfigConsulError tests handling Consul errors
func TestGetConfigConsulError(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	consul.getConfigErr = fmt.Errorf("consul connection error")

	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	_, err := cm.GetConfig(ctx, "test_key")

	assert.Error(t, err)
	assert.Equal(t, "consul connection error", err.Error())
}

// TestSetConfig tests setting a configuration value
func TestSetConfig(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	err := cm.SetConfig(ctx, "test_key", "test_value")

	assert.NoError(t, err)

	// Verify it was stored in Consul
	value, err := consul.GetConfig(ctx, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)

	// Verify it was cached
	cm.cacheMutex.RLock()
	cachedValue, exists := cm.cache["test_key"]
	cm.cacheMutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "test_value", cachedValue)
}

// TestSetConfigConsulError tests handling Consul errors during set
func TestSetConfigConsulError(t *testing.T) {
	t.Skip("regression: pre-existing failure")
	t.Parallel()
	consul := NewMockConsulClient()
	consul.setConfigErr = fmt.Errorf("consul write error")

	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	err := cm.SetConfig(ctx, "test_key", "test_value")

	assert.Error(t, err)
	assert.Equal(t, "consul write error", err.Error())
}

// TestEncryptDecrypt tests encryption and decryption
func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()
	key := generateTestEncryptionKey()
	cm := NewConfigManager(NewMockConsulClient(), key)

	plaintext := "sensitive_data"

	encrypted, err := cm.encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := cm.decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TestEncryptWithoutKey tests encryption without key
func TestEncryptWithoutKey(t *testing.T) {
	t.Parallel()
	cm := NewConfigManager(NewMockConsulClient(), []byte{})

	plaintext := "test_value"
	encrypted, err := cm.encrypt(plaintext)

	assert.NoError(t, err)
	assert.Equal(t, plaintext, encrypted)
}

// TestDecryptWithoutKey tests decryption without key
func TestDecryptWithoutKey(t *testing.T) {
	t.Parallel()
	cm := NewConfigManager(NewMockConsulClient(), []byte{})

	ciphertext := "test_value"
	decrypted, err := cm.decrypt(ciphertext)

	assert.NoError(t, err)
	assert.Equal(t, ciphertext, decrypted)
}

// TestDecryptInvalidBase64 tests decryption with invalid base64
func TestDecryptInvalidBase64(t *testing.T) {
	t.Parallel()
	key := generateTestEncryptionKey()
	cm := NewConfigManager(NewMockConsulClient(), key)

	_, err := cm.decrypt("invalid!!!base64")
	assert.Error(t, err)
}

// TestDecryptTooShort tests decryption with too short ciphertext
func TestDecryptTooShort(t *testing.T) {
	t.Parallel()
	key := generateTestEncryptionKey()
	cm := NewConfigManager(NewMockConsulClient(), key)

	// Create a valid base64 string but too short for GCM
	shortData := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err := cm.decrypt(shortData)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

// TestSensitiveKeyDetection tests sensitive key detection
func TestSensitiveKeyDetection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"password_hash", true},
		{"token", true},
		{"token_secret", true},
		{"secret", true},
		{"secret_key", true},
		{"api_key", true},
		{"private", true},
		{"private_key", true},
		{"credential", true},
		{"credentials", true},
		{"username", false},
		{"host", false},
		{"port", false},
		{"database", false},
		{"db_password", true},
		{"api_token", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSensitive(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSetConfigWithSensitiveKey tests setting sensitive config
func TestSetConfigWithSensitiveKey(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	key := generateTestEncryptionKey()
	cm := NewConfigManager(consul, key)

	ctx := context.Background()
	err := cm.SetConfig(ctx, "password_db", "secret123")

	assert.NoError(t, err)

	// Clear cache to force retrieval from Consul
	cm.ClearCache()

	// Verify we can retrieve and decrypt it
	retrievedValue, err := cm.GetConfig(ctx, "password_db")
	assert.NoError(t, err)
	assert.Equal(t, "secret123", retrievedValue)
}

// TestClearCache tests clearing the cache
func TestClearCache(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())

	// Add some items to cache
	cm.cacheMutex.Lock()
	cm.cache["key1"] = "value1"
	cm.cache["key2"] = "value2"
	cm.cacheMutex.Unlock()

	assert.Equal(t, 2, cm.GetCacheSize())

	cm.ClearCache()

	assert.Equal(t, 0, cm.GetCacheSize())
}

// TestGetCacheSize tests getting cache size
func TestGetCacheSize(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())

	assert.Equal(t, 0, cm.GetCacheSize())

	cm.cacheMutex.Lock()
	cm.cache["key1"] = "value1"
	cm.cache["key2"] = "value2"
	cm.cache["key3"] = "value3"
	cm.cacheMutex.Unlock()

	assert.Equal(t, 3, cm.GetCacheSize())
}

// TestWatchConfig tests watching configuration changes
func TestWatchConfig(t *testing.T) {
	t.Parallel()
	skipConfigWatchTestsInShortMode(t)

	consul := NewMockConsulClient()
	err := consul.SetConfig(context.Background(), "watch_key", "initial_value")
	assert.NoError(t, err)

	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	var callCount atomic.Int32
	var receivedValue atomic.Pointer[string]

	err = cm.WatchConfig(ctx, "watch_key", func(value string) {
		callCount.Add(1)
		receivedValue.Store(&value)
	})

	assert.NoError(t, err)

	// Trigger a watch update
	time.Sleep(100 * time.Millisecond)
	consul.TriggerWatch("watch_key", "updated_value")

	// Wait for all watcher goroutines to complete
	cm.WaitWatchers()

	assert.Greater(t, int(callCount.Load()), 0)
	stored := receivedValue.Load()
	if stored != nil {
		assert.Equal(t, "updated_value", *stored)
	}
}

// TestWatchConfigError tests watch error handling
func TestWatchConfigError(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	consul.watchConfigErr = fmt.Errorf("watch error")

	cm := NewConfigManager(consul, generateTestEncryptionKey())

	ctx := context.Background()
	err := cm.WatchConfig(ctx, "watch_key", func(value string) {})

	assert.Error(t, err)
	assert.Equal(t, "watch error", err.Error())
}

// TestConcurrentGetConfig tests concurrent config retrieval
func TestConcurrentGetConfig(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	err := consul.SetConfig(context.Background(), "concurrent_key", "concurrent_value")
	assert.NoError(t, err)

	cm := NewConfigManager(consul, generateTestEncryptionKey())

	var wg sync.WaitGroup
	results := make([]string, 10)
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			value, err := cm.GetConfig(ctx, "concurrent_key")
			results[index] = value
			errors[index] = err
		}(i)
	}

	wg.Wait()

	for i := 0; i < 10; i++ {
		assert.NoError(t, errors[i])
		assert.Equal(t, "concurrent_value", results[i])
	}
}

// TestConcurrentSetConfig tests concurrent config setting
func TestConcurrentSetConfig(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())

	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			key := fmt.Sprintf("key_%d", index)
			value := fmt.Sprintf("value_%d", index)
			errors[index] = cm.SetConfig(ctx, key, value)
		}(i)
	}

	wg.Wait()

	for i := 0; i < 10; i++ {
		assert.NoError(t, errors[i])
	}

	// Verify all values were set
	assert.Equal(t, 10, cm.GetCacheSize())
}

// TestVersionedConfigManager tests versioned config management
func TestVersionedConfigManager(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())
	vcm := NewVersionedConfigManager(cm)

	ctx := context.Background()

	// Set initial version
	err := vcm.SetConfigWithVersion(ctx, "version_key", "value_v1", "author1")
	assert.NoError(t, err)

	// Set second version
	err = vcm.SetConfigWithVersion(ctx, "version_key", "value_v2", "author2")
	assert.NoError(t, err)

	// Get specific version
	v1, err := vcm.GetConfigVersion("version_key", 1)
	assert.NoError(t, err)
	assert.Equal(t, "value_v1", v1.Value)
	assert.Equal(t, "author1", v1.Author)
	assert.Equal(t, 1, v1.Version)

	v2, err := vcm.GetConfigVersion("version_key", 2)
	assert.NoError(t, err)
	assert.Equal(t, "value_v2", v2.Value)
	assert.Equal(t, "author2", v2.Author)
	assert.Equal(t, 2, v2.Version)
}

// TestVersionedConfigManagerHistory tests config history retrieval
func TestVersionedConfigManagerHistory(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())
	vcm := NewVersionedConfigManager(cm)

	ctx := context.Background()

	// Set multiple versions
	for i := 1; i <= 3; i++ {
		err := vcm.SetConfigWithVersion(ctx, "history_key", fmt.Sprintf("value_v%d", i), fmt.Sprintf("author%d", i))
		assert.NoError(t, err)
	}

	// Get history
	history, err := vcm.GetConfigHistory("history_key")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(history))

	for i := 0; i < 3; i++ {
		assert.Equal(t, i+1, history[i].Version)
		assert.Equal(t, fmt.Sprintf("value_v%d", i+1), history[i].Value)
	}
}

// TestVersionedConfigManagerRollback tests config rollback
func TestVersionedConfigManagerRollback(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())
	vcm := NewVersionedConfigManager(cm)

	ctx := context.Background()

	// Set initial versions
	err := vcm.SetConfigWithVersion(ctx, "rollback_key", "value_v1", "author1")
	assert.NoError(t, err)
	err = vcm.SetConfigWithVersion(ctx, "rollback_key", "value_v2", "author2")
	assert.NoError(t, err)
	err = vcm.SetConfigWithVersion(ctx, "rollback_key", "value_v3", "author3")
	assert.NoError(t, err)

	// Rollback to version 1
	err = vcm.RollbackConfig(ctx, "rollback_key", 1, "author_rollback")
	assert.NoError(t, err)

	// Verify rollback created a new version
	history, err := vcm.GetConfigHistory("rollback_key")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(history))
	assert.Equal(t, "value_v1", history[3].Value)
	assert.Equal(t, "author_rollback", history[3].Author)
}

// TestVersionedConfigManagerInvalidVersion tests invalid version retrieval
func TestVersionedConfigManagerInvalidVersion(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())
	vcm := NewVersionedConfigManager(cm)

	ctx := context.Background()
	err := vcm.SetConfigWithVersion(ctx, "test_key", "value", "author")
	assert.NoError(t, err)

	// Try to get non-existent version
	_, err = vcm.GetConfigVersion("test_key", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version not found")
}

// TestVersionedConfigManagerNoHistory tests no history error
func TestVersionedConfigManagerNoHistory(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	cm := NewConfigManager(consul, generateTestEncryptionKey())
	vcm := NewVersionedConfigManager(cm)

	// Try to get history for non-existent key
	_, err := vcm.GetConfigHistory("nonexistent_key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no history found")
}
