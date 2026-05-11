package core

import (
	"math"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EffectiveGasPrice computes the actual gas price for a transaction.
// Formula for EIP-1559: min(maxFeePerGas, baseFee + maxPriorityFeePerGas)
// For legacy transactions (maxFeePerGas == nil), returns gasPrice as-is.
func EffectiveGasPrice(baseFee, gasPrice, maxFeePerGas, maxPriorityFeePerGas *big.Int) *big.Int {
	if maxFeePerGas == nil {
		// Legacy transaction: effective price is gasPrice
		if gasPrice != nil {
			return new(big.Int).Set(gasPrice)
		}
		// Pre-EIP-155 chain without gasPrice field — fall back to baseFee
		if baseFee != nil {
			return new(big.Int).Set(baseFee)
		}
		return big.NewInt(0)
	}

	if baseFee == nil {
		// Pre-EIP-1559 chain: use maxFeePerGas directly
		return new(big.Int).Set(maxFeePerGas)
	}

	// EIP-1559: effective = min(maxFee, baseFee + priorityFee)
	priorityFee := maxPriorityFeePerGas
	if priorityFee == nil {
		priorityFee = big.NewInt(0)
	}
	tip := new(big.Int).Add(baseFee, priorityFee)

	if tip.Cmp(maxFeePerGas) > 0 {
		// baseFee + priorityFee exceeds maxFeePerGas → capped at maxFeePerGas
		return new(big.Int).Set(maxFeePerGas)
	}

	return tip
}

// PredictNextBaseFee computes the expected base fee for the next block
// using the EIP-1559 base fee formula.
//
// If parentGasUsed < gasTarget (gasLimit/2): baseFee decreases
//   baseFee_next = baseFee * (1 - (gasTarget - gasUsed) / gasTarget / 8)
//
// If parentGasUsed > gasTarget: baseFee increases
//   baseFee_next = baseFee * (1 + (gasUsed - gasTarget) / gasTarget / 8)
//
// Maximum change per block: ~12.5% (1/8 = 0.125)
func PredictNextBaseFee(parentBaseFee *big.Int, gasUsed, gasLimit uint64) *big.Int {
	if parentBaseFee == nil || parentBaseFee.Sign() <= 0 {
		return big.NewInt(0)
	}

	if gasLimit == 0 {
		return new(big.Int).Set(parentBaseFee)
	}

	gasTarget := gasLimit / 2

	// Using exact EIP-1559 spec formula with integer arithmetic:
	//   gasTarget = gasLimit / 2
	//   delta = parentBaseFee * (gasUsed - gasTarget) / gasTarget / 8
	//   nextBaseFee = parentBaseFee + delta
	//
	// We compute delta in two branches to avoid negative big.Int from unsigned subtraction.

	var delta *big.Int
	if gasUsed > gasTarget {
		// Base fee increases: delta = parentBaseFee * (gasUsed - gasTarget) / gasTarget / 8
		delta = new(big.Int).Mul(parentBaseFee, new(big.Int).SetUint64(gasUsed-gasTarget))
		delta.Div(delta, new(big.Int).SetUint64(gasTarget))
		delta.Div(delta, big.NewInt(8))
	} else {
		// Base fee decreases: delta = -parentBaseFee * (gasTarget - gasUsed) / gasTarget / 8
		delta = new(big.Int).Mul(parentBaseFee, new(big.Int).SetUint64(gasTarget-gasUsed))
		delta.Div(delta, new(big.Int).SetUint64(gasTarget))
		delta.Div(delta, big.NewInt(8))
		delta.Neg(delta)
	}

	nextBaseFee := new(big.Int).Add(parentBaseFee, delta)

	// Base fee cannot go below 0
	if nextBaseFee.Sign() < 0 {
		nextBaseFee = big.NewInt(0)
	}

	// Base fee cannot go below 1 wei on mainnet (EIP-1559 spec)
	if nextBaseFee.Sign() == 0 && gasUsed > 0 {
		nextBaseFee = big.NewInt(1)
	}

	return nextBaseFee
}

// GasCost computes the total gas cost for a transaction.
// Formula: gasUsed * effectiveGasPrice
func GasCost(gasUsed uint64, effectiveGasPrice *big.Int) *big.Int {
	if effectiveGasPrice == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), effectiveGasPrice)
}

