package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

// BloomFilter implements a simplified Ethereum Bloom filter for educational purposes.
//
// In Ethereum, each block header contains a 2048-bit (256-byte) Bloom filter (field
// "logsBloom") that encodes all log addresses and topics in the block. This allows
// light clients and indexers to quickly determine whether a block *might* contain
// events of interest without downloading the full block.
//
// How Ethereum's Bloom filter works:
//  1. Each address or topic is hashed with keccak256
//  2. Bits [0..7] and [8..15] and [16..23] of the hash determine 3 bit positions
//  3. The position is taken modulo 2048 (the Bloom filter size)
//  4. Those 3 bits are set to 1 in the 2048-bit bloom
//
// Checking membership:
//   - Compute the same 3 bit positions from the query hash
//   - If ALL 3 bits are set in the bloom, the element MAY be in the block
//   - If ANY bit is NOT set, the element is DEFINITELY NOT in the block
//
// False positive rate: approx (1 - e^(-kn/m))^k where:
//   - m = 2048 (filter size in bits)
//   - k = 3 (hash functions, i.e., bit positions checked)
//   - n = number of elements (addresses + topics) in the block
type BloomFilter struct {
	bits [256]byte // 2048 bits
}

// NewBloomFilter creates an empty Bloom filter.
func NewBloomFilter() *BloomFilter {
	return &BloomFilter{}
}

// BloomFilterBytes creates a Bloom filter from an existing 256-byte representation.
func BloomFilterBytes(b [256]byte) *BloomFilter {
	return &BloomFilter{bits: b}
}

// Add inserts an element (address or topic hash) into the Bloom filter by setting
// the 3 bits determined by keccak256(element).
//
// This matches go-ethereum's crypto.Keccak256Hash() Bloom lookup logic:
//
//	hash = keccak256(element)
//	bit1 = hash[0..1] as uint16 & 0x7FF  (lower 11 bits)
//	bit2 = hash[2..3] as uint16 & 0x7FF
//	bit3 = hash[4..5] as uint16 & 0x7FF
//	set bits[bit1], bits[bit2], bits[bit3]
func (bf *BloomFilter) Add(element []byte) {
	hash := keccak256ForBloom(element)
	for i := 0; i < 3; i++ {
		bitIdx := (uint(hash[2*i])<<8 | uint(hash[2*i+1])) & 0x7FF
		byteIdx := bitIdx / 8
		bitOffset := bitIdx % 8
		bf.bits[byteIdx] |= 1 << (7 - bitOffset) // MSB 0 = first bit (big-endian)
	}
}

// AddAddress adds an Ethereum address to the Bloom filter.
func (bf *BloomFilter) AddAddress(addr common.Address) {
	bf.Add(addr.Bytes())
}

// AddTopic adds an event topic hash to the Bloom filter.
func (bf *BloomFilter) AddTopic(topic common.Hash) {
	bf.Add(topic.Bytes())
}

// Test checks whether an element is possibly in the Bloom filter.
// Returns true if the element MAY be present (could be false positive).
// Returns false if the element is definitely NOT present.
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

// TestAddress checks whether an address is possibly in the Bloom filter.
func (bf *BloomFilter) TestAddress(addr common.Address) bool {
	return bf.Test(addr.Bytes())
}

// TestTopic checks whether a topic hash is possibly in the Bloom filter.
func (bf *BloomFilter) TestTopic(topic common.Hash) bool {
	return bf.Test(topic.Bytes())
}

// Bytes returns the 256-byte Bloom filter representation,
// matching Ethereum's logsBloom block header field format.
func (bf *BloomFilter) Bytes() [256]byte {
	return bf.bits
}

// ToEthereumBloom converts to go-ethereum's types.Bloom for compatibility
// with the existing codebase's types.Bloom fields in block models.
func (bf *BloomFilter) ToEthereumBloom() types.Bloom {
	return types.BytesToBloom(bf.bits[:])
}

// FalsePositiveRate estimates the current false positive probability.
//
// Formula: (1 - e^(-k*n/m))^k
//   - m: filter size in bits (2048)
//   - k: number of hash functions (3)
//   - n: approximate number of elements added
func (bf *BloomFilter) FalsePositiveRate() float64 {
	m := 2048.0
	k := 3.0

	elements := bf.estimateElementCount()
	if elements == 0 {
		return 0.0
	}

	n := float64(elements)
	exp := -k * n / m
	fpr := 1.0 - expApprox(exp)
	result := 1.0
	for i := 0; i < int(k); i++ {
		result *= fpr
	}
	return result
}

// estimateElementCount roughly estimates n from the number of set bits.
// Uses the formula: n ≈ -m/k * ln(1 - X/m) where X = number of set bits.
func (bf *BloomFilter) estimateElementCount() int {
	setBits := 0
	for _, b := range bf.bits {
		setBits += popcount(b)
	}
	if setBits == 0 {
		return 0
	}

	m := 2048.0
	k := 3.0
	x := float64(setBits)

	ratio := x / m
	if ratio >= 1.0 {
		return int(m / k * 2) // saturated
	}

	// ln(1 - ratio) via Taylor: logApprox(x) computes ln(1+x), so call with -ratio
	n := -(m / k) * logApprox(-ratio)
	if n < 0 {
		n = 0
	}
	return int(n + 0.5)
}

// Counting bloom: only used for FPR estimation
func popcount(x byte) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// expApprox computes exp(x) using Taylor series for small x.
func expApprox(x float64) float64 {
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
	}
	return result
}

// logApprox computes ln(1+x) using Taylor series for small x.
func logApprox(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1 // clamped for bloom estimation purposes
	}
	result := 0.0
	term := x
	for i := 1; i < 20; i++ {
		if i%2 == 1 {
			result += term / float64(i)
		} else {
			result -= term / float64(i)
		}
		term *= x
	}
	return result
}

// keccak256ForBloom computes keccak256 hash using only the first 6 bytes
// (matching go-ethereum's bloom lookup: hash[0..5] determine 3 bit positions).
func keccak256ForBloom(data []byte) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	h := hasher.Sum(nil)
	// Only first 6 bytes matter for bloom (3 x 2-byte indices)
	return h[:6]
}

// EthereumBlockBloom creates a Bloom filter from all logs in a block,
// matching the way Ethereum constructs the logsBloom header field.
//
// For each log in the block:
//   - The contract address is added
//   - Each topic in the log is added
//
// This is exactly what geth does in types.NewBloom().
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

// MatchBlockBloom checks whether a block's Bloom filter potentially contains
// logs matching the given address and topics.
//
// This is the bloom-checking step that Ethereum nodes perform BEFORE
// fetching full receipts. If the bloom doesn't match, the block is skipped
// entirely for log queries, making eth_getLogs much more efficient.
func MatchBlockBloom(bloom types.Bloom, addr *common.Address, topics []common.Hash) bool {
	var bloomBytes [256]byte
	copy(bloomBytes[:], bloom.Bytes())
	bf := BloomFilterBytes(bloomBytes)

	if addr != nil {
		if !bf.TestAddress(*addr) {
			return false
		}
	}

	for _, topic := range topics {
		if !bf.TestTopic(topic) {
			return false
		}
	}

	return true
}
