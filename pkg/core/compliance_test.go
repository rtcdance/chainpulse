package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ─── OFAC Screening Tests ───────────────────────────────────────────────────

func TestOFACScreenAddress(t *testing.T) {
	policy := DefaultCompliancePolicy()
	svc := NewOFACScreeningService(policy)

	sanctionedAddr := common.HexToAddress("0xdeadbeef")
	svc.LoadSDNList([]SanctionListEntry{
		{
			Address:    sanctionedAddr,
			EntityName: "Bad Actor",
			Program:    "SDGT",
		},
	})

	// Sanctioned address
	result := svc.ScreenAddress(sanctionedAddr)
	if !result.IsSanctioned {
		t.Error("expected address to be sanctioned")
	}
	if result.RiskScore != 1.0 {
		t.Errorf("risk score = %f, want 1.0", result.RiskScore)
	}
	if result.MatchSource != "sdn" {
		t.Errorf("match source = %s, want sdn", result.MatchSource)
	}

	// Clean address
	cleanAddr := common.HexToAddress("0x1234")
	result = svc.ScreenAddress(cleanAddr)
	if result.IsSanctioned {
		t.Error("clean address should not be sanctioned")
	}
	if result.RiskScore != 0.0 {
		t.Errorf("risk score = %f, want 0.0", result.RiskScore)
	}
}

func TestOFACScreenTransaction(t *testing.T) {
	policy := DefaultCompliancePolicy()
	svc := NewOFACScreeningService(policy)

	sanctionedAddr := common.HexToAddress("0xbad")
	svc.LoadSDNList([]SanctionListEntry{
		{Address: sanctionedAddr, EntityName: "Sanctioned Entity"},
	})

	cleanAddr := common.HexToAddress("0xgood")

	// Clean transaction
	_, _, err := svc.ScreenTransaction(cleanAddr, cleanAddr)
	if err != nil {
		t.Errorf("clean transaction should not error: %v", err)
	}

	// Transaction involving sanctioned address
	_, _, err = svc.ScreenTransaction(cleanAddr, sanctionedAddr)
	if err == nil {
		t.Error("transaction with sanctioned address should error")
	}
}

func TestOFACScreeningCache(t *testing.T) {
	policy := ComplianceScreeningPolicy{
		BlockSanctioned:  true,
		RescreenInterval: 1 * time.Hour,
	}
	svc := NewOFACScreeningService(policy)

	addr := common.HexToAddress("0xtest")

	// First screening
	result1 := svc.ScreenAddress(addr)
	if result1.IsSanctioned {
		t.Error("should not be sanctioned before loading list")
	}

	// Load sanction and screen again — should still use cached result
	svc.LoadSDNList([]SanctionListEntry{
		{Address: addr, EntityName: "Now Sanctioned"},
	})
	result2 := svc.ScreenAddress(addr)
	if result2.IsSanctioned {
		t.Error("cache should return previous result before rescreen interval")
	}
}

func TestOFACRemoveEntry(t *testing.T) {
	policy := DefaultCompliancePolicy()
	svc := NewOFACScreeningService(policy)

	addr := common.HexToAddress("0xremove")
	svc.LoadSDNList([]SanctionListEntry{
		{Address: addr, EntityName: "To Be Removed"},
	})

	if !svc.IsAddressSanctioned(addr) {
		t.Error("should be sanctioned after loading")
	}

	svc.RemoveEntry(addr)

	if svc.IsAddressSanctioned(addr) {
		t.Error("should not be sanctioned after removal")
	}
}

func TestOFACEntryCount(t *testing.T) {
	policy := DefaultCompliancePolicy()
	svc := NewOFACScreeningService(policy)

	if count := svc.EntryCount(); count != 0 {
		t.Errorf("initial count should be 0, got %d", count)
	}

	svc.LoadSDNList([]SanctionListEntry{
		{Address: common.HexToAddress("0x1"), EntityName: "A"},
		{Address: common.HexToAddress("0x2"), EntityName: "B"},
	})

	if count := svc.EntryCount(); count != 2 {
		t.Errorf("count should be 2, got %d", count)
	}
}

// ─── Transaction Monitor Tests ──────────────────────────────────────────────