// CongestionLevel returns a normalized congestion indicator based on gasUsed vs gasTarget.
// gasTarget = gasLimit / 2 (per EIP-1559).
// A value of 1.0 means the block is exactly at target utilization.
// > 1.0 means the block is over target and the next base fee will increase.
func CongestionLevel(gasUsed, gasLimit uint64) float64 {
	gasTarget := gasLimit / 2
	if gasTarget == 0 {
		return 0.0
	}
	return float64(gasUsed) / float64(gasTarget)
}

// CongestionBand returns a human-readable congestion band.
// Thresholds are relative to gasTarget (1.0 = at target utilization).
func CongestionBand(level float64) string {
	switch {
	case level < 0.50:
		return "low"
	case level < 0.75:
		return "moderate"
	case level < 1.00:
		return "high"
	case level < 1.50:
		return "congested"
	default:
		return "severe"
	}
}

// MaxPriorityFeeSuggestion returns a suggested priority fee based on recent transactions.
// It computes the median effective priority fee from the provided transactions.
// The effective priority fee for each tx is: min(maxPriorityFeePerGas, effectiveGasPrice - baseFee)
func MaxPriorityFeeSuggestion(baseFee *big.Int, recentTxs []Transaction) *big.Int {
	if len(recentTxs) == 0 {
		if baseFee != nil {
			// Default suggestion: 1 Gwei
			return new(big.Int).Mul(big.NewInt(1), big.NewInt(1e9))
		}
		return big.NewInt(0)
	}

	var tips []*big.Int
	for i := range recentTxs {
		tx := &recentTxs[i]
		var tip *big.Int

		if tx.MaxPriorityFeePerGas != nil && baseFee != nil {
			// EIP-1559: priority fee is the tip above baseFee
			tip = new(big.Int).Set(tx.MaxPriorityFeePerGas)

			// Cap at what the user actually paid above baseFee
			if tx.MaxFeePerGas != nil {
				effective := EffectiveGasPrice(baseFee, tx.GasPrice, tx.MaxFeePerGas, tx.MaxPriorityFeePerGas)
				actualTip := new(big.Int).Sub(effective, baseFee)
				if actualTip.Sign() < 0 {
					actualTip = big.NewInt(0)
				}
				if actualTip.Cmp(tip) < 0 {
					tip = actualTip
				}
			}
		} else if tx.GasPrice != nil && baseFee != nil {
			// Legacy: tip = gasPrice - baseFee
			tip = new(big.Int).Sub(tx.GasPrice, baseFee)
			if tip.Sign() < 0 {
				tip = big.NewInt(0)
			}
		}

		if tip != nil && tip.Sign() > 0 {
			tips = append(tips, tip)
		}
	}

	if len(tips) == 0 {
		return new(big.Int).Mul(big.NewInt(1), big.NewInt(1e9)) // 1 Gwei default
	}

	// Sort and return median
	sort.Slice(tips, func(i, j int) bool {
		return tips[i].Cmp(tips[j]) < 0
	})
	mid := len(tips) / 2
	return new(big.Int).Set(tips[mid])
}

// IsEIP1559Tx checks whether a transaction uses EIP-1559 gas pricing
// based on the presence of MaxFeePerGas and MaxPriorityFeePerGas fields.
func IsEIP1559Tx(tx *Transaction) bool {
	return tx != nil && tx.MaxFeePerGas != nil && tx.MaxPriorityFeePerGas != nil
}

// EstimateBlockCongestion provides a congestion assessment for a block.
type BlockCongestion struct {
	Level        float64 // gasUsed / gasLimit
	Band         string  // human-readable band
	NextBaseFee  *big.Int // predicted base fee for the next block
	GasTargetPct float64 // gasUsed / gasTarget as percentage
}

