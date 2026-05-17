package cache

import (
	"context"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

type InMemoryCache struct {
	name      string
	version   string
	data      map[string]*cacheItem
	mu        sync.RWMutex
	started   bool
	stats     core.CacheStats
	done      chan struct{}
	closeOnce sync.Once
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		name:    "inmemory-cache",
		version: "1.0.0",
		data:    make(map[string]*cacheItem),
		done:    make(chan struct{}),
	}
}

func (c *InMemoryCache) Name() string    { return c.name }
func (c *InMemoryCache) Version() string { return c.version }

func (c *InMemoryCache) Initialize(config core.Config) error {
	return nil
}

func (c *InMemoryCache) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started = true
	go c.cleanup()
	return nil
}

func (c *InMemoryCache) Stop() error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = false
	c.mu.Unlock()
	c.closeOnce.Do(func() {
		close(c.done)
	})
	c.mu.Lock()
	c.data = make(map[string]*cacheItem)
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) Health() error {
	if !c.started {
		return core.NewSystemError(core.ErrorTypeCritical, core.ErrorCodeInternalError, "cache not started", nil)
	}

	return nil
}

func (c *InMemoryCache) HealthCheck(_ context.Context) error {
	if !c.started {
		return core.NewSystemError(core.ErrorTypeCritical, core.ErrorCodeInternalError, "cache not started", nil)
	}

	return nil
}

func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	item, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || time.Now().After(item.expiresAt) {
		c.mu.Lock()
		c.stats.MissCount++
		c.mu.Unlock()
		return nil, nil
	}

	c.mu.Lock()
	c.stats.HitCount++
	c.mu.Unlock()
	return item.value, nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *InMemoryCache) GetStats() core.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.HitCount + c.stats.MissCount
	if total > 0 {
		c.stats.HitRate = float64(c.stats.HitCount) / float64(total)
	}
	return c.stats
}

func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			if !c.started {
				c.mu.Unlock()
				return
			}

			now := time.Now()
			for key, item := range c.data {
				if now.After(item.expiresAt) {
					delete(c.data, key)
					c.stats.EvictionCount++
				}
			}
			c.mu.Unlock()
		}
	}
}
