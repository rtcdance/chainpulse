package eip

import (
	"math/big"
	"testing"
)

func TestBlobParamsForFork(t *testing.T) {
	dencun, err := BlobParamsForFork(ForkDencun)
	if err != nil {
		t.Fatalf("BlobParamsForFork(dencun) error: %v", err)
	}
	if dencun.MaxBlobCount != 6 {
		t.Errorf("Dencun MaxBlobCount = %d, want 6", dencun.MaxBlobCount)
	}
	if dencun.TargetBlobCount != 3 {
		t.Errorf("Dencun TargetBlobCount = %d, want 3", dencun.TargetBlobCount)
	}

	pectra, err := BlobParamsForFork(ForkPectra)
	if err != nil {
		t.Fatalf("BlobParamsForFork(pectra) error: %v", err)
	}
	if pectra.MaxBlobCount != 9 {
		t.Errorf("Pectra MaxBlobCount = %d, want 9", pectra.MaxBlobCount)
	}
	if pectra.TargetBlobCount != 6 {
		t.Errorf("Pectra TargetBlobCount = %d, want 6", pectra.TargetBlobCount)
	}

	// Unknown fork
	_, err = BlobParamsForFork("unknown")
	if err == nil {
		t.Fatal("expected error for unknown fork")
	}
}

func TestRegisterBlobParams(t *testing.T) {
	custom := BlobParams{
		Fork:                   "test",
		TargetBlobCount:        2,
		MaxBlobCount:           4,
		BlobGasPerBlob:         131072,
		MinBlobGasPrice:        1,
		BlobGasPriceUpdateFrac: 5000000,
	}
	RegisterBlobParams("test", custom)

	got, err := BlobParamsForFork("test")
	if err != nil {
		t.Fatalf("BlobParamsForFork(test) error: %v", err)
	}
	if got.MaxBlobCount != 4 {
		t.Errorf("custom MaxBlobCount = %d, want 4", got.MaxBlobCount)
	}

	// Clean up
	delete(forkBlobParams, "test")
}

func TestBlobParams_TargetBlobGas(t *testing.T) {
	dencun := BlobParamsDencun
	if got := dencun.TargetBlobGas(); got != 3*131072 {
		t.Errorf("Dencun TargetBlobGas = %d, want %d", got, 3*131072)
	}

	pectra := BlobParamsPectra
	if got := pectra.TargetBlobGas(); got != 6*131072 {
		t.Errorf("Pectra TargetBlobGas = %d, want %d", got, 6*131072)
	}
}

func TestBlobParams_MaxBlobGas(t *testing.T) {
	dencun := BlobParamsDencun
	if got := dencun.MaxBlobGas(); got != 6*131072 {
		t.Errorf("Dencun MaxBlobGas = %d, want %d", got, 6*131072)
	}

	pectra := BlobParamsPectra
	if got := pectra.MaxBlobGas(); got != 9*131072 {
		t.Errorf("Pectra MaxBlobGas = %d, want %d", got, 9*131072)
	}
}

func TestCalculateBlobBaseFeeWithParams(t *testing.T) {
	// With zero excess, should return min price
	fee := CalculateBlobBaseFeeWithParams(0, BlobParamsDencun)
	if fee.Uint64() != 1 {
		t.Errorf("zero excess fee = %d, want 1", fee.Uint64())
	}

	// Same excess, different forks should yield different results due to different update fractions
	excess := uint64(3932160) // significant excess
	feeDencun := CalculateBlobBaseFeeWithParams(excess, BlobParamsDencun)
	feePectra := CalculateBlobBaseFeeWithParams(excess, BlobParamsPectra)

	// Both should be > 1 (min price)
	if feeDencun.Uint64() <= 1 {
		t.Errorf("Dencun fee = %d, expected > 1", feeDencun.Uint64())
	}
	if feePectra.Uint64() <= 1 {
		t.Errorf("Pectra fee = %d, expected > 1", feePectra.Uint64())
	}

	// With same excess, Dencun (smaller update fraction = more aggressive increase)
	// should yield higher or equal fee. In practice the difference may be small.
	if feeDencun.Cmp(feePectra) < 0 {
		t.Errorf("Dencun fee (%d) should be >= Pectra fee (%d) for same excess", feeDencun, feePectra)
	}
}

