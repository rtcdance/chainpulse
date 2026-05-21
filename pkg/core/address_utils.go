package core

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ErrInvalidChecksum is returned when an address fails EIP-55 checksum validation.
var ErrInvalidChecksum = errors.New("address fails EIP-55 checksum")

// ToChecksummedAddress returns the EIP-55 mixed-case checksum encoding of an
// Ethereum address. It accepts any 0x-prefixed 40-character hex string (case-insensitive)
// and returns the checksummed form.
//
// Algorithm: https://eips.ethereum.org/EIPS/eip-55
//  1. Take the lowercase hex address (without 0x prefix).
//  2. Hash it with Keccak-256.
//  3. For each character in the address, if the corresponding hex digit of the hash
//     is >= 8, capitalize the letter.
func ToChecksummedAddress(addr string) string {
	if len(addr) == 0 {
		return addr
	}

	// Strip 0x prefix for processing
	raw := addr
	if strings.HasPrefix(strings.ToLower(addr), "0x") {
		raw = addr[2:]
	}

	if len(raw) != 40 {
		return addr
	}

	lower := strings.ToLower(raw)
	hash := hex.EncodeToString(crypto.Keccak256([]byte(lower)))

	var result strings.Builder
	result.WriteString("0x")
	for i, c := range lower {
		if c >= 'a' && c <= 'f' {
			// If the corresponding hash nibble is >= 8, uppercase
			if hash[i] >= '8' {
				result.WriteRune(c - 32)
			} else {
				result.WriteRune(c)
			}
		} else {
			result.WriteRune(c)
		}
	}

	return result.String()
}

// ValidateEIP55Checksum checks whether a 0x-prefixed address with mixed-case
// hex letters conforms to the EIP-55 checksum. All-lowercase and all-uppercase
// addresses are considered valid (they are the unchecksummed form and are
// accepted for backward compatibility).
//
// Returns nil if the address is valid, ErrInvalidChecksum if the checksum
// doesn't match, or an error for malformed addresses.
func ValidateEIP55Checksum(addr string) error {
	if len(addr) == 0 {
		return errors.New("address is empty")
	}

	if !strings.HasPrefix(addr, "0x") && !strings.HasPrefix(addr, "0X") {
		return errors.New("address must be 0x-prefixed")
	}

	raw := addr[2:]
	if len(raw) != 40 {
		return errors.New("address must be 40 hex characters after 0x prefix")
	}

	lower := strings.ToLower(raw)
	// All-lowercase or all-uppercase is valid (unchecksummed form)
	if raw == lower || raw == strings.ToUpper(raw) {
		return nil
	}

	// Mixed case: must match EIP-55 checksum
	expected := ToChecksummedAddress(addr)
	if addr != expected {
		return ErrInvalidChecksum
	}

	return nil
}

// IsChecksummedAddress returns true if the address is in EIP-55 checksummed form.
func IsChecksummedAddress(addr string) bool {
	return addr == ToChecksummedAddress(addr)
}

// ValidateAddressChecksum validates a common.Address using EIP-55 rules.
// Since common.Address.Hex() always returns checksummed output, this is
// primarily useful when validating raw string addresses from external input.
func ValidateAddressChecksum(addr common.Address) error {
	return ValidateEIP55Checksum(addr.Hex())
}
