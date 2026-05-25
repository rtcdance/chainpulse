package pullers

import (
	"testing"
)

func TestHexToUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  uint64
	}{
		{"standard_hex", "0xff", 255},
		{"no_prefix", "ff", 255},
		{"zero", "0x0", 0},
		{"empty_string", "", 0},
		{"single_char", "a", 0},
		{"large_hex", "0xffffffffffffffff", 18446744073709551615},
		{"invalid_hex", "0xGGGG", 0},
		{"decimal_valid", "0x10", 16},
		{"mixed_case", "0xAbCdEf", 11259375},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hexToUint64(tt.input)
			if got != tt.want {
				t.Errorf("hexToUint64(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestUint64ToHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint64
		want  string
	}{
		{"zero", 0, "0x0"},
		{"small", 255, "0xff"},
		{"power_of_two", 16, "0x10"},
		{"max_uint64", 18446744073709551615, "0xffffffffffffffff"},
		{"medium", 123456789, "0x75bcd15"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := uint64ToHex(tt.input)
			if got != tt.want {
				t.Errorf("uint64ToHex(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