// AssessBlockCongestion computes a full congestion assessment for a block.
func AssessBlockCongestion(block *Block) *BlockCongestion {
	if block == nil {
		return &BlockCongestion{
			Level:       0,
			Band:        "unknown",
			NextBaseFee: big.NewInt(0),
		}
	}

	level := CongestionLevel(block.GasUsed, block.GasLimit)
	nextBaseFee := PredictNextBaseFee(block.BaseFee, block.GasUsed, block.GasLimit)

	gasTarget := block.GasLimit / 2
	var gasTargetPct float64
	if gasTarget > 0 {
		gasTargetPct = float64(block.GasUsed) / float64(gasTarget) * 100
	} else {
		gasTargetPct = 0
	}

	return &BlockCongestion{
		Level:        level,
		Band:         CongestionBand(level),
		NextBaseFee:  nextBaseFee,
		GasTargetPct: math.Round(gasTargetPct*100) / 100,
	}
}

// --- EIP-4844 Blob Gas Market ---

// Blob gas constants from EIP-4844 spec (Dencun defaults).
// These are kept for backward compatibility. For fork-versioned parameters,
// use BlobParamsForFork() and the *WithParams() functions in eip4844.go.
const (
	// BlobTxTargetBlobCount is the target number of blobs per block (3, Dencun).
	BlobTxTargetBlobCount = 3
	// BlobTxMaxBlobCount is the maximum number of blobs per block (6, Dencun; 9, Pectra).
	// Deprecated: Use BlobParamsForFork() for fork-aware values.
	BlobTxMaxBlobCount = 6
	// BlobTxBlobGasPerBlob is the gas consumed per blob (2^17 = 131072).
	BlobTxBlobGasPerBlob = 131072
	// BlobTxMinBlobGasPrice is the minimum blob gas price (1 wei).
	BlobTxMinBlobGasPrice = 1
	// BlobTxBlobGasPriceUpdateFraction is the update fraction for the fake-exponential formula.
	// Deprecated: Use BlobParamsForFork() for fork-aware values.
	BlobTxBlobGasPriceUpdateFraction = 3338477
)

// CalculateBlobBaseFee computes the blob base fee using the fake-exponential
// formula from EIP-4844:
//
//	fake_exponential(factor, numerator, denominator)
//
// This follows the spec and geth implementation using uint64 arithmetic where
// overflow naturally terminates the iteration:
//
//	output = 0, accum = factor
//	for i := 1; accum > 0; i++ {
//	  output += accum
//	  accum = accum * numerator / denominator / i
//	}
func CalculateBlobBaseFee(excessBlobGas uint64) *big.Int {
	if excessBlobGas == 0 {
		return big.NewInt(BlobTxMinBlobGasPrice)
	}

	var output, accum uint64 = 0, BlobTxMinBlobGasPrice
	for i := uint64(1); accum > 0; i++ {
		output += accum
		// accum = accum * excessBlobGas / updateFraction / i
		// Multiply first, then divide — uint64 overflow wraps to 0, terminating the loop.
		accum = accum * excessBlobGas / BlobTxBlobGasPriceUpdateFraction / i
	}

	return new(big.Int).SetUint64(output)
}

// PredictNextExcessBlobGas computes the expected excess blob gas for the next block.
// EIP-4844 uses a separate exponential moving average from the regular gas market:
//
//	targetBlobGas = BlobTxTargetBlobCount * BlobTxBlobGasPerBlob
//	if blobGasUsed > targetBlobGas:
//	  excess_next = excess + (blobGasUsed - targetBlobGas)
//	else:
//	  excess_next = max(0, excess - (targetBlobGas - blobGasUsed))
func PredictNextExcessBlobGas(excessBlobGas, blobGasUsed uint64) uint64 {
	targetBlobGas := uint64(BlobTxTargetBlobCount) * uint64(BlobTxBlobGasPerBlob)

	if blobGasUsed > targetBlobGas {
		return excessBlobGas + (blobGasUsed - targetBlobGas)
	}

	if targetBlobGas-blobGasUsed > excessBlobGas {
		return 0
	}

	return excessBlobGas - (targetBlobGas - blobGasUsed)
}

