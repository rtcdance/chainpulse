package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEventSignatureKnownHashes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		eventName  string
		paramTypes []string
		want       string
	}{
		{
			name:       "ERC20 Transfer",
			eventName:  "Transfer",
			paramTypes: []string{"address", "address", "uint256"},
			want:       "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		},
		{
			name:       "ERC20 Approval",
			eventName:  "Approval",
			paramTypes: []string{"address", "address", "uint256"},
			want:       "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
		},
		{
			name:       "ERC721 Transfer",
			eventName:  "Transfer",
			paramTypes: []string{"address", "address", "uint256"},
			want:       "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		},
		{
			name:       "OwnershipTransferred",
			eventName:  "OwnershipTransferred",
			paramTypes: []string{"address", "address"},
			want:       "0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EventSignature(tt.eventName, tt.paramTypes...)
			want := common.HexToHash(tt.want)
			if got != want {
				t.Errorf("EventSignature(%q, %v) = %s, want %s", tt.eventName, tt.paramTypes, got.Hex(), tt.want)
			}
		})
	}
}

func TestEncodeDecodeIndexedAddress(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	topic, err := EncodeIndexedParam(addr, "address")
	if err != nil {
		t.Fatalf("EncodeIndexedParam failed: %v", err)
	}

	// Verify: address should be right-padded with 12 zero bytes at the front
	expectedPrefix := "0x000000000000000000000000"
	if topic.Hex()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected left-padding with zeros, got %s", topic.Hex())
	}

	// Decode back
	decoded, err := DecodeIndexedParam(topic, "address")
	if err != nil {
		t.Fatalf("DecodeIndexedParam failed: %v", err)
	}

	decodedAddr, ok := decoded.(common.Address)
	if !ok {
		t.Fatalf("decoded value is not common.Address, got %T", decoded)
	}
	if decodedAddr != addr {
		t.Errorf("round-trip mismatch: got %s, want %s", decodedAddr.Hex(), addr.Hex())
	}
}

func TestEncodeDecodeIndexedUint256(t *testing.T) {
	t.Parallel()

	val := big.NewInt(1000000)
	topic, err := EncodeIndexedParam(val, "uint256")
	if err != nil {
		t.Fatalf("EncodeIndexedParam failed: %v", err)
	}

	// Decode back
	decoded, err := DecodeIndexedParam(topic, "uint256")
	if err != nil {
		t.Fatalf("DecodeIndexedParam failed: %v", err)
	}

	bigVal, ok := decoded.(*big.Int)
	if !ok {
		t.Fatalf("decoded value is not *big.Int, got %T", decoded)
	}
	if bigVal.Cmp(val) != 0 {
		t.Errorf("round-trip mismatch: got %s, want %s", bigVal.String(), val.String())
	}
}

func TestEncodeDecodeIndexedBool(t *testing.T) {
	t.Parallel()

	for _, b := range []bool{true, false} {
		topic, err := EncodeIndexedParam(b, "bool")
		if err != nil {
			t.Fatalf("EncodeIndexedParam(%v) failed: %v", b, err)
		}

		decoded, err := DecodeIndexedParam(topic, "bool")
		if err != nil {
			t.Fatalf("DecodeIndexedParam failed: %v", err)
		}

		if decoded.(bool) != b {
			t.Errorf("round-trip mismatch: got %v, want %v", decoded, b)
		}
	}
}

func TestEncodeIndexedString(t *testing.T) {
	t.Parallel()

	// Indexed strings are hashed, NOT stored in plaintext
	str := "hello from chainpulse"
	topic, err := EncodeIndexedParam(str, "string")
	if err != nil {
		t.Fatalf("EncodeIndexedParam failed: %v", err)
	}

	// The topic should be keccak256("hello from chainpulse")
	expectedHash := keccak256Hash([]byte(str))
	if topic != expectedHash {
		t.Errorf("indexed string hash mismatch:\ngot:  %s\nwant: %s", topic.Hex(), expectedHash.Hex())
	}

	// Verify it's NOT the original string
	originalStr := common.BytesToHash([]byte(str))
	if topic == originalStr {
		t.Error("indexed string should NOT be stored as plaintext!")
	}
}

func TestEncodeIndexedTypeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  any
		typ  string
	}{
		{"address with wrong type", "not an address", "address"},
		{"bool with wrong type", 42, "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeIndexedParam(tt.val, tt.typ)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestTopic0ForEvent(t *testing.T) {
	t.Parallel()

	// Verify Topic0ForEvent matches EventSignature
	sig := Topic0ForEvent("Transfer", "address", "address", "uint256")
	expected := EventSignature("Transfer", "address", "address", "uint256")
	if sig != expected {
		t.Errorf("Topic0ForEvent != EventSignature: %s vs %s", sig.Hex(), expected.Hex())
	}

	// Verify it matches the canonical Transfer event hash
	canonical := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if sig != canonical {
		t.Errorf("Transfer signature mismatch: got %s, want %s", sig.Hex(), canonical.Hex())
	}
}
