// Package eventsig provides Ethereum event signature computation, ABI encoding
// of indexed parameters, and a registry of known event signature hashes.
//
// These functions were originally defined in pkg/core and are re-exported
// there for backward compatibility.
package eventsig

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// EventSignature computes the keccak256 event signature hash for a Solidity event.
//
// In Solidity, event signatures are computed as:
//
//	keccak256("EventName(type1,type2,...)")
//
// The first topic (topic0) of a log entry is always this hash, which serves as the
// event's unique identifier. This is defined in the Ethereum Yellow Paper (Section 4.3.1).
//
// Example:
//
//	EventSignature("Transfer", []string{"address","address","uint256"})
//	  => common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
func EventSignature(eventName string, paramTypes ...string) common.Hash {
	parts := make([]string, len(paramTypes))
	copy(parts, paramTypes)
	sig := fmt.Sprintf("%s(%s)", eventName, strings.Join(parts, ","))
	return keccak256Hash([]byte(sig))
}

// EncodeIndexedParam encodes a parameter value into a 32-byte topic (common.Hash).
//
// Indexed event parameters are stored in topics[1..3] (maximum 3 indexed params).
// The encoding rules per the Solidity ABI spec:
//   - address: zero-left-padded to 32 bytes (left pad with 12 zero bytes)
//   - uint/int: big-endian padded to 32 bytes
//   - bool: 0x00...00 or 0x00...01
//   - bytes/string: keccak256 hash of the original value (indexed strings are NOT stored in plaintext!)
func EncodeIndexedParam(val any, solidityType string) (common.Hash, error) {
	switch solidityType {
	case "address":
		addr, ok := val.(common.Address)
		if !ok {
			return common.Hash{}, fmt.Errorf("expected common.Address, got %T", val)
		}
		return common.BytesToHash(addr.Bytes()), nil

	case "bool":
		b, ok := val.(bool)
		if !ok {
			return common.Hash{}, fmt.Errorf("expected bool, got %T", val)
		}
		if b {
			return common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), nil
		}
		return common.Hash{}, nil

	case "uint256", "uint":
		switch v := val.(type) {
		case *big.Int:
			return common.BigToHash(v), nil
		case uint64:
			return common.BigToHash(new(big.Int).SetUint64(v)), nil
		case int:
			return common.BigToHash(big.NewInt(int64(v))), nil
		default:
			return common.Hash{}, fmt.Errorf("unsupported uint type: %T", val)
		}

	case "bytes32":
		h, ok := val.(common.Hash)
		if !ok {
			return common.Hash{}, fmt.Errorf("expected common.Hash, got %T", val)
		}
		return h, nil

	case "string":
		s, ok := val.(string)
		if !ok {
			return common.Hash{}, fmt.Errorf("expected string, got %T", val)
		}
		return keccak256Hash([]byte(s)), nil

	default:
		// For other types, try ABI encoding then hash (used for tuples, arrays)
		typed, err := abi.NewType(solidityType, "", nil)
		if err != nil {
			return common.Hash{}, fmt.Errorf("unsupported indexed type: %s", solidityType)
		}
		encoded, err := abi.Arguments{{Type: typed}}.Pack(val)
		if err != nil {
			return common.Hash{}, fmt.Errorf("encode failed for type %s: %w", solidityType, err)
		}
		return keccak256Hash(encoded), nil
	}
}

// DecodeIndexedParam decodes a 32-byte topic back into a Go value.
//
// This is the inverse of EncodeIndexedParam. Note that for indexed strings/bytes,
// the original value cannot be recovered (only its keccak256 hash is stored).
// For those types, DecodeIndexedParam returns the hex-encoded hash.
func DecodeIndexedParam(topic common.Hash, solidityType string) (any, error) {
	switch solidityType {
	case "address":
		return common.BytesToAddress(topic.Bytes()[12:]), nil

	case "bool":
		for i := 0; i < 31; i++ {
			if topic[i] != 0 {
				return nil, fmt.Errorf("invalid bool encoding in topic: %s", topic.Hex())
			}
		}
		return topic[31] == 1, nil

	case "uint256", "uint":
		return new(big.Int).SetBytes(topic.Bytes()), nil

	case "bytes32":
		return topic, nil

	case "string", "bytes":
		// Indexed strings/bytes only store keccak256(value), original is unrecoverable
		return topic.Hex(), fmt.Errorf("indexed %s: only keccak256 hash available, original value unrecoverable", solidityType)

	default:
		return topic.Hex(), fmt.Errorf("unsupported indexed type: %s", solidityType)
	}
}

// Topic0ForEvent computes topic0 for a named event with given ABI types.
//
// This is the canonical way to pre-compute event signature hashes for use in
// eth_getLogs filter queries.
func Topic0ForEvent(eventName string, paramTypes ...string) common.Hash {
	return EventSignature(eventName, paramTypes...)
}

// keccak256Hash computes Keccak-256 (NOT SHA3-256) hash of input data.
// Ethereum uses Keccak-256, which differs from the FIPS-202 SHA3 standard.
func keccak256Hash(data []byte) common.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	return common.BytesToHash(hasher.Sum(nil))
}

// ensure sha256 import is used (for documentation accuracy vs sha3)
var _ = sha256.New
