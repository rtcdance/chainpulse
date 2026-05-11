package core

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
	_, err = HashStruct("Mail", types, map[string]interface{}{
		"from": "some_value",
	})
	assert.Error(t, err)
}

func TestHashStructPrimitiveTypes(t *testing.T) {
	types := apitypes.Types{
		"TestStruct": {
			{Name: "intValue", Type: "uint256"},
			{Name: "boolValue", Type: "bool"},
			{Name: "addrValue", Type: "address"},
		},
	}

	data := map[string]interface{}{
		"intValue":  big.NewInt(42),
		"boolValue": true,
		"addrValue": common.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	hash, err := HashStruct("TestStruct", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestHashDomainSeparator(t *testing.T) {
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
	domain := EIP712Domain{
		Name:    "Test",
		Version: "1",
		ChainID: nil,
	}

	_, err := HashDomainSeparator(domain)
	assert.ErrorIs(t, err, ErrMissingChainID)
}

func TestHashTypedData(t *testing.T) {
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
		Message: map[string]interface{}{
			"from": map[string]interface{}{
				"name":   "Cow",
				"wallet": common.HexToAddress("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"),
			},
			"to": map[string]interface{}{
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
	td := TypedData{
		Types:       apitypes.Types{},
		PrimaryType: "Test",
		Message:     nil,
	}

	_, err := HashTypedData(td)
	assert.ErrorIs(t, err, ErrNilMessage)
}

func TestVerifyTypedDataRoundTrip(t *testing.T) {
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
		Message: map[string]interface{}{
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
	td := TypedData{
		Types:       apitypes.Types{},
		PrimaryType: "Test",
		Domain: EIP712Domain{
			Name:    "Test",
			Version: "1",
			ChainID: big.NewInt(1),
		},
		Message: map[string]interface{}{"foo": "bar"},
	}

	_, err := VerifyTypedData(td, []byte{1, 2, 3}, common.Address{})
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestEncodeValueStringAndBytes(t *testing.T) {
	types := apitypes.Types{
		"Test": {
			{Name: "str", Type: "string"},
			{Name: "bts", Type: "bytes"},
		},
	}

	data := map[string]interface{}{
		"str": "hello",
		"bts": []byte("world"),
	}

	hash, err := HashStruct("Test", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestEncodeValueIntFromString(t *testing.T) {
	types := apitypes.Types{
		"Test": {
			{Name: "amount", Type: "uint256"},
		},
	}

	data := map[string]interface{}{
		"amount": "1000000000000000000",
	}

	hash, err := HashStruct("Test", types, data)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestEncodeValueBool(t *testing.T) {
	types := apitypes.Types{
		"Test": {
			{Name: "flag", Type: "bool"},
		},
	}

	t.Run("true", func(t *testing.T) {
		data := map[string]interface{}{"flag": true}
		hash, err := HashStruct("Test", types, data)
		require.NoError(t, err)
		assert.NotEqual(t, [32]byte{}, hash)
	})

	t.Run("false", func(t *testing.T) {
		data := map[string]interface{}{"flag": false}
		hash, err := HashStruct("Test", types, data)
		require.NoError(t, err)
		// False encodes as all zeros, but the hash includes the type hash so it's non-zero
		assert.NotEqual(t, [32]byte{}, hash)
	})
}
