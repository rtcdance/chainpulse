package core

import "fmt"

// KZGVerifier abstracts KZG commitment verification for blob transactions.
type KZGVerifier interface {
	VerifyBlobProof(commitment []byte, proof []byte, blob []byte) error
}

// SizeOnlyKZGVerifier performs only byte-length validation on KZG inputs.
type SizeOnlyKZGVerifier struct{}

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

const (
	KZGCommitmentSize = 48
	KZGProofSize      = 48
	BlobSize          = 131072
)
