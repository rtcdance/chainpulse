package cache

import (
	"bytes"
	"context"
	"testing"
)

// TestCacheValueEncodingRoundTrip covers the adapter's value encoding directly.
//
// The interesting inputs are the ones a naive `string(value)`/`[]byte(string)`
// pair would corrupt or that would not survive a JSON round trip: NUL bytes,
// invalid UTF-8 and bytes above 0x7F. A cache stores bytes, not text.
func TestCacheValueEncodingRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value []byte
	}{
		{name: "ascii", value: []byte("value1")},
		{name: "empty", value: []byte{}},
		{name: "nil", value: nil},
		{name: "nul bytes", value: []byte{0x00, 0x00, 0x00}},
		{name: "invalid utf8", value: []byte{0xFF, 0xFE, 0x80, 0x81}},
		{name: "high bytes", value: bytes.Repeat([]byte{0xFF}, 256)},
		{name: "binary blob", value: []byte{0x00, 0x91, 0xAB, 0xCD, 0xEF, 0x10}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded := encodeCacheValue(tc.value)
			decoded, err := decodeCacheValue("k", encoded)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !bytes.Equal(tc.value, decoded) {
				t.Errorf("round trip lost bytes: got %v want %v", decoded, tc.value)
			}
		})
	}
}

// TestDecodeCacheValueRejectsUnknownRepresentation pins that an unexpected
// stored type is an error, not a best-effort answer. Returning something else
// would have the caller serve and re-cache the corruption.
func TestDecodeCacheValueRejectsUnknownRepresentation(t *testing.T) {
	t.Parallel()

	if _, err := decodeCacheValue("k", 42); err == nil {
		t.Error("an integer value must be rejected")
	}
	if _, err := decodeCacheValue("k", map[string]string{"a": "b"}); err == nil {
		t.Error("a map value must be rejected")
	}
	if _, err := decodeCacheValue("k", "not base64!!"); err == nil {
		t.Error("a non-base64 string must be rejected")
	}
}

// TestDecodeCacheValueAcceptance documents the two shapes that are accepted, so
// the pass-through case is not mistaken for a silent fallback.
func TestDecodeCacheValueAcceptance(t *testing.T) {
	t.Parallel()

	// A missing value is (nil, nil) — absent, not failed.
	got, err := decodeCacheValue("k", nil)
	if err != nil || got != nil {
		t.Errorf("nil value: got (%v, %v), want (nil, nil)", got, err)
	}

	// Raw bytes are passed through for callers that bypassed the adapter.
	raw := []byte{0x01, 0x02}
	got, err = decodeCacheValue("k", raw)
	if err != nil || !bytes.Equal(raw, got) {
		t.Errorf("raw []byte: got (%v, %v), want (%v, nil)", got, err, raw)
	}
}

// TestRedisCacheAdapterIsPortShaped is a compile-time-shaped reminder: the
// adapter is what satisfies core.CachePlugin, not the raw plugin.
func TestRedisCacheAdapterIsPortShaped(t *testing.T) {
	t.Parallel()

	a := NewRedisCachePluginAdapter(nil, nil)
	if a == nil {
		t.Fatal("expected a non-nil adapter")
	}
	if a.RedisCachePlugin == nil {
		t.Fatal("the adapter must embed a RedisCachePlugin")
	}
}

// TestRedisCacheAdapterHonoursContextCancellation pins that a cancelled caller
// is not silently served: the underlying plugin issues its own
// context.Background() with a timeout, so without this check the caller's
// cancellation would be ignored entirely.
func TestRedisCacheAdapterHonoursContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	a := NewRedisCachePluginAdapter(nil, nil)
	if _, err := a.Get(ctx, "k"); err == nil {
		t.Error("Get must return the context error")
	}
	if err := a.Set(ctx, "k", []byte("v"), 60); err == nil {
		t.Error("Set must return the context error")
	}
	if err := a.Delete(ctx, "k"); err == nil {
		t.Error("Delete must return the context error")
	}
}
