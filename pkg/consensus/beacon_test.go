package consensus

import (
	"testing"
	"time"
)

func TestSlotToEpoch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		slot  uint64
		epoch uint64
	}{
		{"slot_0", 0, 0},
		{"slot_31", 31, 0},
		{"slot_32", 32, 1},
		{"slot_63", 63, 1},
		{"slot_64", 64, 2},
		{"large_slot", 3200, 100},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SlotToEpoch(tc.slot); got != tc.epoch {
				t.Errorf("SlotToEpoch(%d) = %d, want %d", tc.slot, got, tc.epoch)
			}
		})
	}
}

func TestIsEpochBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slot    uint64
		isBound bool
	}{
		{"slot_0", 0, true},
		{"slot_1", 1, false},
		{"slot_31", 31, false},
		{"slot_32", 32, true},
		{"slot_64", 64, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsEpochBoundary(tc.slot); got != tc.isBound {
				t.Errorf("IsEpochBoundary(%d) = %v, want %v", tc.slot, got, tc.isBound)
			}
		})
	}
}

func TestEpochFirstSlot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		epoch uint64
		first uint64
	}{
		{"epoch_0", 0, 0},
		{"epoch_1", 1, 32},
		{"epoch_2", 2, 64},
		{"epoch_100", 100, 3200},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := EpochFirstSlot(tc.epoch); got != tc.first {
				t.Errorf("EpochFirstSlot(%d) = %d, want %d", tc.epoch, got, tc.first)
			}
		})
	}
}

func TestTimestampToSlot(t *testing.T) {
	t.Parallel()

	genesis := int64(1606824023)

	tests := []struct {
		name      string
		timestamp int64
		want      uint64
	}{
		{"genesis_exact", genesis, 0},
		{"before_genesis", genesis - 100, 0},
		{"one_slot_after", genesis + 12, 1},
		{"half_slot", genesis + 6, 0},
		{"ten_slots", genesis + 120, 10},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := TimestampToSlot(tc.timestamp, genesis); got != tc.want {
				t.Errorf("TimestampToSlot(%d, %d) = %d, want %d", tc.timestamp, genesis, got, tc.want)
			}
		})
	}
}

func TestSlotToTimestamp(t *testing.T) {
	t.Parallel()

	genesis := int64(1606824023)

	tests := []struct {
		name string
		slot uint64
		want int64
	}{
		{"slot_0", 0, genesis},
		{"slot_1", 1, genesis + 12},
		{"slot_10", 10, genesis + 120},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SlotToTimestamp(tc.slot, genesis); got != tc.want {
				t.Errorf("SlotToTimestamp(%d, %d) = %d, want %d", tc.slot, genesis, got, tc.want)
			}
		})
	}
}

func TestDetectMissedSlots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		parentSlot  uint64
		currentSlot uint64
		wantLen     int
	}{
		{"consecutive", 5, 6, 0},
		{"same_slot", 5, 5, 0},
		{"child_before_parent", 10, 5, 0},
		{"one_missed", 5, 7, 1},
		{"three_missed", 5, 9, 3},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := DetectMissedSlots(tc.parentSlot, tc.currentSlot)
			if len(got) != tc.wantLen {
				t.Errorf("DetectMissedSlots(%d, %d) len = %d, want %d", tc.parentSlot, tc.currentSlot, len(got), tc.wantLen)
			}
			if tc.wantLen > 0 && got[0] != tc.parentSlot+1 {
				t.Errorf("first missed slot = %d, want %d", got[0], tc.parentSlot+1)
			}
		})
	}
}

func TestExpectedSlotNumber(t *testing.T) {
	t.Parallel()

	genesis := int64(1606824023)
	got := ExpectedSlotNumber(genesis+120, genesis)
	if got != 10 {
		t.Errorf("ExpectedSlotNumber = %d, want 10", got)
	}
}

func TestSlotsUntilNextEpoch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		slot uint64
		want uint64
	}{
		{"slot_0", 0, 32},
		{"slot_1", 1, 31},
		{"slot_31", 31, 1},
		{"slot_32", 32, 32},
		{"slot_63", 63, 1},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SlotsUntilNextEpoch(tc.slot); got != tc.want {
				t.Errorf("SlotsUntilNextEpoch(%d) = %d, want %d", tc.slot, got, tc.want)
			}
		})
	}
}

func TestTimeUntilNextEpoch(t *testing.T) {
	t.Parallel()

	got := TimeUntilNextEpoch(0)
	if got != 32*SlotDuration {
		t.Errorf("TimeUntilNextEpoch(0) = %v, want %v", got, 32*SlotDuration)
	}

	got2 := TimeUntilNextEpoch(31)
	if got2 != SlotDuration {
		t.Errorf("TimeUntilNextEpoch(31) = %v, want %v", got2, SlotDuration)
	}
}

func TestBeaconConstants(t *testing.T) {
	t.Parallel()

	if SlotDuration != 12*time.Second {
		t.Errorf("SlotDuration = %v", SlotDuration)
	}
	if SlotsPerEpoch != 32 {
		t.Errorf("SlotsPerEpoch = %d", SlotsPerEpoch)
	}
	if EpochDuration != 32*12*time.Second {
		t.Errorf("EpochDuration = %v", EpochDuration)
	}
}

func TestNewBeaconBlockInfo(t *testing.T) {
	t.Parallel()

	genesis := int64(1606824023)

	tests := []struct {
		name           string
		blockTimestamp int64
		genesisTime    int64
		parentSlot     uint64
		wantSlot       uint64
		wantEpoch      uint64
		wantMissed     bool
	}{
		{
			name:           "consecutive_no_missed",
			blockTimestamp: genesis + 24,
			genesisTime:    genesis,
			parentSlot:     1,
			wantSlot:       2,
			wantEpoch:      0,
			wantMissed:     false,
		},
		{
			name:           "one_missed_slot",
			blockTimestamp: genesis + 36,
			genesisTime:    genesis,
			parentSlot:     1,
			wantSlot:       3,
			wantEpoch:      0,
			wantMissed:     true,
		},
		{
			name:           "same_slot",
			blockTimestamp: genesis + 12,
			genesisTime:    genesis,
			parentSlot:     1,
			wantSlot:       1,
			wantEpoch:      0,
			wantMissed:     false,
		},
		{
			name:           "epoch_boundary",
			blockTimestamp: genesis + 32*12,
			genesisTime:    genesis,
			parentSlot:     31,
			wantSlot:       32,
			wantEpoch:      1,
			wantMissed:     false,
		},
		{
			name:           "multiple_missed",
			blockTimestamp: genesis + 60,
			genesisTime:    genesis,
			parentSlot:     0,
			wantSlot:       5,
			wantEpoch:      0,
			wantMissed:     true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := NewBeaconBlockInfo(tc.blockTimestamp, tc.genesisTime, tc.parentSlot)
			if info.Slot != tc.wantSlot {
				t.Errorf("Slot = %d, want %d", info.Slot, tc.wantSlot)
			}
			if info.Epoch != tc.wantEpoch {
				t.Errorf("Epoch = %d, want %d", info.Epoch, tc.wantEpoch)
			}
			if info.IsMissedSlot != tc.wantMissed {
				t.Errorf("IsMissedSlot = %v, want %v", info.IsMissedSlot, tc.wantMissed)
			}
		})
	}
}
