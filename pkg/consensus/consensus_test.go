package consensus

import (
	"testing"
	"time"
)

func TestCanTransition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		from, to ValidatorState
		want     bool
	}{
		{ValidatorPending, ValidatorActive, true},
		{ValidatorActive, ValidatorExiting, true},
		{ValidatorActive, ValidatorSlashed, true},
		{ValidatorExiting, ValidatorWithdrawn, true},
		{ValidatorSlashed, ValidatorWithdrawn, true},
		// Invalid transitions
		{ValidatorPending, ValidatorSlashed, false},
		{ValidatorPending, ValidatorWithdrawn, false},
		{ValidatorActive, ValidatorPending, false},
		{ValidatorActive, ValidatorWithdrawn, false},
		{ValidatorWithdrawn, ValidatorActive, false},
		{ValidatorSlashed, ValidatorActive, false},
	}

	for _, tt := range tests {
		got := CanTransition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidatorLifecycleTracker_RegisterAndTransition(t *testing.T) {
	t.Parallel()
	tracker := NewValidatorLifecycleTracker()

	v := &ValidatorInfo{
		Index:            0,
		PublicKey:        "0xabc",
		State:            ValidatorPending,
		ActivationEpoch:  5,
		EffectiveBalance: 32_000_000_000,
	}

	if err := tracker.Register(v); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Duplicate should fail
	if err := tracker.Register(v); err == nil {
		t.Fatal("expected error for duplicate register")
	}

	// Invalid transition: Pending → Slashed
	if err := tracker.Transition(0, ValidatorSlashed); err == nil {
		t.Fatal("expected error for invalid transition Pending → Slashed")
	}

	// Valid transition: Pending → Active
	if err := tracker.Transition(0, ValidatorActive); err != nil {
		t.Fatalf("Transition to Active failed: %v", err)
	}

	got, exists := tracker.GetValidator(0)
	if !exists {
		t.Fatal("validator not found")
	}
	if got.State != ValidatorActive {
		t.Errorf("state = %s, want active", got.State)
	}
}

func TestValidatorLifecycleTracker_ProcessEpoch(t *testing.T) {
	t.Parallel()
	tracker := NewValidatorLifecycleTracker()

	// Register a pending validator with activation epoch 3
	tracker.Register(&ValidatorInfo{
		Index:            1,
		State:            ValidatorPending,
		ActivationEpoch:  3,
		EffectiveBalance: 32_000_000_000,
	})

	// Register an exiting validator with withdrawable epoch 5
	tracker.Register(&ValidatorInfo{
		Index:             2,
		State:             ValidatorExiting,
		WithdrawableEpoch: 5,
		EffectiveBalance:  32_000_000_000,
	})

	// Epoch 2: no changes yet
	tracker.ProcessEpoch(2)
	counts := tracker.StateCounts()
	if counts[ValidatorActive] != 0 {
		t.Errorf("active = %d, want 0 at epoch 2", counts[ValidatorActive])
	}

	// Epoch 3: validator 1 activates
	tracker.ProcessEpoch(3)
	counts = tracker.StateCounts()
	if counts[ValidatorActive] != 1 {
		t.Errorf("active = %d, want 1 at epoch 3", counts[ValidatorActive])
	}
	if counts[ValidatorPending] != 0 {
		t.Errorf("pending = %d, want 0 at epoch 3", counts[ValidatorPending])
	}

	// Epoch 5: validator 2 withdraws
	tracker.ProcessEpoch(5)
	counts = tracker.StateCounts()
	if counts[ValidatorWithdrawn] != 1 {
		t.Errorf("withdrawn = %d, want 1 at epoch 5", counts[ValidatorWithdrawn])
	}
	if counts[ValidatorExiting] != 0 {
		t.Errorf("exiting = %d, want 0 at epoch 5", counts[ValidatorExiting])
	}
}

func TestValidatorLifecycleTracker_SlashedTransition(t *testing.T) {
	t.Parallel()
	tracker := NewValidatorLifecycleTracker()
	tracker.Register(&ValidatorInfo{
		Index:            10,
		State:            ValidatorActive,
		EffectiveBalance: 32_000_000_000,
	})

	if err := tracker.Transition(10, ValidatorSlashed); err != nil {
		t.Fatalf("Transition to Slashed failed: %v", err)
	}

	v, _ := tracker.GetValidator(10)
	if !v.Slashed {
		t.Error("expected Slashed = true")
	}
	if v.State != ValidatorSlashed {
		t.Errorf("state = %s, want slashed", v.State)
	}
}

func TestValidatorInfo_Eligibility(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{
		State:           ValidatorPending,
		ActivationEpoch: 10,
	}
	if v.IsEligibleForActivation(9) {
		t.Error("should not be eligible at epoch 9")
	}
	if !v.IsEligibleForActivation(10) {
		t.Error("should be eligible at epoch 10")
	}

	v2 := &ValidatorInfo{
		State:             ValidatorExiting,
		WithdrawableEpoch: 20,
	}
	if v2.IsEligibleForWithdrawal(19) {
		t.Error("should not be eligible for withdrawal at epoch 19")
	}
	if !v2.IsEligibleForWithdrawal(20) {
		t.Error("should be eligible for withdrawal at epoch 20")
	}
}

func TestValidatorInfo_EffectiveBalanceEth(t *testing.T) {
	t.Parallel()
	v := &ValidatorInfo{EffectiveBalance: 32_000_000_000}
	if got := v.EffectiveBalanceEth(); got != 32.0 {
		t.Errorf("EffectiveBalanceEth() = %f, want 32.0", got)
	}
}

// ─── Attestation Stats Tests ─────────────────────────────────────────────────

func TestAttestationStats_ParticipationRate(t *testing.T) {
	t.Parallel()
	stats := NewAttestationStats()

	if rate := stats.ParticipationRate(); rate != 0 {
		t.Errorf("participation rate = %f, want 0 with no data", rate)
	}

	stats.RecordSlot(true)
	stats.RecordSlot(true)
	stats.RecordSlot(false)

	if rate := stats.ParticipationRate(); rate != 2.0/3.0 {
		t.Errorf("participation rate = %f, want %f", rate, 2.0/3.0)
	}
}

func TestAttestationStats_InclusionDistance(t *testing.T) {
	t.Parallel()
	stats := NewAttestationStats()

	stats.RecordAttestation(1, true, true, true)
	stats.RecordAttestation(3, true, true, false)
	stats.RecordAttestation(5, true, false, false)

	if avg := stats.AvgInclusionDistance(); avg != 3.0 {
		t.Errorf("avg distance = %f, want 3.0", avg)
	}
	if min := stats.MinInclusionDistance(); min != 1 {
		t.Errorf("min distance = %d, want 1", min)
	}
	if max := stats.MaxInclusionDistance(); max != 5 {
		t.Errorf("max distance = %d, want 5", max)
	}
}

func TestAttestationStats_CorrectnessRates(t *testing.T) {
	t.Parallel()
	stats := NewAttestationStats()

	// 4 attestations: all source correct, 3 target, 2 head
	stats.RecordAttestation(1, true, true, true)
	stats.RecordAttestation(1, true, true, true)
	stats.RecordAttestation(1, true, true, false)
	stats.RecordAttestation(1, true, false, false)

	if rate := stats.SourceCorrectRate(); rate != 1.0 {
		t.Errorf("source rate = %f, want 1.0", rate)
	}
	if rate := stats.TargetCorrectRate(); rate != 0.75 {
		t.Errorf("target rate = %f, want 0.75", rate)
	}
	if rate := stats.HeadCorrectRate(); rate != 0.5 {
		t.Errorf("head rate = %f, want 0.5", rate)
	}
}

func TestAttestationRewardEstimate(t *testing.T) {
	t.Parallel()
	baseReward := uint64(10000)

	// Distance 1: full reward (1 - 0/32) * 10000 = 10000
	reward := AttestationRewardEstimate(baseReward, 1)
	if reward != 10000 {
		t.Errorf("distance=1: reward=%d, want 10000", reward)
	}

	// Distance 2: (1 - 1/32) * 10000 = 9687
	reward = AttestationRewardEstimate(baseReward, 2)
	if reward < 9000 || reward > 10000 {
		t.Errorf("distance=2: reward=%d, expected ~9687", reward)
	}

	// Distance 32: (1 - 31/32) * 10000 = 312
	reward = AttestationRewardEstimate(baseReward, 32)
	if reward > 500 || reward == 0 {
		t.Errorf("distance=32: reward=%d, expected ~312", reward)
	}

	// Beyond epoch: no reward
	reward = AttestationRewardEstimate(baseReward, 33)
	if reward != 0 {
		t.Errorf("distance=33: reward=%d, want 0", reward)
	}
}

// ─── Sync Committee Tests ────────────────────────────────────────────────────

func TestSyncCommitteePeriodForEpoch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		epoch uint64
		want  uint64
	}{
		{0, 0},
		{255, 0},
		{256, 1},
		{511, 1},
		{512, 2},
	}
	for _, tt := range tests {
		got := SyncCommitteePeriodForEpoch(tt.epoch)
		if got != tt.want {
			t.Errorf("SyncCommitteePeriodForEpoch(%d) = %d, want %d", tt.epoch, got, tt.want)
		}
	}
}

