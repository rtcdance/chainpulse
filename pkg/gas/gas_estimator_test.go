package gas

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestEffectiveGasPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		baseFee              *big.Int
		gasPrice             *big.Int
		maxFeePerGas         *big.Int
		maxPriorityFeePerGas *big.Int
		expected             *big.Int
	}{
		{
			name:     "legacy tx with gas price",
			gasPrice: big.NewInt(50),
			expected: big.NewInt(50),
		},
		{
			name:                 "eip1559 normal case",
			baseFee:              big.NewInt(100),
			maxFeePerGas:         big.NewInt(120),
			maxPriorityFeePerGas: big.NewInt(5),
			expected:             big.NewInt(105),
		},
		{
			name:                 "eip1559 capped by maxFeePerGas",
			baseFee:              big.NewInt(100),
			maxFeePerGas:         big.NewInt(102),
			maxPriorityFeePerGas: big.NewInt(5),
			expected:             big.NewInt(102),
		},
		{
			name:                 "eip1559 no priority fee",
			baseFee:              big.NewInt(100),
			maxFeePerGas:         big.NewInt(200),
			maxPriorityFeePerGas: nil,
			expected:             big.NewInt(100),
		},
		{
			name:                 "eip1559 no base fee",
			maxFeePerGas:         big.NewInt(120),
			maxPriorityFeePerGas: big.NewInt(5),
			expected:             big.NewInt(120),
		},
		{
			name:     "all nil",
			expected: big.NewInt(0),
		},
		{
			name:     "base fee only, no gas price",
			baseFee:  big.NewInt(50),
			expected: big.NewInt(50),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EffectiveGasPrice(tt.baseFee, tt.gasPrice, tt.maxFeePerGas, tt.maxPriorityFeePerGas)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPredictNextBaseFee(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		parentBF  *big.Int
		gasUsed   uint64
		gasLimit  uint64
		wantAbove *big.Int
		wantBelow *big.Int
	}{
		{
			name:     "gas above target",
			parentBF: big.NewInt(100),
			gasUsed:  20_000_000,
			gasLimit: 30_000_000,
		},
		{
			name:     "gas below target",
			parentBF: big.NewInt(100),
			gasUsed:  10_000_000,
			gasLimit: 30_000_000,
		},
		{
			name:     "gas at target",
			parentBF: big.NewInt(100),
			gasUsed:  15_000_000,
			gasLimit: 30_000_000,
		},
		{
			name:     "nil parent base fee",
			parentBF: nil,
			gasUsed:  10_000_000,
			gasLimit: 30_000_000,
		},
		{
			name:     "zero parent base fee",
			parentBF: big.NewInt(0),
			gasUsed:  10_000_000,
			gasLimit: 30_000_000,
		},
		{
			name:     "zero gas limit",
			parentBF: big.NewInt(100),
			gasUsed:  10_000_000,
			gasLimit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PredictNextBaseFee(tt.parentBF, tt.gasUsed, tt.gasLimit)
			assert.NotNil(t, result)
			if tt.parentBF == nil || tt.parentBF.Sign() <= 0 {
				assert.Equal(t, big.NewInt(0), result)
			}
		})
	}
}

func TestGasCost(t *testing.T) {
	t.Parallel()
	cost := GasCost(21_000, big.NewInt(50e9))
	expected := new(big.Int).Mul(big.NewInt(21_000), big.NewInt(50e9))
	assert.Equal(t, expected, cost)

	nilCost := GasCost(21_000, nil)
	assert.Equal(t, big.NewInt(0), nilCost)

	zeroGas := GasCost(0, big.NewInt(50e9))
	assert.Equal(t, big.NewInt(0), zeroGas)
}

func TestCongestionLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		gasUsed  uint64
		gasLimit uint64
		expected float64
	}{
		{"full target", 15_000_000, 30_000_000, 1.0},
		{"half target", 7_500_000, 30_000_000, 0.5},
		{"over target", 22_500_000, 30_000_000, 1.5},
		{"zero gas limit", 10_000_000, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CongestionLevel(tt.gasUsed, tt.gasLimit)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestCongestionBand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level    float64
		expected string
	}{
		{0.25, "low"},
		{0.60, "moderate"},
		{0.85, "high"},
		{1.20, "congested"},
		{2.00, "severe"},
		{0.49, "low"},
		{0.50, "moderate"},
		{0.74, "moderate"},
		{0.75, "high"},
		{0.99, "high"},
		{1.00, "congested"},
		{1.49, "congested"},
		{1.50, "severe"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := CongestionBand(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAssessBlockCongestion(t *testing.T) {
	t.Parallel()
	block := &blockchain.Block{
		Number:   100,
		Hash:     common.HexToHash("0xabc"),
		BaseFee:  big.NewInt(50e9),
		GasUsed:  10_000_000,
		GasLimit: 30_000_000,
	}

	result := AssessBlockCongestion(block)
	assert.NotNil(t, result)
	assert.Equal(t, "moderate", result.Band)

	nilBlock := AssessBlockCongestion(nil)
	assert.NotNil(t, nilBlock)
	assert.Equal(t, "unknown", nilBlock.Band)
	assert.Equal(t, float64(0), nilBlock.Level)
}

func TestMaxPriorityFeeSuggestion(t *testing.T) {
	t.Parallel()
	baseFee := big.NewInt(50e9)

	// Empty transactions
	result := MaxPriorityFeeSuggestion(baseFee, nil)
	assert.True(t, result.Sign() > 0)

	// With transactions
	txs := []blockchain.Transaction{
		{
			MaxFeePerGas:         big.NewInt(60e9),
			MaxPriorityFeePerGas: big.NewInt(2e9),
		},
		{
			MaxFeePerGas:         big.NewInt(70e9),
			MaxPriorityFeePerGas: big.NewInt(3e9),
		},
	}
	result = MaxPriorityFeeSuggestion(baseFee, txs)
	assert.True(t, result.Sign() > 0)

	// Legacy transactions
	legacyTxs := []blockchain.Transaction{
		{GasPrice: big.NewInt(55e9)},
		{GasPrice: big.NewInt(60e9)},
	}
	result = MaxPriorityFeeSuggestion(baseFee, legacyTxs)
	assert.True(t, result.Sign() > 0)

	// Nil base fee with empty txs
	nilResult := MaxPriorityFeeSuggestion(nil, nil)
	assert.Equal(t, big.NewInt(0), nilResult)
}

func TestIsEIP1559Tx(t *testing.T) {
	t.Parallel()
	assert.False(t, IsEIP1559Tx(nil))
	assert.False(t, IsEIP1559Tx(&blockchain.Transaction{}))
	assert.True(t, IsEIP1559Tx(&blockchain.Transaction{
		MaxFeePerGas:         big.NewInt(100),
		MaxPriorityFeePerGas: big.NewInt(1),
	}))
}

func TestIsCancunBlock(t *testing.T) {
	t.Parallel()
	assert.False(t, IsCancunBlock(nil))
	assert.False(t, IsCancunBlock(&blockchain.Block{}))
	root := common.HexToHash("0x1234")
	assert.True(t, IsCancunBlock(&blockchain.Block{ParentBeaconBlockRoot: &root}))
}

func TestCalculateBlobBaseFee(t *testing.T) {
	t.Parallel()
	// Excess = 0 -> minimum blob gas price
	result := CalculateBlobBaseFee(0)
	assert.Equal(t, big.NewInt(1), result)

	// With some excess
	result = CalculateBlobBaseFee(131072)
	assert.True(t, result.Sign() > 0)

	// Large excess
	result = CalculateBlobBaseFee(10000000)
	assert.True(t, result.Sign() > 0)
}

func TestPredictNextExcessBlobGas(t *testing.T) {
	t.Parallel()
	targetBlobGas := uint64(BlobTxTargetBlobCount) * uint64(BlobTxBlobGasPerBlob)

	// Above target
	next := PredictNextExcessBlobGas(0, targetBlobGas+131072)
	assert.True(t, next > 0)

	// At target
	next = PredictNextExcessBlobGas(0, targetBlobGas)
	assert.Equal(t, uint64(0), next)

	// Below target, excess smaller than gap
	next = PredictNextExcessBlobGas(100, targetBlobGas)
	assert.Equal(t, uint64(0), next)

	// Below target, excess larger than gap
	next = PredictNextExcessBlobGas(200000, targetBlobGas)
	assert.True(t, next > 0)
}

func TestEstimateBlobGasCost(t *testing.T) {
	t.Parallel()
	// Zero blobs
	cost := EstimateBlobGasCost(0, big.NewInt(100), nil)
	assert.Equal(t, big.NewInt(0), cost)

	// Negative blobs should be handled safely
	cost = EstimateBlobGasCost(-1, big.NewInt(100), nil)
	assert.Equal(t, big.NewInt(0), cost)

	// Normal case
	cost = EstimateBlobGasCost(1, big.NewInt(100), nil)
	assert.True(t, cost.Sign() > 0)

	// Nil base fee uses minimum
	cost = EstimateBlobGasCost(1, nil, nil)
	assert.True(t, cost.Sign() > 0)

	// Max fee per blob gas caps
	cost = EstimateBlobGasCost(1, big.NewInt(100), big.NewInt(50))
	assert.True(t, cost.Sign() > 0)
}

func TestBlobCountFromGas(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, BlobCountFromGas(0))
	assert.Equal(t, 1, BlobCountFromGas(uint64(BlobTxBlobGasPerBlob)))
	assert.Equal(t, 3, BlobCountFromGas(3*uint64(BlobTxBlobGasPerBlob)))
}

func TestAccessListGasCost(t *testing.T) {
	t.Parallel()
	// Empty access list
	cost := AccessListGasCost(21_000, nil)
	assert.Equal(t, uint64(21_000), cost)

	// With entries
	entries := []AccessListEntry{
		{
			Address:     common.Address{},
			StorageKeys: []common.Hash{{}},
		},
	}
	cost = AccessListGasCost(21_000, entries)
	assert.True(t, cost > 21_000)
}

func TestAccessListGasSavings(t *testing.T) {
	t.Parallel()
	// No savings with no access
	savings := AccessListGasSavings(0, 0)
	assert.Equal(t, int64(0), savings)

	// With cold accesses, should have savings
	savings = AccessListGasSavings(5, 3)
	assert.True(t, savings > 0)
}

func TestShouldUseAccessList(t *testing.T) {
	t.Parallel()
	// Should not use with low counts
	assert.False(t, ShouldUseAccessList(0, 0, 0))

	// Should use with significant savings
	assert.True(t, ShouldUseAccessList(10, 5, 0))

	// Should not use with min savings exceeding actual
	assert.False(t, ShouldUseAccessList(1, 0, 999999))
}

func TestBuildAccessListForTransfer(t *testing.T) {
	t.Parallel()
	from := common.HexToAddress("0x1234")
	to := common.HexToAddress("0x5678")

	// Without balance slot
	entries := BuildAccessListForTransfer(from, to, false)
	assert.Len(t, entries, 2)

	// With balance slot
	entries = BuildAccessListForTransfer(from, to, true)
	assert.Len(t, entries, 2)
	assert.Len(t, entries[0].StorageKeys, 1)
}

func TestTransientStorageGasCost(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(0), TransientStorageGasCost(0, 0))
	assert.Equal(t, uint64(200), TransientStorageGasCost(1, 1))
	assert.Equal(t, uint64(500), TransientStorageGasCost(3, 2))
}

func TestTransientVsPermanentSavings(t *testing.T) {
	t.Parallel()
	// No savings with no operations
	savings := TransientVsPermanentSavings(0, 0, 0)
	assert.Equal(t, uint64(0), savings)

	// Transient is cheaper than permanent for fresh writes
	savings = TransientVsPermanentSavings(1, 0, 0)
	// Permanent cost = 1*20000 = 20000, Transient cost = 1*100 = 100
	assert.True(t, savings > 0)
}

func TestPredictNextBaseFeeEdgeCases(t *testing.T) {
	t.Parallel()
	// All gas used, base fee zero -> should return 1
	fee := PredictNextBaseFee(big.NewInt(0), 30_000_000, 30_000_000)
	assert.Equal(t, big.NewInt(1), fee)
}

func TestEffectiveGasPriceDeepCopy(t *testing.T) {
	t.Parallel()
	// Verify the result is a copy, not the original
	original := big.NewInt(100)
	result := EffectiveGasPrice(nil, original, nil, nil)
	assert.Equal(t, original, result)
	result.Add(result, big.NewInt(1))
	assert.NotEqual(t, original, result)
}

func TestMaxPriorityFeeSuggestionNilBaseFee(t *testing.T) {
	t.Parallel()
	result := MaxPriorityFeeSuggestion(nil, nil)
	assert.Equal(t, big.NewInt(0), result)
}

func TestMaxPriorityFeeSuggestionTipsBelowBaseFee(t *testing.T) {
	t.Parallel()
	baseFee := big.NewInt(100e9)
	txs := []blockchain.Transaction{
		{
			GasPrice:             big.NewInt(100e9),
			MaxFeePerGas:         big.NewInt(100e9),
			MaxPriorityFeePerGas: big.NewInt(0),
		},
	}
	result := MaxPriorityFeeSuggestion(baseFee, txs)
	assert.True(t, result.Sign() >= 0)
}
