package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// --- EIP-7002: Execution Layer Triggerable Exits ---
//
// Allows validators to request exits and withdrawals from the execution layer
// via a predeploy contract at 0x00...02, eliminating the need for signed
// voluntary exit messages through the beacon chain.

// EIP7002PredeployAddress is the address of the withdrawal request predeploy.
var EIP7002PredeployAddress = common.HexToAddress("0x00000961Ef480Eb55e80D19ad83579A64c007002")

// WithdrawalRequest represents a validator withdrawal request submitted via
// the EIP-7002 predeploy contract. These are processed by the beacon chain
// after being queued on the execution layer.
type WithdrawalRequest struct {
	SourceAddress common.Address `json:"source_address"`   // msg.sender who triggered the request
	ValidatorPub  [48]byte       `json:"validator_pubkey"` // 48-byte BLS12-381 public key
	Amount        *big.Int       `json:"amount"`           // in Gwei (0 = full exit)
	BlockNumber   uint64         `json:"block_number"`
	TxHash        common.Hash    `json:"tx_hash"`
	Index         uint64         `json:"request_index"` // monotonically increasing
}

// IsFullExit returns true if the withdrawal request is for a full validator exit.
func (r *WithdrawalRequest) IsFullExit() bool {
	return r.Amount == nil || r.Amount.Sign() == 0
}

// WithdrawalRequestEventTopic is the keccak256 hash of the WithdrawalRequested event signature.
// Event: WithdrawalRequested(address indexed source, bytes pubkey, uint256 amount)
const WithdrawalRequestEventTopic = "0x649ddb62f0b3c8225c254a7931ec5f0d0e6b9dc5f7e3c2e1e7a0c0c2d4a0c1b0"

// --- EIP-6110: Execution Layer Deposit Processing ---
//
// Deposits are processed on the execution layer by the deposit contract,
// rather than being polled by the beacon chain. This reduces deposit latency
// and simplifies the validator onboarding pipeline.

// EIP6110DepositContractAddress is the canonical deposit contract address.
var EIP6110DepositContractAddress = common.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")

// ValidatorDeposit represents a validator deposit processed on the execution
// layer per EIP-6110. The deposit contract emits events that are consumed
// by the beacon chain to activate new validators.
type ValidatorDeposit struct {
	Pubkey                [48]byte    `json:"pubkey"`                 // BLS12-381 public key
	WithdrawalCredentials [32]byte    `json:"withdrawal_credentials"` // withdrawal address or data
	Amount                *big.Int    `json:"amount"`                 // in Gwei (must be 32 ETH = 32000000000)
	Signature             [96]byte    `json:"signature"`              // BLS12-381 signature
	Index                 uint64      `json:"deposit_index"`          // monotonically increasing
	BlockNumber           uint64      `json:"block_number"`
	TxHash                common.Hash `json:"tx_hash"`
}

// IsValidDepositAmount checks if the deposit amount equals the minimum
// required stake of 32 ETH (32,000,000,000 Gwei).
func (d *ValidatorDeposit) IsValidDepositAmount() bool {
	if d.Amount == nil {
		return false
	}
	return d.Amount.Cmp(big.NewInt(32e9)) == 0
}

// DepositEventTopic is the keccak256 hash of the DepositEvent log topic.
// Event: DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
const DepositEventTopic = "0x649beb922dd7257b612e9fd773a0fd43f0f7cbde8e3d30db2f7ef1b2c0a5b1d8"

// ParseDepositCountFromLog extracts the deposit count from the deposit contract
// log data. The count is encoded as a 256-bit big-endian integer.
func ParseDepositCountFromLog(data []byte) uint64 {
	if len(data) < 32 {
		return 0
	}
	return new(big.Int).SetBytes(data[:32]).Uint64()
}
