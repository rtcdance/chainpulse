package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DistributedLock represents a distributed lock
type DistributedLock struct {
	mu              sync.RWMutex
	locks           map[string]*LockInfo
	lockTimeout     time.Duration
	metrics         *LockMetrics
}

// LockInfo contains information about a lock
type LockInfo struct {
	Key           string
	Owner         string
	AcquiredAt    time.Time
	ExpiresAt     time.Time
	RenewalCount  int
	Status        string // "locked", "released", "expired"
}

// LockMetrics tracks lock metrics
type LockMetrics struct {
	mu              sync.RWMutex
	LocksAcquired   int64
	LocksReleased   int64
	LocksFailed     int64
	LocksExpired    int64
	DeadlocksDetected int64
	AverageWaitTime time.Duration
	TotalWaitTime   time.Duration
}

// LockManager manages distributed locks
type LockManager struct {
	locks            *DistributedLock
	waitQueues       map[string][]*LockWaiter
	deadlockDetector *DeadlockDetector
	metrics          *LockMetrics
}

// LockWaiter represents a waiter for a lock
type LockWaiter struct {
	Owner    string
	WaitTime time.Time
	Done     chan bool
}

// DeadlockDetector detects deadlock situations
type DeadlockDetector struct {
	mu              sync.RWMutex
	lockGraph       map[string][]string // lock -> owners waiting for it
	ownerGraph      map[string][]string // owner -> locks it holds
	checkInterval   time.Duration
	lastCheckTime   time.Time
}

// NewDistributedLock creates a new distributed lock
func NewDistributedLock(lockTimeout time.Duration) *DistributedLock {
	return &DistributedLock{
		locks:       make(map[string]*LockInfo),
		lockTimeout: lockTimeout,
		metrics: &LockMetrics{},
	}
}

// AcquireLock acquires a lock
func (dl *DistributedLock) AcquireLock(ctx context.Context, key, owner string) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Check if lock already exists
	if lockInfo, exists := dl.locks[key]; exists {
		// Check if lock has expired
		if time.Now().After(lockInfo.ExpiresAt) {
			// Lock has expired, acquire it
			dl.locks[key] = &LockInfo{
				Key:        key,
				Owner:      owner,
				AcquiredAt: time.Now(),
				ExpiresAt:  time.Now().Add(dl.lockTimeout),
				Status:     "locked",
			}

			dl.metrics.mu.Lock()
			dl.metrics.LocksAcquired++
			dl.metrics.mu.Unlock()

			return nil
		}

		// Lock is still held by another owner
		if lockInfo.Owner != owner {
			dl.metrics.mu.Lock()
			dl.metrics.LocksFailed++
			dl.metrics.mu.Unlock()
			return fmt.Errorf("lock is held by %s", lockInfo.Owner)
		}

		// Same owner, renew the lock
		lockInfo.ExpiresAt = time.Now().Add(dl.lockTimeout)
		lockInfo.RenewalCount++
		return nil
	}

	// Lock doesn't exist, create it
	dl.locks[key] = &LockInfo{
		Key:        key,
		Owner:      owner,
		AcquiredAt: time.Now(),
		ExpiresAt:  time.Now().Add(dl.lockTimeout),
		Status:     "locked",
	}

	dl.metrics.mu.Lock()
	dl.metrics.LocksAcquired++
	dl.metrics.mu.Unlock()

	return nil
}

// ReleaseLock releases a lock
func (dl *DistributedLock) ReleaseLock(ctx context.Context, key, owner string) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	lockInfo, exists := dl.locks[key]
	if !exists {
		return fmt.Errorf("lock not found")
	}

	if lockInfo.Owner != owner {
		return fmt.Errorf("lock is held by %s, not %s", lockInfo.Owner, owner)
	}

	lockInfo.Status = "released"
	delete(dl.locks, key)

	dl.metrics.mu.Lock()
	dl.metrics.LocksReleased++
	dl.metrics.mu.Unlock()

	return nil
}

// IsLocked checks if a lock is held
func (dl *DistributedLock) IsLocked(key string) bool {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	lockInfo, exists := dl.locks[key]
	if !exists {
		return false
	}

	// Check if lock has expired
	if time.Now().After(lockInfo.ExpiresAt) {
		return false
	}

	return true
}

// GetLockInfo retrieves lock information
func (dl *DistributedLock) GetLockInfo(key string) (*LockInfo, error) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	lockInfo, exists := dl.locks[key]
	if !exists {
		return nil, fmt.Errorf("lock not found")
	}

	return lockInfo, nil
}

// CleanupExpiredLocks removes expired locks
func (dl *DistributedLock) CleanupExpiredLocks() {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now()
	for key, lockInfo := range dl.locks {
		if now.After(lockInfo.ExpiresAt) {
			lockInfo.Status = "expired"
			delete(dl.locks, key)

			dl.metrics.mu.Lock()
			dl.metrics.LocksExpired++
			dl.metrics.mu.Unlock()
		}
	}
}

