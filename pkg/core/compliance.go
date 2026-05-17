package core

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ─── OFAC Sanctions Screening ────────────────────────────────────────────────

// ScreeningResult represents the result of a compliance screening.
type ScreeningResult struct {
	Address      common.Address `json:"address"`
	IsSanctioned bool           `json:"is_sanctioned"`
	RiskScore    float64        `json:"risk_score"`              // 0.0 (safe) to 1.0 (high risk)
	MatchSource  string         `json:"match_source,omitempty"`  // "sdn", "entity", "fuzzy"
	MatchedEntry string         `json:"matched_entry,omitempty"` // the matched list entry
	ScreenedAt   time.Time      `json:"screened_at"`
}

// SanctionListEntry represents an entry on the OFAC SDN (Specially Designated Nationals) list.
type SanctionListEntry struct {
	Address    common.Address `json:"address"`
	EntityName string         `json:"entity_name"`
	Program    string         `json:"program"` // e.g., "SDGT", "IRAN", "DPRK"
	ListedDate time.Time      `json:"listed_date"`
	Aliases    []string       `json:"aliases,omitempty"`
}

// ComplianceScreeningPolicy defines the policy for compliance screening.
type ComplianceScreeningPolicy struct {
	BlockSanctioned    bool          `json:"block_sanctioned"`     // block transactions involving sanctioned addresses
	RiskThreshold      float64       `json:"risk_threshold"`       // risk score threshold for flagging (0.0-1.0)
	EnableFuzzyMatch   bool          `json:"enable_fuzzy_match"`   // enable fuzzy matching on entity names
	FuzzyMatchDistance int           `json:"fuzzy_match_distance"` // Levenshtein distance for fuzzy match
	RequireScreening   bool          `json:"require_screening"`    // require screening before transaction processing
	RescreenInterval   time.Duration `json:"rescreen_interval"`    // how often to re-screen addresses
}

// DefaultCompliancePolicy returns a sensible default policy.
func DefaultCompliancePolicy() ComplianceScreeningPolicy {
	return ComplianceScreeningPolicy{
		BlockSanctioned:  true,
		RiskThreshold:    0.7,
		EnableFuzzyMatch: false,
		RequireScreening: true,
		RescreenInterval: 24 * time.Hour,
	}
}

// OFACScreeningService performs address screening against sanction lists.
type OFACScreeningService struct {
	mu     sync.RWMutex
	policy ComplianceScreeningPolicy

	// address → SanctionListEntry (direct match)
	sdnList map[common.Address]*SanctionListEntry

	// entity name → address for fuzzy matching
	entityIndex map[string]common.Address

	// address → last screening time + result
	screenedCache map[common.Address]*ScreeningResult
}

// NewOFACScreeningService creates a new screening service.
func NewOFACScreeningService(policy ComplianceScreeningPolicy) *OFACScreeningService {
	return &OFACScreeningService{
		policy:        policy,
		sdnList:       make(map[common.Address]*SanctionListEntry),
		entityIndex:   make(map[string]common.Address),
		screenedCache: make(map[common.Address]*ScreeningResult),
	}
}

// LoadSDNList loads OFAC SDN entries into the screening service.
func (s *OFACScreeningService) LoadSDNList(entries []SanctionListEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range entries {
		entry := &entries[i]
		s.sdnList[entry.Address] = entry
		s.entityIndex[entry.EntityName] = entry.Address
		for _, alias := range entry.Aliases {
			s.entityIndex[alias] = entry.Address
		}
	}
}

// ScreenAddress screens an address against the sanction list.
func (s *OFACScreeningService) ScreenAddress(addr common.Address) ScreeningResult {
	// Fast path: check cache under read lock
	s.mu.RLock()
	if cached, ok := s.screenedCache[addr]; ok {
		if time.Since(cached.ScreenedAt) < s.policy.RescreenInterval {
			s.mu.RUnlock()
			return *cached
		}
	}
	s.mu.RUnlock()

	// Slow path: perform screening, then cache under write lock
	result := ScreeningResult{
		Address:    addr,
		ScreenedAt: time.Now(),
		RiskScore:  0.0,
	}

	// Check SDN list under read lock (sdnList is read-only after construction)
	s.mu.RLock()
	if entry, found := s.sdnList[addr]; found {
		result.IsSanctioned = true
		result.RiskScore = 1.0
		result.MatchSource = "sdn"
		result.MatchedEntry = entry.EntityName
	}
	s.mu.RUnlock()

	// Fuzzy matching: if exact match failed and fuzzy is enabled, check entity
	// names via Levenshtein distance. This catches near-matches like typos or
	// transliteration variants of sanctioned entity names.
	if !result.IsSanctioned && s.policy.EnableFuzzyMatch && s.policy.FuzzyMatchDistance > 0 {
		s.mu.RLock()
		bestDist := s.policy.FuzzyMatchDistance + 1 // start above threshold
		bestName := ""
		for entityName := range s.entityIndex {
			dist := levenshteinDistance(entityName, addr.Hex())
			if dist < bestDist {
				bestDist = dist
				bestName = entityName
			}
		}
		s.mu.RUnlock()

		if bestDist <= s.policy.FuzzyMatchDistance {
			result.IsSanctioned = true
			result.MatchSource = "fuzzy"
			result.MatchedEntry = bestName
			// Graduated risk: closer match → higher risk score
			maxLen := len(bestName)
			if len(addr.Hex()) > maxLen {
				maxLen = len(addr.Hex())
			}
			if maxLen > 0 {
				result.RiskScore = 1.0 - float64(bestDist)/float64(maxLen)
			}
			if result.RiskScore < s.policy.RiskThreshold {
				// Below threshold: downgrade from sanctioned to high-risk flag
				result.IsSanctioned = false
				result.RiskScore = 1.0 - float64(bestDist)/float64(maxLen)
			}
		}
	}

	// Cache the result under write lock
	s.mu.Lock()
	s.screenedCache[addr] = &result
	s.mu.Unlock()

	return result
}

