package core

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestBloomAddAndTest(t *testing.T) {
	t.Parallel()

	bf := NewBloomFilter()
	addr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")

	// Before adding: should return false (definitely not present)
	if bf.TestAddress(addr) {
		t.Error("expected false before adding address")
	}

	bf.AddAddress(addr)

	// After adding: should return true (possibly present)
	if !bf.TestAddress(addr) {
		t.Error("expected true after adding address")
	}
}

func TestBloomTopic(t *testing.T) {
	t.Parallel()

	bf := NewBloomFilter()
	topic := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	bf.AddTopic(topic)
	if !bf.TestTopic(topic) {
		t.Error("expected true after adding topic")
	}
}

func TestBloomDefiniteNegative(t *testing.T) {
	t.Parallel()

	bf := NewBloomFilter()
	addr1 := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	addr2 := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

	bf.AddAddress(addr1)

	// addr2 was NOT added - bloom should correctly report "definitely not present"
	// (assuming no false positive from addr1's bits overlapping addr2's)
	if bf.TestAddress(addr2) {
		// This COULD be a false positive (bloom is probabilistic)
		// But with only 1 element and 2048 bits, probability is very low
		t.Log("note: possible false positive (extremely unlikely with 1 element)")
	}
}

func TestBloomFalsePositiveRate(t *testing.T) {
	t.Parallel()

	bf := NewBloomFilter()
	rng := rand.New(rand.NewSource(42))

	// Add 100 random addresses
	added := make([]common.Address, 100)
	for i := range added {
		var addr common.Address
		rng.Read(addr[:])
		added[i] = addr
		bf.AddAddress(addr)
	}

	// Test 100 random addresses that were NOT added
	falsePositives := 0
	notAdded := 100
	for i := 0; i < notAdded; i++ {
		var addr common.Address
		rng.Read(addr[:])
		// Skip if it happens to match an added one (extremely unlikely)
		isAdded := false
		for _, a := range added {
			if addr == a {
				isAdded = true
				break
			}
		}
		if isAdded {
			continue
		}
		if bf.TestAddress(addr) {
			falsePositives++
		}
	}

	fpr := float64(falsePositives) / float64(notAdded)
	t.Logf("false positive rate: %d/%d = %.4f%%", falsePositives, notAdded, fpr*100)
	t.Logf("estimated FPR from formula: %.4f%%", bf.FalsePositiveRate()*100)

	// With 100 elements in 2048 bits, expected FPR ≈ 0.7%
	// Allow some margin: should be under 5%
	if fpr > 0.05 {
		t.Errorf("false positive rate too high: %.4f (expected < 0.05)", fpr)
	}
}

func TestEthereumBlockBloom(t *testing.T) {
	t.Parallel()

	logs := []*types.Log{
		{
			Address: common.HexToAddress("0x1234"),
			Topics: []common.Hash{
				common.HexToHash("0xaabb"),
				common.HexToHash("0xccdd"),
			},
		},
		{
			Address: common.HexToAddress("0x5678"),
			Topics: []common.Hash{
				common.HexToHash("0xeeff"),
			},
		},
	}

	bloom := EthereumBlockBloom(logs)

	if !bloom.TestAddress(common.HexToAddress("0x1234")) {
		t.Error("expected bloom to contain 0x1234")
	}
	if !bloom.TestTopic(common.HexToHash("0xaabb")) {
		t.Error("expected bloom to contain topic 0xaabb")
	}

	// Non-added address should NOT match
	if bloom.TestAddress(common.HexToAddress("0xDEAD")) {
		t.Log("note: possible false positive (expected)")

		// Convert to go-ethereum bloom
		ethBloom := bloom.ToEthereumBloom()
		if !ethBloom.Test(common.HexToAddress("0x1234").Bytes()) {
			t.Error("go-ethereum bloom should match existing address")
		}
	}
}

func TestMatchBlockBloom(t *testing.T) {
	t.Parallel()

	logs := []*types.Log{
		{
			Address: common.HexToAddress("0xabc"),
			Topics: []common.Hash{
				common.HexToHash("0xdef"),
			},
		},
	}

	ethBloom := EthereumBlockBloom(logs).ToEthereumBloom()

	// Matching query
	addr := common.HexToAddress("0xabc")
	if !MatchBlockBloom(ethBloom, &addr, nil) {
		t.Error("MatchBlockBloom should match existing address")
	}

	// Non-matching query
	nonAddr := common.HexToAddress("0x999")
	if MatchBlockBloom(ethBloom, &nonAddr, nil) {
		t.Log("note: possible false positive for non-matching address")
	}

	// Both address and topics
	topic := common.HexToHash("0xdef")
	if !MatchBlockBloom(ethBloom, &addr, []common.Hash{topic}) {
		t.Error("MatchBlockBloom should match existing address+topic")
	}
}

func TestBloomBytesRoundTrip(t *testing.T) {
	t.Parallel()

	original := NewBloomFilter()
	addr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	original.AddAddress(addr)

	bytes := original.Bytes()
	restored := BloomFilterBytes(bytes)

	if !restored.TestAddress(addr) {
		t.Error("round-trip through Bytes lost bloom data")
	}
}

func TestBloomEmpty(t *testing.T) {
	t.Parallel()

	bf := NewBloomFilter()

	// Empty bloom should match nothing
	addr := common.HexToAddress("0x1234")
	if bf.TestAddress(addr) {
		t.Error("empty bloom should not match anything")
	}

	// FPR should be 0 for empty bloom
	if fpr := bf.FalsePositiveRate(); fpr != 0.0 {
		t.Errorf("empty bloom FPR should be 0, got %f", fpr)
	}
}
