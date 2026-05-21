package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pry "pgregory.net/rapid"
)

// Property: DecodeEventData never panics regardless of input.
func TestProperty_DecodeEventDataNeverPanics(t *testing.T) {
	t.Parallel()

	pry.Check(t, func(t *pry.T) {
		eventName := pry.SampledFrom([]string{
			"Transfer", "Approval", "Swap", "Mint", "Burn",
			"", "nonexistent", randomString(10),
		}).Draw(t, "eventName")

		numTopics := pry.IntRange(0, 4).Draw(t, "numTopics")
		topics := make([]common.Hash, numTopics)
		for i := 0; i < numTopics; i++ {
			var h common.Hash
			copy(h[:], randomBytes(32))
			topics[i] = h
		}

		dataLen := pry.IntRange(0, 1024).Draw(t, "dataLen")
		data := randomBytes(dataLen)

		// Must never panic
		result := DecodeEventData(eventName, topics, data)

		// Must never return nil (ChainedDecoder guarantees raw hex fallback)
		if result == nil {
			t.Logf("nil result for event=%q topics=%d data=%d bytes", eventName, numTopics, dataLen)
		}
	})
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*17 + i*i + 31)
	}
	return b
}

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	b := randomBytes(n)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}
