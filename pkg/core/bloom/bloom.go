// Package bloom implements an Ethereum-compatible Bloom filter for log event matching.
package bloom

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

type BloomFilter struct {
	bits [256]byte
}

func NewBloomFilter() *BloomFilter { return &BloomFilter{} }

func BloomFilterBytes(b [256]byte) *BloomFilter { return &BloomFilter{bits: b} }

func (bf *BloomFilter) Add(element []byte) {
	hash := keccak256ForBloom(element)
	for i := 0; i < 3; i++ {
		bitIdx := (uint(hash[2*i])<<8 | uint(hash[2*i+1])) & 0x7FF
		byteIdx := bitIdx / 8
		bitOffset := bitIdx % 8
		bf.bits[byteIdx] |= 1 << (7 - bitOffset)
	}
}

func (bf *BloomFilter) AddAddress(addr common.Address) { bf.Add(addr.Bytes()) }
func (bf *BloomFilter) AddTopic(topic common.Hash)      { bf.Add(topic.Bytes()) }

func (bf *BloomFilter) Test(element []byte) bool {
	hash := keccak256ForBloom(element)
	for i := 0; i < 3; i++ {
		bitIdx := (uint(hash[2*i])<<8 | uint(hash[2*i+1])) & 0x7FF
		byteIdx := bitIdx / 8
		bitOffset := bitIdx % 8
		if bf.bits[byteIdx]&(1<<(7-bitOffset)) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) TestAddress(addr common.Address) bool { return bf.Test(addr.Bytes()) }
func (bf *BloomFilter) TestTopic(topic common.Hash) bool     { return bf.Test(topic.Bytes()) }
func (bf *BloomFilter) Bytes() [256]byte                     { return bf.bits }

func (bf *BloomFilter) ToEthereumBloom() types.Bloom {
	return types.BytesToBloom(bf.bits[:])
}

func keccak256ForBloom(data []byte) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	return hasher.Sum(nil)[:6]
}

func EthereumBlockBloom(logs []*types.Log) *BloomFilter {
	bf := NewBloomFilter()
	for _, log := range logs {
		bf.AddAddress(log.Address)
		for _, topic := range log.Topics {
			bf.AddTopic(topic)
		}
	}
	return bf
}

func MatchBlockBloom(bloom types.Bloom, addr *common.Address, topics []common.Hash) bool {
	var bloomBytes [256]byte
	copy(bloomBytes[:], bloom.Bytes())
	bf := BloomFilterBytes(bloomBytes)
	if addr != nil && !bf.TestAddress(*addr) {
		return false
	}
	for _, topic := range topics {
		if !bf.TestTopic(topic) {
			return false
		}
	}
	return true
}