package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// StakeInfo represents the stake deposited by an entity (factory, paymaster,
// or aggregator) on the EntryPoint contract. Staking is required for entities
// that store data or validate UserOperations, ensuring they have economic
// skin in the game.
type StakeInfo struct {
	Address       common.Address `json:"address"`
	StakedAmount  *big.Int       `json:"staked_amount"` // in wei
	UnstakeDelay  uint64         `json:"unstake_delay"` // seconds before withdrawal
	WithdrawTime  time.Time      `json:"withdraw_time"` // when funds become withdrawable
	IsStaked      bool           `json:"is_staked"`
	DepositEntity string         `json:"deposit_entity"` // "factory", "paymaster", or "aggregator"
}

// MinimumStakeThreshold is the minimum stake required by the EntryPoint
// for entities that access global storage (factories, paymasters, aggregators).
const MinimumStakeThreshold = 1e18 // 1 ETH (varies by EntryPoint version)

// IsStakeSufficient checks if the staked amount meets the minimum threshold
// and the unstake delay is at least the required minimum.
func IsStakeSufficient(info *StakeInfo, minStake *big.Int, minUnstakeDelay uint64) bool {
	if info == nil || !info.IsStaked {
		return false
	}
	if minStake == nil {
		minStake = big.NewInt(MinimumStakeThreshold)
	}
	if info.StakedAmount == nil || info.StakedAmount.Cmp(minStake) < 0 {
		return false
	}
	if info.UnstakeDelay < minUnstakeDelay {
		return false
	}
	return true
}

// IsWithdrawInProgress checks if the entity has initiated withdrawal
// (unstake delay has started counting down).
func (s *StakeInfo) IsWithdrawInProgress() bool {
	return !s.WithdrawTime.IsZero() && time.Now().Before(s.WithdrawTime)
}

// StakeLockedEvent is emitted when an entity adds stake or increases
// the unstake delay on the EntryPoint contract.
type StakeLockedEvent struct {
	Address      common.Address `json:"address"`
	StakedAmount *big.Int       `json:"staked_amount"`
	UnstakeDelay uint64         `json:"unstake_delay"`
	BlockNumber  uint64         `json:"block_number"`
	TxHash       common.Hash    `json:"tx_hash"`
}

// StakeUnlockedEvent is emitted when the unstake delay has passed and
// the entity can withdraw their stake.
type StakeUnlockedEvent struct {
	Address      common.Address `json:"address"`
	WithdrawTime time.Time      `json:"withdraw_time"`
	BlockNumber  uint64         `json:"block_number"`
	TxHash       common.Hash    `json:"tx_hash"`
}

// StakeWithdrawnEvent is emitted when an entity withdraws their staked ETH.
type StakeWithdrawnEvent struct {
	Address     common.Address `json:"address"`
	Amount      *big.Int       `json:"amount"`
	WithdrawTo  common.Address `json:"withdraw_to"`
	BlockNumber uint64         `json:"block_number"`
	TxHash      common.Hash    `json:"tx_hash"`
}
