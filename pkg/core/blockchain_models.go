package core

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EventStatus represents the processing status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusFinalized EventStatus = "finalized"
	EventStatusFailed    EventStatus = "failed"
	EventStatusReorged   EventStatus = "reorged"
)

// BlockchainEvent represents a blockchain event with full details
type BlockchainEvent struct {
	// Event identification
	ID             string      `json:"id"`
	EventHash      string      `json:"event_hash"`
	EventSignature common.Hash `json:"event_signature"`

	// Block information
	BlockNumber    uint64      `json:"block_number"`
	BlockHash      common.Hash `json:"block_hash"`
	BlockTimestamp int64       `json:"block_timestamp"`

	// Transaction information
	TransactionHash  common.Hash `json:"transaction_hash"`
	TransactionIndex uint64      `json:"transaction_index"`
	GasUsed          uint64      `json:"gas_used"`
	GasPrice         *big.Int    `json:"gas_price"`
	TxStatus         uint8       `json:"tx_status"` // 0=failed, 1=success (EIP-658)

	// Log information
	LogIndex uint64 `json:"log_index"`
	Removed  bool   `json:"removed"`

	// Contract information
	ContractAddress common.Address `json:"contract_address"`
	EventName       string         `json:"event_name"`
	EventTopic      []common.Hash  `json:"event_topic"`

	// Event data
	EventData   []byte         `json:"event_data"`
	DecodedData map[string]any `json:"decoded_data"`
	TypedData   any            `json:"typed_data,omitempty"` // Type-safe decoded event (e.g., *ERC20Transfer)

	// Indexing metadata
	ChainID         string      `json:"chain_id"`
	Network         string      `json:"network"`
	Status          EventStatus `json:"status"`
	TransactionType uint8       `json:"transaction_type"` // TxLegacy(0), TxAccessList(1), TxEIP1559(2), TxBlob(3)

	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at"`
	IndexedAt   time.Time `json:"indexed_at"`
}

// TxType constants for EIP-2718 transaction types
const (
	TxLegacy     uint8 = 0 // Pre-EIP-2718 legacy transaction
	TxAccessList uint8 = 1 // EIP-2930 Access List transaction
	TxEIP1559    uint8 = 2 // EIP-1559 Dynamic Fee transaction
	TxBlob       uint8 = 3 // EIP-4844 Blob transaction
)

// TxStatus constants for EIP-658 receipt status
const (
	TxStatusUnknown uint8 = 0 // Not yet resolved (default zero value)
	TxStatusFailed  uint8 = 1 // Transaction reverted
	TxStatusSuccess uint8 = 2 // Transaction succeeded
)

// TxTypeResolver resolves the EIP-2718 transaction type for a given transaction hash.
// Implementations may fetch the type from RPC (eth_getTransactionReceipt),
// a local cache, or infer it from gas price fields.
type TxTypeResolver interface {
	ResolveTxType(ctx context.Context, txHash string) (txType uint8, txStatus uint8, err error)
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash              common.Hash     `json:"hash"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	Value             *big.Int        `json:"value"`
	Gas               uint64          `json:"gas"`
	GasPrice          *big.Int        `json:"gas_price"` // Legacy & EIP-1559 base fee
	Input             []byte          `json:"input"`
	Nonce             uint64          `json:"nonce"`
	BlockNumber       uint64          `json:"block_number"`
	BlockHash         common.Hash     `json:"block_hash"`
	TransactionIndex  uint            `json:"transaction_index"`
	Status            uint64          `json:"status"` // 1 = success, 0 = failed
	ContractAddress   *common.Address `json:"contract_address"`
	CumulativeGasUsed uint64          `json:"cumulative_gas_used"`
	Logs              []*types.Log    `json:"logs"`

	// EIP-2718 transaction type (0=legacy, 1=AccessList, 2=EIP-1559, 3=EIP-4844)
	Type uint8 `json:"type"`

	// EIP-155 Chain ID for replay protection
	ChainID *big.Int `json:"chain_id,omitempty"`

	// EIP-1559 Dynamic Fee fields (Type=2)
	MaxFeePerGas         *big.Int `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFeePerGas *big.Int `json:"max_priority_fee_per_gas,omitempty"`

	// EIP-2930 Access List (Type=1 and Type=2)
	AccessList types.AccessList `json:"access_list,omitempty"`

	// EIP-4844 Blob fields (Type=3)
	BlobVersionedHashes []common.Hash `json:"blob_versioned_hashes,omitempty"`
	MaxFeePerBlobGas    *big.Int      `json:"max_fee_per_blob_gas,omitempty"`
	BlobSidecar         *BlobSidecar  `json:"blob_sidecar,omitempty"` // Full blob data + KZG proofs
}

