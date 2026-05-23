package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransition_Valid(t *testing.T) {
	t.Parallel()
	assert.True(t, CanTransition(ValidatorPending, ValidatorActive))
	assert.True(t, CanTransition(ValidatorActive, ValidatorExiting))
	assert.True(t, CanTransition(ValidatorActive, ValidatorSlashed))
	assert.True(t, CanTransition(ValidatorExiting, ValidatorWithdrawn))
	assert.True(t, CanTransition(ValidatorSlashed, ValidatorWithdrawn))
}

func TestCanTransition_Invalid(t *testing.T) {
	t.Parallel()
	assert.False(t, CanTransition(ValidatorPending, ValidatorWithdrawn))
	assert.False(t, CanTransition(ValidatorActive, ValidatorPending))
	assert.False(t, CanTransition(ValidatorWithdrawn, ValidatorActive))
	assert.False(t, CanTransition(ValidatorWithdrawn, ValidatorPending))
	assert.False(t, CanTransition(ValidatorExiting, ValidatorPending))
}

func TestCanTransition_UnknownState(t *testing.T) {
	t.Parallel()
	assert.False(t, CanTransition("nonexistent", ValidatorActive))
}

func TestCanTransition_WithdrawnIsTerminal(t *testing.T) {
	t.Parallel()
	assert.False(t, CanTransition(ValidatorWithdrawn, ValidatorActive))
	assert.False(t, CanTransition(ValidatorWithdrawn, ValidatorPending))
	assert.False(t, CanTransition(ValidatorWithdrawn, ValidatorExiting))
}

func TestValidatorInfo_IsEligibleForActivation(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{
		Index:           1,
		State:           ValidatorPending,
		ActivationEpoch: 10,
	}

	assert.False(t, v.IsEligibleForActivation(5))
	assert.True(t, v.IsEligibleForActivation(10))
	assert.True(t, v.IsEligibleForActivation(15))
}

func TestValidatorInfo_IsEligibleForActivationNotPending(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{
		Index:           1,
		State:           ValidatorActive,
		ActivationEpoch: 10,
	}

	assert.False(t, v.IsEligibleForActivation(10))
}

func TestValidatorInfo_IsEligibleForWithdrawal(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{
		Index:             1,
		State:             ValidatorExiting,
		WithdrawableEpoch: 20,
	}

	assert.False(t, v.IsEligibleForWithdrawal(15))
	assert.True(t, v.IsEligibleForWithdrawal(20))
	assert.True(t, v.IsEligibleForWithdrawal(25))
}

func TestValidatorInfo_IsEligibleForWithdrawalSlashed(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{
		Index:             1,
		State:             ValidatorSlashed,
		WithdrawableEpoch: 20,
	}

	assert.True(t, v.IsEligibleForWithdrawal(20))
	assert.False(t, v.IsEligibleForWithdrawal(10))
}

func TestValidatorInfo_IsEligibleForWithdrawalNotExiting(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{
		Index:             1,
		State:             ValidatorActive,
		WithdrawableEpoch: 20,
	}

	assert.False(t, v.IsEligibleForWithdrawal(20))
}

func TestValidatorInfo_EffectiveBalanceEth(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{EffectiveBalance: 32_000_000_000}
	assert.InDelta(t, 32.0, v.EffectiveBalanceEth(), 0.001)

	v2 := &ValidatorInfo{EffectiveBalance: 16_000_000_000}
	assert.InDelta(t, 16.0, v2.EffectiveBalanceEth(), 0.001)

	v3 := &ValidatorInfo{EffectiveBalance: 0}
	assert.Equal(t, 0.0, v3.EffectiveBalanceEth())
}

func TestNewValidatorLifecycleTracker(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()
	assert.NotNil(t, tracker)
	assert.Equal(t, 0, tracker.TotalCount())
	assert.Equal(t, 0, tracker.ActiveCount())
}

