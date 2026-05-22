package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestDetectBlockBuilder(t *testing.T) {
	t.Run("known Flashbots builder", func(t *testing.T) {
		block := &Block{
			Number: 17000000,
			Miner:  common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"),
		}
		builder := DetectBlockBuilder(block)
		assert.NotNil(t, builder)
		assert.Equal(t, "Flashbots", builder.BuilderName)
		assert.True(t, builder.IsMevBoost)
		assert.Equal(t, block.Miner, builder.BuilderAddress)
	})

	t.Run("known Builder0x69", func(t *testing.T) {
		block := &Block{
			Number: 17500000,
			Miner:  common.HexToAddress("0x388C818CA8B9251b393131C08a736A67ccB19297"),
		}
		builder := DetectBlockBuilder(block)
		assert.NotNil(t, builder)
		assert.Equal(t, "Builder0x69", builder.BuilderName)
		assert.True(t, builder.IsMevBoost)
	})

	t.Run("unknown miner returns nil", func(t *testing.T) {
		block := &Block{
			Number: 15000000,
			Miner:  common.HexToAddress("0x1234567890123456789012345678901234567890"),
		}
		builder := DetectBlockBuilder(block)
		assert.Nil(t, builder)
	})

	t.Run("nil block returns nil", func(t *testing.T) {
		builder := DetectBlockBuilder(nil)
		assert.Nil(t, builder)
	})

	t.Run("zero address returns nil", func(t *testing.T) {
		block := &Block{Number: 1, Miner: common.Address{}}
		builder := DetectBlockBuilder(block)
		assert.Nil(t, builder)
	})
}

func TestIsMevBoostBlock(t *testing.T) {
	t.Run("MEV block", func(t *testing.T) {
		block := &Block{
			Miner: common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"),
		}
		assert.True(t, IsMevBoostBlock(block))
	})

	t.Run("non-MEV block", func(t *testing.T) {
		block := &Block{
			Miner: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		}
		assert.False(t, IsMevBoostBlock(block))
	})
}

func TestBlockBuilderFieldOnBlock(t *testing.T) {
	block := &Block{
		Number: 17000000,
		Miner:  common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"),
	}

	// Before detection
	assert.Nil(t, block.Builder)

	// After detection
	builder := DetectBlockBuilder(block)
	block.Builder = builder
	assert.NotNil(t, block.Builder)
	assert.Equal(t, "Flashbots", block.Builder.BuilderName)
	assert.True(t, block.Builder.IsMevBoost)
}
