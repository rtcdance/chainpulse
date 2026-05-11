package config

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// ConsulClientInterface defines the interface for Consul client operations
type ConsulClientInterface interface {
	GetConfig(ctx context.Context, key string) (string, error)
	SetConfig(ctx context.Context, key, value string) error
	WatchConfig(ctx context.Context, key string, handler func(string)) error
	Health(ctx context.Context) error
	Close() error
}

// ConfigManager manages centralized configuration.
//
// Renaming would break many external uses.
type ConfigManager struct {
	consul        ConsulClientInterface
	cache         map[string]string
	cacheMutex    sync.RWMutex
	encryptionKey []byte
	watchers      map[string][]func(string)
	watcherMutex  sync.RWMutex
	watcherWg     sync.WaitGroup
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(consul ConsulClientInterface, encryptionKey []byte) *ConfigManager {
	return &ConfigManager{
		consul:        consul,
		cache:         make(map[string]string),
		encryptionKey: encryptionKey,
		watchers:      make(map[string][]func(string)),
	}
}

// GetConfig retrieves a configuration value
func (cm *ConfigManager) GetConfig(ctx context.Context, key string) (string, error) {
	// Check cache first
	cm.cacheMutex.RLock()
	if val, exists := cm.cache[key]; exists {
		cm.cacheMutex.RUnlock()
		return val, nil
	}
	cm.cacheMutex.RUnlock()

	// Fetch from Consul
	val, err := cm.consul.GetConfig(ctx, key)
	if err != nil {
		return "", err
	}

	// Decrypt if necessary
	if isSensitive(key) {
		decrypted, err := cm.decrypt(val)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt config: %w", err)
		}
		val = decrypted
	}

	// Cache the value
	cm.cacheMutex.Lock()
	cm.cache[key] = val
	cm.cacheMutex.Unlock()

	return val, nil
}

// SetConfig sets a configuration value
func (cm *ConfigManager) SetConfig(ctx context.Context, key, value string) error {
	if key == "" {
		return fmt.Errorf("config key must not be empty")
	}

	// Save old cache value for rollback in case Consul write succeeds
	// but cache update fails (should never happen, but defensive programming).
	cm.cacheMutex.RLock()
	oldValue, hadOld := cm.cache[key]
	cm.cacheMutex.RUnlock()

	// Encrypt if necessary
	if isSensitive(key) {
		encrypted, err := cm.encrypt(value)
		if err != nil {
			return fmt.Errorf("failed to encrypt config: %w", err)
		}
		value = encrypted
	}

	// Store in Consul
	if err := cm.consul.SetConfig(ctx, key, value); err != nil {
		return err
	}

	// Update cache
	cm.cacheMutex.Lock()
	cm.cache[key] = value
	cm.cacheMutex.Unlock()

	// Validate the written value by reading it back from Consul.
	// If the read-back fails or the value differs, roll back the cache.
	readback, err := cm.consul.GetConfig(ctx, key)
	if err != nil || readback != value {
		cm.cacheMutex.Lock()
		if hadOld {
			cm.cache[key] = oldValue
		} else {
			delete(cm.cache, key)
		}
		cm.cacheMutex.Unlock()
		if err != nil {
			return fmt.Errorf("config validation failed: read-back error: %w", err)
		}
		return fmt.Errorf("config validation failed: read-back value mismatch for key %s", key)
	}

	// Notify watchers
	cm.notifyWatchers(key, value)

	return nil
}

// WatchConfig watches for configuration changes
func (cm *ConfigManager) WatchConfig(ctx context.Context, key string, handler func(string)) error {
	// Register watcher
	cm.watcherMutex.Lock()
	cm.watchers[key] = append(cm.watchers[key], handler)
	cm.watcherMutex.Unlock()

	// Start watching in Consul
	return cm.consul.WatchConfig(ctx, key, func(value string) {
		// Decrypt if necessary
		if isSensitive(key) {
			decrypted, err := cm.decrypt(value)
			if err != nil {
				slog.Warn("failed to decrypt config", "error", err)
				return
			}
			value = decrypted
		}

		// Update cache
		cm.cacheMutex.Lock()
		cm.cache[key] = value
		cm.cacheMutex.Unlock()

		// Notify watchers
		cm.notifyWatchers(key, value)
	})
}

