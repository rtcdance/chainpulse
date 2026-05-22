// Package gas provides gas estimation and fee market analysis.
package gas

import (
	"math"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// EffectiveGasPrice computes the actual gas price for a transaction.
func EffectiveGasPrice(baseFee, gasPrice, maxFeePerGas, maxPriorityFeePerGas *big.Int) *big.Int {
	if maxFeePerGas == nil {
		if gasPrice != nil {
			return new(big.Int).Set(gasPrice)
		}
		if baseFee != nil {
			return new(big.Int).Set(baseFee)
		}
		return big.NewInt(0)
	}
	if baseFee == nil {
		return new(big.Int).Set(maxFeePerGas)
	}
	priorityFee := maxPriorityFeePerGas
	if priorityFee == nil {
		priorityFee = big.NewInt(0)
	}
	tip := new(big.Int).Add(baseFee, priorityFee)
	if tip.Cmp(maxFeePerGas) > 0 {
		return new(big.Int).Set(maxFeePerGas)
	}
	return tip
}

// PredictNextBaseFee computes the expected base fee for the next block
// using the EIP-1559 base fee formula.
func PredictNextBaseFee(parentBaseFee *big.Int, gasUsed, gasLimit uint64) *big.Int {
	if parentBaseFee == nil || parentBaseFee.Sign() <= 0 {
		return big.NewInt(0)
	}
	if gasLimit == 0 {
		return new(big.Int).Set(parentBaseFee)
	}
	gasTarget := gasLimit / 2
	var delta *big.Int
	if gasUsed > gasTarget {
		delta = new(big.Int).Mul(parentBaseFee, new(big.Int).SetUint64(gasUsed-gasTarget))
		delta.Div(delta, new(big.Int).SetUint64(gasTarget))
		delta.Div(delta, big.NewInt(8))
	} else {
		delta = new(big.Int).Mul(parentBaseFee, new(big.Int).SetUint64(gasTarget-gasUsed))
		delta.Div(delta, new(big.Int).SetUint64(gasTarget))
		delta.Div(delta, big.NewInt(8))
		delta.Neg(delta)
	}
	nextBaseFee := new(big.Int).Add(parentBaseFee, delta)
	if nextBaseFee.Sign() < 0 {
		nextBaseFee = big.NewInt(0)
	}
	if nextBaseFee.Sign() == 0 && gasUsed > 0 {
		nextBaseFee = big.NewInt(1)
	}
	return nextBaseFee
}

// GasCost computes the total gas cost for a transaction.
func GasCost(gasUsed uint64, effectiveGasPrice *big.Int) *big.Int {
	if effectiveGasPrice == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), effectiveGasPrice)
}

// CongestionLevel returns a normalized congestion indicator.
func CongestionLevel(gasUsed, gasLimit uint64) float64 {
	gasTarget := gasLimit / 2
	if gasTarget == 0 {
		return 0.0
	}
	return float64(gasUsed) / float64(gasTarget)
}

// CongestionBand returns a human-readable congestion band.
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

// BlockCongestion provides a congestion assessment for a block.
type BlockCongestion struct {
	Level        float64
	Band         string
	NextBaseFee  *big.Int
	GasTargetPct float64
}

// AssessBlockCongestion computes a full congestion assessment for a block.
func AssessBlockCongestion(block *blockchain.Block) *BlockCongestion {
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
	}
	return &BlockCongestion{
		Level:        level,
		Band:         CongestionBand(level),
		NextBaseFee:  nextBaseFee,
		GasTargetPct: math.Round(gasTargetPct*100) / 100,
	}
}

