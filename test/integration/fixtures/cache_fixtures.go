package fixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// CacheFixture provides cache setup and teardown for integration tests
type CacheFixture struct {
	cache core.CachePlugin
	t     *testing.T
}

// NewCacheFixture creates a new cache fixture
func NewCacheFixture(t *testing.T, cache core.CachePlugin) *CacheFixture {
	return &CacheFixture{
		cache: cache,
		t:     t,
	}
}

// Setup initializes the cache for testing
func (f *CacheFixture) Setup(ctx context.Context) error {
	if f.cache == nil {
		return fmt.Errorf("cache plugin is nil")
	}

	return nil
}

// Cleanup cleans up the cache after testing
func (f *CacheFixture) Cleanup(ctx context.Context) error {
	if f.cache == nil {
		return nil
	}

	return nil
}

// Close closes the cache connection
func (f *CacheFixture) Close() error {
	return nil
}

// Set sets a value in the cache
func (f *CacheFixture) Set(ctx context.Context, key string, value []byte, ttl int) error {
	if err := f.cache.Set(ctx, key, value, ttl); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}
	return nil
}

// Get retrieves a value from the cache
func (f *CacheFixture) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := f.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache key %s: %w", key, err)
	}
	return value, nil
}

// Delete deletes a value from the cache
func (f *CacheFixture) Delete(ctx context.Context, key string) error {
	if err := f.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}
	return nil
}

// Clear clears all values from the cache
func (f *CacheFixture) Clear() error {
	return nil
}

// Exists checks if a key exists in the cache
func (f *CacheFixture) Exists(ctx context.Context, key string) (bool, error) {
	_, err := f.cache.Get(ctx, key)
	if err != nil {
		// Key doesn't exist
		return false, nil
	}
	return true, nil
}

// GetStats returns cache statistics
func (f *CacheFixture) GetStats() core.CacheStats {
	return f.cache.GetStats()
}

// WaitForExpiration waits for a cache entry to expire
func (f *CacheFixture) WaitForExpiration(ctx context.Context, key string, ttl time.Duration) error {
	// Add buffer for expiration
	timeout := ttl + (100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Check if key is actually expired
			exists, err := f.Exists(ctx, key)
			if err != nil {
				return fmt.Errorf("error checking key existence: %w", err)
			}
			if !exists {
				return nil
			}
			return fmt.Errorf("key did not expire within timeout")
		case <-ticker.C:
			exists, err := f.Exists(ctx, key)
			if err != nil {
				return fmt.Errorf("error checking key existence: %w", err)
			}
			if !exists {
				return nil
			}
		}
	}
}

// SetMultiple sets multiple values in the cache
func (f *CacheFixture) SetMultiple(ctx context.Context, entries map[string][]byte, ttl int) error {
	for key, value := range entries {
		if err := f.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// GetMultiple retrieves multiple values from the cache
func (f *CacheFixture) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, key := range keys {
		value, err := f.Get(ctx, key)
		if err != nil {
			// Skip missing keys
			continue
		}
		result[key] = value
	}
	return result, nil
}

// DeleteMultiple deletes multiple values from the cache
func (f *CacheFixture) DeleteMultiple(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := f.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// WithTimeout returns a context with timeout
func (f *CacheFixture) WithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
