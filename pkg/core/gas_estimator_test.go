package core

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEffectiveGasPrice_EIP1559(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		baseFee  *big.Int
		maxFee   *big.Int
		priority *big.Int
		want     int64
	}{
		{
			name:     "tip_below_maxFee",
			baseFee:  big.NewInt(100),
			maxFee:   big.NewInt(200),
			priority: big.NewInt(10),
			want:     110, // baseFee + priority = 100 + 10 = 110 < 200
		},
		{
			name:     "tip_exceeds_maxFee_capped",
			baseFee:  big.NewInt(190),
			maxFee:   big.NewInt(200),
			priority: big.NewInt(50),
			want:     200, // baseFee + priority = 190 + 50 = 240 > 200, capped at 200
		},
		{
			name:     "exact_match",
			baseFee:  big.NewInt(100),
			maxFee:   big.NewInt(200),
			priority: big.NewInt(100),
			want:     200, // baseFee + priority = 100 + 100 = 200 == maxFee
		},
		{
			name:     "zero_priority_fee",
			baseFee:  big.NewInt(100),
			maxFee:   big.NewInt(200),
			priority: big.NewInt(0),
			want:     100, // baseFee + 0 = 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffectiveGasPrice(tt.baseFee, nil, tt.maxFee, tt.priority)
			if got.Int64() != tt.want {
				t.Errorf("EffectiveGasPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveGasPrice_LegacyTx(t *testing.T) {
	t.Parallel()
	// Legacy transaction: maxFeePerGas is nil, gasPrice is set
	gasPrice := big.NewInt(500)
	got := EffectiveGasPrice(nil, gasPrice, nil, nil)
	if got.Cmp(gasPrice) != 0 {
		t.Errorf("EffectiveGasPrice for legacy tx = %v, want %v", got, gasPrice)
	}
}

func TestEffectiveGasPrice_NilBaseFee(t *testing.T) {
	t.Parallel()
	// Pre-EIP-1559 chain: baseFee is nil
	maxFee := big.NewInt(200)
	got := EffectiveGasPrice(nil, nil, maxFee, big.NewInt(10))
	if got.Cmp(maxFee) != 0 {
		t.Errorf("EffectiveGasPrice with nil baseFee = %v, want %v", got, maxFee)
	}
}

func TestEffectiveGasPrice_NilPriorityFee(t *testing.T) {
	t.Parallel()
	// EIP-1559 tx with nil maxPriorityFeePerGas — should default to 0
	baseFee := big.NewInt(30)
	maxFee := big.NewInt(100)
	got := EffectiveGasPrice(baseFee, nil, maxFee, nil)
	if got.Cmp(baseFee) != 0 {
		t.Errorf("EffectiveGasPrice with nil priorityFee = %v, want %v (baseFee)", got, baseFee)
	}
}

func TestPredictNextBaseFee(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		baseFee  *big.Int
		gasUsed  uint64
		gasLimit uint64
		wantGTE  int64 // result must be >= this
		wantLTE  int64 // result must be <= this
	}{
		{
			name:     "exactly_half_utilization_no_change",
			baseFee:  big.NewInt(100),
			gasUsed:  7500000, // exactly half of 15M
			gasLimit: 15000000,
			wantGTE:  99,
			wantLTE:  101,
		},
		{
			name:     "full_block_baseFee_increases",
			baseFee:  big.NewInt(100),
			gasUsed:  15000000, // 100% utilization
			gasLimit: 15000000,
			wantGTE:  112, // ~12.5% increase max
			wantLTE:  113,
		},
		{
			name:     "empty_block_baseFee_decreases",
			baseFee:  big.NewInt(100),
			gasUsed:  0,
			gasLimit: 15000000,
			wantGTE:  87, // ~12.5% decrease max
			wantLTE:  88,
		},
		{
			name:     "above_target_slight_increase",
			baseFee:  big.NewInt(1e9), // 1 Gwei
			gasUsed:  10000000,
			gasLimit: 15000000,
			wantGTE:  1.04e9,
			wantLTE:  1.05e9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PredictNextBaseFee(tt.baseFee, tt.gasUsed, tt.gasLimit)
			if got.Int64() < tt.wantGTE || got.Int64() > tt.wantLTE {
				t.Errorf("PredictNextBaseFee() = %v, want [%v, %v]", got, tt.wantGTE, tt.wantLTE)
			}
		})
	}
}

func TestPredictNextBaseFee_EdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("nil_baseFee", func(t *testing.T) {
		got := PredictNextBaseFee(nil, 100, 200)
		if got.Int64() != 0 {
			t.Errorf("PredictNextBaseFee(nil) = %v, want 0", got)
		}
	})

	t.Run("zero_gasLimit", func(t *testing.T) {
		got := PredictNextBaseFee(big.NewInt(100), 50, 0)
		if got.Int64() != 100 {
			t.Errorf("PredictNextBaseFee with gasLimit=0 = %v, want 100 (unchanged)", got)
		}
	})

	t.Run("negative_baseFee_returns_zero", func(t *testing.T) {
		got := PredictNextBaseFee(big.NewInt(-1), 100, 200)
		if got.Int64() != 0 {
			t.Errorf("PredictNextBaseFee with negative baseFee = %v, want 0", got)
		}
	})
}

