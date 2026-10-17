package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addressTopic builds a topic whose low 20 bytes are a non-zero address.
func addressTopic(seed byte) common.Hash {
	var topic common.Hash
	for i := 12; i < 32; i++ {
		topic[i] = seed
	}
	return topic
}

// TestDecodeTypedEventRejectsMalformedLogs is the T09 malformed-input corpus.
//
// A truncated or under-specified log must NOT become a typed event. A
// partially-filled struct is indistinguishable from a real event whose fields
// happen to be zero, so the log has to fall through to the raw-only path
// instead of being reported as decoded.
func TestDecodeTypedEventRejectsMalformedLogs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		event   string
		topics  int // total topics including topic0
		dataLen int
	}{
		{"transfer/missing-to-topic", "Transfer", 2, 32},
		{"transfer/short-value", "Transfer", 3, 31},
		{"transfer/no-data", "Transfer", 3, 0},

		{"approval/short-value", "Approval", 3, 16},
		{"approval/no-data", "Approval", 3, 0},

		{"approval-for-all/short", "ApprovalForAll", 3, 0},
		{"approval-for-all/missing-operator", "ApprovalForAll", 2, 32},

		{"transfer-single/short-ids", "TransferSingle", 4, 63},
		{"transfer-single/missing-to-topic", "TransferSingle", 3, 64},

		{"transfer-batch/no-data", "TransferBatch", 4, 0},

		{"uri/no-data", "URI", 2, 0},

		{"swap/short-data", "Swap", 3, 159},
		{"swap/missing-recipient", "Swap", 2, 160},
		{"swap/no-data", "Swap", 3, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			topic0 := topic0ForName(tc.event)
			require.NotEqual(t, common.Hash{}, topic0,
				"the ABI for %q must be registered; a rename would silently disable this case", tc.event)

			topics := make([]common.Hash, 0, tc.topics)
			topics = append(topics, topic0)
			for i := 1; i < tc.topics; i++ {
				topics = append(topics, addressTopic(byte(0xa0+i)))
			}

			event, ok := DecodeTypedEvent(topics, make([]byte, tc.dataLen))

			assert.False(t, ok, "a malformed %s log must not decode", tc.event)
			assert.Nil(t, event, "no partially-filled event may be returned")
		})
	}
}

// TestDecodeTypedEventAcceptsWellFormedLogs is the positive control: the corpus
// above is only meaningful if the same events DO decode when well-formed.
func TestDecodeTypedEventAcceptsWellFormedLogs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		event   string
		topics  int
		dataLen int
	}{
		{"Transfer", 3, 32},
		{"Approval", 3, 32},
		{"ApprovalForAll", 3, 32},
		{"TransferSingle", 4, 64},
		{"Swap", 3, 160},
	}

	for _, tc := range cases {
		t.Run(tc.event, func(t *testing.T) {
			t.Parallel()

			topic0 := topic0ForName(tc.event)
			require.NotEqual(t, common.Hash{}, topic0, "the ABI for %q must be registered", tc.event)

			topics := make([]common.Hash, 0, tc.topics)
			topics = append(topics, topic0)
			for i := 1; i < tc.topics; i++ {
				topics = append(topics, addressTopic(byte(0xb0+i)))
			}

			event, ok := DecodeTypedEvent(topics, make([]byte, tc.dataLen))
			require.True(t, ok, "a well-formed %s log must decode", tc.event)
			require.NotNil(t, event)
		})
	}
}

// TestDecodeTypedEventUnknownTopicIsRawOnly pins the raw-only policy: an
// unrecognised topic0 is not an error, it simply has no typed decoder.
func TestDecodeTypedEventUnknownTopicIsRawOnly(t *testing.T) {
	t.Parallel()

	event, ok := DecodeTypedEvent(
		[]common.Hash{common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")},
		make([]byte, 160),
	)
	assert.False(t, ok)
	assert.Nil(t, event)
}
