package cache

import (
	"container/list"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Ensure list package is properly imported
var _ *list.List

// AdvancedInMemoryCachePlugin provides advanced in-memory cache with LRU eviction
type AdvancedInMemoryCachePlugin struct {
	*BaseCachePlugin
	data            map[string]*CacheEntry
	lruList         *list.List
	lruMap          map[string]*list.Element
	maxSize         int64
	currentSize     int64
	maxEntries      int64
	currentEntries  int64
	evictionPolicy  string // "LRU" or "FIFO"
	cleanupInterval time.Duration
	stopCleanup     chan bool
	cleanupDone     chan struct{}
}

// NewAdvancedInMemoryCachePlugin creates a new advanced in-memory cache plugin
func NewAdvancedInMemoryCachePlugin(logger core.Logger, metricsCollector core.MetricsCollector) *AdvancedInMemoryCachePlugin {
	return &AdvancedInMemoryCachePlugin{
		BaseCachePlugin: NewBaseCachePlugin(logger, metricsCollector),
		data:            make(map[string]*CacheEntry),
		lruList:         list.New(),
		lruMap:          make(map[string]*list.Element),
		maxSize:         1024 * 1024 * 100, // 100MB default
		maxEntries:      10000,             // 10k entries default
		evictionPolicy:  "LRU",
		cleanupInterval: 30 * time.Second,
		stopCleanup:     make(chan bool, 1),
		cleanupDone:     make(chan struct{}),
	}
}

// Initialize initializes the advanced cache plugin
func (p *AdvancedInMemoryCachePlugin) Initialize(config *core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("advanced cache plugin already initialized")
	}

	if config == nil {
		return fmt.Errorf("config is required")
	}

	p.config = config
	p.initialized = true

	// Configure from environment if available
	// Note: core.Config doesn't have CacheMaxSize/CacheMaxEntries fields
	// These are set to defaults in NewAdvancedInMemoryCachePlugin

	p.logger.Info("Advanced in-memory cache plugin initialized", core.LogKeyComponent, "advanced_cache", "maxSize", p.maxSize, "maxEntries", p.maxEntries, "evictionPolicy", p.evictionPolicy)

	return nil
}

// Start starts the advanced cache plugin
func (p *AdvancedInMemoryCachePlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("advanced cache plugin not initialized")
	}

	if p.running {
		return fmt.Errorf("advanced cache plugin already running")
	}

	p.running = true
	p.lastHealthCheck = &core.HealthStatus{
		Status:    "healthy",
		Message:   "Advanced cache plugin started",
		Timestamp: time.Now(),
	}

	// Start cleanup goroutine
	go p.cleanupExpiredEntries()

	p.logger.Info("Advanced cache plugin started", core.LogKeyComponent, "advanced_cache")

	return nil
}

// Stop stops the advanced cache plugin
func (p *AdvancedInMemoryCachePlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("advanced cache plugin not running")
	}

	p.running = false

	// Stop cleanup goroutine
	select {
	case p.stopCleanup <- true:
	default:
	}

	p.lastHealthCheck = &core.HealthStatus{
		Status:    "healthy",
		Message:   "Advanced cache plugin stopped",
		Timestamp: time.Now(),
	}

	p.logger.Info("Advanced cache plugin stopped", core.LogKeyComponent, "advanced_cache")

	return nil
}

// Get retrieves a value from cache
func (p *AdvancedInMemoryCachePlugin) Get(key string) (*CacheEntry, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	p.mu.Lock()

	if !p.running {
		p.mu.Unlock()
		return nil, fmt.Errorf("advanced cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		p.mu.Unlock()
		p.RecordMiss()
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("advanced_cache_miss", 1, map[string]string{})
		}
		return nil, nil
	}

	// Check if entry has expired
	if entry.ExpiresAt.Before(time.Now()) {
		p.removeEntry(key)
		p.mu.Unlock()
		p.RecordEviction()
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("advanced_cache_eviction", 1, map[string]string{})
		}
		return nil, nil
	}

	// Update LRU position
	if elem, ok := p.lruMap[key]; ok {
		p.lruList.MoveToFront(elem)
	}

	p.mu.Unlock()
	p.RecordHit()
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("advanced_cache_hit", 1, map[string]string{})
	}

	return entry, nil
}

// Set stores a value in cache with TTL
func (p *AdvancedInMemoryCachePlugin) Set(entry *CacheEntry) error {
	if entry == nil {
		return fmt.Errorf("entry is required")
	}

	if entry.Key == "" {
		return fmt.Errorf("entry key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("advanced cache plugin not running")
	}

	// Calculate expiration time
	if entry.TTL > 0 {
		entry.ExpiresAt = time.Now().Add(time.Duration(entry.TTL) * time.Second)
	} else {
		entry.ExpiresAt = time.Now().Add(24 * time.Hour) // default 24 hours
	}

	// Calculate entry size
	entrySize := p.calculateEntrySize(entry)

	// Check if we need to evict entries
	if p.currentSize+entrySize > p.maxSize || p.currentEntries+1 > p.maxEntries {
		p.evictEntries(entrySize)
	}

	// Remove old entry if exists
	if oldEntry, exists := p.data[entry.Key]; exists {
		p.currentSize -= p.calculateEntrySize(oldEntry)
		p.currentEntries--
		p.removeFromLRU(entry.Key)
	}

	// Add new entry
	p.data[entry.Key] = entry
	p.currentSize += entrySize
	p.currentEntries++

	// Update LRU
	elem := p.lruList.PushFront(entry.Key)
	p.lruMap[entry.Key] = elem

	p.metricsCollector.RecordCounter("advanced_cache_set", 1, map[string]string{})
	p.metricsCollector.RecordGauge("advanced_cache_size", float64(p.currentSize), map[string]string{})
	p.metricsCollector.RecordGauge("advanced_cache_entries", float64(p.currentEntries), map[string]string{})

	return nil
}

// Delete removes a value from cache
func (p *AdvancedInMemoryCachePlugin) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("advanced cache plugin not running")
	}

	p.removeEntry(key)

	p.metricsCollector.RecordCounter("advanced_cache_delete", 1, map[string]string{})
	p.metricsCollector.RecordGauge("advanced_cache_size", float64(p.currentSize), map[string]string{})
	p.metricsCollector.RecordGauge("advanced_cache_entries", float64(p.currentEntries), map[string]string{})

	return nil
}