// ScreenTransaction screens all parties in a transaction.
func (s *OFACScreeningService) ScreenTransaction(from, to common.Address) (fromResult, toResult ScreeningResult, err error) {
	fromResult = s.ScreenAddress(from)
	toResult = s.ScreenAddress(to)

	if s.policy.BlockSanctioned && (fromResult.IsSanctioned || toResult.IsSanctioned) {
		sanctioned := fromResult
		if toResult.IsSanctioned {
			sanctioned = toResult
		}
		return fromResult, toResult, fmt.Errorf("transaction involves sanctioned address: %s (%s)",
			sanctioned.Address.Hex(), sanctioned.MatchedEntry)
	}

	return fromResult, toResult, nil
}

// IsAddressSanctioned is a convenience method for quick checks.
func (s *OFACScreeningService) IsAddressSanctioned(addr common.Address) bool {
	return s.ScreenAddress(addr).IsSanctioned
}

// RemoveEntry removes a sanction entry (e.g., after delisting).
func (s *OFACScreeningService) RemoveEntry(addr common.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.sdnList[addr]; ok {
		delete(s.entityIndex, entry.EntityName)
		for _, alias := range entry.Aliases {
			delete(s.entityIndex, alias)
		}
	}
	delete(s.sdnList, addr)
	delete(s.screenedCache, addr)
}

// EntryCount returns the number of entries in the SDN list.
func (s *OFACScreeningService) EntryCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sdnList)
}

// ─── Transaction Monitoring ─────────────────────────────────────────────────

// MonitoringAlert represents a compliance alert triggered by transaction monitoring.
type MonitoringAlert struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"` // "sanction_match", "high_risk", "unusual_volume", "structuring"
	Address     common.Address `json:"address"`
	TxHash      common.Hash    `json:"tx_hash"`
	Severity    string         `json:"severity"` // "low", "medium", "high", "critical"
	Description string         `json:"description"`
	Timestamp   time.Time      `json:"timestamp"`
	Resolved    bool           `json:"resolved"`
}

// TransactionMonitor monitors transactions for suspicious patterns.
type TransactionMonitor struct {
	mu     sync.Mutex
	policy ComplianceScreeningPolicy
	alerts []MonitoringAlert

	// Volume tracking per address for unusual volume detection
	addressVolume map[common.Address]*volumeTracker
}

type volumeTracker struct {
	dailyTotal *big.Int
	lastReset  time.Time
	txCount    int
}

// NewTransactionMonitor creates a new transaction monitor.
func NewTransactionMonitor(policy ComplianceScreeningPolicy) *TransactionMonitor {
	return &TransactionMonitor{
		policy:        policy,
		alerts:        make([]MonitoringAlert, 0),
		addressVolume: make(map[common.Address]*volumeTracker),
	}
}

// CheckUnusualVolume checks if an address has unusual transaction volume.
// Returns true if the address exceeds the daily threshold.
func (tm *TransactionMonitor) CheckUnusualVolume(addr common.Address, amount *big.Int, dailyThreshold *big.Int) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	vt, ok := tm.addressVolume[addr]
	if !ok || time.Since(vt.lastReset) > 24*time.Hour {
		vt = &volumeTracker{
			dailyTotal: big.NewInt(0),
			lastReset:  time.Now(),
		}
		tm.addressVolume[addr] = vt
	}

	vt.dailyTotal.Add(vt.dailyTotal, amount)
	vt.txCount++

	return vt.dailyTotal.Cmp(dailyThreshold) > 0
}

// RecordAlert records a new compliance alert.
func (tm *TransactionMonitor) RecordAlert(alert MonitoringAlert) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.alerts = append(tm.alerts, alert)
}

// GetAlerts returns all unresolved alerts.
func (tm *TransactionMonitor) GetAlerts() []MonitoringAlert {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var unresolved []MonitoringAlert
	for _, a := range tm.alerts {
		if !a.Resolved {
			unresolved = append(unresolved, a)
		}
	}
	return unresolved
}

// ResolveAlert marks an alert as resolved.
func (tm *TransactionMonitor) ResolveAlert(id string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for i := range tm.alerts {
		if tm.alerts[i].ID == id {
			tm.alerts[i].Resolved = true
			return
		}
	}
}

// levenshteinDistance computes the Levenshtein edit distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions,
// substitutions) required to transform a into b.
func levenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two-row optimization for O(min(m,n)) space
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}

	prev := make([]int, la+1)
	curr := make([]int, la+1)

	for i := 0; i <= la; i++ {
		prev[i] = i
	}

	for j := 1; j <= lb; j++ {
		curr[0] = j
		for i := 1; i <= la; i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[i] = min(
				prev[i]+1,      // deletion
				curr[i-1]+1,    // insertion
				prev[i-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[la]
}