func TestValidatorLifecycleTracker_Register(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	v := &ValidatorInfo{Index: 1, State: ValidatorPending}
	err := tracker.Register(v)
	assert.NoError(t, err)
	assert.Equal(t, 1, tracker.TotalCount())

	got, ok := tracker.GetValidator(1)
	assert.True(t, ok)
	assert.Equal(t, uint64(1), got.Index)
}

func TestValidatorLifecycleTracker_RegisterDuplicate(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	v := &ValidatorInfo{Index: 1, State: ValidatorPending}
	_ = tracker.Register(v)
	err := tracker.Register(v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestValidatorLifecycleTracker_Transition(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorPending})

	err := tracker.Transition(1, ValidatorActive)
	assert.NoError(t, err)

	got, ok := tracker.GetValidator(1)
	assert.True(t, ok)
	assert.Equal(t, ValidatorActive, got.State)
}

func TestValidatorLifecycleTracker_TransitionInvalid(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorPending})

	err := tracker.Transition(1, ValidatorWithdrawn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
}

func TestValidatorLifecycleTracker_TransitionNotFound(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	err := tracker.Transition(999, ValidatorActive)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidatorLifecycleTracker_TransitionSetsSlashed(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorActive})
	_ = tracker.Transition(1, ValidatorSlashed)

	got, _ := tracker.GetValidator(1)
	assert.True(t, got.Slashed)
}

func TestValidatorLifecycleTracker_StateCounts(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorPending})
	_ = tracker.Register(&ValidatorInfo{Index: 2, State: ValidatorActive})
	_ = tracker.Register(&ValidatorInfo{Index: 3, State: ValidatorActive})

	counts := tracker.StateCounts()
	assert.Equal(t, 1, counts[ValidatorPending])
	assert.Equal(t, 2, counts[ValidatorActive])
}

func TestValidatorLifecycleTracker_ActiveCount(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorActive})
	_ = tracker.Register(&ValidatorInfo{Index: 2, State: ValidatorActive})

	assert.Equal(t, 2, tracker.ActiveCount())
}

func TestValidatorLifecycleTracker_TotalCount(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorPending})
	_ = tracker.Register(&ValidatorInfo{Index: 2, State: ValidatorActive})
	_ = tracker.Register(&ValidatorInfo{Index: 3, State: ValidatorExiting})

	assert.Equal(t, 3, tracker.TotalCount())
}

func TestValidatorLifecycleTracker_GetValidatorNotFound(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()
	_, ok := tracker.GetValidator(999)
	assert.False(t, ok)
}

func TestValidatorLifecycleTracker_GetValidatorReturnsCopy(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()
	v := &ValidatorInfo{Index: 1, State: ValidatorPending, EffectiveBalance: 100}
	_ = tracker.Register(v)

	got, _ := tracker.GetValidator(1)
	got.EffectiveBalance = 200

	got2, _ := tracker.GetValidator(1)
	assert.Equal(t, uint64(100), got2.EffectiveBalance)
}

func TestValidatorLifecycleTracker_ProcessEpoch(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorPending, ActivationEpoch: 10})
	_ = tracker.Register(&ValidatorInfo{Index: 2, State: ValidatorExiting, WithdrawableEpoch: 10})
	_ = tracker.Register(&ValidatorInfo{Index: 3, State: ValidatorSlashed, WithdrawableEpoch: 10})
	_ = tracker.Register(&ValidatorInfo{Index: 4, State: ValidatorActive})

	tracker.ProcessEpoch(10)

	got1, _ := tracker.GetValidator(1)
	assert.Equal(t, ValidatorActive, got1.State)

	got2, _ := tracker.GetValidator(2)
	assert.Equal(t, ValidatorWithdrawn, got2.State)

	got3, _ := tracker.GetValidator(3)
	assert.Equal(t, ValidatorWithdrawn, got3.State)

	got4, _ := tracker.GetValidator(4)
	assert.Equal(t, ValidatorActive, got4.State)
}

