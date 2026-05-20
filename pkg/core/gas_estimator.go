package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/gas"
)

type BlockCongestion = gas.BlockCongestion
type AccessListEntry = gas.AccessListEntry

// Gas constants forwarded to pkg/gas
const (
	BlobTxTargetBlobCount              = gas.BlobTxTargetBlobCount
	BlobTxMaxBlobCount                 = gas.BlobTxMaxBlobCount
	BlobTxBlobGasPerBlob               = gas.BlobTxBlobGasPerBlob
	BlobTxMinBlobGasPrice       uint64 = gas.BlobTxMinBlobGasPrice
	BlobTxBlobGasPriceUpdateFraction   = gas.BlobTxBlobGasPriceUpdateFraction
	ColdSloadCost                      = gas.ColdSloadCost
	WarmStorageReadCost                = gas.WarmStorageReadCost
	ColdAccountAccessCost              = gas.ColdAccountAccessCost
	WarmStorageReadCostEIP2930         = gas.WarmStorageReadCostEIP2930
	AccessListAddressCost              = gas.AccessListAddressCost
	AccessListStorageKeyCost           = gas.AccessListStorageKeyCost
	TransientSloadCost                 = gas.TransientSloadCost
	TransientSstoreCost                = gas.TransientSstoreCost
)

func EffectiveGasPrice(baseFee, gasPrice, maxFeePerGas, maxPriorityFeePerGas *big.Int) *big.Int {
	return gas.EffectiveGasPrice(baseFee, gasPrice, maxFeePerGas, maxPriorityFeePerGas)
}

func PredictNextBaseFee(parentBaseFee *big.Int, gasUsed, gasLimit uint64) *big.Int {
	return gas.PredictNextBaseFee(parentBaseFee, gasUsed, gasLimit)
}

func GasCost(gasUsed uint64, effectiveGasPrice *big.Int) *big.Int {
	return gas.GasCost(gasUsed, effectiveGasPrice)
}

func CongestionLevel(gasUsed, gasLimit uint64) float64 {
	return gas.CongestionLevel(gasUsed, gasLimit)
}

func CongestionBand(level float64) string {
	return gas.CongestionBand(level)
}

func AssessBlockCongestion(block *Block) *BlockCongestion {
	return gas.AssessBlockCongestion((*blockchain.Block)(block))
}

func MaxPriorityFeeSuggestion(baseFee *big.Int, recentTxs []Transaction) *big.Int {
	txs := make([]blockchain.Transaction, len(recentTxs))
	for i, tx := range recentTxs {
		txs[i] = blockchain.Transaction(tx)
	}
	return gas.MaxPriorityFeeSuggestion(baseFee, txs)
}

func IsEIP1559Tx(tx *Transaction) bool {
	return gas.IsEIP1559Tx((*blockchain.Transaction)(tx))
}

func IsCancunBlock(block *Block) bool {
	return gas.IsCancunBlock((*blockchain.Block)(block))
}

func CalculateBlobBaseFee(excessBlobGas uint64) *big.Int {
	return gas.CalculateBlobBaseFee(excessBlobGas)
}

func PredictNextExcessBlobGas(excessBlobGas, blobGasUsed uint64) uint64 {
	return gas.PredictNextExcessBlobGas(excessBlobGas, blobGasUsed)
}

func EstimateBlobGasCost(numBlobs int, blobBaseFee, maxFeePerBlobGas *big.Int) *big.Int {
	return gas.EstimateBlobGasCost(numBlobs, blobBaseFee, maxFeePerBlobGas)
}

func BlobCountFromGas(blobGasUsed uint64) int {
	return gas.BlobCountFromGas(blobGasUsed)
}

func AccessListGasCost(baseGas uint64, accessList []AccessListEntry) uint64 {
	return gas.AccessListGasCost(baseGas, accessList)
}

func AccessListGasSavings(coldSloads, coldAccountAccesses uint64) int64 {
	return gas.AccessListGasSavings(coldSloads, coldAccountAccesses)
}

func ShouldUseAccessList(coldSloads, coldAccountAccesses uint64, minSavings uint64) bool {
	return gas.ShouldUseAccessList(coldSloads, coldAccountAccesses, minSavings)
}

func BuildAccessListForTransfer(from, to common.Address, includeBalanceSlot bool) []AccessListEntry {
	return gas.BuildAccessListForTransfer(from, to, includeBalanceSlot)
}

func TransientStorageGasCost(tloads, tstores uint64) uint64 {
	return gas.TransientStorageGasCost(tloads, tstores)
}

func TransientVsPermanentSavings(freshWrites, warmWrites, reads uint64) uint64 {
	return gas.TransientVsPermanentSavings(freshWrites, warmWrites, reads)
}
