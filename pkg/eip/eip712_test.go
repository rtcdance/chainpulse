package eip

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeType(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"EIP712Domain": {},
		"Mail": {
			{Name: "from", Type: "Person"},
			{Name: "to", Type: "Person"},
			{Name: "contents", Type: "string"},
		},
		"Person": {
			{Name: "name", Type: "string"},
			{Name: "wallet", Type: "address"},
		},
	}

	encoded, err := EncodeType("Mail", types)
	require.NoError(t, err)

	expected := "Mail(Person from,Person to,string contents)Person(string name,address wallet)"
	assert.Equal(t, expected, encoded)
}

func TestTypeHash(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Mail": {
			{Name: "from", Type: "Person"},
			{Name: "to", Type: "Person"},
			{Name: "contents", Type: "string"},
		},
		"Person": {
			{Name: "name", Type: "string"},
			{Name: "wallet", Type: "address"},
		},
	}

	hash, err := TypeHash("Mail", types)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash, "type hash should not be zero")
}

func TestEncodeTypeCircularReference(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"A": {
			{Name: "b", Type: "B"},
		},
		"B": {
			{Name: "a", Type: "A"},
		},
	}

	_, err := EncodeType("A", types)
	assert.ErrorIs(t, err, ErrCircularTypeReference)
}

func TestEncodeTypeUnknownType(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Mail": {
			{Name: "from", Type: "NonExistent"},
		},
	}

	// EncodeType treats unknown types as atomic (they appear in the type string).
	// The error surfaces during HashStruct/encodeData when trying to encode the value.
	encoded, err := EncodeType("Mail", types)
	require.NoError(t, err)
	assert.Contains(t, encoded, "NonExistent")

	// HashStruct should fail because "NonExistent" is not a known type or primitive
	_, err = HashStruct("Mail", types, map[string]any{
		"from": "some_value",
	})
	assert.Error(t, err)
}

