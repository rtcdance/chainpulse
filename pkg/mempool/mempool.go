// Package mempool provides an ERC-4337 UserOperation mempool with priority fee ordering.
package mempool

import (
	"container/heap"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// AAMempoolEntry represents a UserOperation in the alternative mempool.
type AAMempoolEntry struct {
	UserOp         *blockchain.UserOperation
	PreValidation  *PreValidationResult
	EntryPointAddr common.Address
	SubmittedAt    time.Time
	PriorityFee    *big.Int
	Sender         common.Address
	Hash           string
}

// PreValidationResult captures the outcome of off-chain simulation.
type PreValidationResult struct {
	Success           bool             `json:"success"`
	SimulationGasUsed uint64           `json:"simulation_gas_used"`
	GasOverhead       uint64           `json:"gas_overhead"`
	StakeValid        bool             `json:"stake_valid"`
	FailureReason     string           `json:"failure_reason,omitempty"`
	AccessedAddresses []common.Address `json:"accessed_addresses,omitempty"`
	AccessedSlots     []common.Hash    `json:"accessed_slots,omitempty"`
}

type priorityFeeHeap []*AAMempoolEntry

func (h priorityFeeHeap) Len() int { return len(h) }
func (h priorityFeeHeap) Less(i, j int) bool {
	if h[i].PriorityFee != nil && h[j].PriorityFee != nil {
		if c := h[i].PriorityFee.Cmp(h[j].PriorityFee); c != 0 {
			return c > 0
		}
	}
	return h[i].SubmittedAt.Before(h[j].SubmittedAt)
}
func (h priorityFeeHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *priorityFeeHeap) Push(x any)   { *h = append(*h, x.(*AAMempoolEntry)) }
func (h *priorityFeeHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return item
}

// AAMempool maintains an ordered collection of pending UserOperations.
type AAMempool struct {
	mu       sync.RWMutex
	entries  map[string]*AAMempoolEntry
	ordered  priorityFeeHeap
	maxSize  int
	entryTTL time.Duration
}

// NewAAMempool creates an alternative mempool with the given capacity.
func NewAAMempool(maxSize int, entryTTL time.Duration) *AAMempool {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if entryTTL <= 0 {
		entryTTL = 5 * time.Minute
	}
	return &AAMempool{
		entries:  make(map[string]*AAMempoolEntry),
		ordered:  make(priorityFeeHeap, 0, maxSize),
		maxSize:  maxSize,
		entryTTL: entryTTL,
	}
}

// AddEntry adds a UserOperation to the mempool.
func (m *AAMempool) AddEntry(entry *AAMempoolEntry) bool {
	if entry == nil || entry.UserOp == nil {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entries[entry.Hash]; exists {
		return false
	}

	if len(m.entries) >= m.maxSize {
		m.evictOldest()
	}

	m.entries[entry.Hash] = entry
	heap.Push(&m.ordered, entry)
	return true
}

// RemoveEntry removes a UserOperation from the mempool.
func (m *AAMempool) RemoveEntry(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entries[hash]; exists {
		delete(m.entries, hash)
		for i, e := range m.ordered {
			if e.Hash == hash {
				heap.Remove(&m.ordered, i)
				return
			}
		}
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

	cp := make(priorityFeeHeap, len(m.ordered))
	for i, e := range m.ordered {
		cp[i] = e
	}

	result := make([]*AAMempoolEntry, n)
	for i := 0; i < n; i++ {
		result[i] = heap.Pop(&cp).(*AAMempoolEntry)
	}
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

func (m *AAMempool) evictOldest() {
	if len(m.ordered) == 0 {
		return
	}
	minIdx := 0
	for i, e := range m.ordered {
		if e.PriorityFee != nil && m.ordered[minIdx].PriorityFee != nil &&
			e.PriorityFee.Cmp(m.ordered[minIdx].PriorityFee) < 0 {
			minIdx = i
		}
	}
	removed := heap.Remove(&m.ordered, minIdx).(*AAMempoolEntry)
	delete(m.entries, removed.Hash)
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
		m.ordered = make(priorityFeeHeap, 0, len(m.entries))
		for _, e := range m.entries {
			m.ordered = append(m.ordered, e)
		}
		heap.Init(&m.ordered)
	}
}
