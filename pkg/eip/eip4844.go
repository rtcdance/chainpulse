package eip

import (
	"fmt"
	"math/big"
	"sync"

	kzg4844 "github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// ForkName identifies an Ethereum execution-layer upgrade.
type ForkName string

const (
	ForkDencun ForkName = "dencun"
	ForkPectra ForkName = "pectra"
)

// BlobParams holds the fork-dependent EIP-4844 parameters.
type BlobParams struct {
	Fork                   ForkName
	TargetBlobCount        uint64
	MaxBlobCount           uint64
	BlobGasPerBlob         uint64
	MinBlobGasPrice        uint64
	BlobGasPriceUpdateFrac uint64
}

var (
	BlobParamsDencun = BlobParams{
		Fork:                   ForkDencun,
		TargetBlobCount:        3,
		MaxBlobCount:           6,
		BlobGasPerBlob:         131072,
		MinBlobGasPrice:        1,
		BlobGasPriceUpdateFrac: 3338477,
	}

	BlobParamsPectra = BlobParams{
		Fork:                   ForkPectra,
		TargetBlobCount:        6,
		MaxBlobCount:           9,
		BlobGasPerBlob:         131072,
		MinBlobGasPrice:        1,
		BlobGasPriceUpdateFrac: 3825718,
	}
)

var (
	forkBlobParams = map[ForkName]BlobParams{
		ForkDencun: BlobParamsDencun,
		ForkPectra: BlobParamsPectra,
	}
	forkBlobParamsMu sync.RWMutex
)

func BlobParamsForFork(fork ForkName) (BlobParams, error) {
	forkBlobParamsMu.RLock()
	defer forkBlobParamsMu.RUnlock()
	params, ok := forkBlobParams[fork]
	if !ok {
		return BlobParams{}, fmt.Errorf("no blob params registered for fork %q", fork)
	}
	return params, nil
}

func RegisterBlobParams(fork ForkName, params BlobParams) {
	forkBlobParamsMu.Lock()
	defer forkBlobParamsMu.Unlock()
	forkBlobParams[fork] = params
}

func (p BlobParams) TargetBlobGas() uint64 {
	return p.TargetBlobCount * p.BlobGasPerBlob
}

func (p BlobParams) MaxBlobGas() uint64 {
	return p.MaxBlobCount * p.BlobGasPerBlob
}

func CalculateBlobBaseFeeWithParams(excessBlobGas uint64, params BlobParams) *big.Int {
	if excessBlobGas == 0 {
		return big.NewInt(int64(params.MinBlobGasPrice))
	}

	var output, accum uint64 = 0, params.MinBlobGasPrice
	for i := uint64(1); accum > 0; i++ {
		output += accum
		accum = accum * excessBlobGas / params.BlobGasPriceUpdateFrac / i
	}

	return new(big.Int).SetUint64(output)
}

func PredictNextExcessBlobGasWithParams(excessBlobGas, blobGasUsed uint64, params BlobParams) uint64 {
	targetBlobGas := params.TargetBlobGas()

	if blobGasUsed > targetBlobGas {
		return excessBlobGas + (blobGasUsed - targetBlobGas)
	}

	if targetBlobGas-blobGasUsed > excessBlobGas {
		return 0
	}

	return excessBlobGas - (targetBlobGas - blobGasUsed)
}

func EstimateBlobGasCostWithParams(numBlobs int, blobBaseFee, maxFeePerBlobGas *big.Int, params BlobParams) *big.Int {
	if numBlobs <= 0 {
		return big.NewInt(0)
	}
	if blobBaseFee == nil {
		blobBaseFee = big.NewInt(int64(params.MinBlobGasPrice))
	}

	effectiveBlobGasPrice := new(big.Int).Set(blobBaseFee)
	if maxFeePerBlobGas != nil && maxFeePerBlobGas.Cmp(blobBaseFee) < 0 {
		effectiveBlobGasPrice.Set(maxFeePerBlobGas)
	}

	totalBlobGas := new(big.Int).Mul(big.NewInt(int64(numBlobs)), big.NewInt(int64(params.BlobGasPerBlob)))
	return new(big.Int).Mul(totalBlobGas, effectiveBlobGasPrice)
}

func BlobCountFromGasWithParams(blobGasUsed uint64, params BlobParams) int {
	if blobGasUsed == 0 || params.BlobGasPerBlob == 0 {
		return 0
	}
	return int(blobGasUsed / params.BlobGasPerBlob)
}

// KZGVerifier abstracts KZG commitment verification for blob transactions.
type KZGVerifier interface {
	VerifyBlobProof(commitment []byte, proof []byte, blob []byte) error
}

type SizeOnlyKZGVerifier struct{}

const (
	KZGCommitmentSize = 48
	KZGProofSize      = 48
	BlobSize          = 131072
)

func (d *SizeOnlyKZGVerifier) VerifyBlobProof(commitment []byte, proof []byte, blob []byte) error {
	if len(commitment) != KZGCommitmentSize {
		return fmt.Errorf("invalid commitment size: got %d, want %d", len(commitment), KZGCommitmentSize)
	}
	if len(proof) != KZGProofSize {
		return fmt.Errorf("invalid proof size: got %d, want %d", len(proof), KZGProofSize)
	}
	if len(blob) != BlobSize {
		return fmt.Errorf("invalid blob size: got %d, want %d", len(blob), BlobSize)
	}
	return nil
}

type GethKZGVerifier struct{}

func (g *GethKZGVerifier) VerifyBlobProof(commitment []byte, proof []byte, blob []byte) error {
	if len(commitment) != KZGCommitmentSize {
		return fmt.Errorf("invalid commitment size: got %d, want %d", len(commitment), KZGCommitmentSize)
	}
	if len(proof) != KZGProofSize {
		return fmt.Errorf("invalid proof size: got %d, want %d", len(proof), KZGProofSize)
	}
	if len(blob) != BlobSize {
		return fmt.Errorf("invalid blob size: got %d, want %d", len(blob), BlobSize)
	}

	var b kzg4844.Blob
	copy(b[:], blob)
	var c kzg4844.Commitment
	copy(c[:], commitment)
	var p kzg4844.Proof
	copy(p[:], proof)

	return kzg4844.VerifyBlobProof(&b, c, p)
}
