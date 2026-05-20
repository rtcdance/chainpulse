package consensus

import (
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// Beacon chain constants for post-Merge Ethereum.
// After The Merge, Ethereum consensus operates on a slot-based system
// where validators propose blocks in 12-second slots, grouped into
// 32-slot epochs. Finality advances at epoch boundaries, not per-block.
const (
	// SlotDuration is the time between consecutive beacon chain slots (12 seconds).
	SlotDuration = 12 * time.Second

	// SlotsPerEpoch is the number of slots in one epoch (32).
	SlotsPerEpoch = 32

	// EpochDuration is the duration of one epoch: 32 * 12s = 384s ≈ 6.4 minutes.
	EpochDuration = SlotDuration * SlotsPerEpoch

	// MainnetGenesisTime is the Ethereum mainnet beacon chain genesis timestamp (Dec 1, 2020 12:00:23 UTC).
	MainnetGenesisTime int64 = 1606824023

	// MaxMissedSlots is the threshold for alerting on consecutive missed slots.
	MaxMissedSlots = 10
)

// SlotToEpoch converts a beacon chain slot number to its corresponding epoch.
func SlotToEpoch(slot uint64) uint64 {
	return slot / SlotsPerEpoch
}

// IsEpochBoundary checks if the given slot is the first slot of an epoch.
// Epoch boundaries are where finality decisions are made.
func IsEpochBoundary(slot uint64) bool {
	return slot%SlotsPerEpoch == 0
}

// EpochFirstSlot returns the first slot of the given epoch.
func EpochFirstSlot(epoch uint64) uint64 {
	return epoch * SlotsPerEpoch
}

// TimestampToSlot computes the beacon chain slot number for a given Unix timestamp
// relative to the genesis time.
// Formula: slot = (timestamp - genesisTime) / 12
func TimestampToSlot(timestamp, genesisTime int64) uint64 {
	if timestamp <= genesisTime {
		return 0
	}
	return uint64((timestamp - genesisTime) / int64(SlotDuration.Seconds()))
}

// SlotToTimestamp computes the Unix timestamp for the start of a given slot.
// Formula: timestamp = genesisTime + slot * 12
func SlotToTimestamp(slot uint64, genesisTime int64) int64 {
	return genesisTime + int64(slot)*int64(SlotDuration.Seconds())
}

// DetectMissedSlots returns a slice of slot numbers that were skipped between
// the parent slot and the current slot. If no slots were missed, returns nil.
// A gap indicates that no validator proposed a block for those slots.
func DetectMissedSlots(parentSlot, currentSlot uint64) []uint64 {
	if currentSlot <= parentSlot+1 {
		return nil
	}
	missed := make([]uint64, 0, currentSlot-parentSlot-1)
	for s := parentSlot + 1; s < currentSlot; s++ {
		missed = append(missed, s)
	}
	return missed
}

// ExpectedSlotNumber computes the expected beacon slot for a block based on
// its timestamp and the chain's genesis time.
func ExpectedSlotNumber(blockTimestamp, genesisTime int64) uint64 {
	return TimestampToSlot(blockTimestamp, genesisTime)
}

// NewBeaconBlockInfo creates a BeaconBlockInfo from a block's timestamp and
// the parent's slot number. It computes the current slot, epoch, and
// whether any slots were missed between the parent and current.
func NewBeaconBlockInfo(blockTimestamp, genesisTime int64, parentSlot uint64) *blockchain.BeaconBlockInfo {
	currentSlot := TimestampToSlot(blockTimestamp, genesisTime)
	missed := DetectMissedSlots(parentSlot, currentSlot)

	return &blockchain.BeaconBlockInfo{
		Slot:         currentSlot,
		Epoch:        SlotToEpoch(currentSlot),
		IsMissedSlot: len(missed) > 0,
	}
}

// SlotsUntilNextEpoch returns the number of slots remaining until the next
// epoch boundary from the given slot.
func SlotsUntilNextEpoch(slot uint64) uint64 {
	nextEpochStart := (SlotToEpoch(slot) + 1) * SlotsPerEpoch
	return nextEpochStart - slot
}

// TimeUntilNextEpoch returns the estimated duration until the next epoch
// boundary from the given slot.
func TimeUntilNextEpoch(slot uint64) time.Duration {
	return time.Duration(SlotsUntilNextEpoch(slot)) * SlotDuration
}
