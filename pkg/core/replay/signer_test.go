package replay

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignerTypeString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Homestead", SignerHomestead.String())
	assert.Equal(t, "EIP-155", SignerEIP155.String())
	assert.Equal(t, "Unknown", SignerType(99).String())
}

func TestValidateChainIDReplayProtectionNil(t *testing.T) {
	t.Parallel()
	err := ValidateChainIDReplayProtection(nil, 0)
	assert.NoError(t, err)
}

func TestValidateChainIDReplayProtectionZeroExpected(t *testing.T) {
	t.Parallel()
	err := ValidateChainIDReplayProtection(big.NewInt(1), 0)
	assert.NoError(t, err)
}

func TestValidateChainIDReplayProtectionMismatch(t *testing.T) {
	t.Parallel()
	err1 := ValidateChainIDReplayProtection(big.NewInt(1), 5)
	assert.Error(t, err1)
	assert.Contains(t, err1.Error(), "chain ID mismatch")

	err2 := ValidateChainIDReplayProtection(big.NewInt(5), 1)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "chain ID mismatch")
}

func TestValidateChainIDReplayProtectionMatch(t *testing.T) {
	t.Parallel()
	err := ValidateChainIDReplayProtection(big.NewInt(1), 1)
	assert.NoError(t, err)
}

func TestValidateChainIDReplayProtectionPreEIP155(t *testing.T) {
	t.Parallel()
	err := ValidateChainIDReplayProtection(nil, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pre-EIP-155 signing")
}

func TestInferEIP155SignerType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, SignerHomestead, InferEIP155SignerType(nil))
	assert.Equal(t, SignerHomestead, InferEIP155SignerType(big.NewInt(0)))
	assert.Equal(t, SignerEIP155, InferEIP155SignerType(big.NewInt(1)))
	assert.Equal(t, SignerEIP155, InferEIP155SignerType(big.NewInt(137)))
}

func TestIsReplayVulnerable(t *testing.T) {
	t.Parallel()
	assert.True(t, IsReplayVulnerable(27))
	assert.True(t, IsReplayVulnerable(28))
	assert.False(t, IsReplayVulnerable(35))
	assert.False(t, IsReplayVulnerable(36))
	assert.False(t, IsReplayVulnerable(0))
}

func TestExtractChainIDFromV(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ExtractChainIDFromV(27))
	assert.Nil(t, ExtractChainIDFromV(28))
	assert.Nil(t, ExtractChainIDFromV(0))
	assert.Nil(t, ExtractChainIDFromV(5))

	chainID := ExtractChainIDFromV(37)
	assert.NotNil(t, chainID)
	assert.Equal(t, int64(1), chainID.Int64())

	chainID2 := ExtractChainIDFromV(35)
	assert.Equal(t, int64(0), chainID2.Int64())
}

func TestExtractChainIDFromVEIP155(t *testing.T) {
	t.Parallel()
	chainID := ExtractChainIDFromV(37)
	assert.Equal(t, int64(1), chainID.Int64())

	chainID = ExtractChainIDFromV(39)
	assert.Equal(t, int64(2), chainID.Int64())

	chainID = ExtractChainIDFromV(35 + 2*137)
	assert.NotNil(t, chainID)
}

func TestValidateSignatureVSuccess(t *testing.T) {
	t.Parallel()
	err := ValidateSignatureV(37, 1)
	assert.NoError(t, err)
}

func TestValidateSignatureVExpectedZero(t *testing.T) {
	t.Parallel()
	err := ValidateSignatureV(27, 0)
	assert.NoError(t, err)
}

func TestValidateSignatureVPreEIP155(t *testing.T) {
	t.Parallel()
	err := ValidateSignatureV(27, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pre-EIP-155 signing")
}

func TestValidateSignatureVMismatch(t *testing.T) {
	t.Parallel()
	err := ValidateSignatureV(37, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encodes chain ID")
}
