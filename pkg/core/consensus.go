package core

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// ─── Validator Lifecycle ─────────────────────────────────────────────────────

// ValidatorState represents the lifecycle state of a beacon chain validator.
// Transitions follow the Ethereum spec: Pending → Active → (Exiting | Slashed) → Withdrawn.
type ValidatorState string

const (
	// ValidatorPending means the validator is in the activation queue.
	// Transition to Active happens when the activation queue is processed at epoch boundary.
	ValidatorPending ValidatorState = "pending"

	// ValidatorActive means the validator is actively attesting and proposing.
	// Transition to Exiting (voluntary) or Slashed (forcible).
	ValidatorActive ValidatorState = "active"

	// ValidatorExiting means the validator submitted a voluntary exit.
	// Transition to Withdrawn after exit delay (MIN_VALIDATOR_WITHDRAWABILITY_DELAY + CHURN_LIMIT).
	ValidatorExiting ValidatorState = "exiting"

	// ValidatorSlashed means the validator was slashed for provable misbehavior.
	// Slashed validators suffer a penalty and must wait through a longer withdrawal delay.
	ValidatorSlashed ValidatorState = "slashed"

	// ValidatorWithdrawn means the validator has exited and funds are withdrawable.
	// This is a terminal state.
	ValidatorWithdrawn ValidatorState = "withdrawn"
)

// ValidTransitions defines the allowed state transitions for a validator.
var ValidTransitions = map[ValidatorState][]ValidatorState{
	ValidatorPending:   {ValidatorActive},
	ValidatorActive:    {ValidatorExiting, ValidatorSlashed},
	ValidatorExiting:   {ValidatorWithdrawn},
	ValidatorSlashed:   {ValidatorWithdrawn},
	ValidatorWithdrawn: {}, // terminal
}

// CanTransition returns whether transitioning from current to next is allowed.
func CanTransition(current, next ValidatorState) bool {
	allowed, exists := ValidTransitions[current]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// ValidatorInfo holds metadata and state for a single beacon chain validator.
type ValidatorInfo struct {
	Index             uint64         `json:"index"`
	PublicKey         string         `json:"public_key"`
	State             ValidatorState `json:"state"`
	ActivationEpoch   uint64         `json:"activation_epoch"`
	ExitEpoch         uint64         `json:"exit_epoch"`
	WithdrawableEpoch uint64         `json:"withdrawable_epoch"`
	EffectiveBalance  uint64         `json:"effective_balance"` // in Gwei (max 32 ETH = 32_000_000_000)
	Slashed           bool           `json:"slashed"`
}

// IsEligibleForActivation returns true if the validator is pending and has reached its activation epoch.
func (v *ValidatorInfo) IsEligibleForActivation(currentEpoch uint64) bool {
	return v.State == ValidatorPending && v.ActivationEpoch <= currentEpoch
}

// IsEligibleForWithdrawal returns true if the validator has exited and reached its withdrawable epoch.
func (v *ValidatorInfo) IsEligibleForWithdrawal(currentEpoch uint64) bool {
	return (v.State == ValidatorExiting || v.State == ValidatorSlashed) &&
		v.WithdrawableEpoch <= currentEpoch
}

// EffectiveBalanceEth returns the effective balance in ETH (1 ETH = 1e9 Gwei).
func (v *ValidatorInfo) EffectiveBalanceEth() float64 {
	return float64(v.EffectiveBalance) / 1e9
}

// ─── Validator Lifecycle Tracker ─────────────────────────────────────────────

// ValidatorLifecycleTracker tracks validator state transitions and computes
// aggregate statistics for the active validator set.
type ValidatorLifecycleTracker struct {
	mu          sync.RWMutex
	validators  map[uint64]*ValidatorInfo
	stateCounts map[ValidatorState]int
}

// NewValidatorLifecycleTracker creates a new tracker.
func NewValidatorLifecycleTracker() *ValidatorLifecycleTracker {
	return &ValidatorLifecycleTracker{
		validators:  make(map[uint64]*ValidatorInfo),
		stateCounts: make(map[ValidatorState]int),
	}
}

// Register adds a validator to the tracker.
func (t *ValidatorLifecycleTracker) Register(v *ValidatorInfo) error {
	if _, exists := t.validators[v.Index]; exists {
		return fmt.Errorf("validator %d already registered", v.Index)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.validators[v.Index] = v
	t.stateCounts[v.State]++
	return nil
}

// Transition moves a validator from one state to another if the transition is valid.
func (t *ValidatorLifecycleTracker) Transition(index uint64, newState ValidatorState) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	v, exists := t.validators[index]
	if !exists {
		return fmt.Errorf("validator %d not found", index)
	}

	if !CanTransition(v.State, newState) {
		return fmt.Errorf("invalid transition: %s → %s for validator %d", v.State, newState, index)
	}

	t.stateCounts[v.State]--
	v.State = newState
	t.stateCounts[v.State]++

	if newState == ValidatorSlashed {
		v.Slashed = true
	}

	return nil
}

// GetValidator returns a copy of the validator info.
func (t *ValidatorLifecycleTracker) GetValidator(index uint64) (*ValidatorInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	v, exists := t.validators[index]
	if !exists {
		return nil, false
	}
	cp := *v
	return &cp, true
}

// StateCounts returns the current count of validators in each state.
func (t *ValidatorLifecycleTracker) StateCounts() map[ValidatorState]int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cp := make(map[ValidatorState]int, len(t.stateCounts))
	for k, v := range t.stateCounts {
		cp[k] = v
	}
	return cp
}