// notifyWatchers notifies all watchers of a configuration change
func (cm *ConfigManager) notifyWatchers(key, value string) {
	cm.watcherMutex.RLock()
	watchers := cm.watchers[key]
	cm.watcherMutex.RUnlock()

	for _, watcher := range watchers {
		cm.watcherWg.Add(1)
		go func(fn func(string)) {
			defer cm.watcherWg.Done()
			fn(value)
		}(watcher)
	}
}

// WaitWatchers waits for all in-flight watcher goroutines to complete
func (cm *ConfigManager) WaitWatchers() {
	cm.watcherWg.Wait()
}

// encrypt encrypts a value using AES-256-GCM
func (cm *ConfigManager) encrypt(plaintext string) (string, error) {
	if len(cm.encryptionKey) == 0 {
		return plaintext, nil
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a value using AES-256-GCM
func (cm *ConfigManager) decrypt(ciphertext string) (string, error) {
	if len(cm.encryptionKey) == 0 {
		return ciphertext, nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := data[:nonceSize]
	ciphertextBytes := data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ClearCache clears the configuration cache
func (cm *ConfigManager) ClearCache() {
	cm.cacheMutex.Lock()
	cm.cache = make(map[string]string)
	cm.cacheMutex.Unlock()
}

// GetCacheSize returns the size of the cache
func (cm *ConfigManager) GetCacheSize() int {
	cm.cacheMutex.RLock()
	defer cm.cacheMutex.RUnlock()
	return len(cm.cache)
}

// isSensitive checks if a configuration key is sensitive
func isSensitive(key string) bool {
	sensitiveKeys := map[string]bool{
		"password":   true,
		"token":      true,
		"secret":     true,
		"api_key":    true,
		"private":    true,
		"credential": true,
	}

	for sensitiveKey := range sensitiveKeys {
		if len(key) >= len(sensitiveKey) && key[:len(sensitiveKey)] == sensitiveKey {
			return true
		}
	}

	return false
}

// ConfigVersion represents a configuration version.
//
// Renaming would break many external uses.
type ConfigVersion struct {
	Key       string
	Value     string
	Version   int
	Timestamp time.Time
	Author    string
}

// VersionedConfigManager manages configuration versions
type VersionedConfigManager struct {
	cm       *ConfigManager
	versions map[string][]*ConfigVersion
	vMutex   sync.RWMutex
}

// NewVersionedConfigManager creates a new versioned configuration manager
func NewVersionedConfigManager(cm *ConfigManager) *VersionedConfigManager {
	return &VersionedConfigManager{
		cm:       cm,
		versions: make(map[string][]*ConfigVersion),
	}
}

// SetConfigWithVersion sets a configuration value with versioning
func (vcm *VersionedConfigManager) SetConfigWithVersion(ctx context.Context, key, value, author string) error {
	if err := vcm.cm.SetConfig(ctx, key, value); err != nil {
		return err
	}

	vcm.vMutex.Lock()
	defer vcm.vMutex.Unlock()

	versions := vcm.versions[key]
	version := len(versions) + 1

	configVersion := &ConfigVersion{
		Key:       key,
		Value:     value,
		Version:   version,
		Timestamp: time.Now(),
		Author:    author,
	}

	vcm.versions[key] = append(versions, configVersion)
	return nil
}

// GetConfigVersion retrieves a specific version of a configuration
func (vcm *VersionedConfigManager) GetConfigVersion(key string, version int) (*ConfigVersion, error) {
	vcm.vMutex.RLock()
	defer vcm.vMutex.RUnlock()

	versions := vcm.versions[key]
	if version < 1 || version > len(versions) {
		return nil, fmt.Errorf("version not found: %d", version)
	}

	return versions[version-1], nil
}

// GetConfigHistory retrieves the history of a configuration
func (vcm *VersionedConfigManager) GetConfigHistory(key string) ([]*ConfigVersion, error) {
	vcm.vMutex.RLock()
	defer vcm.vMutex.RUnlock()

	versions := vcm.versions[key]
	if len(versions) == 0 {
		return nil, fmt.Errorf("no history found for key: %s", key)
	}

	return versions, nil
}

// RollbackConfig rolls back to a previous version
func (vcm *VersionedConfigManager) RollbackConfig(ctx context.Context, key string, version int, author string) error {
	configVersion, err := vcm.GetConfigVersion(key, version)
	if err != nil {
		return err
	}

	return vcm.SetConfigWithVersion(ctx, key, configVersion.Value, author)
}
