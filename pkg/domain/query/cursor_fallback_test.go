package query

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncodePageCursor_JSONMarshalFallback(t *testing.T) {
	t.Parallel()
	encoded := EncodePageCursor(42, 1, "evt_test")
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("expected valid base64, got error: %v", err)
	}
	body := string(b)
	if !strings.Contains(body, `"bn":42`) {
		t.Errorf("expected block number 42 in JSON, got: %s", body)
	}
	if !strings.Contains(body, `"li":1`) {
		t.Errorf("expected log index 1 in JSON, got: %s", body)
	}
	if !strings.Contains(body, `"id":"evt_test"`) {
		t.Errorf("expected event ID evt_test in JSON, got: %s", body)
	}
}

func TestDecodePageCursor_FallbackFormat(t *testing.T) {
	t.Parallel()
	// The fallback format "block:log:eventID" should NOT decode via base64.
	// It should return ok=false since it's not valid base64.
	_, ok := DecodePageCursor("invalid!")
	if ok {
		t.Error("expected ok=false for invalid base64 input")
	}
}

func TestDecodePageCursor_ValidBase64InvalidJSON(t *testing.T) {
	t.Parallel()
	// "aW52YWxpZCBqc29u" = "invalid json" in base64
	_, ok := DecodePageCursor("aW52YWxpZCBqc29u")
	if ok {
		t.Error("expected ok=false for valid base64 but invalid JSON")
	}
}

func TestEncodePageCursor_Deterministic(t *testing.T) {
	t.Parallel()
	c1 := EncodePageCursor(100, 2, "evt_deterministic")
	c2 := EncodePageCursor(100, 2, "evt_deterministic")
	if c1 != c2 {
		t.Errorf("expected deterministic encoding, got %q vs %q", c1, c2)
	}
}

func TestDecodePageCursor_MalformedBase64(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"not base64", "!!!not-base64!!!"},
		{"truncated base64", "AAAA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, ok := DecodePageCursor(tc.cursor)
			if ok {
				t.Errorf("expected ok=false for %q", tc.cursor)
			}
		})
	}
}