// EstimateBlobGasCost estimates the total gas cost for a blob transaction.
// Cost = numBlobs * BlobTxBlobGasPerBlob * max(blobBaseFee, maxFeePerBlobGas)
// This is the blob portion only; execution gas is separate.
func EstimateBlobGasCost(numBlobs int, blobBaseFee, maxFeePerBlobGas *big.Int) *big.Int {
	if numBlobs <= 0 {
		return big.NewInt(0)
	}
	if blobBaseFee == nil {
		blobBaseFee = big.NewInt(BlobTxMinBlobGasPrice)
	}

	// Effective blob gas price = min(blobBaseFee, maxFeePerBlobGas)
	effectiveBlobGasPrice := new(big.Int).Set(blobBaseFee)
	if maxFeePerBlobGas != nil && maxFeePerBlobGas.Cmp(blobBaseFee) < 0 {
		effectiveBlobGasPrice.Set(maxFeePerBlobGas)
	}

	// Total blob gas = numBlobs * BlobTxBlobGasPerBlob
	totalBlobGas := new(big.Int).Mul(big.NewInt(int64(numBlobs)), big.NewInt(BlobTxBlobGasPerBlob))

	return new(big.Int).Mul(totalBlobGas, effectiveBlobGasPrice)
}

// IsCancunBlock checks if a block has Cancun-specific fields (post-Dencun).
// A Cancun block will have ParentBeaconBlockRoot set.
func IsCancunBlock(block *Block) bool {
	return block != nil && block.ParentBeaconBlockRoot != nil
}

// BlobCountFromGas computes how many blobs are represented by the given blob gas.
func BlobCountFromGas(blobGasUsed uint64) int {
	if blobGasUsed == 0 {
		return 0
	}
	return int(blobGasUsed / uint64(BlobTxBlobGasPerBlob))
}

// --- EIP-2930 Access List Gas Savings ---

// Access list gas constants from EIP-2930 and EIP-2200.
const (
	// ColdSloadCost is the gas cost for accessing a storage slot that has not
	// been accessed in the current transaction (2100 gas per EIP-2929).
	ColdSloadCost uint64 = 2100
	// WarmStorageReadCost is the gas cost for accessing a storage slot that
	// has already been accessed in the current transaction (100 gas).
	WarmStorageReadCost uint64 = 100
	// ColdAccountAccessCost is the gas cost for accessing an address that has
	// not been accessed in the current transaction (2600 gas per EIP-2929).
	ColdAccountAccessCost uint64 = 2600
	// WarmStorageReadCostEIP2930 is the cost of a warm SLOAD after the
	// access list pre-payment (same as WarmStorageReadCost).
	WarmStorageReadCostEIP2930 uint64 = 100
	// AccessListAddressCost is the fixed gas cost per address in an access list (2400 gas).
	AccessListAddressCost uint64 = 2400
	// AccessListStorageKeyCost is the fixed gas cost per storage key in an access list (1900 gas).
	AccessListStorageKeyCost uint64 = 1900
)

// AccessListEntry represents a single entry in an EIP-2930 access list.
type AccessListEntry struct {
	Address     common.Address   `json:"address"`
	StorageKeys []common.Hash    `json:"storageKeys"`
}

// AccessListGasCost computes the total gas cost for a transaction with an access list.
// The access list adds a fixed cost per address and per storage key, but reduces
// the per-access cost for cold slots/addresses during execution.
//
// Cost = baseGas + (numAddresses * AccessListAddressCost) + (numStorageKeys * AccessListStorageKeyCost)
// + (execution gas with warm costs instead of cold costs)
func AccessListGasCost(baseGas uint64, accessList []AccessListEntry) uint64 {
	cost := baseGas
	for _, entry := range accessList {
		cost += AccessListAddressCost
		cost += uint64(len(entry.StorageKeys)) * AccessListStorageKeyCost
	}
	return cost
}