func TestHashStructPrimitiveTypes(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"TestStruct": {
			{Name: "intValue", Type: "uint256"},
			{Name: "boolValue", Type: "bool"},
			{Name: "addrValue", Type: "address"},
		},
	}

	data := map[string]any{
		"intValue":  big.NewInt(42),
		"boolValue": true,
		"addrValue": common.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	hash, err := HashStruct("TestStruct", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestHashDomainSeparator(t *testing.T) {
	t.Parallel()
	domain := EIP712Domain{
		Name:              "Ether Mail",
		Version:           "1",
		ChainID:           big.NewInt(1),
		VerifyingContract: common.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"),
	}

	hash, err := HashDomainSeparator(domain)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestHashDomainSeparatorMissingChainID(t *testing.T) {
	t.Parallel()
	domain := EIP712Domain{
		Name:    "Test",
		Version: "1",
		ChainID: nil,
	}

	_, err := HashDomainSeparator(domain)
	assert.ErrorIs(t, err, ErrMissingChainID)
}

func TestHashTypedData(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Mail": {
			{Name: "from", Type: "Person"},
			{Name: "to", Type: "Person"},
			{Name: "contents", Type: "string"},
		},
		"Person": {
			{Name: "name", Type: "string"},
			{Name: "wallet", Type: "address"},
		},
	}

	td := TypedData{
		Types:       types,
		PrimaryType: "Mail",
		Domain: EIP712Domain{
			Name:              "Ether Mail",
			Version:           "1",
			ChainID:           big.NewInt(1),
			VerifyingContract: common.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"),
		},
		Message: map[string]any{
			"from": map[string]any{
				"name":   "Cow",
				"wallet": common.HexToAddress("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"),
			},
			"to": map[string]any{
				"name":   "Bob",
				"wallet": common.HexToAddress("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"),
			},
			"contents": "Hello, Bob!",
		},
	}

	hash, err := HashTypedData(td)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestHashTypedDataNilMessage(t *testing.T) {
	t.Parallel()
	td := TypedData{
		Types:       apitypes.Types{},
		PrimaryType: "Test",
		Message:     nil,
	}

	_, err := HashTypedData(td)
	assert.ErrorIs(t, err, ErrNilMessage)
}

func TestVerifyTypedDataRoundTrip(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"SimpleMsg": {
			{Name: "message", Type: "string"},
			{Name: "value", Type: "uint256"},
		},
	}

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	signerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	td := TypedData{
		Types:       types,
		PrimaryType: "SimpleMsg",
		Domain: EIP712Domain{
			Name:              "TestApp",
			Version:           "1",
			ChainID:           big.NewInt(1),
			VerifyingContract: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		},
		Message: map[string]any{
			"message": "hello world",
			"value":   big.NewInt(100),
		},
	}

	// Hash and sign
	hash, err := HashTypedData(td)
	require.NoError(t, err)

	sig, err := crypto.Sign(hash[:], privateKey)
	require.NoError(t, err)

	// V from crypto.Sign is 0 or 1; convert to 27/28 for Ethereum
	sig[64] += 27

	// Verify
	valid, err := VerifyTypedData(td, sig, signerAddr)
	require.NoError(t, err)
	assert.True(t, valid, "signature should verify against the signer address")

	// Verify against wrong address
	wrongAddr := common.HexToAddress("0x0000000000000000000000000000000000000002")
	valid, err = VerifyTypedData(td, sig, wrongAddr)
	require.NoError(t, err)
	assert.False(t, valid, "signature should NOT verify against a different address")
}

func TestVerifyTypedDataInvalidSignatureLength(t *testing.T) {
	t.Parallel()
	td := TypedData{
		Types:       apitypes.Types{},
		PrimaryType: "Test",
		Domain: EIP712Domain{
			Name:    "Test",
			Version: "1",
			ChainID: big.NewInt(1),
		},
		Message: map[string]any{"foo": "bar"},
	}

	_, err := VerifyTypedData(td, []byte{1, 2, 3}, common.Address{})
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestEncodeValueStringAndBytes(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Test": {
			{Name: "str", Type: "string"},
			{Name: "bts", Type: "bytes"},
		},
	}

	data := map[string]any{
		"str": "hello",
		"bts": []byte("world"),
	}

	hash, err := HashStruct("Test", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestEncodeValueIntFromString(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Test": {
			{Name: "amount", Type: "uint256"},
		},
	}

	data := map[string]any{
		"amount": "1000000000000000000",
	}

	hash, err := HashStruct("Test", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestEncodeValueBool(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Test": {
			{Name: "flag", Type: "bool"},
		},
	}

	t.Run("true", func(t *testing.T) {
		data := map[string]any{"flag": true}
		hash, err := HashStruct("Test", types, data)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, hash)
	})

	t.Run("false", func(t *testing.T) {
		data := map[string]any{"flag": false}
		hash, err := HashStruct("Test", types, data)
		require.NoError(t, err)
		// False encodes as all zeros, but the hash includes the type hash so it's non-zero
		assert.NotEqual(t, [32]byte{}, hash)
	})
}

func TestEncodeFixedBytesValue(t *testing.T) {
	t.Parallel()

	t.Run("bytes", func(t *testing.T) {
		result, err := encodeFixedBytesValue([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 32, len(result))
	})

	t.Run("string", func(t *testing.T) {
		result, err := encodeFixedBytesValue("world")
		require.NoError(t, err)
		assert.Equal(t, 32, len(result))
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := encodeFixedBytesValue(42)
		assert.Error(t, err)
	})
}

func TestEncodeIntValue(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) {
		result, err := encodeIntValue(int(42))
		require.NoError(t, err)
		assert.Equal(t, 32, len(result))
	})

	t.Run("int64", func(t *testing.T) {
		result, err := encodeIntValue(int64(99))
		require.NoError(t, err)
		assert.Equal(t, 32, len(result))
	})

	t.Run("invalid string", func(t *testing.T) {
		_, err := encodeIntValue("not_a_number")
		assert.Error(t, err)
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := encodeIntValue(true)
		assert.Error(t, err)
	})
}

func TestEncodeValueFixedBytes(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Test": {
			{Name: "data", Type: "bytes32"},
		},
	}

	data := map[string]any{
		"data": []byte("fixed-32-byte-data-here!!!!!!!"),
	}

	hash, err := HashStruct("Test", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestHashStructMissingField(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Test": {
			{Name: "field1", Type: "string"},
		},
	}

	_, err := HashStruct("Test", types, map[string]any{})
	assert.Error(t, err)
}

func TestTypeHashUnknownType(t *testing.T) {
	t.Parallel()
	types := apitypes.Types{
		"Test": {
			{Name: "field", Type: "NonExistent"},
		},
	}

	_, err := TypeHash("NonExistent", types)
	assert.Error(t, err)
}
