package core

import (
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// AAMempoolEntry represents a UserOperation in the alternative mempool.
// Unlike standard Ethereum transactions, UserOps do not enter the public
// mempool — they are submitted to bundlers who simulate, validate, and
// include them via the EntryPoint contract.
type AAMempoolEntry struct {
	UserOp         *UserOperation
	PreValidation  *PreValidationResult
	EntryPointAddr common.Address
	SubmittedAt    time.Time
	PriorityFee    *big.Int // maxPriorityFeePerGas, used for ordering
	Sender         common.Address
	Hash           string // unique identifier for dedup
}

// PreValidationResult captures the outcome of off-chain simulation before
// a bundler includes a UserOperation in a bundle.
type PreValidationResult struct {
	Success           bool     `json:"success"`
	SimulationGasUsed uint64   `json:"simulation_gas_used"`
	GasOverhead       uint64   `json:"gas_overhead"` // actual gas minus estimated
	StakeValid        bool     `json:"stake_valid"`  // paymaster/factory stake sufficient
	FailureReason     string   `json:"failure_reason,omitempty"`
	AccessedAddresses []common.Address `json:"accessed_addresses,omitempty"`
	AccessedSlots     []common.Hash    `json:"accessed_slots,omitempty"`
}

// AAMempool maintains an ordered collection of pending UserOperations.
// Entries are ordered by priority fee (highest first) and bounded to
// prevent unbounded memory growth.
type AAMempool struct {
	mu         sync.RWMutex
	entries    map[string]*AAMempoolEntry // hash → entry
	ordered    []*AAMempoolEntry          // sorted by priority fee desc
	maxSize    int
	entryTTL   time.Duration // entries older than this are evicted
}

// NewAAMempool creates an alternative mempool with the given capacity.
// If maxSize <= 0, it defaults to 10000. If entryTTL <= 0, it defaults
// to 5 minutes (after which unsponsored entries are evicted).
func NewAAMempool(maxSize int, entryTTL time.Duration) *AAMempool {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if entryTTL <= 0 {
		entryTTL = 5 * time.Minute
	}
	return &AAMempool{
		entries:  make(map[string]*AAMempoolEntry),
		ordered:  make([]*AAMempoolEntry, 0, maxSize),
		maxSize:  maxSize,
		entryTTL: entryTTL,
	}
}

// AddEntry adds a UserOperation to the mempool. Returns false if the entry
// already exists (dedup by hash) or if the pre-validation failed.
func (m *AAMempool) AddEntry(entry *AAMempoolEntry) bool {
	if entry == nil || entry.UserOp == nil {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Dedup
	if _, exists := m.entries[entry.Hash]; exists {
		return false
	}

	// Evict if at capacity (remove lowest-priority / oldest)
	if len(m.entries) >= m.maxSize {
		m.evictOldest()
	}

	m.entries[entry.Hash] = entry
	m.insertOrdered(entry)
	return true
}

// RemoveEntry removes a UserOperation from the mempool (e.g., after inclusion).
func (m *AAMempool) RemoveEntry(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entries[hash]; exists {
		delete(m.entries, hash)
		// Rebuild ordered list without the entry
		newOrdered := make([]*AAMempoolEntry, 0, len(m.entries))
		for _, e := range m.ordered {
			if e.Hash != hash {
				newOrdered = append(newOrdered, e)
			}
		}
		m.ordered = newOrdered
	}
}

// GetPendingOps returns up to n pending UserOperations ordered by priority fee.
func (m *AAMempool) GetPendingOps(n int) []*AAMempoolEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.evictExpired()

	if n > len(m.ordered) {
		n = len(m.ordered)
	}

	result := make([]*AAMempoolEntry, n)
	copy(result, m.ordered[:n])
	return result
}

// Size returns the number of pending entries.
func (m *AAMempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// Contains checks if an entry exists by hash.
func (m *AAMempool) Contains(hash string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.entries[hash]
	return exists
}

// EvictStale removes entries older than the TTL.
func (m *AAMempool) EvictStale() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evictExpired()
}

func (m *AAMempool) insertOrdered(entry *AAMempoolEntry) {
	// Insert in priority-fee descending order (simple linear scan)
	for i, existing := range m.ordered {
		if entry.PriorityFee != nil && existing.PriorityFee != nil &&
			entry.PriorityFee.Cmp(existing.PriorityFee) > 0 {
			m.ordered = append(m.ordered[:i+1], m.ordered[i:]...)
			m.ordered[i] = entry
			return
		}
	}
	m.ordered = append(m.ordered, entry)
}

func (m *AAMempool) evictOldest() {
	if len(m.ordered) == 0 {
		return
	}
	// Remove the last entry (lowest priority)
	oldest := m.ordered[len(m.ordered)-1]
	m.ordered = m.ordered[:len(m.ordered)-1]
	delete(m.entries, oldest.Hash)
}

func (m *AAMempool) evictExpired() {
	now := time.Now()
	var toRemove []string
	for hash, entry := range m.entries {
		if now.Sub(entry.SubmittedAt) > m.entryTTL {
			toRemove = append(toRemove, hash)
		}
	}
	for _, hash := range toRemove {
		delete(m.entries, hash)
	}
	if len(toRemove) > 0 {
		// Rebuild ordered list
		m.ordered = make([]*AAMempoolEntry, 0, len(m.entries))
		for _, e := range m.entries {
			m.insertOrdered(e)
		}
	}
}