// ActiveCount returns the number of active validators.
func (t *ValidatorLifecycleTracker) ActiveCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stateCounts[ValidatorActive]
}

// TotalCount returns the total number of tracked validators.
func (t *ValidatorLifecycleTracker) TotalCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	total := 0
	for _, c := range t.stateCounts {
		total += c
	}
	return total
}

// ProcessEpoch processes pending state transitions for a given epoch.
func (t *ValidatorLifecycleTracker) ProcessEpoch(epoch uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, v := range t.validators {
		// Pending → Active
		if v.State == ValidatorPending && v.ActivationEpoch <= epoch {
			t.stateCounts[ValidatorPending]--
			v.State = ValidatorActive
			t.stateCounts[ValidatorActive]++
		}

		// Exiting → Withdrawn
		if v.State == ValidatorExiting && v.WithdrawableEpoch <= epoch {
			t.stateCounts[ValidatorExiting]--
			v.State = ValidatorWithdrawn
			t.stateCounts[ValidatorWithdrawn]++
		}

		// Slashed → Withdrawn
		if v.State == ValidatorSlashed && v.WithdrawableEpoch <= epoch {
			t.stateCounts[ValidatorSlashed]--
			v.State = ValidatorWithdrawn
			t.stateCounts[ValidatorWithdrawn]++
		}
	}
}

// ─── Attestation Statistics ──────────────────────────────────────────────────

// AttestationStats tracks attestation participation and inclusion quality.
type AttestationStats struct {
	mu sync.RWMutex

	// Per-epoch attestation participation
	TotalSlots    uint64 `json:"total_slots"`
	AttestedSlots uint64 `json:"attested_slots"`
	MissedSlots   uint64 `json:"missed_slots"`

	// Inclusion distance tracking
	// An attestation's inclusion distance is the number of slots between the
	// attested slot and the slot where the attestation is included on-chain.
	// Optimal = 1 (next slot), max meaningful = SLOTS_PER_EPOCH (32).
	totalDistance uint64
	distanceCount uint64
	maxDistance   uint64
	minDistance   uint64

	// Source/Target/Head correctness (simplified)
	SourceCorrect     uint64 `json:"source_correct"`
	TargetCorrect     uint64 `json:"target_correct"`
	HeadCorrect       uint64 `json:"head_correct"`
	TotalAttestations uint64 `json:"total_attestations"`
}

// NewAttestationStats creates a new attestation stats tracker.
func NewAttestationStats() *AttestationStats {
	return &AttestationStats{
		minDistance: math.MaxUint64,
	}
}