func TestPredictNextExcessBlobGasWithParams(t *testing.T) {
	// Dencun: target = 3 blobs * 131072 = 393216
	excess := PredictNextExcessBlobGasWithParams(0, 393216, BlobParamsDencun)
	if excess != 0 {
		t.Errorf("at target: excess = %d, want 0", excess)
	}

	// Over target: excess increases
	excess = PredictNextExcessBlobGasWithParams(0, 500000, BlobParamsDencun)
	if excess != 500000-393216 {
		t.Errorf("over target: excess = %d, want %d", excess, 500000-393216)
	}

	// Pectra: target = 6 blobs * 131072 = 786432
	excess = PredictNextExcessBlobGasWithParams(0, 786432, BlobParamsPectra)
	if excess != 0 {
		t.Errorf("Pectra at target: excess = %d, want 0", excess)
	}
}

func TestEstimateBlobGasCostWithParams(t *testing.T) {
	blobBaseFee := big.NewInt(100) // 100 wei per blob gas

	cost := EstimateBlobGasCostWithParams(3, blobBaseFee, nil, BlobParamsDencun)
	expected := int64(3 * 131072 * 100)
	if cost.Int64() != expected {
		t.Errorf("Dencun cost = %d, want %d", cost.Int64(), expected)
	}

	// With maxFeePerBlobGas lower than blobBaseFee
	maxFee := big.NewInt(50)
	cost = EstimateBlobGasCostWithParams(3, blobBaseFee, maxFee, BlobParamsDencun)
	expectedCapped := int64(3 * 131072 * 50)
	if cost.Int64() != expectedCapped {
		t.Errorf("capped cost = %d, want %d", cost.Int64(), expectedCapped)
	}
}

func TestBlobCountFromGasWithParams(t *testing.T) {
	// 3 blobs = 3 * 131072 = 393216
	count := BlobCountFromGasWithParams(393216, BlobParamsDencun)
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}

	// Zero
	count = BlobCountFromGasWithParams(0, BlobParamsDencun)
	if count != 0 {
		t.Errorf("zero count = %d, want 0", count)
	}
}

// ─── KZG Verifier Tests ──────────────────────────────────────────────────────

func TestSizeOnlyKZGVerifier_ValidInput(t *testing.T) {
	verifier := &SizeOnlyKZGVerifier{}

	commitment := make([]byte, KZGCommitmentSize)
	proof := make([]byte, KZGProofSize)
	blob := make([]byte, BlobSize)

	if err := verifier.VerifyBlobProof(commitment, proof, blob); err != nil {
		t.Errorf("valid input should pass: %v", err)
	}
}

func TestSizeOnlyKZGVerifier_InvalidCommitmentSize(t *testing.T) {
	verifier := &SizeOnlyKZGVerifier{}

	err := verifier.VerifyBlobProof(
		make([]byte, 32), // wrong size
		make([]byte, KZGProofSize),
		make([]byte, BlobSize),
	)
	if err == nil {
		t.Fatal("expected error for wrong commitment size")
	}
}

func TestSizeOnlyKZGVerifier_InvalidProofSize(t *testing.T) {
	verifier := &SizeOnlyKZGVerifier{}

	err := verifier.VerifyBlobProof(
		make([]byte, KZGCommitmentSize),
		make([]byte, 16), // wrong size
		make([]byte, BlobSize),
	)
	if err == nil {
		t.Fatal("expected error for wrong proof size")
	}
}

func TestSizeOnlyKZGVerifier_InvalidBlobSize(t *testing.T) {
	verifier := &SizeOnlyKZGVerifier{}

	err := verifier.VerifyBlobProof(
		make([]byte, KZGCommitmentSize),
		make([]byte, KZGProofSize),
		make([]byte, 100), // wrong size
	)
	if err == nil {
		t.Fatal("expected error for wrong blob size")
	}
}