// Clear clears all cache entries
func (p *AdvancedInMemoryCachePlugin) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("advanced cache plugin not running")
	}

	p.data = make(map[string]*CacheEntry)
	p.lruList = list.New()
	p.lruMap = make(map[string]*list.Element)
	p.currentSize = 0
	p.currentEntries = 0

	p.metricsCollector.RecordCounter("advanced_cache_clear", 1, map[string]string{})

	return nil
}

// GetStats returns cache statistics
func (p *AdvancedInMemoryCachePlugin) GetStats() *CacheStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &CacheStats{
		HitCount:      p.hitCount,
		MissCount:     p.missCount,
		EvictionCount: p.evictionCount,
		TotalSize:     p.currentSize,
		EntryCount:    p.currentEntries,
	}
}

// GetMaxSize returns the maximum cache size
func (p *AdvancedInMemoryCachePlugin) GetMaxSize() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.maxSize
}

// SetMaxSize sets the maximum cache size
func (p *AdvancedInMemoryCachePlugin) SetMaxSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("max size must be positive")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.maxSize = size

	// Evict if necessary
	if p.currentSize > p.maxSize {
		p.evictEntries(0)
	}

	return nil
}

// GetMaxEntries returns the maximum number of entries
func (p *AdvancedInMemoryCachePlugin) GetMaxEntries() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.maxEntries
}

// SetMaxEntries sets the maximum number of entries
func (p *AdvancedInMemoryCachePlugin) SetMaxEntries(count int64) error {
	if count <= 0 {
		return fmt.Errorf("max entries must be positive")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.maxEntries = count

	// Evict if necessary
	if p.currentEntries > p.maxEntries {
		p.evictEntries(0)
	}

	return nil
}

// GetEvictionPolicy returns the eviction policy
func (p *AdvancedInMemoryCachePlugin) GetEvictionPolicy() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.evictionPolicy
}

// SetEvictionPolicy sets the eviction policy
func (p *AdvancedInMemoryCachePlugin) SetEvictionPolicy(policy string) error {
	if policy != "LRU" && policy != "FIFO" {
		return fmt.Errorf("invalid eviction policy: %s", policy)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.evictionPolicy = policy

	return nil
}

// GetCurrentSize returns the current cache size
func (p *AdvancedInMemoryCachePlugin) GetCurrentSize() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentSize
}

// GetCurrentEntries returns the current number of entries
func (p *AdvancedInMemoryCachePlugin) GetCurrentEntries() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentEntries
}

// Private helper methods

func (p *AdvancedInMemoryCachePlugin) calculateEntrySize(entry *CacheEntry) int64 {
	if entry == nil {
		return 0
	}

	size := int64(len(entry.Key))

	if entry.Value != nil {
		if data, err := json.Marshal(entry.Value); err == nil {
			size += int64(len(data))
		}
	}

	return size
}

func (p *AdvancedInMemoryCachePlugin) removeEntry(key string) {
	if entry, exists := p.data[key]; exists {
		p.currentSize -= p.calculateEntrySize(entry)
		p.currentEntries--
		delete(p.data, key)
		p.removeFromLRU(key)
	}
}

func (p *AdvancedInMemoryCachePlugin) removeFromLRU(key string) {
	if elem, ok := p.lruMap[key]; ok {
		p.lruList.Remove(elem)
		delete(p.lruMap, key)
	}
}

func (p *AdvancedInMemoryCachePlugin) evictEntries(requiredSize int64) {
	// Evict expired entries first
	p.evictExpiredEntries()

	// If still need space, evict based on policy
	for p.currentSize+requiredSize > p.maxSize || p.currentEntries >= p.maxEntries {
		if p.lruList.Len() == 0 {
			break
		}

		// Get the least recently used (back of list)
		elem := p.lruList.Back()
		if elem == nil {
			break
		}

		key, ok := elem.Value.(string)
		if !ok {
			break
		}
		p.removeEntry(key)
		p.recordEvictionUnlocked()
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("advanced_cache_lru_eviction", 1, map[string]string{})
		}
	}
}

func (p *AdvancedInMemoryCachePlugin) evictExpiredEntries() {
	now := time.Now()
	keysToDelete := make([]string, 0)

	for key, entry := range p.data {
		if entry.ExpiresAt.Before(now) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		p.removeEntry(key)
		p.recordEvictionUnlocked()
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("advanced_cache_expiration_eviction", 1, map[string]string{})
		}
	}
}

// recordEvictionUnlocked records an eviction without acquiring a lock (must be called while holding lock)
func (p *AdvancedInMemoryCachePlugin) recordEvictionUnlocked() {
	p.evictionCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_eviction", 1, map[string]string{})
	}
}

func (p *AdvancedInMemoryCachePlugin) cleanupExpiredEntries() {
	defer close(p.cleanupDone)

	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			if p.running {
				p.evictExpiredEntries()
			}
			p.mu.Unlock()

		case <-p.stopCleanup:
			return
		}
	}
}