func TestIsSyncCommitteeRotationEpoch(t *testing.T) {
	t.Parallel()
	if !IsSyncCommitteeRotationEpoch(0) {
		t.Error("epoch 0 should be rotation epoch")
	}
	if !IsSyncCommitteeRotationEpoch(256) {
		t.Error("epoch 256 should be rotation epoch")
	}
	if IsSyncCommitteeRotationEpoch(100) {
		t.Error("epoch 100 should not be rotation epoch")
	}
}

func TestNewSyncCommitteeInfo(t *testing.T) {
	t.Parallel()
	// Create validator list of 512
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}

	info, err := NewSyncCommitteeInfo(0, validators)
	if err != nil {
		t.Fatalf("NewSyncCommitteeInfo failed: %v", err)
	}

	if info.PeriodStartEpoch != 0 {
		t.Errorf("PeriodStartEpoch = %d, want 0", info.PeriodStartEpoch)
	}
	if info.PeriodEndEpoch != 255 {
		t.Errorf("PeriodEndEpoch = %d, want 255", info.PeriodEndEpoch)
	}

	// Check subcommittees
	for i := 0; i < 4; i++ {
		if len(info.Subcommittees[i]) != SyncCommitteeSubcommitteeSize {
			t.Errorf("subcommittee %d len = %d, want %d", i, len(info.Subcommittees[i]), SyncCommitteeSubcommitteeSize)
		}
	}

	// IsMember
	if !info.IsMember(0) {
		t.Error("validator 0 should be a member")
	}
	if info.IsMember(999) {
		t.Error("validator 999 should not be a member")
	}

	// SubcommitteeIndex
	if idx := info.SubcommitteeIndex(0); idx != 0 {
		t.Errorf("SubcommitteeIndex(0) = %d, want 0", idx)
	}
	if idx := info.SubcommitteeIndex(128); idx != 1 {
		t.Errorf("SubcommitteeIndex(128) = %d, want 1", idx)
	}
	if idx := info.SubcommitteeIndex(999); idx != -1 {
		t.Errorf("SubcommitteeIndex(999) = %d, want -1", idx)
	}
}