// MaxPriorityFeeSuggestion returns a suggested priority fee based on recent transactions.
func MaxPriorityFeeSuggestion(baseFee *big.Int, recentTxs []blockchain.Transaction) *big.Int {
	if len(recentTxs) == 0 {
		if baseFee != nil {
			return new(big.Int).Mul(big.NewInt(1), big.NewInt(1e9))
		}
		return big.NewInt(0)
	}
	var tips []*big.Int
	for i := range recentTxs {
		tx := &recentTxs[i]
		var tip *big.Int
		if tx.MaxPriorityFeePerGas != nil && baseFee != nil {
			tip = new(big.Int).Set(tx.MaxPriorityFeePerGas)
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
		return new(big.Int).Mul(big.NewInt(1), big.NewInt(1e9))
	}
	sort.Slice(tips, func(i, j int) bool {
		return tips[i].Cmp(tips[j]) < 0
	})
	mid := len(tips) / 2
	return new(big.Int).Set(tips[mid])
}

// IsEIP1559Tx checks whether a transaction uses EIP-1559 gas pricing.
func IsEIP1559Tx(tx *blockchain.Transaction) bool {
	return tx != nil && tx.MaxFeePerGas != nil && tx.MaxPriorityFeePerGas != nil
}

// IsCancunBlock checks if a block has Cancun-specific fields.
func IsCancunBlock(block *blockchain.Block) bool {
	return block != nil && block.ParentBeaconBlockRoot != nil
}

// EIP-4844 blob gas constants
const (
	BlobTxTargetBlobCount                   = 3
	BlobTxMaxBlobCount                      = 6
	BlobTxBlobGasPerBlob                    = 131072
	BlobTxMinBlobGasPrice            uint64 = 1
	BlobTxBlobGasPriceUpdateFraction        = 3338477
)

// CalculateBlobBaseFee computes the blob base fee using EIP-4844 formula.
// Implements fake_exponential(MIN_BLOB_GAS_PRICE, excess_blob_gas, BLOB_GAS_PRICE_UPDATE_FRACTION)
// as specified in the EIP-4844 spec.
func CalculateBlobBaseFee(excessBlobGas uint64) *big.Int {
	minPrice := big.NewInt(int64(BlobTxMinBlobGasPrice))
	denom := big.NewInt(int64(BlobTxBlobGasPriceUpdateFraction))
	numer := big.NewInt(int64(excessBlobGas))

	// fake_exponential(factor, numerator, denominator):
	//   output = 0
	//   accum = factor * denominator
	//   for i = 1; accum > 0; i++:
	//     output += accum / denominator
	//     accum = (accum * numerator) / denominator / i
	output := new(big.Int)
	accum := new(big.Int).Mul(minPrice, denom)
	one := big.NewInt(1)
	i := big.NewInt(1)

	for accum.Sign() > 0 {
		// output += accum / denominator
		term := new(big.Int).Div(accum, denom)
		output.Add(output, term)

		// accum = accum * numerator / denominator / i
		accum.Mul(accum, numer)
		accum.Div(accum, denom)
		accum.Div(accum, i)

		i.Add(i, one)

		// Safety: prevent infinite loop on pathological inputs
		if i.BitLen() > 256 {
			break
		}
	}

	return output
}

// PredictNextExcessBlobGas computes the expected excess blob gas for the next block.
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
func EstimateBlobGasCost(numBlobs int, blobBaseFee, maxFeePerBlobGas *big.Int) *big.Int {
	if numBlobs <= 0 {
		return big.NewInt(0)
	}
	if blobBaseFee == nil {
		blobBaseFee = big.NewInt(int64(BlobTxMinBlobGasPrice))
	}
	effectiveBlobGasPrice := new(big.Int).Set(blobBaseFee)
	if maxFeePerBlobGas != nil && maxFeePerBlobGas.Cmp(blobBaseFee) < 0 {
		effectiveBlobGasPrice.Set(maxFeePerBlobGas)
	}
	totalBlobGas := new(big.Int).Mul(big.NewInt(int64(numBlobs)), big.NewInt(BlobTxBlobGasPerBlob))
	return new(big.Int).Mul(totalBlobGas, effectiveBlobGasPrice)
}

// BlobCountFromGas computes how many blobs are represented by the given blob gas.
func BlobCountFromGas(blobGasUsed uint64) int {
	if blobGasUsed == 0 {
		return 0
	}
	return int(blobGasUsed / uint64(BlobTxBlobGasPerBlob))
}

// EIP-2930 access list gas constants
const (
	ColdSloadCost              uint64 = 2100
	WarmStorageReadCost        uint64 = 100
	ColdAccountAccessCost      uint64 = 2600
	WarmStorageReadCostEIP2930 uint64 = 100
	AccessListAddressCost      uint64 = 2400
	AccessListStorageKeyCost   uint64 = 1900
)

// AccessListEntry represents a single entry in an EIP-2930 access list.
type AccessListEntry struct {
	Address     common.Address `json:"address"`
	StorageKeys []common.Hash  `json:"storageKeys"`
}

// AccessListGasCost computes the total gas cost for a transaction with an access list.
func AccessListGasCost(baseGas uint64, accessList []AccessListEntry) uint64 {
	cost := baseGas
	for _, entry := range accessList {
		cost += AccessListAddressCost
		cost += uint64(len(entry.StorageKeys)) * AccessListStorageKeyCost
	}
	return cost
}

// AccessListGasSavings estimates the gas savings from using an access list.
func AccessListGasSavings(coldSloads, coldAccountAccesses uint64) int64 {
	storageSavings := coldSloads * (ColdSloadCost - AccessListStorageKeyCost - WarmStorageReadCost)
	addressSavings := coldAccountAccesses * (ColdAccountAccessCost - AccessListAddressCost - WarmStorageReadCost)
	return int64(storageSavings + addressSavings)
}

// ShouldUseAccessList determines whether including an access list is beneficial.
func ShouldUseAccessList(coldSloads, coldAccountAccesses uint64, minSavings uint64) bool {
	if minSavings == 0 {
		minSavings = 1000
	}
	savings := AccessListGasSavings(coldSloads, coldAccountAccesses)
	return savings > 0 && uint64(savings) >= minSavings
}

// BuildAccessListForTransfer builds a minimal access list for a simple ETH transfer.
func BuildAccessListForTransfer(from, to common.Address, includeBalanceSlot bool) []AccessListEntry {
	entries := []AccessListEntry{
		{Address: from},
		{Address: to},
	}
	if includeBalanceSlot {
		var keyBytes [32]byte
		copy(keyBytes[12:], from[:])
		var slotBytes [32]byte
		combined := append(keyBytes[:], slotBytes[:]...)
		storageKey := common.BytesToHash(crypto.Keccak256(combined))
		entries[0].StorageKeys = []common.Hash{storageKey}
	}
	return entries
}

// EIP-1153 transient storage gas constants
const (
	TransientSloadCost  uint64 = 100
	TransientSstoreCost uint64 = 100
)

// TransientStorageGasCost computes the total gas cost for using transient storage.
func TransientStorageGasCost(tloads, tstores uint64) uint64 {
	return tloads*TransientSloadCost + tstores*TransientSstoreCost
}

// TransientVsPermanentSavings estimates the gas savings from using transient storage.
func TransientVsPermanentSavings(freshWrites, warmWrites, reads uint64) uint64 {
	permanentCost := freshWrites*20000 + warmWrites*2900 + reads*WarmStorageReadCost
	transientCost := (freshWrites+warmWrites)*TransientSstoreCost + reads*TransientSloadCost
	if permanentCost < transientCost {
		return 0
	}
	return permanentCost - transientCost
}
