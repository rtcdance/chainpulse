package query

import (
	"testing"
)

func FuzzPageCursorRoundTrip(f *testing.F) {
	// Seed with valid cursor values
	seeds := []string{
		EncodePageCursor(0, 0, ""),
		EncodePageCursor(1, 1, "evt_1"),
		EncodePageCursor(999999, 255, "evt_long_id_with_special_chars"),
		EncodePageCursor(^uint64(0), ^uint64(0), "max_values"),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	// Also test arbitrary strings (invalid cursors should return false)
	f.Add("")
	f.Add("not-valid-base64!!!")
	f.Add("aW52YWxpZCBqc29u") // valid base64 but not a PageCursor JSON
	f.Add("eyJibiI6MX0=")     // {"bn":1} — partial PageCursor
	f.Add("///")              // edge-case base64

	f.Fuzz(func(t *testing.T, cursor string) {
		pc, ok := DecodePageCursor(cursor)
		if !ok {
			// Invalid cursor — must not panic, must return zero PageCursor
			if pc != (PageCursor{}) {
				t.Errorf("invalid cursor %q returned non-zero PageCursor: %+v", cursor, pc)
			}
			return
		}

		// Valid cursor — re-encoding must produce a cursor that decodes identically
		encoded := EncodePageCursor(pc.BlockNumber, pc.LogIndex, pc.EventID)
		pc2, ok2 := DecodePageCursor(encoded)
		if !ok2 {
			t.Fatalf("re-encoded cursor %q failed to decode", encoded)
		}
		if pc2.BlockNumber != pc.BlockNumber || pc2.LogIndex != pc.LogIndex || pc2.EventID != pc.EventID {
			t.Errorf("round-trip mismatch: got %+v, want %+v", pc2, pc)
		}
	})
}
