package eventsig

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pry "pgregory.net/rapid"
)

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

func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*17 + i*i + 31)
	}
	return b
}