// AccessListGasSavings estimates the gas savings from using an access list.
// Without an access list, each cold SLOAD costs ColdSloadCost (2100) and each
// cold account access costs ColdAccountAccessCost (2600). With an access list,
// these accesses are pre-warmed, so they cost WarmStorageReadCost (100) instead.
//
// Savings per cold access = coldCost - (accessListPrepayment + warmCost)
// For storage: savings per key = ColdSloadCost - (AccessListStorageKeyCost + WarmStorageReadCost)
// For address: savings per address = ColdAccountAccessCost - (AccessListAddressCost + WarmStorageReadCost)
func AccessListGasSavings(coldSloads, coldAccountAccesses uint64) int64 {
	// Storage key savings: cold_sload (2100) - prepayment (1900) - warm_read (100) = 100
	storageSavings := coldSloads * (ColdSloadCost - AccessListStorageKeyCost - WarmStorageReadCost)

	// Address savings: cold_account (2600) - prepayment (2400) - warm_read (100) = 100
	addressSavings := coldAccountAccesses * (ColdAccountAccessCost - AccessListAddressCost - WarmStorageReadCost)

	return int64(storageSavings + addressSavings)
}

// ShouldUseAccessList determines whether including an access list is beneficial.
// It returns true if the estimated savings exceed the overhead of computing
// and including the access list in the transaction.
func ShouldUseAccessList(coldSloads, coldAccountAccesses uint64, minSavings uint64) bool {
	if minSavings == 0 {
		minSavings = 1000 // default threshold
	}
	savings := AccessListGasSavings(coldSloads, coldAccountAccesses)
	return savings > 0 && uint64(savings) >= minSavings
}

// BuildAccessListForTransfer builds a minimal access list for a simple ETH transfer.
// It includes the sender and recipient addresses (pre-warm them) and optionally
// the sender's balance storage slot if the caller wants to warm it.
func BuildAccessListForTransfer(from, to common.Address, includeBalanceSlot bool) []AccessListEntry {
	entries := []AccessListEntry{
		{Address: from},
		{Address: to},
	}

	if includeBalanceSlot {
		// The balance mapping uses keccak256(pad32(address) ++ pad32(0))
		// For a simple transfer, we want to warm the sender's balance slot
		var keyBytes [32]byte
		copy(keyBytes[12:], from[:])
		var slotBytes [32]byte // slot 0
		combined := append(keyBytes[:], slotBytes[:]...)
		storageKey := common.BytesToHash(crypto.Keccak256(combined))

		entries[0].StorageKeys = []common.Hash{storageKey}
	}

	return entries
}

// --- EIP-1153 Transient Storage ---

// Transient storage gas constants from EIP-1153 (live since Dencun).
// Transient storage is cleared after each transaction, making it ideal for
// reentrancy guards, single-transaction locks, and callback tracking.
const (
	// TransientSloadCost is the gas cost for TLOAD (reading transient storage).
	// At 100 gas, it is the same as a warm SLOAD, making it very cheap.
	TransientSloadCost uint64 = 100
	// TransientSstoreCost is the gas cost for TSTORE (writing transient storage).
	// At 100 gas for a fresh or cleared slot, it is far cheaper than regular SSTORE.
	TransientSstoreCost uint64 = 100
)

// TransientStorageGasCost computes the total gas cost for using transient storage.
// Each TLOAD costs TransientSloadCost, each TSTORE costs TransientSstoreCost.
// Unlike regular storage, there is no refund for clearing transient storage
// because it is automatically cleared at the end of each transaction.
func TransientStorageGasCost(tloads, tstores uint64) uint64 {
	return tloads*TransientSloadCost + tstores*TransientSstoreCost
}

// TransientVsPermanentSavings estimates the gas savings from using transient
// storage instead of permanent storage for values that only need to exist
// within a single transaction (e.g., reentrancy guards).
//
// Permanent SSTORE: 20000 (fresh) or 2900 (warm overwrite)
// Transient TSTORE: 100
// Permanent SLOAD: 100 (warm) or 2100 (cold)
// Transient TLOAD: 100
func TransientVsPermanentSavings(freshWrites, warmWrites, reads uint64) uint64 {
	permanentCost := freshWrites*20000 + warmWrites*2900 + reads*WarmStorageReadCost
	transientCost := (freshWrites+warmWrites)*TransientSstoreCost + reads*TransientSloadCost
	if permanentCost < transientCost {
		return 0
	}
	return permanentCost - transientCost
}