func TestValidatorLifecycleTracker_ProcessEpochNotYetEligible(t *testing.T) {
	tracker := NewValidatorLifecycleTracker()

	_ = tracker.Register(&ValidatorInfo{Index: 1, State: ValidatorPending, ActivationEpoch: 20})

	tracker.ProcessEpoch(10)

	got, _ := tracker.GetValidator(1)
	assert.Equal(t, ValidatorPending, got.State)
}

func TestNewAttestationStats(t *testing.T) {
	stats := NewAttestationStats()
	assert.NotNil(t, stats)
	assert.Equal(t, uint64(0), stats.TotalAttestations)
}

func TestAttestationStats_RecordAttestation(t *testing.T) {
	stats := NewAttestationStats()

	stats.RecordAttestation(2, true, true, true)
	assert.Equal(t, uint64(1), stats.TotalAttestations)
	assert.Equal(t, uint64(1), stats.SourceCorrect)
	assert.Equal(t, uint64(1), stats.TargetCorrect)
	assert.Equal(t, uint64(1), stats.HeadCorrect)
}

func TestAttestationStats_RecordAttestationMixed(t *testing.T) {
	stats := NewAttestationStats()

	stats.RecordAttestation(2, true, false, true)
	assert.Equal(t, uint64(1), stats.SourceCorrect)
	assert.Equal(t, uint64(0), stats.TargetCorrect)
	assert.Equal(t, uint64(1), stats.HeadCorrect)
}

func TestAttestationStats_InclusionDistance(t *testing.T) {
	stats := NewAttestationStats()

	stats.RecordAttestation(5, true, true, true)
	stats.RecordAttestation(3, true, true, true)

	assert.Equal(t, float64(4), stats.AvgInclusionDistance())
	assert.Equal(t, uint64(3), stats.MinInclusionDistance())
	assert.Equal(t, uint64(5), stats.MaxInclusionDistance())
}

func TestAttestationStats_ParticipationRate(t *testing.T) {
	stats := NewAttestationStats()

	stats.RecordSlot(true)
	stats.RecordSlot(true)
	stats.RecordSlot(false)

	assert.Equal(t, 2.0/3.0, stats.ParticipationRate())
}

func TestAttestationStats_ParticipationRateEmpty(t *testing.T) {
	stats := NewAttestationStats()
	assert.Equal(t, 0.0, stats.ParticipationRate())
}

func TestAttestationStats_AvgInclusionDistanceEmpty(t *testing.T) {
	stats := NewAttestationStats()
	assert.Equal(t, 0.0, stats.AvgInclusionDistance())
}

func TestAttestationStats_MinInclusionDistanceEmpty(t *testing.T) {
	stats := NewAttestationStats()
	assert.Equal(t, uint64(0), stats.MinInclusionDistance())
}

func TestAttestationStats_SourceCorrectRate(t *testing.T) {
	stats := NewAttestationStats()

	stats.RecordAttestation(1, true, true, true)
	stats.RecordAttestation(1, false, true, true)

	assert.Equal(t, 0.5, stats.SourceCorrectRate())
	assert.Equal(t, 1.0, stats.TargetCorrectRate())
	assert.Equal(t, 1.0, stats.HeadCorrectRate())
}

func TestAttestationStats_SourceCorrectRateEmpty(t *testing.T) {
	stats := NewAttestationStats()
	assert.Equal(t, 0.0, stats.SourceCorrectRate())
}

func TestAttestationRewardEstimate(t *testing.T) {
	t.Parallel()
	reward := AttestationRewardEstimate(1000, 1)
	assert.Greater(t, reward, uint64(0))

	reward = AttestationRewardEstimate(1000, 32)
	assert.GreaterOrEqual(t, reward, uint64(0))

	reward = AttestationRewardEstimate(1000, 33)
	assert.Equal(t, uint64(0), reward)

	reward = AttestationRewardEstimate(1000, 0)
	assert.Greater(t, reward, uint64(0))
}

