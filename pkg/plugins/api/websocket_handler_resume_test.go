package api

import "testing"

// TestSubscriptionHubResumeAdjudication proves the hub rejects stale and
// gapped reconnect cursors instead of handing back a partial replay, and
// accepts in-window cursors (with and without an epoch).
func TestSubscriptionHubResumeAdjudication(t *testing.T) {
	t.Parallel()

	t.Run("no cursor is fresh", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		hub.BufferEvent("events", "a")
		status, entries := hub.Resume("events", "", 0, false)
		if status != resumeStatusFresh {
			t.Fatalf("expected %q, got %q", resumeStatusFresh, status)
		}
		if entries != nil {
			t.Fatalf("fresh resume must not replay, got %d entries", len(entries))
		}
	})

	t.Run("in-window cursor with matching epoch replays", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		for i := 0; i < 5; i++ {
			hub.BufferEvent("events", i)
		}
		status, entries := hub.Resume("events", hub.Epoch(), 3, true)
		if status != resumeStatusOK {
			t.Fatalf("expected %q, got %q", resumeStatusOK, status)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 replayed entries (seq 4,5), got %d", len(entries))
		}
		if entries[0].Seq != 4 || entries[1].Seq != 5 {
			t.Fatalf("expected seq 4,5 got %d,%d", entries[0].Seq, entries[1].Seq)
		}
	})

	t.Run("legacy bare seq cursor keeps working inside the window", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		for i := 0; i < 5; i++ {
			hub.BufferEvent("events", i)
		}
		status, entries := hub.Resume("events", "", 3, true)
		if status != resumeStatusOK {
			t.Fatalf("expected %q, got %q", resumeStatusOK, status)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 replayed entries, got %d", len(entries))
		}
	})

	t.Run("epoch mismatch is stale", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		hub.BufferEvent("events", "a")
		status, entries := hub.Resume("events", "not-the-real-epoch", 1, true)
		if status != resumeStatusStaleEpoch {
			t.Fatalf("expected %q, got %q", resumeStatusStaleEpoch, status)
		}
		if entries != nil {
			t.Fatalf("stale resume must not replay, got %d entries", len(entries))
		}
	})

	t.Run("cursor beyond current sequence is stale", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		hub.BufferEvent("events", "a")
		status, entries := hub.Resume("events", "", 99, true)
		if status != resumeStatusStaleEpoch {
			t.Fatalf("expected %q, got %q", resumeStatusStaleEpoch, status)
		}
		if entries != nil {
			t.Fatalf("stale resume must not replay, got %d entries", len(entries))
		}
	})

	t.Run("cursor older than retained window is a gap", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		total := replayBufferCapacity + 10
		for i := 0; i < total; i++ {
			hub.BufferEvent("events", i)
		}
		// Oldest retained seq is 11; cursor 9 leaves a hole at seq 10.
		status, entries := hub.Resume("events", hub.Epoch(), 9, true)
		if status != resumeStatusGap {
			t.Fatalf("expected %q, got %q", resumeStatusGap, status)
		}
		if entries != nil {
			t.Fatalf("gap resume must not replay, got %d entries", len(entries))
		}
	})

	t.Run("cursor exactly contiguous with retained window replays", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		total := replayBufferCapacity + 10
		for i := 0; i < total; i++ {
			hub.BufferEvent("events", i)
		}
		// Oldest retained seq is 11; cursor 10 is contiguous — no gap.
		status, entries := hub.Resume("events", hub.Epoch(), 10, true)
		if status != resumeStatusOK {
			t.Fatalf("expected %q, got %q", resumeStatusOK, status)
		}
		if len(entries) != replayBufferCapacity {
			t.Fatalf("expected full retained window (%d), got %d", replayBufferCapacity, len(entries))
		}
	})

	t.Run("fresh hub with zero cursor is ok with nothing to replay", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		status, entries := hub.Resume("events", "", 0, true)
		if status != resumeStatusOK {
			t.Fatalf("expected %q, got %q", resumeStatusOK, status)
		}
		if entries != nil {
			t.Fatalf("expected no entries, got %d", len(entries))
		}
	})

	t.Run("epoch is stable across calls", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		if hub.Epoch() == "" {
			t.Fatal("hub epoch must not be empty")
		}
		if hub.Epoch() != hub.Epoch() {
			t.Fatal("hub epoch must be stable")
		}
	})
}
