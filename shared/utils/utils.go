package utils

import (
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ParseHex converts a hex string to bytes
func ParseHex(hexStr string) ([]byte, error) {
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		hexStr = hexStr[2:]
	}
	return hex.DecodeString(hexStr)
}

// HexToBigInt converts a hex string to big.Int
func HexToBigInt(hexStr string) (*big.Int, error) {
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		hexStr = hexStr[2:]
	}
	n := new(big.Int)
	_, err := n.SetString(hexStr, 16)
	return n, err
}

// IsValidAddress checks if the given string is a valid Ethereum address
func IsValidAddress(address string) bool {
	return common.IsHexAddress(address)
}

// FormatAddress formats an address to lowercase
func FormatAddress(address string) string {
	return strings.ToLower(address)
}

// FormatBigInt formats a big.Int to string
func FormatBigInt(n *big.Int) string {
	if n == nil {
		return "0"
	}
	return n.String()
}

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
	MaxDelay   time.Duration
	Factor     float64
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		Delay:      time.Second,
		MaxDelay:   10 * time.Second,
		Factor:     2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(fn func() error, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.Delay

	for i := 0; i <= config.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i == config.MaxRetries {
				break
			}

			// Wait before retrying with exponential backoff
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * config.Factor)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		} else {
			// Success
			return nil
		}
	}

	return lastErr
}