func TestGasCost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		gasUsed uint64
		price   *big.Int
		want    int64
	}{
		{"normal", 21000, big.NewInt(100), 2100000},
		{"zero_gas", 0, big.NewInt(100), 0},
		{"nil_price", 21000, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GasCost(tt.gasUsed, tt.price)
			if got.Int64() != tt.want {
				t.Errorf("GasCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCongestionLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		gasUsed  uint64
		gasLimit uint64
		want     float64
	}{
		{"half_full", 7500000, 15000000, 1.0}, // exactly at target (gasLimit/2)
		{"empty", 0, 15000000, 0.0},
		{"full", 15000000, 15000000, 2.0},        // 2x target
		{"over_full", 20000000, 15000000, 2.667}, // well above target
		{"zero_limit", 100, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CongestionLevel(tt.gasUsed, tt.gasLimit)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("CongestionLevel() = %v, want ~%v", got, tt.want)
			}
		})
	}
}

func TestCongestionBand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level float64
		want  string
	}{
		{0.1, "low"},
		{0.6, "moderate"},
		{0.9, "high"},
		{1.2, "congested"},
		{1.6, "severe"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := CongestionBand(tt.level)
			if got != tt.want {
				t.Errorf("CongestionBand(%v) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestMaxPriorityFeeSuggestion(t *testing.T) {
	t.Parallel()
	t.Run("empty_transactions_returns_default", func(t *testing.T) {
		got := MaxPriorityFeeSuggestion(big.NewInt(100), nil)
		want := big.NewInt(1e9) // 1 Gwei default
		if got.Cmp(want) != 0 {
			t.Errorf("MaxPriorityFeeSuggestion(empty) = %v, want %v", got, want)
		}
	})

	t.Run("nil_baseFee_returns_zero", func(t *testing.T) {
		got := MaxPriorityFeeSuggestion(nil, nil)
		if got.Int64() != 0 {
			t.Errorf("MaxPriorityFeeSuggestion(nil baseFee) = %v, want 0", got)
		}
	})

	t.Run("computes_median_tip", func(t *testing.T) {
		baseFee := big.NewInt(100)
		txs := []Transaction{
			{MaxFeePerGas: big.NewInt(200), MaxPriorityFeePerGas: big.NewInt(5)},  // tip=5
			{MaxFeePerGas: big.NewInt(200), MaxPriorityFeePerGas: big.NewInt(10)}, // tip=10
			{MaxFeePerGas: big.NewInt(200), MaxPriorityFeePerGas: big.NewInt(15)}, // tip=15
		}
		got := MaxPriorityFeeSuggestion(baseFee, txs)
		if got.Int64() != 10 {
			t.Errorf("MaxPriorityFeeSuggestion(median) = %v, want 10", got)
		}
	})

	t.Run("legacy_tx_tip_from_gasPrice", func(t *testing.T) {
		baseFee := big.NewInt(100)
		txs := []Transaction{
			{GasPrice: big.NewInt(115)}, // tip = 115 - 100 = 15
			{GasPrice: big.NewInt(110)}, // tip = 110 - 100 = 10
		}
		got := MaxPriorityFeeSuggestion(baseFee, txs)
		if got.Int64() != 15 { // median of [10, 15] at index len/2=1 is 15
			t.Errorf("MaxPriorityFeeSuggestion(legacy) = %v, want 15", got)
		}
	})

	t.Run("tip_capped_at_effective", func(t *testing.T) {
		baseFee := big.NewInt(195)
		txs := []Transaction{
			{MaxFeePerGas: big.NewInt(200), MaxPriorityFeePerGas: big.NewInt(50)},
			// effective = min(200, 195+50) = 200, actual tip = 200-195 = 5 (not 50)
		}
		got := MaxPriorityFeeSuggestion(baseFee, txs)
		if got.Int64() != 5 {
			t.Errorf("MaxPriorityFeeSuggestion(capped) = %v, want 5", got)
		}
	})
}

func TestIsEIP1559Tx(t *testing.T) {
	t.Parallel()
	t.Run("eip1559_tx", func(t *testing.T) {
		tx := &Transaction{MaxFeePerGas: big.NewInt(200), MaxPriorityFeePerGas: big.NewInt(10)}
		if !IsEIP1559Tx(tx) {
			t.Error("IsEIP1559Tx() = false, want true")
		}
	})

	t.Run("legacy_tx", func(t *testing.T) {
		tx := &Transaction{GasPrice: big.NewInt(100)}
		if IsEIP1559Tx(tx) {
			t.Error("IsEIP1559Tx() = true, want false")
		}
	})

	t.Run("nil_tx", func(t *testing.T) {
		if IsEIP1559Tx(nil) {
			t.Error("IsEIP1559Tx(nil) = true, want false")
		}
	})
}

func TestAssessBlockCongestion(t *testing.T) {
	t.Parallel()
	t.Run("normal_block", func(t *testing.T) {
		block := &Block{
			GasUsed:  10000000,
			GasLimit: 15000000,
			BaseFee:  big.NewInt(1e9),
		}
		assess := AssessBlockCongestion(block)
		// CongestionLevel = gasUsed / gasTarget, gasTarget = gasLimit/2 = 7.5M
		// Level = 10M / 7.5M = 1.333
		if assess.Level < 1.33 || assess.Level > 1.34 {
			t.Errorf("Level = %v, want ~1.333", assess.Level)
		}
		if assess.Band != "congested" {
			t.Errorf("Band = %v, want congested", assess.Band)
		}
		if assess.GasTargetPct < 133 || assess.GasTargetPct > 134 {
			t.Errorf("GasTargetPct = %v, want ~133.3", assess.GasTargetPct)
		}
		if assess.NextBaseFee == nil || assess.NextBaseFee.Int64() <= 1e9 {
			t.Errorf("NextBaseFee = %v, want > 1e9 (should increase)", assess.NextBaseFee)
		}
	})

	t.Run("nil_block", func(t *testing.T) {
		assess := AssessBlockCongestion(nil)
		if assess.Level != 0 {
			t.Errorf("Level = %v, want 0", assess.Level)
		}
		if assess.Band != "unknown" {
			t.Errorf("Band = %v, want unknown", assess.Band)
		}
	})
}

func TestEffectiveGasPrice_LargeValues(t *testing.T) {
	t.Parallel()
	// Test with realistic Gwei values
	baseFee := new(big.Int).Mul(big.NewInt(30), big.NewInt(1e9)) // 30 Gwei
	maxFee := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e9)) // 100 Gwei
	priority := new(big.Int).Mul(big.NewInt(2), big.NewInt(1e9)) // 2 Gwei

	got := EffectiveGasPrice(baseFee, nil, maxFee, priority)
	want := new(big.Int).Mul(big.NewInt(32), big.NewInt(1e9)) // 32 Gwei
	if got.Cmp(want) != 0 {
		t.Errorf("EffectiveGasPrice(gwei) = %v, want %v", got, want)
	}
}

// --- EIP-4844 Blob Gas Tests ---

func TestCalculateBlobBaseFee(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		excessBlobGas uint64
		wantGTE       int64 // result must be >= this
	}{
		{"zero_excess", 0, 1},         // minimum 1 wei
		{"low_excess", 100, 1},        // ~1 wei
		{"medium_excess", 1000000, 1}, // ~1 wei (still small relative to update fraction)
		{"high_excess", 100000000, 1}, // higher excess, still >= 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBlobBaseFee(tt.excessBlobGas)
			if got.Cmp(big.NewInt(tt.wantGTE)) < 0 {
				t.Errorf("CalculateBlobBaseFee(%d) = %v, want >= %d", tt.excessBlobGas, got, tt.wantGTE)
			}
		})
	}

	// Verify monotonicity: higher excess → higher fee
	// Use realistic mainnet-scale excess values. With factor=1 and updateFraction=3338477,
	// excess values below ~3.3M produce output=1 (integer division rounds to 0).
	// At higher excess, the fake-exponential growth becomes visible.
	fee0 := CalculateBlobBaseFee(0)
	fee1 := CalculateBlobBaseFee(10_000_000)
	fee2 := CalculateBlobBaseFee(50_000_000)
	fee3 := CalculateBlobBaseFee(100_000_000)
	if fee1.Cmp(fee0) <= 0 {
		t.Errorf("fee(10M)=%v should be > fee(0)=%v", fee1, fee0)
	}
	if fee2.Cmp(fee1) <= 0 {
		t.Errorf("fee(50M)=%v should be > fee(10M)=%v", fee2, fee1)
	}
	if fee3.Cmp(fee2) <= 0 {
		t.Errorf("fee(100M)=%v should be > fee(50M)=%v", fee3, fee2)
	}

	// Verify spec-known value: fake_exponential(1, 0, 3338477) = 1
	if v := CalculateBlobBaseFee(0); v.Int64() != 1 {
		t.Errorf("CalculateBlobBaseFee(0) = %d, want 1", v.Int64())
	}
}

