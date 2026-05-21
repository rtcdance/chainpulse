package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// FuzzDecodeTypedEvent verifies that DecodeTypedEvent never panics
// regardless of the input topics and data.
func FuzzDecodeTypedEvent(f *testing.F) {
	// Seed with known-good Transfer event
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	topics := []common.Hash{
		topic0ForName("Transfer"),
		common.BytesToHash(from.Bytes()),
		common.BytesToHash(to.Bytes()),
	}
	data := encodeUint256Helper(big.NewInt(100))
	f.Add(topics[0][:], topics[1][:], topics[2][:], data)

	// Seed with empty data
	f.Add([]byte{}, []byte{}, []byte{}, []byte{})

	// Seed with random-looking bytes
	f.Add(
		make([]byte, 32),
		make([]byte, 32),
		make([]byte, 32),
		make([]byte, 64),
	)

	f.Fuzz(func(t *testing.T, topic0, topic1, topic2, data []byte) {
		topics := make([]common.Hash, 0, 3)
		if len(topic0) >= 32 {
			var h common.Hash
			copy(h[:], topic0[:32])
			topics = append(topics, h)
		}
		if len(topic1) >= 32 {
			var h common.Hash
			copy(h[:], topic1[:32])
			topics = append(topics, h)
		}
		if len(topic2) >= 32 {
			var h common.Hash
			copy(h[:], topic2[:32])
			topics = append(topics, h)
		}

		// Must not panic
		result, ok := DecodeTypedEvent(topics, data)
		_ = result
		_ = ok
	})
}

// FuzzDecodeEventData verifies that DecodeEventData never panics
// regardless of the event name, topics, and data.
func FuzzDecodeEventData(f *testing.F) {
	f.Add("Transfer", []byte{}, []byte{}, []byte{})
	f.Add("UnknownEvent", make([]byte, 32), make([]byte, 32), make([]byte, 32))
	f.Add("", []byte{}, []byte{}, []byte{})
	f.Add("Swap", make([]byte, 32), make([]byte, 32), make([]byte, 160))

	f.Fuzz(func(t *testing.T, eventName string, topic0, topic1, data []byte) {
		topics := make([]common.Hash, 0, 2)
		if len(topic0) >= 32 {
			var h common.Hash
			copy(h[:], topic0[:32])
			topics = append(topics, h)
		}
		if len(topic1) >= 32 {
			var h common.Hash
			copy(h[:], topic1[:32])
			topics = append(topics, h)
		}

		// Must not panic
		result := DecodeEventData(eventName, topics, data)
		_ = result
	})
}

func encodeUint256Helper(v *big.Int) []byte {
	b := v.Bytes()
	data := make([]byte, 32)
	copy(data[32-len(b):], b)
	return data
}