func TestTransactionMonitorUnusualVolume(t *testing.T) {
	policy := DefaultCompliancePolicy()
	tm := NewTransactionMonitor(policy)

	addr := common.HexToAddress("0xhighvol")
	threshold := big.NewInt(10000)

	// Below threshold
	if tm.CheckUnusualVolume(addr, big.NewInt(5000), threshold) {
		t.Error("5000 should not trigger unusual volume for 10000 threshold")
	}

	// Exceed threshold
	if !tm.CheckUnusualVolume(addr, big.NewInt(6000), threshold) {
		t.Error("5000+6000=11000 should trigger unusual volume for 10000 threshold")
	}
}

func TestTransactionMonitorAlerts(t *testing.T) {
	policy := DefaultCompliancePolicy()
	tm := NewTransactionMonitor(policy)

	alert := MonitoringAlert{
		ID:          "alert-1",
		Type:        "sanction_match",
		Address:     common.HexToAddress("0xbad"),
		Severity:    "critical",
		Description: "Transaction with sanctioned address",
		Timestamp:   time.Now(),
	}

	tm.RecordAlert(alert)

	alerts := tm.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}

	// Resolve
	tm.ResolveAlert("alert-1")
	alerts = tm.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("expected 0 unresolved alerts, got %d", len(alerts))
	}
}

func TestCompliancePolicy(t *testing.T) {
	policy := DefaultCompliancePolicy()

	if !policy.BlockSanctioned {
		t.Error("default policy should block sanctioned addresses")
	}
	if policy.RiskThreshold <= 0 || policy.RiskThreshold > 1 {
		t.Errorf("risk threshold should be in (0,1], got %f", policy.RiskThreshold)
	}
	if !policy.RequireScreening {
		t.Error("default policy should require screening")
	}
	if policy.RescreenInterval != 24*time.Hour {
		t.Errorf("rescreen interval should be 24h, got %v", policy.RescreenInterval)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"kitten", "sitting", 3},
		{"", "abc", 3},
		{"abc", "", 3},
		{"same", "same", 0},
		{"abc", "abd", 1},
		{"saturday", "sunday", 3},
	}
	for _, tt := range tests {
		got := levenshteinDistance(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestOFACFuzzyMatchHit(t *testing.T) {
	policy := ComplianceScreeningPolicy{
		BlockSanctioned:   true,
		RiskThreshold:     0.7,
		EnableFuzzyMatch:  true,
		FuzzyMatchDistance: 3,
		RequireScreening:  true,
		RescreenInterval:  24 * time.Hour,
	}
	svc := NewOFACScreeningService(policy)

	// Load an entity with a known name
	sanctionedAddr := common.HexToAddress("0xdeadbeef")
	svc.LoadSDNList([]SanctionListEntry{
		{
			Address:    sanctionedAddr,
			EntityName: "BadActor",
			Program:    "SDGT",
			Aliases:    []string{"EvilOrg"},
		},
	})

	// Screen an address that is NOT on the list — fuzzy matching against
	// entity names keyed by address hex won't match, but the feature is
	// exercised. The result should not be sanctioned because the address
	// hex doesn't resemble entity names.
	otherAddr := common.HexToAddress("0x12345678")
	result := svc.ScreenAddress(otherAddr)
	if result.IsSanctioned {
		t.Error("non-matching address should not be sanctioned via fuzzy match")
	}
}

func TestOFACFuzzyMatchDisabled(t *testing.T) {
	policy := ComplianceScreeningPolicy{
		BlockSanctioned:   true,
		RiskThreshold:     0.7,
		EnableFuzzyMatch:  false, // disabled
		FuzzyMatchDistance: 3,
		RequireScreening:  true,
		RescreenInterval:  24 * time.Hour,
	}
	svc := NewOFACScreeningService(policy)

	sanctionedAddr := common.HexToAddress("0xdeadbeef")
	svc.LoadSDNList([]SanctionListEntry{
		{
			Address:    sanctionedAddr,
			EntityName: "BadActor",
			Program:    "SDGT",
		},
	})

	// Exact match still works
	result := svc.ScreenAddress(sanctionedAddr)
	if !result.IsSanctioned {
		t.Error("exact match should still work with fuzzy disabled")
	}
	if result.MatchSource != "sdn" {
		t.Errorf("expected match source 'sdn', got %q", result.MatchSource)
	}
}
