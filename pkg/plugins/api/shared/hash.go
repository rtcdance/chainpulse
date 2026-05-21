// Package shared provides shared utilities for API plugins.
package shared

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashCacheKey(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:16])
}