// RecordAttestation records a single attestation with its inclusion distance and
// correctness flags.
func (a *AttestationStats) RecordAttestation(inclusionDistance uint64, sourceCorrect, targetCorrect, headCorrect bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.TotalAttestations++
	a.totalDistance += inclusionDistance
	a.distanceCount++

	if inclusionDistance > a.maxDistance {
		a.maxDistance = inclusionDistance
	}
	if inclusionDistance < a.minDistance {
		a.minDistance = inclusionDistance
	}

	if sourceCorrect {
		a.SourceCorrect++
	}
	if targetCorrect {
		a.TargetCorrect++
	}
	if headCorrect {
		a.HeadCorrect++
	}
}

// RecordSlot records whether a slot was attested or missed.
func (a *AttestationStats) RecordSlot(attested bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.TotalSlots++
	if attested {
		a.AttestedSlots++
	} else {
		a.MissedSlots++
	}
}

// ParticipationRate returns the fraction of slots that were attested (0.0 – 1.0).
func (a *AttestationStats) ParticipationRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.TotalSlots == 0 {
		return 0
	}
	return float64(a.AttestedSlots) / float64(a.TotalSlots)
}

// AvgInclusionDistance returns the average inclusion distance across all recorded attestations.
func (a *AttestationStats) AvgInclusionDistance() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.distanceCount == 0 {
		return 0
	}
	return float64(a.totalDistance) / float64(a.distanceCount)
}

// MinInclusionDistance returns the minimum observed inclusion distance.
func (a *AttestationStats) MinInclusionDistance() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.distanceCount == 0 {
		return 0
	}
	return a.minDistance
}

// MaxInclusionDistance returns the maximum observed inclusion distance.
func (a *AttestationStats) MaxInclusionDistance() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.maxDistance
}

// SourceCorrectRate returns the fraction of attestations with correct source vote.
func (a *AttestationStats) SourceCorrectRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.TotalAttestations == 0 {
		return 0
	}
	return float64(a.SourceCorrect) / float64(a.TotalAttestations)
}

// TargetCorrectRate returns the fraction of attestations with correct target vote.
func (a *AttestationStats) TargetCorrectRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.TotalAttestations == 0 {
		return 0
	}
	return float64(a.TargetCorrect) / float64(a.TotalAttestations)
}

// HeadCorrectRate returns the fraction of attestations with correct head vote.
func (a *AttestationStats) HeadCorrectRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.TotalAttestations == 0 {
		return 0
	}
	return float64(a.HeadCorrect) / float64(a.TotalAttestations)
}

// AttestationRewardEstimate estimates the attestation reward for a validator
// based on inclusion distance. Shorter distances yield higher rewards.
// Based on the Ethereum spec: base_reward * (1 - (distance-1)/SLOTS_PER_EPOCH).
func AttestationRewardEstimate(baseRewardGwei uint64, inclusionDistance uint64) uint64 {
	if inclusionDistance == 0 {
		inclusionDistance = 1
	}
	if inclusionDistance > uint64(SlotsPerEpoch) {
		return 0 // too late, no reward
	}
	penaltyFactor := float64(inclusionDistance-1) / float64(SlotsPerEpoch)
	return uint64(float64(baseRewardGwei) * (1 - penaltyFactor))
}

// ─── Sync Committee ──────────────────────────────────────────────────────────

// Sync committee constants
const (
	// SyncCommitteeSize is the number of validators in a sync committee (512).
	SyncCommitteeSize = 512

	// SyncCommitteeSubcommitteeSize is the size of each subcommittee (128 = 512/4).
	SyncCommitteeSubcommitteeSize = SyncCommitteeSize / 4

	// SyncCommitteePeriodEpochs is the number of epochs in one sync committee period (256).
	SyncCommitteePeriodEpochs = 256

	// SyncCommitteePeriodDuration is the wall-clock duration of one sync committee period.
	SyncCommitteePeriodDuration = EpochDuration * SyncCommitteePeriodEpochs
)