func TestSyncCommitteePeriodForEpoch(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(0), SyncCommitteePeriodForEpoch(0))
	assert.Equal(t, uint64(0), SyncCommitteePeriodForEpoch(255))
	assert.Equal(t, uint64(1), SyncCommitteePeriodForEpoch(256))
	assert.Equal(t, uint64(1), SyncCommitteePeriodForEpoch(511))
}

func TestIsSyncCommitteeRotationEpoch(t *testing.T) {
	t.Parallel()
	assert.True(t, IsSyncCommitteeRotationEpoch(0))
	assert.True(t, IsSyncCommitteeRotationEpoch(256))
	assert.True(t, IsSyncCommitteeRotationEpoch(512))
	assert.False(t, IsSyncCommitteeRotationEpoch(1))
	assert.False(t, IsSyncCommitteeRotationEpoch(100))
}

func TestNewSyncCommitteeInfo(t *testing.T) {
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}

	info, err := NewSyncCommitteeInfo(0, validators)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, uint64(0), info.PeriodStartEpoch)
	assert.Equal(t, uint64(255), info.PeriodEndEpoch)
	assert.Len(t, info.Subcommittees[0], SyncCommitteeSubcommitteeSize)
	assert.Len(t, info.Subcommittees[1], SyncCommitteeSubcommitteeSize)
	assert.Len(t, info.Subcommittees[2], SyncCommitteeSubcommitteeSize)
	assert.Len(t, info.Subcommittees[3], SyncCommitteeSubcommitteeSize)
}

func TestNewSyncCommitteeInfo_WrongSize(t *testing.T) {
	validators := make([]uint64, 100)
	_, err := NewSyncCommitteeInfo(0, validators)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have 512 validators")
}

func TestSyncCommitteeInfo_IsMember(t *testing.T) {
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i * 2)
	}

	info, _ := NewSyncCommitteeInfo(0, validators)
	assert.True(t, info.IsMember(0))
	assert.True(t, info.IsMember(2))
	assert.False(t, info.IsMember(1))
	assert.False(t, info.IsMember(9999))
}

func TestSyncCommitteeInfo_SubcommitteeIndex(t *testing.T) {
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}

	info, _ := NewSyncCommitteeInfo(0, validators)
	assert.Equal(t, 0, info.SubcommitteeIndex(0))
	assert.Equal(t, 1, info.SubcommitteeIndex(128))
	assert.Equal(t, 2, info.SubcommitteeIndex(256))
	assert.Equal(t, 3, info.SubcommitteeIndex(384))
	assert.Equal(t, -1, info.SubcommitteeIndex(9999))
}

func TestSyncCommitteeInfo_RecordParticipation(t *testing.T) {
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}

	info, _ := NewSyncCommitteeInfo(0, validators)

	assert.Equal(t, 0, info.ParticipationCount())

	info.RecordParticipation(0)
	info.RecordParticipation(1)
	info.RecordParticipation(100)

	assert.Equal(t, 3, info.ParticipationCount())
}

func TestSyncCommitteeInfo_RecordParticipationOutOfRange(t *testing.T) {
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}

	info, _ := NewSyncCommitteeInfo(0, validators)

	info.RecordParticipation(-1)
	info.RecordParticipation(512)

	assert.Equal(t, 0, info.ParticipationCount())
}

func TestSyncCommitteeInfo_ParticipationRate(t *testing.T) {
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}

	info, _ := NewSyncCommitteeInfo(0, validators)

	half := SyncCommitteeSize / 2
	for i := 0; i < half; i++ {
		info.RecordParticipation(i)
	}

	assert.InDelta(t, 0.5, info.ParticipationRate(), 0.01)

	for i := half; i < SyncCommitteeSize; i++ {
		info.RecordParticipation(i)
	}

	assert.Equal(t, 1.0, info.ParticipationRate())
}

func TestTimeUntilNextSyncCommitteePeriod(t *testing.T) {
	slot := uint64(100)
	d := TimeUntilNextSyncCommitteePeriod(slot)
	assert.Greater(t, d.Milliseconds(), int64(0))
}
