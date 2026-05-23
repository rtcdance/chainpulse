package crypto

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEthSignHash(t *testing.T) {
	t.Parallel()
	hash := EthSignHash([]byte("hello world"))
	assert.NotEqual(t, common.Hash{}, hash)
}

func TestEthSignHashEmpty(t *testing.T) {
	t.Parallel()
	hash := EthSignHash([]byte{})
	assert.NotEqual(t, common.Hash{}, hash)
}

func TestEthSignHashDeterministic(t *testing.T) {
	t.Parallel()
	h1 := EthSignHash([]byte("test"))
	h2 := EthSignHash([]byte("test"))
	assert.Equal(t, h1, h2)
}

func TestEthSignHashDifferentMessages(t *testing.T) {
	t.Parallel()
	h1 := EthSignHash([]byte("message1"))
	h2 := EthSignHash([]byte("message2"))
	assert.NotEqual(t, h1, h2)
}

func TestEthSignHashPrefix(t *testing.T) {
	t.Parallel()
	hash := EthSignHash([]byte("hello"))

	rawHash := ethcrypto.Keccak256Hash([]byte("hello"))
	assert.NotEqual(t, rawHash, hash)
}

func TestSignAndRecover(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	message := []byte("sign this message")
	sig, err := SignMessage(message, key)
	require.NoError(t, err)
	assert.Len(t, sig, 65)

	recovered, err := RecoverAddress(message, sig)
	require.NoError(t, err)
	assert.Equal(t, addr, recovered)
}

func TestSignAndVerify(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	message := []byte("verify me")
	sig, err := SignMessage(message, key)
	require.NoError(t, err)

	err = VerifySignature(message, sig, addr)
	assert.NoError(t, err)
}

func TestVerifySignatureWrongAddress(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	message := []byte("message")
	sig, err := SignMessage(message, key)
	require.NoError(t, err)

	wrongAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	err = VerifySignature(message, sig, wrongAddr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recovered")
}

func TestRecoverAddressInvalidSigLength(t *testing.T) {
	_, err := RecoverAddress([]byte("msg"), []byte{0x01, 0x02})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature length")
}

func TestRecoverAddressInvalidSignature(t *testing.T) {
	invalidSig := make([]byte, 65)
	_, err := RecoverAddress([]byte("msg"), invalidSig)
	assert.Error(t, err)
}

func TestRecoverAddressFromHash(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	hash := EthSignHash([]byte("test hash recovery"))
	sig, err := SignMessage([]byte("test hash recovery"), key)
	require.NoError(t, err)

	sig[64] -= 27

	recovered, err := RecoverAddressFromHash(hash, sig)
	require.NoError(t, err)
	assert.Equal(t, addr, recovered)
}

func TestRecoverAddressFromHashInvalidSigLength(t *testing.T) {
	hash := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	_, err := RecoverAddressFromHash(hash, []byte{0x01})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature length")
}

func TestSignMessage(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	sig, err := SignMessage([]byte("hello"), key)
	require.NoError(t, err)
	assert.Len(t, sig, 65)
	assert.GreaterOrEqual(t, sig[64], uint8(27))
}

func TestSignMessageConsistency(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	sig1, err := SignMessage([]byte("same"), key)
	require.NoError(t, err)
	sig2, err := SignMessage([]byte("same"), key)
	require.NoError(t, err)

	assert.Len(t, sig1, 65)
	assert.Len(t, sig2, 65)
}

func TestPublicKeyToAddress(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	addr1 := PublicKeyToAddress(&key.PublicKey)
	addr2 := ethcrypto.PubkeyToAddress(key.PublicKey)
	assert.Equal(t, addr2, addr1)
}

func TestRecoverAddressHighV(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	message := []byte("high v test")
	sig, err := SignMessage(message, key)
	require.NoError(t, err)

	recovered, err := RecoverAddress(message, sig)
	require.NoError(t, err)
	assert.Equal(t, addr, recovered)
}

func roundTrip(t *testing.T, msg string) {
	t.Helper()
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	message := []byte(msg)
	sig, err := SignMessage(message, key)
	require.NoError(t, err)

	err = VerifySignature(message, sig, addr)
	assert.NoError(t, err)

	recovered, err := RecoverAddress(message, sig)
	require.NoError(t, err)
	assert.Equal(t, addr, recovered)
}

func TestSignVerifyRecoverRoundTrip(t *testing.T) {
	roundTrip(t, "simple")
	roundTrip(t, "longer message with more content")
	roundTrip(t, "")
	roundTrip(t, "1234567890")
}

func TestEthSignHashLengthEncoding(t *testing.T) {
	t.Parallel()
	h1 := EthSignHash([]byte("abc"))
	h2 := EthSignHash([]byte("abcd"))
	assert.NotEqual(t, h1, h2)
}

func TestRecoverAddressRecoveryIDAdjustment(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	msg := []byte("recovery id test")
	sig, err := SignMessage(msg, key)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sig[64], uint8(27))

	recovered, err := RecoverAddress(msg, sig)
	require.NoError(t, err)
	assert.Equal(t, addr, recovered)
}
