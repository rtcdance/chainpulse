package query

import (
	"testing"
)

func TestEncodeDecodePageCursor_RoundTrip(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		blockNumber uint64
		logIndex    uint64
		eventID     string
	}{
		{"basic", 12345, 3, "evt_abc123"},
		{"zero block", 0, 0, "evt_0"},
		{"large block", 999999999999, 255, "evt_long_id_with_special_chars_!@#"},
		{"max uint64", ^uint64(0), ^uint64(0), "evt_max"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded := EncodePageCursor(tc.blockNumber, tc.logIndex, tc.eventID)

			pc, ok := DecodePageCursor(encoded)
			if !ok {
				t.Fatalf("DecodePageCursor(%q) returned ok=false", encoded)
			}
			if pc.BlockNumber != tc.blockNumber {
				t.Errorf("BlockNumber: got %d, want %d", pc.BlockNumber, tc.blockNumber)
			}
			if pc.LogIndex != tc.logIndex {
				t.Errorf("LogIndex: got %d, want %d", pc.LogIndex, tc.logIndex)
			}
			if pc.EventID != tc.eventID {
				t.Errorf("EventID: got %q, want %q", pc.EventID, tc.eventID)
			}
		})
	}
}

func TestDecodePageCursor_InvalidInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"invalid base64", "not!valid!base64!"},
		{"valid base64 but not JSON", "aW52YWxpZCBqc29u"},
		{"random string", "zzzzzzzz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, ok := DecodePageCursor(tc.cursor)
			if ok {
				t.Errorf("DecodePageCursor(%q) should return ok=false, got ok=true", tc.cursor)
			}
		})
	}
}

func TestEncodePageCursor_Opacity(t *testing.T) {
	t.Parallel()
	// Cursors should not expose array indices or be guessable
	c1 := EncodePageCursor(100, 0, "evt_a")
	c2 := EncodePageCursor(100, 1, "evt_b")

	// Different events must produce different cursors
	if c1 == c2 {
		t.Error("cursors for different events should differ")
	}

	// Cursor should not contain raw block number as plain text
	// (base64 encoding obscures it, but let's verify it's not a simple "100" string)
	if c1 == "100" {
		t.Error("cursor should be opaque, not a raw block number")
	}
}

func TestEncodePageCursor_Stability(t *testing.T) {
	t.Parallel()
	// Same input must always produce the same cursor
	c1 := EncodePageCursor(500, 7, "evt_stable")
	c2 := EncodePageCursor(500, 7, "evt_stable")
	if c1 != c2 {
		t.Errorf("identical inputs produced different cursors: %q vs %q", c1, c2)
	}
}
