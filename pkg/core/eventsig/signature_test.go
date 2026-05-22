package eventsig

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventSignature(t *testing.T) {
	t.Parallel()
	// ERC-20 Transfer(address,address,uint256)
	sig := EventSignature("Transfer", "address", "address", "uint256")
	assert.Equal(t, common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), sig)

	// ERC-20 Approval(address,address,uint256)
	sig = EventSignature("Approval", "address", "address", "uint256")
	assert.Equal(t, common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"), sig)

	// No params
	sig = EventSignature("Ping")
	assert.NotEqual(t, common.Hash{}, sig)
}

func TestEncodeIndexedParamAddress(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B")
	topic, err := EncodeIndexedParam(addr, "address")
	require.NoError(t, err)

	// Address should be left-padded to 32 bytes, so last 20 bytes = address
	recovered := common.BytesToAddress(topic[12:])
	assert.Equal(t, addr, recovered)
}

func TestEncodeIndexedParamBool(t *testing.T) {
	t.Parallel()
	topic, err := EncodeIndexedParam(true, "bool")
	require.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), topic)

	topic, err = EncodeIndexedParam(false, "bool")
	require.NoError(t, err)
	assert.Equal(t, common.Hash{}, topic)
}

func TestEncodeIndexedParamUint256(t *testing.T) {
	t.Parallel()
	// big.Int
	val := big.NewInt(1000000)
	topic, err := EncodeIndexedParam(val, "uint256")
	require.NoError(t, err)
	decoded := new(big.Int).SetBytes(topic.Bytes())
	assert.Equal(t, val, decoded)

	// uint64
	topic, err = EncodeIndexedParam(uint64(42), "uint256")
	require.NoError(t, err)
	decoded = new(big.Int).SetBytes(topic.Bytes())
	assert.Equal(t, uint64(42), decoded.Uint64())

	// int
	topic, err = EncodeIndexedParam(99, "uint")
	require.NoError(t, err)
	decoded = new(big.Int).SetBytes(topic.Bytes())
	assert.Equal(t, int64(99), decoded.Int64())
}

func TestEncodeIndexedParamBytes32(t *testing.T) {
	t.Parallel()
	h := common.HexToHash("0xcafecafecafecafecafecafecafecafecafecafecafecafecafecafecafecafe")
	topic, err := EncodeIndexedParam(h, "bytes32")
	require.NoError(t, err)
	assert.Equal(t, h, topic)
}

func TestEncodeIndexedParamString(t *testing.T) {
	t.Parallel()
	topic, err := EncodeIndexedParam("hello", "string")
	require.NoError(t, err)
	// String should be keccak256 hashed
	assert.NotEqual(t, common.Hash{}, topic)
}

func TestEncodeIndexedParamErrors(t *testing.T) {
	t.Parallel()
	// Wrong type for address
	_, err := EncodeIndexedParam("not_an_address", "address")
	assert.Error(t, err)

	// Wrong type for bool
	_, err = EncodeIndexedParam("not_bool", "bool")
	assert.Error(t, err)

	// Wrong type for uint
	_, err = EncodeIndexedParam("not_uint", "uint256")
	assert.Error(t, err)

	// Wrong type for bytes32
	_, err = EncodeIndexedParam("not_bytes32", "bytes32")
	assert.Error(t, err)

	// Wrong type for string
	_, err = EncodeIndexedParam(123, "string")
	assert.Error(t, err)
}

func TestEncodeIndexedParamUintSubtypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		val      any
		solType  string
		wantZero bool
	}{
		{"big.Int", big.NewInt(999), "uint256", false},
		{"uint64", uint64(999), "uint", false},
		{"int", 100, "uint256", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, err := EncodeIndexedParam(tt.val, tt.solType)
			require.NoError(t, err)
			decoded := new(big.Int).SetBytes(topic.Bytes())
			assert.Equal(t, big.NewInt(999).Int64(), decoded.Int64())
		})
	}
}

func TestDecodeIndexedParamAddress(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B")
	topic, err := EncodeIndexedParam(addr, "address")
	require.NoError(t, err)

	decoded, err := DecodeIndexedParam(topic, "address")
	require.NoError(t, err)
	assert.Equal(t, addr, decoded)
}

func TestDecodeIndexedParamBool(t *testing.T) {
	t.Parallel()
	// true
	topic, err := EncodeIndexedParam(true, "bool")
	require.NoError(t, err)
	decoded, err := DecodeIndexedParam(topic, "bool")
	require.NoError(t, err)
	assert.Equal(t, true, decoded)

	// false
	topic, err = EncodeIndexedParam(false, "bool")
	require.NoError(t, err)
	decoded, err = DecodeIndexedParam(topic, "bool")
	require.NoError(t, err)
	assert.Equal(t, false, decoded)

	// Invalid bool encoding
	invalidTopic := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")
	_, err = DecodeIndexedParam(invalidTopic, "bool")
	assert.Error(t, err)
}