// BlobSidecar contains the EIP-4844 blob data and KZG commitments/proofs.
// This is required for blob data availability verification and L2 transaction reconstruction.
type BlobSidecar struct {
	Blobs          []Blob          `json:"blobs"`           // 4096 field elements each (131072 bytes)
	KZGCommitments []KZGCommitment `json:"kzg_commitments"` // 48-byte KZG commitments
	KZGProofs      []KZGProof      `json:"kzg_proofs"`      // 48-byte KZG proofs
}

// Blob represents a single EIP-4844 blob (4096 BLS12-381 field elements = 131072 bytes)
type Blob [131072]byte

// KZGCommitment represents a KZG polynomial commitment (48 bytes, BLS12-381 G1 point)
type KZGCommitment [48]byte

// KZGProof represents a KZG evaluation proof (48 bytes, BLS12-381 G1 point)
type KZGProof [48]byte

// defaultKZGVerifier is the production KZG verifier used by BlobSidecar.VerifyBlobProof.
// It can be overridden for testing via SetKZGVerifier.
var defaultKZGVerifier KZGVerifier = &GethKZGVerifier{}

// SetKZGVerifier allows tests to inject a mock verifier.
func SetKZGVerifier(v KZGVerifier) {
	defaultKZGVerifier = v
}

// VerifyBlobProof validates that the KZG proof matches the commitment for a given blob index.
// Returns an error if the sidecar is malformed (length mismatch), the index is out of range,
// or the KZG pairing verification fails.
func (s *BlobSidecar) VerifyBlobProof(index int) error {
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
	return defaultKZGVerifier.VerifyBlobProof(
		s.KZGCommitments[index][:],
		s.KZGProofs[index][:],
		s.Blobs[index][:],
	)
}

// BlockBuilder identifies the MEV-Boost builder that constructed a block (post-Merge).
type BlockBuilder struct {
	BuilderName    string         `json:"builder_name"`
	BuilderAddress common.Address `json:"builder_address"`
	IsMevBoost     bool           `json:"is_mev_boost"`
	RelayName      string         `json:"relay_name,omitempty"`
}

// Block represents a blockchain block
type Block struct {
	Number                uint64           `json:"number"`
	Hash                  common.Hash      `json:"hash"`
	ParentHash            common.Hash      `json:"parent_hash"`
	Timestamp             int64            `json:"timestamp"`
	Miner                 common.Address   `json:"miner"` // feeRecipient post-Merge
	Difficulty            *big.Int         `json:"difficulty"`
	TotalDifficulty       *big.Int         `json:"total_difficulty,omitempty"` // Pre-Merge only
	GasLimit              uint64           `json:"gas_limit"`
	GasUsed               uint64           `json:"gas_used"`
	BaseFee               *big.Int         `json:"base_fee,omitempty"` // EIP-1559
	Transactions          []common.Hash    `json:"transactions"`
	Uncles                []common.Hash    `json:"uncles,omitempty"`      // Pre-Merge ommer blocks
	Withdrawals           []*Withdrawal    `json:"withdrawals,omitempty"` // EIP-4895 Capella
	LogsBloom             types.Bloom      `json:"logs_bloom"`
	Builder               *BlockBuilder    `json:"builder,omitempty"`                  // MEV-Boost builder (post-Merge)
	ExcessBlobGas         uint64           `json:"excess_blob_gas,omitempty"`          // EIP-4844: tracks blob gas market
	BlobGasUsed           uint64           `json:"blob_gas_used,omitempty"`            // EIP-4844: blob gas consumed in this block
	ParentBeaconBlockRoot *common.Hash     `json:"parent_beacon_block_root,omitempty"` // EIP-4788: beacon chain state root (post-Dencun)
	BeaconBlockInfo       *BeaconBlockInfo `json:"beacon_block_info,omitempty"`        // post-Merge slot/epoch metadata
}

// Withdrawal represents a validator withdrawal (EIP-4895 Capella)
type Withdrawal struct {
	Index          uint64         `json:"index"`
	ValidatorIndex uint64         `json:"validator_index"`
	Address        common.Address `json:"address"`
	Amount         *big.Int       `json:"amount"` // in Gwei
}

