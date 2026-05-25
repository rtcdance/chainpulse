package api

import (
	"testing"
)

func TestValidateEthereumAddress_Valid(t *testing.T) {
	t.Parallel()
	if err := validateEthereumAddress("0x1234567890123456789012345678901234567890"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateEthereumAddress_Empty(t *testing.T) {
	t.Parallel()
	err := validateEthereumAddress("")
	if err == nil {
		t.Error("expected error for empty address")
	}
}

func TestValidateEthereumAddress_InvalidFormat(t *testing.T) {
	t.Parallel()
	tests := []string{
		"0x123",
		"not-an-address",
		"0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",
		"0X1234567890123456789012345678901234567890",
	}
	for _, addr := range tests {
		err := validateEthereumAddress(addr)
		if err == nil {
			t.Errorf("expected error for %q", addr)
		}
	}
}

func TestValidateChainID_ValidKnown(t *testing.T) {
	t.Parallel()
	if err := validateChainID("1"); err != nil {
		t.Errorf("expected no error for chain 1, got %v", err)
	}
	if err := validateChainID("137"); err != nil {
		t.Errorf("expected no error for chain 137, got %v", err)
	}
}

func TestValidateChainID_Empty(t *testing.T) {
	t.Parallel()
	err := validateChainID("")
	if err == nil {
		t.Error("expected error for empty chain ID")
	}
}

func TestValidateChainID_NotANumber(t *testing.T) {
	t.Parallel()
	err := validateChainID("abc")
	if err == nil {
		t.Error("expected error for non-numeric chain ID")
	}
}

func TestValidateChainID_Unknown(t *testing.T) {
	t.Parallel()
	err := validateChainID("999")
	if err == nil {
		t.Error("expected error for unknown chain ID")
	}
}

func TestValidateChainID_Negative(t *testing.T) {
	t.Parallel()
	err := validateChainID("-1")
	if err == nil {
		t.Error("expected error for negative chain ID")
	}
}