func TestPredictNextExcessBlobGas(t *testing.T) {
	t.Parallel()
	targetGas := uint64(BlobTxTargetBlobCount) * uint64(BlobTxBlobGasPerBlob) // 3 * 131072

	tests := []struct {
		name        string
		excess      uint64
		blobGasUsed uint64
		want        uint64
	}{
		{
			name:        "at_target_no_change",
			excess:      1000,
			blobGasUsed: targetGas, // exactly target
			want:        1000,      // no change
		},
		{
			name:        "above_target_excess_increases",
			excess:      1000,
			blobGasUsed: targetGas + 131072, // 1 extra blob
			want:        1000 + 131072,
		},
		{
			name:        "below_target_excess_decreases",
			excess:      200000,
			blobGasUsed: targetGas - 131072, // 1 fewer blob
			want:        200000 - 131072,
		},
		{
			name:        "excess_clamps_to_zero",
			excess:      100,
			blobGasUsed: 0, // no blobs
			want:        0, // can't go negative
		},
		{
			name:        "zero_excess_stays_near_zero",
			excess:      0,
			blobGasUsed: 0,
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PredictNextExcessBlobGas(tt.excess, tt.blobGasUsed)
			if got != tt.want {
				t.Errorf("PredictNextExcessBlobGas(%d, %d) = %d, want %d", tt.excess, tt.blobGasUsed, got, tt.want)
			}
		})
	}
}

