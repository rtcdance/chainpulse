package api

import (
	"testing"
)

func FuzzValidateEthereumAddress(f *testing.F) {
	// Seed corpus
	seeds := []string{
		"",
		"0x",
		"0x1234567890abcdef1234567890abcdef12345678",
		"0x1234567890ABCDEF1234567890ABCDEF12345678",
		"0xGGGG567890abcdef1234567890abcdef12345678",
		"1234567890abcdef1234567890abcdef12345678",
		"0x1234",
		"0x1234567890abcdef1234567890abcdef1234567890",
		"0x0000000000000000000000000000000000000000",
		"0xFFfFfFffFFfffFFfFFfFFFFFffFFFffffFfFFFfF",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, addr string) {
		err := validateEthereumAddress(addr)
		// The function should never panic.
		// Valid addresses must pass, invalid ones must return an error.
		_ = err
	})
}

func FuzzValidateChainID(f *testing.F) {
	seeds := []string{
		"",
		"1",
		"137",
		"56",
		"0",
		"-1",
		"999999",
		"abc",
		"1.5",
		"42161",
		"18446744073709551615", // max uint64
		"18446744073709551616", // overflow
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, chainID string) {
		err := validateChainID(chainID)
		// The function should never panic.
		_ = err
	})
}
