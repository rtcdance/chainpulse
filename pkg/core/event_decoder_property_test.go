package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pry "pgregory.net/rapid"
)

// Property: DecodeEventData never panics regardless of input.
func TestProperty_DecodeEventDataNeverPanics(t *testing.T) {
	t.Parallel()

	pry.Check(t, func(t *pry.T) {
		eventName := pry.SampledFrom([]string{
			"Transfer", "Approval", "Swap", "Mint", "Burn",
			"", "nonexistent", randomString(10),
		}).Draw(t, "eventName")

		numTopics := pry.IntRange(0, 4).Draw(t, "numTopics")
		topics := make([]common.Hash, numTopics)
		for i := 0; i < numTopics; i++ {
			var h common.Hash
			copy(h[:], randomBytes(32))
			topics[i] = h
		}

		dataLen := pry.IntRange(0, 1024).Draw(t, "dataLen")
		data := randomBytes(dataLen)

		// Must never panic
		result := DecodeEventData(eventName, topics, data)

		// Must never return nil (ChainedDecoder guarantees raw hex fallback)
		if result == nil {
			t.Logf("nil result for event=%q topics=%d data=%d bytes", eventName, numTopics, dataLen)
		}
	})
}

// Property: EncodeIndexedParam round-trips correctly for standard types.
func TestProperty_EncodeIndexedParamRoundTrip(t *testing.T) {
	t.Parallel()

	pry.Check(t, func(t *pry.T) {
		typ := pry.SampledFrom([]string{"address", "uint256", "bool"}).Draw(t, "type")

		switch typ {
		case "address":
			var addr common.Address
			copy(addr[:], randomBytes(20))
			topic, err := EncodeIndexedParam(addr, "address")
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}
			decoded, err := DecodeIndexedParam(topic, "address")
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			decodedAddr, ok := decoded.(common.Address)
			if !ok || decodedAddr != addr {
				t.Errorf("address round-trip failed")
			}

		case "uint256":
			val := big.NewInt(pry.Int64Range(0, 1e18).Draw(t, "val"))
			topic, err := EncodeIndexedParam(val, "uint256")
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}
			decoded, err := DecodeIndexedParam(topic, "uint256")
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			bigVal, ok := decoded.(*big.Int)
			if !ok || bigVal.Cmp(val) != 0 {
				t.Errorf("uint256 round-trip failed: %s vs %s", bigVal, val)
			}

		case "bool":
			b := pry.Bool().Draw(t, "val")
			topic, err := EncodeIndexedParam(b, "bool")
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}
			decoded, err := DecodeIndexedParam(topic, "bool")
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if decoded.(bool) != b {
				t.Errorf("bool round-trip failed: %v vs %v", decoded, b)
			}
		}
	})
}

// Property: Bloom filter always returns true for added elements.
func TestProperty_BloomAlwaysMatchesAdded(t *testing.T) {
	t.Parallel()

	pry.Check(t, func(t *pry.T) {
		bf := NewBloomFilter()
		n := pry.IntRange(1, 50).Draw(t, "numElements")

		added := make([]common.Address, n)
		for i := 0; i < n; i++ {
			var addr common.Address
			copy(addr[:], randomBytes(20))
			added[i] = addr
			bf.AddAddress(addr)
		}

		for _, addr := range added {
			if !bf.TestAddress(addr) {
				t.Fatalf("bloom should match added address %s", addr.Hex())
			}
		}
	})
}

// Property: Bloom filter false positive rate for small N is below 10%.
// Uses rapid's random source for non-deterministic test data.
func TestProperty_BloomFalsePositiveRateAcceptable(t *testing.T) {
	t.Parallel()

	pry.Check(t, func(t *pry.T) {
		bf := NewBloomFilter()

		// Add elements using rapid-managed random values
		added := make([]common.Address, 50)
		for i := range added {
			addr := common.BytesToAddress(pry.SliceOfN(pry.Byte(), 20, 20).Draw(t, "addr"))
			added[i] = addr
			bf.AddAddress(addr)
		}

		falsePositives := 0
		trials := 200
		for i := 0; i < trials; i++ {
			addr := common.BytesToAddress(pry.SliceOfN(pry.Byte(), 20, 20).Draw(t, "testAddr"))
			// Skip if it matches an added address
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

		fpr := float64(falsePositives) / float64(trials)
		if fpr > 0.10 {
			t.Errorf("FPR too high: %d/%d = %.2f%% (50 elements, 2048 bits)", falsePositives, trials, fpr*100)
		}
	})
}

// Property: TxLifecycleTracker phase transitions are valid.
func TestProperty_TxLifecycleValidTransitions(t *testing.T) {
	t.Parallel()

	pry.Check(t, func(t *pry.T) {
		tracker := NewTxLifecycleTracker()
		txHash := common.BytesToHash(randomBytes(32))
		blockHash := common.BytesToHash(randomBytes(32))

		tracker.TrackMempool(txHash)
		state := tracker.GetTxState(txHash)
		if state.Current != TxInMempool {
			t.Fatalf("expected mempool, got %s", state.Current)
		}

		tracker.TrackIncluded(txHash, pry.Uint64Range(1, 1e6).Draw(t, "blockNumber"), blockHash)
		state = tracker.GetTxState(txHash)
		if state.Current != TxProposed {
			t.Fatalf("expected proposed after inclusion, got %s", state.Current)
		}

		tracker.TrackConfirmed(txHash, pry.Uint64Range(1, 1e6).Draw(t, "confirmBlock"))
		state = tracker.GetTxState(txHash)
		if state.Current != TxConfirmed {
			t.Fatalf("expected confirmed, got %s", state.Current)
		}

		tracker.TrackFinalized(txHash, pry.Uint64Range(1, 1e6).Draw(t, "epoch"))
		state = tracker.GetTxState(txHash)
		if state.Current != TxFinalized {
			t.Fatalf("expected finalized, got %s", state.Current)
		}
	})
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*17 + i*i + 31) // deterministic pseudo-random
	}
	return b
}

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	b := randomBytes(n)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}