func TestNewSyncCommitteeInfo_WrongSize(t *testing.T) {
	t.Parallel()
	_, err := NewSyncCommitteeInfo(0, make([]uint64, 100))
	if err == nil {
		t.Fatal("expected error for wrong size")
	}
}

func TestSyncCommitteeInfo_Participation(t *testing.T) {
	t.Parallel()
	validators := make([]uint64, SyncCommitteeSize)
	for i := range validators {
		validators[i] = uint64(i)
	}
	info, _ := NewSyncCommitteeInfo(0, validators)

	// No participation yet
	if count := info.ParticipationCount(); count != 0 {
		t.Errorf("participation count = %d, want 0", count)
	}
	if rate := info.ParticipationRate(); rate != 0 {
		t.Errorf("participation rate = %f, want 0", rate)
	}

	// Record some participation
	info.RecordParticipation(0)
	info.RecordParticipation(1)
	info.RecordParticipation(511)

	if count := info.ParticipationCount(); count != 3 {
		t.Errorf("participation count = %d, want 3", count)
	}

	expectedRate := 3.0 / float64(SyncCommitteeSize)
	if rate := info.ParticipationRate(); rate != expectedRate {
		t.Errorf("participation rate = %f, want %f", rate, expectedRate)
	}
}

func TestSyncCommitteeInfo_RecordParticipation_Bounds(t *testing.T) {
	t.Parallel()
	validators := make([]uint64, SyncCommitteeSize)
	info, _ := NewSyncCommitteeInfo(0, validators)

	// Out of bounds — should not panic
	info.RecordParticipation(-1)
	info.RecordParticipation(SyncCommitteeSize)

	if count := info.ParticipationCount(); count != 0 {
		t.Errorf("participation count = %d, want 0 after out-of-bounds", count)
	}
}

func TestTimeUntilNextSyncCommitteePeriod(t *testing.T) {
	t.Parallel()
	// At epoch 0, slot 0: next period starts at epoch 256
	// Slots until = 256 * 32 = 8192
	// Duration = 8192 * 12s = 98304s
	dur := TimeUntilNextSyncCommitteePeriod(0)
	expected := time.Duration(8192) * SlotDuration
	if dur != expected {
		t.Errorf("TimeUntilNextSyncCommitteePeriod(0) = %v, want %v", dur, expected)
	}
}