func TestEstimateBlobGasCost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		numBlobs      int
		blobBaseFee   *big.Int
		maxFeePerBlob *big.Int
		wantGTE       int64
	}{
		{"zero_blobs", 0, big.NewInt(100), nil, 0},
		{"one_blob_base_fee", 1, big.NewInt(100), nil, 13107200},          // 1 * 131072 * 100
		{"three_blobs_base_fee", 3, big.NewInt(100), nil, 39321600},       // 3 * 131072 * 100
		{"max_fee_caps", 2, big.NewInt(1000), big.NewInt(500), 131072000}, // 2 * 131072 * 500 (capped)
		{"nil_base_fee_uses_min", 1, nil, nil, 131072},                    // 1 * 131072 * 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateBlobGasCost(tt.numBlobs, tt.blobBaseFee, tt.maxFeePerBlob)
			if got.Int64() != tt.wantGTE {
				t.Errorf("EstimateBlobGasCost() = %v, want %v", got, tt.wantGTE)
			}
		})
	}
}

func TestIsCancunBlock(t *testing.T) {
	t.Parallel()
	t.Run("nil_block", func(t *testing.T) {
		if IsCancunBlock(nil) {
			t.Error("IsCancunBlock(nil) = true, want false")
		}
	})

	t.Run("no_beacon_root", func(t *testing.T) {
		block := &Block{Number: 19000000}
		if IsCancunBlock(block) {
			t.Error("IsCancunBlock without beacon root should be false")
		}
	})

	t.Run("with_beacon_root", func(t *testing.T) {
		root := common.HexToHash("0x1234")
		block := &Block{Number: 19426587, ParentBeaconBlockRoot: &root}
		if !IsCancunBlock(block) {
			t.Error("IsCancunBlock with beacon root should be true")
		}
	})
}

func TestBlobCountFromGas(t *testing.T) {
	t.Parallel()
	tests := []struct {
		blobGas uint64
		want    int
	}{
		{0, 0},
		{131072, 1}, // exactly 1 blob
		{262144, 2}, // exactly 2 blobs
		{393216, 3}, // exactly 3 blobs (target)
		{786432, 6}, // exactly 6 blobs (max)
	}

	for _, tt := range tests {
		got := BlobCountFromGas(tt.blobGas)
		if got != tt.want {
			t.Errorf("BlobCountFromGas(%d) = %d, want %d", tt.blobGas, got, tt.want)
		}
	}
}