// SyncCommitteeInfo holds metadata about the current sync committee.
type SyncCommitteeInfo struct {
	PeriodStartEpoch  uint64      `json:"period_start_epoch"`
	PeriodEndEpoch    uint64      `json:"period_end_epoch"`
	Validators        []uint64    `json:"validators"`         // validator indices
	Subcommittees     [4][]uint64 `json:"subcommittees"`      // 4 subcommittees of 128 each
	AggregateBitfield []byte      `json:"aggregate_bitfield"` // participation bitfield
}

// SyncCommitteePeriodForEpoch returns the sync committee period for a given epoch.
func SyncCommitteePeriodForEpoch(epoch uint64) uint64 {
	return epoch / SyncCommitteePeriodEpochs
}

// IsSyncCommitteeRotationEpoch returns true if the given epoch is the first epoch
// of a new sync committee period, i.e., when the sync committee rotates.
func IsSyncCommitteeRotationEpoch(epoch uint64) bool {
	return epoch%SyncCommitteePeriodEpochs == 0
}

// NewSyncCommitteeInfo creates a SyncCommitteeInfo for a given period.
// The validators list must contain exactly SyncCommitteeSize entries.
func NewSyncCommitteeInfo(periodEpoch uint64, validators []uint64) (*SyncCommitteeInfo, error) {
	if len(validators) != SyncCommitteeSize {
		return nil, fmt.Errorf("sync committee must have %d validators, got %d", SyncCommitteeSize, len(validators))
	}

	info := &SyncCommitteeInfo{
		PeriodStartEpoch:  periodEpoch,
		PeriodEndEpoch:    periodEpoch + SyncCommitteePeriodEpochs - 1,
		Validators:        validators,
		AggregateBitfield: make([]byte, SyncCommitteeSize/8), // 64 bytes
	}

	// Split into 4 subcommittees
	for i, vIdx := range validators {
		subIdx := i / SyncCommitteeSubcommitteeSize
		info.Subcommittees[subIdx] = append(info.Subcommittees[subIdx], vIdx)
	}

	return info, nil
}

// IsMember returns true if the given validator index is a member of this sync committee.
func (s *SyncCommitteeInfo) IsMember(validatorIndex uint64) bool {
	for _, v := range s.Validators {
		if v == validatorIndex {
			return true
		}
	}
	return false
}

// SubcommitteeIndex returns which subcommittee (0-3) a validator belongs to,
// or -1 if the validator is not a member.
func (s *SyncCommitteeInfo) SubcommitteeIndex(validatorIndex uint64) int {
	for i, sub := range s.Subcommittees {
		for _, v := range sub {
			if v == validatorIndex {
				return i
			}
		}
	}
	return -1
}

// RecordParticipation marks a validator as having participated in the sync committee
// for the current slot. The bitfield is indexed by position in the committee, not
// by validator index.
func (s *SyncCommitteeInfo) RecordParticipation(committeePosition int) {
	if committeePosition < 0 || committeePosition >= SyncCommitteeSize {
		return
	}
	byteIdx := committeePosition / 8
	bitIdx := uint(committeePosition % 8)
	s.AggregateBitfield[byteIdx] |= 1 << bitIdx
}

// ParticipationCount returns how many sync committee members participated.
func (s *SyncCommitteeInfo) ParticipationCount() int {
	count := 0
	for _, b := range s.AggregateBitfield {
		for b != 0 {
			count++
			b &= b - 1 // clear lowest set bit
		}
	}
	return count
}

// ParticipationRate returns the sync committee participation fraction (0.0 – 1.0).
func (s *SyncCommitteeInfo) ParticipationRate() float64 {
	return float64(s.ParticipationCount()) / float64(SyncCommitteeSize)
}

// TimeUntilNextSyncCommitteePeriod returns the duration until the next sync
// committee rotation from the given slot.
func TimeUntilNextSyncCommitteePeriod(slot uint64) time.Duration {
	currentEpoch := SlotToEpoch(slot)
	currentPeriod := SyncCommitteePeriodForEpoch(currentEpoch)
	nextPeriodStartEpoch := (currentPeriod + 1) * SyncCommitteePeriodEpochs
	slotsUntilNext := EpochFirstSlot(nextPeriodStartEpoch) - slot
	return time.Duration(slotsUntilNext) * SlotDuration
}
