package api

import (
	"fmt"
	"regexp"
	"strconv"
)

// Known chain IDs that ChainPulse supports
var knownChainIDs = map[uint64]bool{
	1: true, 137: true, 56: true, 97: true,
	42161: true, 421614: true, 10: true, 11155420: true,
	8453: true, 84532: true, 43114: true, 43113: true,
}

// ethereumAddressRegex matches 0x-prefixed 40-character hex strings
var ethereumAddressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// validateEthereumAddress checks if an address matches the Ethereum format
func validateEthereumAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("address is required")
	}
	if !ethereumAddressRegex.MatchString(addr) {
		return fmt.Errorf("invalid Ethereum address format: must be 0x-prefixed 40-character hex")
	}
	return nil
}

// validateChainID checks if a chain ID is a known, supported chain
func validateChainID(chainIDStr string) error {
	if chainIDStr == "" {
		return fmt.Errorf("chain ID is required")
	}
	id, err := strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chain ID: must be a positive integer")
	}
	if !knownChainIDs[id] {
		return fmt.Errorf("unsupported chain ID: %s", chainIDStr)
	}
	return nil
}