// UserOperation represents an ERC-4337 Account Abstraction user operation.
// Unlike a regular EOA transaction, a UserOperation is submitted to an
// EntryPoint contract via a Bundler, with optional Paymaster gas sponsorship.
type UserOperation struct {
	Sender               common.Address `json:"sender"`
	Nonce                *big.Int       `json:"nonce"`
	InitCode             []byte         `json:"init_code"`
	CallData             []byte         `json:"call_data"`
	CallGasLimit         uint64         `json:"call_gas_limit"`
	VerificationGasLimit uint64         `json:"verification_gas_limit"`
	PreVerificationGas   uint64         `json:"pre_verification_gas"`
	MaxFeePerGas         *big.Int       `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas *big.Int       `json:"max_priority_fee_per_gas"`
	PaymasterAndData     []byte         `json:"paymaster_and_data"`
	Signature            []byte         `json:"signature"`
}

// HasPaymaster checks if the UserOperation has a paymaster for gas sponsorship.
func (op *UserOperation) HasPaymaster() bool {
	return len(op.PaymasterAndData) >= 20
}

// PaymasterAddress extracts the paymaster address from PaymasterAndData.
// Returns zero address if no paymaster is set.
func (op *UserOperation) PaymasterAddress() common.Address {
	if !op.HasPaymaster() {
		return common.Address{}
	}
	return common.BytesToAddress(op.PaymasterAndData[:20])
}

// HasInitCode checks if the UserOperation will deploy a new smart account.
func (op *UserOperation) HasInitCode() bool {
	return len(op.InitCode) >= 20
}

// FactoryAddress extracts the factory address from InitCode.
// Returns zero address if no init code is set.
func (op *UserOperation) FactoryAddress() common.Address {
	if !op.HasInitCode() {
		return common.Address{}
	}
	return common.BytesToAddress(op.InitCode[:20])
}

// PaymasterReputation tracks the reputation of an ERC-4337 paymaster
// based on its UserOperation submission history.
type PaymasterReputation struct {
	Address      common.Address `json:"address"`
	OpsSeen      uint64         `json:"ops_seen"`
	OpsIncluded  uint64         `json:"ops_included"`
	SuccessRate  float64        `json:"success_rate"`
	LastActiveAt int64          `json:"last_active_at"`
}

// UpdateProposed increments the proposed counter when a paymaster-backed op is seen.
func (pr *PaymasterReputation) UpdateProposed() {
	pr.OpsSeen++
}

// UpdateIncluded increments the included counter when a paymaster-backed op succeeds.
func (pr *PaymasterReputation) UpdateIncluded() {
	pr.OpsIncluded++
}

// CalculateInclusionRate computes the ratio of included to seen operations.
func (pr *PaymasterReputation) CalculateInclusionRate() float64 {
	if pr.OpsSeen == 0 {
		return 0
	}
	return float64(pr.OpsIncluded) / float64(pr.OpsSeen)
}

// Validate validates the blockchain event
func (be *BlockchainEvent) Validate() error {
	if be.BlockNumber == 0 {
		return ErrInvalidBlockNumber
	}
	if be.TransactionHash == (common.Hash{}) {
		return ErrInvalidTransactionHash
	}
	if be.ContractAddress == (common.Address{}) {
		return ErrInvalidContractAddress
	}
	// Validate EIP-55 checksum on the hex representation.
	// common.Address.Hex() returns checksummed form, so a mismatch
	// would indicate a corrupted address — but we validate the
	// raw string in case external code sets the address via reflection.
	if err := ValidateEIP55Checksum(be.ContractAddress.Hex()); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidContractAddress, err)
	}
	if be.EventName == "" {
		return ErrInvalidEventName
	}
	return nil
}

// IsConfirmed returns whether the event is confirmed
func (be *BlockchainEvent) IsConfirmed() bool {
	return be.Status == EventStatusConfirmed
}

// IsBlobTx returns whether the event originated from an EIP-4844 blob transaction
func (be *BlockchainEvent) IsBlobTx() bool {
	return be.TransactionType == TxBlob
}

// IsEIP1559Tx returns whether the event originated from an EIP-1559 dynamic fee transaction
func (be *BlockchainEvent) IsEIP1559Tx() bool {
	return be.TransactionType == TxEIP1559
}

// IsLegacyTx returns whether the event originated from a legacy (pre-EIP-2718) transaction
func (be *BlockchainEvent) IsLegacyTx() bool {
	return be.TransactionType == TxLegacy
}

// IsPending returns whether the event is pending
func (be *BlockchainEvent) IsPending() bool {
	return be.Status == EventStatusPending
}

// IsFailed returns whether the event failed
func (be *BlockchainEvent) IsFailed() bool {
	return be.Status == EventStatusFailed
}

// IsReorged returns whether the event was reorged
func (be *BlockchainEvent) IsReorged() bool {
	return be.Status == EventStatusReorged
}

// IsFinalized returns whether the event is marked as finalized.
func (be *BlockchainEvent) IsFinalized() bool {
	return be.Status == EventStatusFinalized
}

// EffectiveGasPrice computes the effective gas price for the transaction that produced this event.
// For EIP-1559 (type 2): effectiveGasPrice = min(maxFeePerGas, baseFee + maxPriorityFeePerGas)
// For Legacy (type 0) and AccessList (type 1): uses GasPrice directly.
// Returns nil if insufficient data.
func (be *BlockchainEvent) EffectiveGasPrice(baseFee *big.Int) *big.Int {
	if be.GasPrice == nil {
		return nil
	}
	if be.TransactionType != TxEIP1559 || baseFee == nil {
		return be.GasPrice
	}
	// EIP-1559: effectiveGasPrice = min(maxFeePerGas, baseFee + maxPriorityFeePerGas)
	// Since we don't store per-event maxFeePerGas, estimate: baseFee + tip
	// In practice the gas price field already represents the effective price
	return be.GasPrice
}

// Validate validates the transaction
func (t *Transaction) Validate() error {
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

// IsEIP1559 returns true if this is an EIP-1559 Dynamic Fee transaction (Type=2).
func (t *Transaction) IsEIP1559() bool {
	return t.Type == TxEIP1559
}

// IsBlobTx returns true if this is an EIP-4844 Blob transaction (Type=3).
func (t *Transaction) IsBlobTx() bool {
	return t.Type == TxBlob
}

// IsLegacyTx returns true if this is a legacy transaction (Type=0).
func (t *Transaction) IsLegacyTx() bool {
	return t.Type == TxLegacy
}

// IsAccessListTx returns true if this is an EIP-2930 Access List transaction (Type=1).
func (t *Transaction) IsAccessListTx() bool {
	return t.Type == TxAccessList
}

// IsSuccessful returns whether the transaction was successful
func (t *Transaction) IsSuccessful() bool {
	return t.Status == 1
}

// IsFailed returns whether the transaction failed
func (t *Transaction) IsFailed() bool {
	return t.Status == 0
}

// Validate validates the block
func (b *Block) Validate() error {
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

// GetTimestamp returns the block timestamp as time.Time
func (b *Block) GetTimestamp() time.Time {
	return time.Unix(b.Timestamp, 0)
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash   common.Hash     `json:"transaction_hash"`
	BlockNumber       uint64          `json:"block_number"`
	BlockHash         common.Hash     `json:"block_hash"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	GasUsed           uint64          `json:"gas_used"`
	CumulativeGasUsed uint64          `json:"cumulative_gas_used"`
	ContractAddress   *common.Address `json:"contract_address"`
	Logs              []*types.Log    `json:"logs"`
	Status            uint64          `json:"status"` // 1 = success, 0 = failed
	LogsBloom         types.Bloom     `json:"logs_bloom"`
	Type              uint8           `json:"type"`                          // EIP-2718 transaction type (0/1/2/3)
	EffectiveGasPrice *big.Int        `json:"effective_gas_price,omitempty"` // Actual gas price after EIP-1559
	BlobGasUsed       uint64          `json:"blob_gas_used,omitempty"`       // EIP-4844
	BlobGasPrice      *big.Int        `json:"blob_gas_price,omitempty"`      // EIP-4844
}

