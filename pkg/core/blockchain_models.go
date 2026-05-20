package core

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// Event status values
type EventStatus = blockchain.EventStatus

const (
	EventStatusPending   EventStatus = blockchain.EventStatusPending
	EventStatusConfirmed EventStatus = blockchain.EventStatusConfirmed
	EventStatusFinalized EventStatus = blockchain.EventStatusFinalized
	EventStatusFailed    EventStatus = blockchain.EventStatusFailed
	EventStatusReorged   EventStatus = blockchain.EventStatusReorged
)

// Transaction type constants for EIP-2718
const (
	TxLegacy     = blockchain.TxLegacy
	TxAccessList = blockchain.TxAccessList
	TxEIP1559    = blockchain.TxEIP1559
	TxBlob       = blockchain.TxBlob
)

// Transaction receipt status constants
const (
	TxStatusUnknown = blockchain.TxStatusUnknown
	TxStatusFailed  = blockchain.TxStatusFailed
	TxStatusSuccess = blockchain.TxStatusSuccess
)

// BlockchainEvent type alias
type BlockchainEvent = blockchain.BlockchainEvent

// TxTypeResolver type alias
type TxTypeResolver = blockchain.TxTypeResolver

// Transaction type alias
type Transaction = blockchain.Transaction

// BlobSidecar type alias
type BlobSidecar = blockchain.BlobSidecar

// Blob type alias
type Blob = blockchain.Blob

// KZGCommitment type alias
type KZGCommitment = blockchain.KZGCommitment

// KZGProof type alias
type KZGProof = blockchain.KZGProof

// BlockBuilder type alias
type BlockBuilder = blockchain.BlockBuilder

// BeaconBlockInfo type alias
type BeaconBlockInfo = blockchain.BeaconBlockInfo

// Block type alias
type Block = blockchain.Block

// Withdrawal type alias
type Withdrawal = blockchain.Withdrawal

// UserOperation type alias
type UserOperation = blockchain.UserOperation

// PaymasterReputation type alias
type PaymasterReputation = blockchain.PaymasterReputation

// TransactionReceipt type alias
type TransactionReceipt = blockchain.TransactionReceipt

// ReorgDetectedMessage type alias
type ReorgDetectedMessage = blockchain.ReorgDetectedMessage

// EntryPointVersion type alias
type EntryPointVersion = blockchain.EntryPointVersion

const (
	EntryPointV06 EntryPointVersion = blockchain.EntryPointV06
	EntryPointV07 EntryPointVersion = blockchain.EntryPointV07
)

// EntryPointAddresses type alias
var EntryPointAddresses = blockchain.EntryPointAddresses

// UserOperationV07 type alias
type UserOperationV07 = blockchain.UserOperationV07

func EntryPointVersionForAddress(addr common.Address) EntryPointVersion {
	return blockchain.EntryPointVersionForAddress(addr)
}