// GetMetrics returns lock metrics
func (dl *DistributedLock) GetMetrics() map[string]interface{} {
	dl.metrics.mu.RLock()
	defer dl.metrics.mu.RUnlock()

	return map[string]interface{}{
		"locks_acquired":      dl.metrics.LocksAcquired,
		"locks_released":      dl.metrics.LocksReleased,
		"locks_failed":        dl.metrics.LocksFailed,
		"locks_expired":       dl.metrics.LocksExpired,
		"deadlocks_detected":  dl.metrics.DeadlocksDetected,
		"average_wait_time":   dl.metrics.AverageWaitTime.String(),
		"total_wait_time":     dl.metrics.TotalWaitTime.String(),
	}
}

// NewLockManager creates a new lock manager
func NewLockManager(lockTimeout time.Duration) *LockManager {
	return &LockManager{
		locks:      NewDistributedLock(lockTimeout),
		waitQueues: make(map[string][]*LockWaiter),
		deadlockDetector: &DeadlockDetector{
			lockGraph:     make(map[string][]string),
			ownerGraph:    make(map[string][]string),
			checkInterval: 30 * time.Second,
			lastCheckTime: time.Now(),
		},
		metrics: &LockMetrics{},
	}
}

// AcquireLockWithWait acquires a lock with waiting
func (lm *LockManager) AcquireLockWithWait(ctx context.Context, key, owner string, maxWait time.Duration) error {
	startTime := time.Now()

	for {
		// Try to acquire lock
		err := lm.locks.AcquireLock(ctx, key, owner)
		if err == nil {
			// Lock acquired
			waitTime := time.Since(startTime)
			lm.recordWaitTime(waitTime)
			return nil
		}

		// Check if we've exceeded max wait time
		if time.Since(startTime) > maxWait {
			lm.metrics.mu.Lock()
			lm.metrics.LocksFailed++
			lm.metrics.mu.Unlock()
			return fmt.Errorf("lock acquisition timeout")
		}

		// Wait a bit before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue trying
		}
	}
}

// ReleaseLock releases a lock
func (lm *LockManager) ReleaseLock(ctx context.Context, key, owner string) error {
	return lm.locks.ReleaseLock(ctx, key, owner)
}

// recordWaitTime records lock wait time
func (lm *LockManager) recordWaitTime(waitTime time.Duration) {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()

	lm.metrics.TotalWaitTime += waitTime
	if lm.metrics.LocksAcquired > 0 {
		lm.metrics.AverageWaitTime = lm.metrics.TotalWaitTime / time.Duration(lm.metrics.LocksAcquired)
	}
}

// DetectDeadlocks detects potential deadlock situations
func (lm *LockManager) DetectDeadlocks(ctx context.Context) error {
	lm.deadlockDetector.mu.Lock()
	defer lm.deadlockDetector.mu.Unlock()

	// Check if enough time has passed since last check
	if time.Since(lm.deadlockDetector.lastCheckTime) < lm.deadlockDetector.checkInterval {
		return nil
	}

	// Perform deadlock detection
	// This is a simplified version; in production, use cycle detection algorithm
	lm.deadlockDetector.lastCheckTime = time.Now()

	return nil
}

// GetMetrics returns lock manager metrics
func (lm *LockManager) GetMetrics() map[string]interface{} {
	return lm.locks.GetMetrics()
}

// NewDeadlockDetector creates a new deadlock detector
func NewDeadlockDetector() *DeadlockDetector {
	return &DeadlockDetector{
		lockGraph:     make(map[string][]string),
		ownerGraph:    make(map[string][]string),
		checkInterval: 30 * time.Second,
		lastCheckTime: time.Now(),
	}
}

// AddWaitEdge adds a wait edge to the graph
func (dd *DeadlockDetector) AddWaitEdge(owner, lock string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	// Add edge: owner waits for lock
	dd.ownerGraph[owner] = append(dd.ownerGraph[owner], lock)
	dd.lockGraph[lock] = append(dd.lockGraph[lock], owner)
}

// RemoveWaitEdge removes a wait edge from the graph
func (dd *DeadlockDetector) RemoveWaitEdge(owner, lock string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	// Remove edge: owner no longer waits for lock
	if owners, exists := dd.lockGraph[lock]; exists {
		for i, o := range owners {
			if o == owner {
				dd.lockGraph[lock] = append(owners[:i], owners[i+1:]...)
				break
			}
		}
	}

	if locks, exists := dd.ownerGraph[owner]; exists {
		for i, l := range locks {
			if l == lock {
				dd.ownerGraph[owner] = append(locks[:i], locks[i+1:]...)
				break
			}
		}
	}
}

// DetectCycle detects cycles in the wait graph
func (dd *DeadlockDetector) DetectCycle() bool {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for owner := range dd.ownerGraph {
		if !visited[owner] {
			if dd.hasCycle(owner, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCycle is a helper function for cycle detection
func (dd *DeadlockDetector) hasCycle(owner string, visited, recStack map[string]bool) bool {
	visited[owner] = true
	recStack[owner] = true

	// Check all locks this owner waits for
	for _, lock := range dd.ownerGraph[owner] {
		// Check all owners waiting for this lock
		for _, nextOwner := range dd.lockGraph[lock] {
			if !visited[nextOwner] {
				if dd.hasCycle(nextOwner, visited, recStack) {
					return true
				}
			} else if recStack[nextOwner] {
				return true
			}
		}
	}

	recStack[owner] = false
	return false
}
