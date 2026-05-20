package consensus

import (
	"testing"
	"time"
)

func TestSlotToEpoch(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{0, 0},
		{31, 0},
		{32, 1},
		{63, 1},
		{64, 2},
		{100, 3},
	}
	for _, tt := range tests {
		got := SlotToEpoch(tt.slot)
		if got != tt.epoch {
			t.Errorf("SlotToEpoch(%d) = %d, want %d", tt.slot, got, tt.epoch)
		}
	}
}

func TestIsEpochBoundary(t *testing.T) {
	tests := []struct {
		slot uint64
		want bool
	}{
		{0, true},
		{1, false},
		{31, false},
		{32, true},
		{64, true},
		{100, false},
	}
	for _, tt := range tests {
		got := IsEpochBoundary(tt.slot)
		if got != tt.want {
			t.Errorf("IsEpochBoundary(%d) = %v, want %v", tt.slot, got, tt.want)
		}
	}
}

func TestEpochFirstSlot(t *testing.T) {
	tests := []struct {
		epoch uint64
		slot  uint64
	}{
		{0, 0},
		{1, 32},
		{2, 64},
		{10, 320},
	}
	for _, tt := range tests {
		got := EpochFirstSlot(tt.epoch)
		if got != tt.slot {
			t.Errorf("EpochFirstSlot(%d) = %d, want %d", tt.epoch, got, tt.slot)
		}
	}
}

func TestTimestampToSlot(t *testing.T) {
	genesis := MainnetGenesisTime

	// At genesis time, slot = 0
	if s := TimestampToSlot(genesis, genesis); s != 0 {
		t.Errorf("TimestampToSlot(genesis) = %d, want 0", s)
	}

	// 12 seconds after genesis = slot 1
	if s := TimestampToSlot(genesis+12, genesis); s != 1 {
		t.Errorf("TimestampToSlot(genesis+12) = %d, want 1", s)
	}

	// 384 seconds after genesis = slot 32 (epoch 1)
	if s := TimestampToSlot(genesis+384, genesis); s != 32 {
		t.Errorf("TimestampToSlot(genesis+384) = %d, want 32", s)
	}

	// Before genesis = 0
	if s := TimestampToSlot(genesis-1, genesis); s != 0 {
		t.Errorf("TimestampToSlot(genesis-1) = %d, want 0", s)
	}
}

func TestSlotToTimestamp(t *testing.T) {
	genesis := MainnetGenesisTime

	// Slot 0 = genesis time
	if ts := SlotToTimestamp(0, genesis); ts != genesis {
		t.Errorf("SlotToTimestamp(0) = %d, want %d", ts, genesis)
	}

	// Slot 1 = genesis + 12
	if ts := SlotToTimestamp(1, genesis); ts != genesis+12 {
		t.Errorf("SlotToTimestamp(1) = %d, want %d", ts, genesis+12)
	}

	// Slot 32 = genesis + 384
	if ts := SlotToTimestamp(32, genesis); ts != genesis+384 {
		t.Errorf("SlotToTimestamp(32) = %d, want %d", ts, genesis+384)
	}
}

func TestTimestampToSlotRoundTrip(t *testing.T) {
	genesis := MainnetGenesisTime
	for _, slot := range []uint64{0, 1, 31, 32, 100, 1000} {
		ts := SlotToTimestamp(slot, genesis)
		recovered := TimestampToSlot(ts, genesis)
		if recovered != slot {
			t.Errorf("round-trip slot %d: got %d", slot, recovered)
		}
	}
}

func TestDetectMissedSlots(t *testing.T) {
	tests := []struct {
		name       string
		parentSlot uint64
		current    uint64
		wantCount  int
	}{
		{"consecutive", 100, 101, 0},
		{"same_slot", 100, 100, 0},
		{"one_missed", 100, 102, 1},
		{"many_missed", 100, 105, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missed := DetectMissedSlots(tt.parentSlot, tt.current)
			if len(missed) != tt.wantCount {
				t.Errorf("DetectMissedSlots(%d, %d) = %d missed, want %d",
					tt.parentSlot, tt.current, len(missed), tt.wantCount)
			}
			// Verify missed slots are correct
			if tt.wantCount > 0 {
				first := tt.parentSlot + 1
				if missed[0] != first {
					t.Errorf("first missed slot = %d, want %d", missed[0], first)
				}
			}
		})
	}
}

func TestNewBeaconBlockInfo(t *testing.T) {
	genesis := MainnetGenesisTime
	ts := genesis + 120 // slot 10

	info := NewBeaconBlockInfo(ts, genesis, 9)
	if info.Slot != 10 {
		t.Errorf("Slot = %d, want 10", info.Slot)
	}
	if info.Epoch != 0 {
		t.Errorf("Epoch = %d, want 0", info.Epoch)
	}
	if info.IsMissedSlot {
		t.Error("IsMissedSlot should be false for consecutive slots")
	}

	// Missed slot: parent at slot 8, current at slot 10
	info2 := NewBeaconBlockInfo(ts, genesis, 8)
	if !info2.IsMissedSlot {
		t.Error("IsMissedSlot should be true when slot 9 is missed")
	}
}

func TestSlotsUntilNextEpoch(t *testing.T) {
	tests := []struct {
		slot uint64
		want uint64
	}{
		{0, 32},
		{1, 31},
		{31, 1},
		{32, 32},
		{50, 14},
	}
	for _, tt := range tests {
		got := SlotsUntilNextEpoch(tt.slot)
		if got != tt.want {
			t.Errorf("SlotsUntilNextEpoch(%d) = %d, want %d", tt.slot, got, tt.want)
		}
	}
}

func TestTimeUntilNextEpoch(t *testing.T) {
	// Slot 0: 32 slots * 12s = 384s until next epoch
	d := TimeUntilNextEpoch(0)
	if d != 384*time.Second {
		t.Errorf("TimeUntilNextEpoch(0) = %v, want 384s", d)
	}

	// Slot 31: 1 slot * 12s = 12s until next epoch
	d = TimeUntilNextEpoch(31)
	if d != 12*time.Second {
		t.Errorf("TimeUntilNextEpoch(31) = %v, want 12s", d)
	}
}

func TestEpochDuration(t *testing.T) {
	if EpochDuration != 384*time.Second {
		t.Errorf("EpochDuration = %v, want 384s", EpochDuration)
	}
}