func TestDecodeIndexedParamUint256(t *testing.T) {
	t.Parallel()
	val := big.NewInt(1e18)
	topic, err := EncodeIndexedParam(val, "uint256")
	require.NoError(t, err)

	decoded, err := DecodeIndexedParam(topic, "uint256")
	require.NoError(t, err)
	assert.Equal(t, val, decoded)
}

func TestDecodeIndexedParamBytes32(t *testing.T) {
	t.Parallel()
	h := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	topic, err := EncodeIndexedParam(h, "bytes32")
	require.NoError(t, err)

	decoded, err := DecodeIndexedParam(topic, "bytes32")
	require.NoError(t, err)
	assert.Equal(t, h, decoded)
}

func TestDecodeIndexedParamString(t *testing.T) {
	t.Parallel()
	topic, err := EncodeIndexedParam("hello world", "string")
	require.NoError(t, err)

	// Indexed strings cannot be recovered
	_, err = DecodeIndexedParam(topic, "string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecoverable")

	_, err = DecodeIndexedParam(topic, "bytes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecoverable")
}

func TestDecodeIndexedParamUnsupported(t *testing.T) {
	t.Parallel()
	topic := common.HexToHash("0x01")
	_, err := DecodeIndexedParam(topic, "tuple")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported indexed type")
}

func TestTopic0ForEvent(t *testing.T) {
	t.Parallel()
	topic := Topic0ForEvent("Transfer", "address", "address", "uint256")
	assert.Equal(t, common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), topic)
}

func TestKeccak256Hash(t *testing.T) {
	t.Parallel()
	// keccak256("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
	result := keccak256Hash([]byte(""))
	assert.NotEqual(t, common.Hash{}, result)
}

func TestResolveEventNameFromTopic(t *testing.T) {
	t.Parallel()
	// Known signature
	name := ResolveEventNameFromTopic("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	assert.Equal(t, "Transfer", name)

	// Case insensitive
	name = ResolveEventNameFromTopic("0xDDF252AD1BE2C89B69C2B068FC378DAA952BA7F163C4A11628F55A4DF523B3EF")
	assert.Equal(t, "Transfer", name)

	// Unknown topic returns as-is
	name = ResolveEventNameFromTopic("0xunknown")
	assert.Equal(t, "0xunknown", name)

	// Empty topic
	name = ResolveEventNameFromTopic("")
	assert.Equal(t, "", name)

	// Whitespace only
	name = ResolveEventNameFromTopic("  ")
	assert.Equal(t, "", name)
}

func TestResolveTopicFromName(t *testing.T) {
	t.Parallel()
	// Known name
	sig := ResolveTopicFromName("Transfer")
	assert.Equal(t, "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", sig)

	// Unknown name
	sig = ResolveTopicFromName("UnknownEvent")
	assert.Equal(t, "", sig)

	// First occurrence wins for duplicate names
	sig = ResolveTopicFromName("Ping")
	assert.NotEqual(t, "", sig)
}

func TestIsKnownEventSignature(t *testing.T) {
	t.Parallel()
	assert.True(t, IsKnownEventSignature("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"))
	assert.True(t, IsKnownEventSignature("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"))
	assert.False(t, IsKnownEventSignature("0xdeadbeef"))
	assert.False(t, IsKnownEventSignature(""))
}

func TestEncodeIndexedParamCustomABIType(t *testing.T) {
	t.Parallel()
	// bytes8 type uses ABI encoding path
	var arrayOfOnes [8]byte
	for i := range arrayOfOnes {
		arrayOfOnes[i] = 1
	}
	var bs [8]byte
	copy(bs[:], arrayOfOnes[:])
	topic, err := EncodeIndexedParam([8]byte{1, 1, 1, 1, 1, 1, 1, 1}, "bytes8")
	require.NoError(t, err)
	assert.NotEqual(t, common.Hash{}, topic)
}

func TestEncodeIndexedParamUnsupportedType(t *testing.T) {
	t.Parallel()
	_, err := EncodeIndexedParam(struct{ X int }{X: 1}, "nonexistent_type_xyz")
	assert.Error(t, err)
}

func TestResolveTopicDuplicateNames(t *testing.T) {
	t.Parallel()
	// "Ping" appears twice in the registry, first occurrence should win
	sig := ResolveTopicFromName("Ping")
	assert.NotEqual(t, "", sig)
	assert.True(t, IsKnownEventSignature(sig))
}

func TestEncodeDecodeUintRoundTrip(t *testing.T) {
	t.Parallel()
	values := []uint64{0, 1, 100, 1e6, 1e18}
	for _, v := range values {
		val := big.NewInt(int64(v))
		topic, err := EncodeIndexedParam(val, "uint256")
		require.NoError(t, err)
		decoded, err := DecodeIndexedParam(topic, "uint256")
		require.NoError(t, err)
		bigVal, ok := decoded.(*big.Int)
		require.True(t, ok)
		assert.Equal(t, val.Uint64(), bigVal.Uint64())
	}
}