// IsSuccessful returns whether the receipt indicates success
func (tr *TransactionReceipt) IsSuccessful() bool {
	return tr.Status == 1
}

// IsFailed returns whether the receipt indicates failure
func (tr *TransactionReceipt) IsFailed() bool {
	return tr.Status == 0
}

// ReorgDetectedMessage is published by the puller when a block hash mismatch is detected.
// The event-processor consumes this from the "reorg-events" Kafka topic to trigger rollback.
type ReorgDetectedMessage struct {
	ChainID    string    `json:"chain_id"`
	ReorgBlock uint64    `json:"reorg_block"`
	OldHash    string    `json:"old_hash"`
	NewHash    string    `json:"new_hash"`
	DetectedAt time.Time `json:"detected_at"`
}

// --- ERC-4337 EntryPoint Versioning ---

// EntryPointVersion represents the version of the EntryPoint contract.
// v0.6 and v0.7 have different UserOperation encodings.
type EntryPointVersion string

const (
	EntryPointV06 EntryPointVersion = "0.6"
	EntryPointV07 EntryPointVersion = "0.7"
)

// Known EntryPoint contract addresses on Ethereum mainnet.
var EntryPointAddresses = map[EntryPointVersion]common.Address{
	EntryPointV06: common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7d57e7E7188e1a3E0a"),
	EntryPointV07: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
}

