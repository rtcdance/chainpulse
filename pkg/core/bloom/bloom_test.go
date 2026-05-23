package bloom

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBloomFilter(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	assert.NotNil(t, bf)
	assert.Equal(t, [256]byte{}, bf.Bytes())
}

func TestBloomFilterBytes(t *testing.T) {
	t.Parallel()
	bits := [256]byte{}
	bits[0] = 0xFF
	bf := BloomFilterBytes(bits)
	assert.Equal(t, bits, bf.Bytes())
}

func TestAddAndTest(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	data := []byte("hello world")

	assert.False(t, bf.Test(data))

	bf.Add(data)
	assert.True(t, bf.Test(data))
}

func TestAddAndTestDifferentData(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	bf.Add([]byte("alpha"))

	assert.True(t, bf.Test([]byte("alpha")))
	assert.False(t, bf.Test([]byte("beta")))
}

func TestAddAddress(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	assert.False(t, bf.TestAddress(addr))
	bf.AddAddress(addr)
	assert.True(t, bf.TestAddress(addr))
}

func TestAddTopic(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	topic := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	assert.False(t, bf.TestTopic(topic))
	bf.AddTopic(topic)
	assert.True(t, bf.TestTopic(topic))
}

func TestToEthereumBloom(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	bf.Add([]byte("test data"))

	ethBloom := bf.ToEthereumBloom()
	assert.Equal(t, types.Bloom(bf.Bytes()), ethBloom)
}

func TestEthereumBlockBloom(t *testing.T) {
	t.Parallel()
	logs := []*types.Log{
		{
			Address: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			Topics: []common.Hash{
				common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			},
		},
		{
			Address: common.HexToAddress("0x2222222222222222222222222222222222222222"),
			Topics: []common.Hash{
				common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
				common.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
			},
		},
	}

	bf := EthereumBlockBloom(logs)

	t.Run("contains address from first log", func(t *testing.T) {
		assert.True(t, bf.TestAddress(common.HexToAddress("0x1111111111111111111111111111111111111111")))
	})

	t.Run("contains address from second log", func(t *testing.T) {
		assert.True(t, bf.TestAddress(common.HexToAddress("0x2222222222222222222222222222222222222222")))
	})

	t.Run("contains topic from first log", func(t *testing.T) {
		assert.True(t, bf.TestTopic(common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")))
	})

	t.Run("contains topics from second log", func(t *testing.T) {
		assert.True(t, bf.TestTopic(common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")))
		assert.True(t, bf.TestTopic(common.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")))
	})

	t.Run("does not contain unrelated address", func(t *testing.T) {
		assert.False(t, bf.TestAddress(common.HexToAddress("0x3333333333333333333333333333333333333333")))
	})
}

func TestEthereumBlockBloomEmpty(t *testing.T) {
	t.Parallel()
	bf := EthereumBlockBloom(nil)
	assert.Equal(t, [256]byte{}, bf.Bytes())
}

func TestMatchBlockBloom(t *testing.T) {
	t.Parallel()
	logs := []*types.Log{
		{
			Address: common.HexToAddress("0xdead000000000000000000000000000000000000"),
			Topics: []common.Hash{
				common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000"),
			},
		},
	}

	bf := EthereumBlockBloom(logs)
	bfBytes := bf.Bytes()
	bloom := types.BytesToBloom(bfBytes[:])

	t.Run("match address and topic", func(t *testing.T) {
		addr := common.HexToAddress("0xdead000000000000000000000000000000000000")
		topics := []common.Hash{
			common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000"),
		}
		assert.True(t, MatchBlockBloom(bloom, &addr, topics))
	})

	t.Run("match address only", func(t *testing.T) {
		addr := common.HexToAddress("0xdead000000000000000000000000000000000000")
		assert.True(t, MatchBlockBloom(bloom, &addr, nil))
	})

	t.Run("no match wrong address", func(t *testing.T) {
		addr := common.HexToAddress("0xbeef000000000000000000000000000000000000")
		assert.False(t, MatchBlockBloom(bloom, &addr, nil))
	})

	t.Run("match topic only", func(t *testing.T) {
		topics := []common.Hash{
			common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000"),
		}
		assert.True(t, MatchBlockBloom(bloom, nil, topics))
	})

	t.Run("no match wrong topic", func(t *testing.T) {
		topics := []common.Hash{
			common.HexToHash("0xcafebabe00000000000000000000000000000000000000000000000000000000"),
		}
		assert.False(t, MatchBlockBloom(bloom, nil, topics))
	})

	t.Run("empty bloom never matches addr", func(t *testing.T) {
		emptyBloom := types.Bloom{}
		addr := common.HexToAddress("0xdead000000000000000000000000000000000000")
		assert.False(t, MatchBlockBloom(emptyBloom, &addr, nil))
	})

	t.Run("empty bloom never matches topic", func(t *testing.T) {
		emptyBloom := types.Bloom{}
		topics := []common.Hash{
			common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000"),
		}
		assert.False(t, MatchBlockBloom(emptyBloom, nil, topics))
	})

	t.Run("nil addr and nil topics returns true", func(t *testing.T) {
		assert.True(t, MatchBlockBloom(bloom, nil, nil))
	})
}

func TestKeccak256ForBloom(t *testing.T) {
	t.Parallel()
	result := keccak256ForBloom([]byte("test"))
	require.NotNil(t, result)
	assert.Len(t, result, 6)
}

func TestKeccak256ForBloomDeterministic(t *testing.T) {
	t.Parallel()
	a := keccak256ForBloom([]byte("hello"))
	b := keccak256ForBloom([]byte("hello"))
	assert.True(t, bytes.Equal(a, b))
}

func TestKeccak256ForBloomDifferentInputs(t *testing.T) {
	t.Parallel()
	a := keccak256ForBloom([]byte("hello"))
	b := keccak256ForBloom([]byte("world"))
	assert.False(t, bytes.Equal(a, b))
}

func TestBloomFilterNoFalseNegatives(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	testData := [][]byte{
		[]byte("one"), []byte("two"), []byte("three"),
		[]byte("four"), []byte("five"),
	}
	for _, d := range testData {
		bf.Add(d)
	}
	for _, d := range testData {
		assert.True(t, bf.Test(d), "should contain %s", string(d))
	}
}

func TestBloomFilterMultipleAdds(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter()
	bf.Add([]byte("alpha"))
	bf.Add([]byte("alpha"))
	assert.True(t, bf.Test([]byte("alpha")))
}
