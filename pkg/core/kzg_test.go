package core

import (
	"testing"
)

func TestSizeOnlyKZGVerifier_VerifyBlobProof_Valid(t *testing.T) {
	v := &SizeOnlyKZGVerifier{}
	commitment := make([]byte, KZGCommitmentSize)
	proof := make([]byte, KZGProofSize)
	blob := make([]byte, BlobSize)
	err := v.VerifyBlobProof(commitment, proof, blob)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestSizeOnlyKZGVerifier_InvalidCommitmentSize(t *testing.T) {
	v := &SizeOnlyKZGVerifier{}
	commitment := make([]byte, 1)
	proof := make([]byte, KZGProofSize)
	blob := make([]byte, BlobSize)
	err := v.VerifyBlobProof(commitment, proof, blob)
	if err == nil {
		t.Fatal("expected error for invalid commitment size")
	}
}

func TestSizeOnlyKZGVerifier_InvalidProofSize(t *testing.T) {
	v := &SizeOnlyKZGVerifier{}
	commitment := make([]byte, KZGCommitmentSize)
	proof := make([]byte, 1)
	blob := make([]byte, BlobSize)
	err := v.VerifyBlobProof(commitment, proof, blob)
	if err == nil {
		t.Fatal("expected error for invalid proof size")
	}
}

func TestSizeOnlyKZGVerifier_InvalidBlobSize(t *testing.T) {
	v := &SizeOnlyKZGVerifier{}
	commitment := make([]byte, KZGCommitmentSize)
	proof := make([]byte, KZGProofSize)
	blob := make([]byte, 1)
	err := v.VerifyBlobProof(commitment, proof, blob)
	if err == nil {
		t.Fatal("expected error for invalid blob size")
	}
}
