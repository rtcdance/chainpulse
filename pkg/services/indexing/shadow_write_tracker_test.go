package indexing

import (
	"testing"

	"chainpulse/pkg/core"
)

func TestShadowWriteTracker_MarkAndConsume(t *testing.T) {
	tracker := NewShadowWriteTracker()

	event := &core.BlockchainEvent{ID: "evt-1", ChainID: "1", BlockNumber: 100}
	tracker.Mark(event)

	if !tracker.Consume(event) {
		t.Fatal("Consume should return true for marked event")
	}

	if tracker.Consume(event) {
		t.Fatal("Consume should return false after first consumption")
	}
}

func TestShadowWriteTracker_MultipleEvents(t *testing.T) {
	tracker := NewShadowWriteTracker()

	e1 := &core.BlockchainEvent{ID: "evt-1", ChainID: "1", BlockNumber: 100}
	e2 := &core.BlockchainEvent{ID: "evt-2", ChainID: "1", BlockNumber: 101}
	e3 := &core.BlockchainEvent{ID: "evt-3", ChainID: "1", BlockNumber: 102}

	tracker.Mark(e1)
	tracker.Mark(e2)
	tracker.Mark(e3)

	if !tracker.Consume(e2) {
		t.Fatal("Consume(e2) should return true")
	}
	if !tracker.Consume(e1) {
		t.Fatal("Consume(e1) should return true")
	}
	if tracker.Consume(e1) {
		t.Fatal("Consume(e1) again should return false")
	}
	if !tracker.Consume(e3) {
		t.Fatal("Consume(e3) should return true")
	}
}

func TestShadowWriteTracker_NilSafety(t *testing.T) {
	tracker := NewShadowWriteTracker()

	tracker.Mark(nil)

	if tracker.Consume(nil) {
		t.Fatal("Consume(nil) should return false")
	}
}

func TestShadowWriteTracker_UnmarkedEvent(t *testing.T) {
	tracker := NewShadowWriteTracker()

	event := &core.BlockchainEvent{ID: "evt-1", ChainID: "1", BlockNumber: 100}

	if tracker.Consume(event) {
		t.Fatal("Consume should return false for unmarked event")
	}
}

func TestShadowWriteTracker_PointerBased(t *testing.T) {
	tracker := NewShadowWriteTracker()

	event := &core.BlockchainEvent{ID: "evt-1", ChainID: "1", BlockNumber: 100}

	// Pointer-based map: marking via one variable, consuming via another
	// But they point to the same event, so it should match
	alias := event
	tracker.Mark(alias)

	if !tracker.Consume(event) {
		t.Fatal("Consume should match by pointer, not by value")
	}
}

func TestShadowWriteTracker_TrackThenConsumeAll(t *testing.T) {
	tracker := NewShadowWriteTracker()

	n := 50
	events := make([]*core.BlockchainEvent, n)
	for i := range n {
		events[i] = &core.BlockchainEvent{ID: "evt", ChainID: "1", BlockNumber: uint64(i)}
		tracker.Mark(events[i])
	}

	for i := range n {
		if !tracker.Consume(events[i]) {
			t.Fatalf("Consume(%d) should return true", i)
		}
	}

	for i := range n {
		if tracker.Consume(events[i]) {
			t.Fatalf("Consume(%d) second time should return false", i)
		}
	}
}

func TestDefaultTracker(t *testing.T) {
	event := &core.BlockchainEvent{ID: "evt-def", ChainID: "1", BlockNumber: 200}
	defaultTracker.Mark(event)

	if !defaultTracker.Consume(event) {
		t.Fatal("default tracker: Consume should return true")
	}

	defaultTracker.Mark(event)
	if !consumeShadowWrite(event) {
		t.Fatal("consumeShadowWrite should delegate to default tracker")
	}
}
