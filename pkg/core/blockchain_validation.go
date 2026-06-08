package core

import (
	"fmt"
	"github.com/rtcdance/chainpulse/pkg/blockchain"

	"github.com/ethereum/go-ethereum/common"
)

// ValidateBlockchainEvent validates a blockchain event.
func ValidateBlockchainEvent(be *blockchain.BlockchainEvent) error {
	if be.BlockNumber == 0 {
		return ErrInvalidBlockNumber
	}
	if be.TransactionHash == (common.Hash{}) {
		return ErrInvalidTransactionHash
	}
	if be.ContractAddress == (common.Address{}) {
		return ErrInvalidContractAddress
	}
	if err := ValidateEIP55Checksum(be.ContractAddress.Hex()); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidContractAddress, err)
	}
	if be.EventName == "" {
		return ErrInvalidEventName
	}
	return nil
}

// ValidateTransaction validates a transaction.
func ValidateTransaction(t *blockchain.Transaction) error {
	if t.Hash == (common.Hash{}) {
		return ErrInvalidTransactionHash
	}
	if t.From == (common.Address{}) {
		return ErrInvalidAddress
	}
	if t.BlockNumber == 0 {
		return ErrInvalidBlockNumber
	}
	return nil
}

// ValidateBlock validates a block.
func ValidateBlock(b *blockchain.Block) error {
	if b.Number == 0 {
		return ErrInvalidBlockNumber
	}
	if b.Hash == (common.Hash{}) {
		return ErrInvalidBlockHash
	}
	if b.Timestamp == 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

// VerifyBlobSidecarProof validates that the KZG proof in a blockchain.BlobSidecar matches
// the commitment for a given blob index.
func VerifyBlobSidecarProof(s *blockchain.BlobSidecar, verifier KZGVerifier, index int) error {
	if s == nil {
		return fmt.Errorf("blob sidecar is nil")
	}
	if index < 0 || index >= len(s.Blobs) {
		return fmt.Errorf("blob index %d out of range [0, %d)", index, len(s.Blobs))
	}
	if len(s.Blobs) != len(s.KZGCommitments) {
		return fmt.Errorf("blobs count (%d) != commitments count (%d)", len(s.Blobs), len(s.KZGCommitments))
	}
	if len(s.Blobs) != len(s.KZGProofs) {
		return fmt.Errorf("blobs count (%d) != proofs count (%d)", len(s.Blobs), len(s.KZGProofs))
	}
	return verifier.VerifyBlobProof(
		s.KZGCommitments[index][:],
		s.KZGProofs[index][:],
		s.Blobs[index][:],
	)
}