// --- EIP-2930 Access List Gas Tests ---

func TestAccessListGasCost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		baseGas    uint64
		accessList []AccessListEntry
		want       uint64
	}{
		{
			name:       "no_access_list",
			baseGas:    21000,
			accessList: nil,
			want:       21000,
		},
		{
			name:    "one_address_no_keys",
			baseGas: 21000,
			accessList: []AccessListEntry{
				{Address: common.Address{1}},
			},
			want: 21000 + 2400,
		},
		{
			name:    "one_address_two_keys",
			baseGas: 21000,
			accessList: []AccessListEntry{
				{Address: common.Address{1}, StorageKeys: []common.Hash{{}, {}}},
			},
			want: 21000 + 2400 + 2*1900,
		},
		{
			name:    "two_addresses",
			baseGas: 21000,
			accessList: []AccessListEntry{
				{Address: common.Address{1}, StorageKeys: []common.Hash{{}}},
				{Address: common.Address{2}, StorageKeys: []common.Hash{{}, {}}},
			},
			want: 21000 + 2400 + 1900 + 2400 + 2*1900,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AccessListGasCost(tt.baseGas, tt.accessList)
			if got != tt.want {
				t.Errorf("AccessListGasCost() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAccessListGasSavings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		coldSloads          uint64
		coldAccountAccesses uint64
		wantPositive        bool
	}{
		{"no_accesses", 0, 0, false},
		{"one_cold_sload", 1, 0, true},         // 100 savings > 0
		{"many_cold_sloads", 10, 0, true},      // 10 * 100 = 1000 savings
		{"one_address_access", 0, 1, true},     // 100 savings
		{"many_address_accesses", 0, 10, true}, // 10 * 100 = 1000 savings
		{"mixed", 5, 5, true},                  // 500 + 500 = 1000 savings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			savings := AccessListGasSavings(tt.coldSloads, tt.coldAccountAccesses)
			if (savings > 0) != tt.wantPositive {
				t.Errorf("AccessListGasSavings(%d, %d) = %d, wantPositive=%v", tt.coldSloads, tt.coldAccountAccesses, savings, tt.wantPositive)
			}
		})
	}

	// Verify exact savings per access type
	// Storage: 2100 - 1900 - 100 = 100 per key
	storageSavings := AccessListGasSavings(1, 0)
	if storageSavings != 100 {
		t.Errorf("per-key storage savings = %d, want 100", storageSavings)
	}
	// Address: 2600 - 2400 - 100 = 100 per address
	addressSavings := AccessListGasSavings(0, 1)
	if addressSavings != 100 {
		t.Errorf("per-address savings = %d, want 100", addressSavings)
	}
}

func TestShouldUseAccessList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		coldSloads          uint64
		coldAccountAccesses uint64
		minSavings          uint64
		want                bool
	}{
		{"no_accesses", 0, 0, 0, false},
		{"below_threshold", 5, 0, 1000, false}, // 500 < 1000
		{"at_threshold", 10, 0, 1000, true},    // 1000 >= 1000
		{"above_threshold", 20, 0, 1000, true}, // 2000 >= 1000
		{"default_threshold", 10, 0, 0, true},  // uses default 1000
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUseAccessList(tt.coldSloads, tt.coldAccountAccesses, tt.minSavings)
			if got != tt.want {
				t.Errorf("ShouldUseAccessList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildAccessListForTransfer(t *testing.T) {
	t.Parallel()
	from := common.Address{0x01}
	to := common.Address{0x02}

	t.Run("without_balance_slot", func(t *testing.T) {
		al := BuildAccessListForTransfer(from, to, false)
		if len(al) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(al))
		}
		if al[0].Address != from {
			t.Errorf("entry[0].Address = %v, want %v", al[0].Address, from)
		}
		if al[1].Address != to {
			t.Errorf("entry[1].Address = %v, want %v", al[1].Address, to)
		}
		if len(al[0].StorageKeys) != 0 {
			t.Errorf("expected no storage keys, got %d", len(al[0].StorageKeys))
		}
	})

	t.Run("with_balance_slot", func(t *testing.T) {
		al := BuildAccessListForTransfer(from, to, true)
		if len(al) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(al))
		}
		if len(al[0].StorageKeys) != 1 {
			t.Errorf("expected 1 storage key, got %d", len(al[0].StorageKeys))
		}
		// Verify the storage key is deterministic (keccak256 of padded address + slot 0)
		if al[0].StorageKeys[0] == (common.Hash{}) {
			t.Error("storage key should not be zero hash")
		}
	})
}
