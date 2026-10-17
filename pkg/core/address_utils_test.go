package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToChecksummedAddress(t *testing.T) {
	// Official EIP-55 test vectors from https://eips.ethereum.org/EIPS/eip-55
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"all lowercase", "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"},
		{"mid uppercase", "0xfb6916095ca1df60bb79ce92ce3ea74c37c5d359", "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359"},
		{"leading uppercase", "0xdbf03b407c01e7cd3cbea99509d93f8dddc8c6fb", "0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB"},
		{"repeating pattern", "0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb", "0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToChecksummedAddress(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}

	t.Run("empty string passthrough", func(t *testing.T) {
		assert.Equal(t, "", ToChecksummedAddress(""))
	})

	t.Run("invalid length passthrough", func(t *testing.T) {
		assert.Equal(t, "0x1234", ToChecksummedAddress("0x1234"))
	})

	t.Run("without 0x prefix", func(t *testing.T) {
		result := ToChecksummedAddress("5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
		assert.Equal(t, "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", result)
	})

	t.Run("already checksummed", func(t *testing.T) {
		input := "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
		assert.Equal(t, input, ToChecksummedAddress(input))
	})
}

func TestValidateEIP55Checksum(t *testing.T) {
	t.Run("valid checksummed address", func(t *testing.T) {
		err := ValidateEIP55Checksum("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
		assert.NoError(t, err)
	})

	t.Run("all lowercase is valid", func(t *testing.T) {
		err := ValidateEIP55Checksum("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
		assert.NoError(t, err)
	})

	t.Run("all uppercase is valid", func(t *testing.T) {
		err := ValidateEIP55Checksum("0x5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED")
		assert.NoError(t, err)
	})

	t.Run("invalid checksum", func(t *testing.T) {
		err := ValidateEIP55Checksum("0x5aAeb6053F3E94C9b9A09f33669435e7ef1BeAed")
		assert.ErrorIs(t, err, ErrInvalidChecksum)
	})

	t.Run("empty address", func(t *testing.T) {
		err := ValidateEIP55Checksum("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("missing 0x prefix", func(t *testing.T) {
		err := ValidateEIP55Checksum("5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "0x")
	})

	t.Run("wrong length", func(t *testing.T) {
		err := ValidateEIP55Checksum("0x1234")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "40 hex")
	})
}

func TestIsChecksummedAddress(t *testing.T) {
	assert.True(t, IsChecksummedAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"))
	assert.False(t, IsChecksummedAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"))
}

func TestValidateAddressChecksum(t *testing.T) {
	addr := common.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	err := ValidateAddressChecksum(addr)
	require.NoError(t, err, "common.Address.Hex() returns checksummed form, should always pass")
}

func TestBlockchainEventValidateWithChecksum(t *testing.T) {
	t.Run("valid contract address passes", func(t *testing.T) {
		event := &blockchain.BlockchainEvent{
			BlockNumber:     1,
			TransactionHash: common.HexToHash("0x01"),
			ContractAddress: common.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
			EventName:       "Transfer",
		}
		assert.NoError(t, ValidateBlockchainEvent(event))
	})

	t.Run("zero contract address still fails", func(t *testing.T) {
		event := &blockchain.BlockchainEvent{
			BlockNumber:     1,
			TransactionHash: common.HexToHash("0x01"),
			ContractAddress: common.Address{},
			EventName:       "Transfer",
		}
		assert.ErrorIs(t, ValidateBlockchainEvent(event), ErrInvalidContractAddress)
	})
}
