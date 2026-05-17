package core

import (
	"fmt"
	"math/big"
	"sync"

	kzg4844 "github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// ─── Fork-Versioned Blob Parameters ─────────────────────────────────────────

// ForkName identifies an Ethereum execution-layer upgrade.
type ForkName string

const (
	ForkDencun ForkName = "dencun" // EIP-4844 introduced (Mar 2024)
	ForkPectra ForkName = "pectra" // EIP-7668 raises max blobs (May 2025)
)

// BlobParams holds the fork-dependent EIP-4844 parameters.
// Dencun shipped with target=3 / max=6 blobs. Pectra (EIP-7668) raises
// max to 9, keeping the same update fraction for backward compatibility.
type BlobParams struct {
	Fork                   ForkName
	TargetBlobCount        uint64 // target blobs per block
	MaxBlobCount           uint64 // max blobs per block
	BlobGasPerBlob         uint64 // gas consumed per blob (2^17 = 131072)
	MinBlobGasPrice        uint64 // minimum blob gas price (1 wei)
	BlobGasPriceUpdateFrac uint64 // update fraction for fake-exponential
}

// Pre-defined fork parameters.
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

// forkBlobParams is the registry mapping fork names to their blob parameters.
// It can be extended for future forks (e.g., Osaka).
var (
	forkBlobParams = map[ForkName]BlobParams{
		ForkDencun: BlobParamsDencun,
		ForkPectra: BlobParamsPectra,
	}
	forkBlobParamsMu sync.RWMutex
)

// BlobParamsForFork returns the blob parameters for the given fork.
// Returns an error if the fork is not registered.
func BlobParamsForFork(fork ForkName) (BlobParams, error) {
	forkBlobParamsMu.RLock()
	defer forkBlobParamsMu.RUnlock()
	params, ok := forkBlobParams[fork]
	if !ok {
		return BlobParams{}, fmt.Errorf("no blob params registered for fork %q", fork)
	}
	return params, nil
}

// RegisterBlobParams allows registering custom blob parameters for a fork
// (e.g., for test chains or future upgrades).
func RegisterBlobParams(fork ForkName, params BlobParams) {
	forkBlobParamsMu.Lock()
	defer forkBlobParamsMu.Unlock()
	forkBlobParams[fork] = params
}

// TargetBlobGas returns the target blob gas per block for these params.
func (p BlobParams) TargetBlobGas() uint64 {
	return p.TargetBlobCount * p.BlobGasPerBlob
}

// MaxBlobGas returns the maximum blob gas per block for these params.
func (p BlobParams) MaxBlobGas() uint64 {
	return p.MaxBlobCount * p.BlobGasPerBlob
}

// CalculateBlobBaseFeeWithParams computes the blob base fee using the
// fake-exponential formula from EIP-4844, parameterized by fork-specific values.
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

// PredictNextExcessBlobGasWithParams computes the expected excess blob gas
// for the next block, using fork-specific target blob count.
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

// EstimateBlobGasCostWithParams estimates the total blob gas cost using
// fork-specific parameters.
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

// BlobCountFromGasWithParams computes how many blobs are represented by
// the given blob gas using fork-specific BlobGasPerBlob.
func BlobCountFromGasWithParams(blobGasUsed uint64, params BlobParams) int {
	if blobGasUsed == 0 || params.BlobGasPerBlob == 0 {
		return 0
	}
	return int(blobGasUsed / params.BlobGasPerBlob)
}

// ─── KZG Commitment Abstraction ──────────────────────────────────────────────

// KZGVerifier abstracts KZG commitment verification for blob transactions.
// This interface allows swapping between a production implementation (using
// go-ethereum's crypto/kzg4844) and a test/mock implementation.
type KZGVerifier interface {
	// VerifyBlobProof verifies that a blob's KZG commitment matches its proof.
	// Returns nil if verification succeeds.
	VerifyBlobProof(commitment []byte, proof []byte, blob []byte) error
}

// SizeOnlyKZGVerifier performs only byte-length validation on KZG inputs.
// This should ONLY be used in tests — it does NOT verify the cryptographic pairing.
// Production code must use GethKZGVerifier instead.
type SizeOnlyKZGVerifier struct{}

// KZGCommitmentSize is the expected size of a KZG commitment (48 bytes).
const KZGCommitmentSize = 48

// KZGProofSize is the expected size of a KZG proof (48 bytes).
const KZGProofSize = 48

// BlobSize is the expected size of a blob (4096 field elements * 32 bytes = 131072).
const BlobSize = 131072

// VerifyBlobProof performs basic format validation on KZG inputs.
// WARNING: This does NOT perform real cryptographic verification. Use GethKZGVerifier for production.
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

// GethKZGVerifier performs real KZG pairing verification using go-ethereum's kzg4844 package.
// This is the production verifier that should be used in all non-test code.
type GethKZGVerifier struct{}

// VerifyBlobProof verifies a KZG proof by performing the full elliptic curve pairing check.
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
