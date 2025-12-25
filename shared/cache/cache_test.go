package cache

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cache == nil {
		t.Error("Expected cache instance, got nil")
	}
}

func TestCacheSetAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Skipf("skipping test: could not connect to Redis: %v", err)
	}

	key := "test_key"
	value := "test_value"

	// Test setting a value
	err = cache.Set(key, value, 10*time.Second)
	if err != nil {
		t.Errorf("Expected no error when setting value, got %v", err)
	}

	// Test getting the value
	result, err := cache.Get(key)
	if err != nil {
		t.Errorf("Expected no error when getting value, got %v", err)
	}

	if result != value {
		t.Errorf("Expected value %s, got %s", value, result)
	}
}

func TestCacheGetNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Skipf("skipping test: could not connect to Redis: %v", err)
	}

	key := "non_existent_key"

	// Test getting a non-existent value
	result, err := cache.Get(key)
	if err != nil {
		// Depending on implementation, this might return an error or empty string
		// For our implementation, we expect an empty string with no error
		t.Errorf("Expected no error when getting non-existent value, got %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty string for non-existent key, got %s", result)
	}
}

func TestCacheDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Skipf("skipping test: could not connect to Redis: %v", err)
	}

	key := "test_delete_key"
	value := "test_delete_value"

	// Set a value first
	err = cache.Set(key, value, 10*time.Second)
	if err != nil {
		t.Errorf("Expected no error when setting value, got %v", err)
	}

	// Delete the key
	err = cache.Delete(key)
	if err != nil {
		t.Errorf("Expected no error when deleting key, got %v", err)
	}

	// Try to get the deleted key
	result, err := cache.Get(key)
	if err != nil {
		t.Errorf("Expected no error when getting deleted value, got %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty string after deletion, got %s", result)
	}
}

func TestCacheExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Skipf("skipping test: could not connect to Redis: %v", err)
	}

	key := "test_exists_key"
	value := "test_exists_value"

	// Set a value first
	err = cache.Set(key, value, 10*time.Second)
	if err != nil {
		t.Errorf("Expected no error when setting value, got %v", err)
	}

	// Check if key exists
	exists, err := cache.Exists(key)
	if err != nil {
		t.Errorf("Expected no error when checking existence, got %v", err)
	}

	if !exists {
		t.Error("Expected key to exist, but it doesn't")
	}

	// Check non-existent key
	nonExistentKey := "non_existent_key"
	exists, err = cache.Exists(nonExistentKey)
	if err != nil {
		t.Errorf("Expected no error when checking non-existent key, got %v", err)
	}

	if exists {
		t.Error("Expected key to not exist, but it does")
	}
}

func TestCacheSetWithExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Skipf("skipping test: could not connect to Redis: %v", err)
	}

	key := "test_expiration_key"
	value := "test_expiration_value"

	// Set a value with 1 second expiration
	err = cache.Set(key, value, 1*time.Second)
	if err != nil {
		t.Errorf("Expected no error when setting value, got %v", err)
	}

	// Check that it exists initially
	result, err := cache.Get(key)
	if err != nil {
		t.Errorf("Expected no error when getting value, got %v", err)
	}
	if result != value {
		t.Errorf("Expected value %s, got %s", value, result)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Check that it no longer exists
	result, err = cache.Get(key)
	if err != nil {
		t.Errorf("Expected no error when getting expired value, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string after expiration, got %s", result)
	}
}

func TestCacheClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache test in short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewCache(redisURL)
	if err != nil {
		t.Skipf("skipping test: could not connect to Redis: %v", err)
	}

	// Test that Close can be called without error
	err = cache.Close()
	if err != nil {
		t.Errorf("Expected no error when closing cache, got %v", err)
	}
}