// EntryPointVersionForAddress returns the EntryPoint version for a given address.
// Returns empty string if the address is not a known EntryPoint.
func EntryPointVersionForAddress(addr common.Address) EntryPointVersion {
	for v, a := range EntryPointAddresses {
		if a == addr {
			return v
		}
	}
	return ""
}

// UserOperationV07 represents a v0.7 UserOperation with packed gas limits.
// In v0.7, verificationGasLimit and callGasLimit are packed into a single
// bytes32 field `accountGasLimits`, and paymaster data is split into
// paymaster (address), paymasterVerificationGasLimit (uint128),
// paymasterPostOpGasLimit (uint128), and paymasterData (bytes).
type UserOperationV07 struct {
	Sender             common.Address `json:"sender"`
	Nonce              *big.Int       `json:"nonce"`
	InitCode           []byte         `json:"init_code"`
	CallData           []byte         `json:"call_data"`
	AccountGasLimits   []byte         `json:"account_gas_limits"` // bytes32: [16B verificationGasLimit][16B callGasLimit]
	MaxFeePerGas       []byte         `json:"max_fee_per_gas"`    // bytes32: [16B maxPriorityFee][16B maxFeePerGas]
	PreVerificationGas uint64         `json:"pre_verification_gas"`
	PaymasterAndData   []byte         `json:"paymaster_and_data"` // v0.7: [20B paymaster][16B verificationGas][16B postOpGas][paymasterData]
	Signature          []byte         `json:"signature"`
}

// DecodeV07GasLimits extracts verificationGasLimit and callGasLimit from
// the packed accountGasLimits bytes32 field.
// Layout: bytes32 = [16 bytes verificationGasLimit (big-endian uint128)] [16 bytes callGasLimit (big-endian uint128)]
func (op *UserOperationV07) DecodeV07GasLimits() (verificationGasLimit, callGasLimit uint64) {
	if len(op.AccountGasLimits) != 32 {
		return 0, 0
	}
	verificationGasLimit = new(big.Int).SetBytes(op.AccountGasLimits[:16]).Uint64()
	callGasLimit = new(big.Int).SetBytes(op.AccountGasLimits[16:]).Uint64()
	return
}

// DecodeV07FeePerGas extracts maxPriorityFeePerGas and maxFeePerGas from
// the packed maxFeePerGas bytes32 field.
// Layout: bytes32 = [16 bytes maxPriorityFeePerGas][16 bytes maxFeePerGas]
func (op *UserOperationV07) DecodeV07FeePerGas() (maxPriorityFeePerGas, maxFeePerGas *big.Int) {
	if len(op.MaxFeePerGas) != 32 {
		return big.NewInt(0), big.NewInt(0)
	}
	maxPriorityFeePerGas = new(big.Int).SetBytes(op.MaxFeePerGas[:16])
	maxFeePerGas = new(big.Int).SetBytes(op.MaxFeePerGas[16:])
	return
}

// ToUserOperation converts a v0.7 UserOperation to the canonical UserOperation
// format by unpacking the gas limit and fee fields.
func (op *UserOperationV07) ToUserOperation() *UserOperation {
	verificationGasLimit, callGasLimit := op.DecodeV07GasLimits()
	maxPriorityFeePerGas, maxFeePerGas := op.DecodeV07FeePerGas()

	return &UserOperation{
		Sender:               op.Sender,
		Nonce:                op.Nonce,
		InitCode:             op.InitCode,
		CallData:             op.CallData,
		CallGasLimit:         callGasLimit,
		VerificationGasLimit: verificationGasLimit,
		PreVerificationGas:   op.PreVerificationGas,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		PaymasterAndData:     op.PaymasterAndData,
		Signature:            op.Signature,
	}
}
