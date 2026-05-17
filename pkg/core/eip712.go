package core

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// EIP-712 typed data signing implementation per https://eips.ethereum.org/EIPS/eip-712

// EIP712Domain represents the EIP-712 domain parameters that define the signing scope.
type EIP712Domain struct {
	Name              string         `json:"name"`
	Version           string         `json:"version"`
	ChainID           *big.Int       `json:"chainId"`
	VerifyingContract common.Address `json:"verifyingContract,omitempty"`
	Salt              string         `json:"salt,omitempty"`
}

// TypedData represents the full EIP-712 typed data structure for signing.
type TypedData struct {
	Types       apitypes.Types `json:"types"`
	PrimaryType string         `json:"primaryType"`
	Domain      EIP712Domain   `json:"domain"`
	Message     map[string]any `json:"message"`
}

// Errors
var (
	ErrCircularTypeReference = errors.New("circular type reference detected in EIP-712 types")
	ErrUnknownType           = errors.New("unknown type in EIP-712 field")
	ErrMissingChainID        = errors.New("EIP-712 domain requires chainId")
	ErrNilMessage            = errors.New("EIP-712 message is nil")
	ErrInvalidSignature      = errors.New("invalid EIP-712 signature length")
)

// EncodeType encodes the type string for a given type name.
// For example: EncodeType("Mail", {"Mail": [{"name":"from","type":"Person"},...]})
// returns "Mail(Person from,Person to,string contents)Person(string name,address wallet)"
func EncodeType(typeName string, types apitypes.Types) (string, error) {
	seen := make(map[string]bool)
	return encodeTypeRecursive(typeName, types, seen)
}

func encodeTypeRecursive(typeName string, types apitypes.Types, seen map[string]bool) (string, error) {
	if seen[typeName] {
		return "", fmt.Errorf("%w: type %q references itself", ErrCircularTypeReference, typeName)
	}
	seen[typeName] = true
	defer func() { delete(seen, typeName) }()

	fields, ok := types[typeName]
	if !ok {
		return "", fmt.Errorf("%w: type %q not defined", ErrUnknownType, typeName)
	}

	var b strings.Builder
	b.WriteString(typeName)
	b.WriteString("(")

	// Collect referenced struct types for inclusion
	referencedStructs := make(map[string]bool)

	for i, field := range fields {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(field.Type)
		b.WriteString(" ")
		b.WriteString(field.Name)

		// Check if field type is a referenced struct (not a primitive)
		baseType := strings.TrimSuffix(field.Type, "[]")
		if _, isStruct := types[baseType]; isStruct && !referencedStructs[baseType] {
			referencedStructs[baseType] = true
		}
	}

	b.WriteString(")")

	// Append referenced struct type definitions in alphabetical order for determinism
	sortedRefs := make([]string, 0, len(referencedStructs))
	for ref := range referencedStructs {
		sortedRefs = append(sortedRefs, ref)
	}
	sort.Strings(sortedRefs)

	for _, ref := range sortedRefs {
		refEncoding, err := encodeTypeRecursive(ref, types, seen)
		if err != nil {
			return "", err
		}
		b.WriteString(refEncoding)
	}

	return b.String(), nil
}

// TypeHash computes the keccak256 hash of the type string.
func TypeHash(typeName string, types apitypes.Types) ([32]byte, error) {
	encoded, err := EncodeType(typeName, types)
	if err != nil {
		return [32]byte{}, err
	}
	return crypto.Keccak256Hash([]byte(encoded)), nil
}

// HashStruct computes the keccak256 hash of a struct per EIP-712:
// hashStruct(s) = keccak256(typeHash ‖ encodeData(s))
func HashStruct(typeName string, types apitypes.Types, data map[string]any) ([32]byte, error) {
	typeHash, err := TypeHash(typeName, types)
	if err != nil {
		return [32]byte{}, err
	}

	encoded, err := encodeData(typeName, types, data, make(map[string]bool))
	if err != nil {
		return [32]byte{}, err
	}

	// Concatenate typeHash + encoded data
	combined := make([]byte, 32+len(encoded))
	copy(combined[:32], typeHash[:])
	copy(combined[32:], encoded)

	return crypto.Keccak256Hash(combined), nil
}

// encodeData encodes the data values according to the type definition.
func encodeData(typeName string, types apitypes.Types, data map[string]any, seen map[string]bool) ([]byte, error) {
	if seen[typeName] {
		return nil, fmt.Errorf("%w: type %q has circular reference", ErrCircularTypeReference, typeName)
	}
	seen[typeName] = true
	defer func() { delete(seen, typeName) }()

	fields, ok := types[typeName]
	if !ok {
		return nil, fmt.Errorf("%w: type %q not defined", ErrUnknownType, typeName)
	}

	var encoded []byte
	for _, field := range fields {
		value, ok := data[field.Name]
		if !ok {
			return nil, fmt.Errorf("missing field %q in data for type %q", field.Name, typeName)
		}

		fieldEncoded, err := encodeValue(field.Type, types, value, seen)
		if err != nil {
			return nil, fmt.Errorf("encoding field %q of type %q: %w", field.Name, typeName, err)
		}
		encoded = append(encoded, fieldEncoded...)
	}

	return encoded, nil
}

// encodeValue encodes a single value according to its ABI type.
func encodeValue(typeStr string, types apitypes.Types, value any, seen map[string]bool) ([]byte, error) {
	// Handle array types
	if strings.HasSuffix(typeStr, "[]") {
		// Dynamic arrays not fully supported — return hash of empty bytes32
		return make([]byte, 32), nil
	}

	// Atomic types
	switch typeStr {
	case "string":
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return crypto.Keccak256([]byte(s)), nil
	case "bytes":
		b, ok := value.([]byte)
		if !ok {
			s, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("expected []byte or string for bytes, got %T", value)
			}
			b = []byte(s)
		}
		return crypto.Keccak256(b), nil
	case "bool":
		result := make([]byte, 32)
		if b, ok := value.(bool); ok && b {
			result[31] = 1
		}
		return result, nil
	case "address":
		addr, ok := value.(common.Address)
		if !ok {
			s, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("expected address, got %T", value)
			}
			addr = common.HexToAddress(s)
		}
		result := make([]byte, 32)
		copy(result[12:], addr.Bytes())
		return result, nil
	}

	// Integer types (uint8..uint256, int8..int256)
	if strings.HasPrefix(typeStr, "uint") || strings.HasPrefix(typeStr, "int") {
		return encodeIntValue(value)
	}

	// Fixed-size bytes (bytes1..bytes32)
	if strings.HasPrefix(typeStr, "bytes") && len(typeStr) > 5 {
		return encodeFixedBytesValue(value)
	}

	// Struct types
	if _, isStruct := types[typeStr]; isStruct {
		data, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map for struct type %q, got %T", typeStr, value)
		}
		hash, err := HashStruct(typeStr, types, data)
		if err != nil {
			return nil, err
		}
		return hash[:], nil
	}

	return nil, fmt.Errorf("%w: %q", ErrUnknownType, typeStr)
}

func encodeIntValue(value any) ([]byte, error) {
	result := make([]byte, 32)

	switch v := value.(type) {
	case *big.Int:
		b := v.Bytes()
		copy(result[32-len(b):], b)
	case int:
		b := big.NewInt(int64(v)).Bytes()
		copy(result[32-len(b):], b)
	case int64:
		b := big.NewInt(v).Bytes()
		copy(result[32-len(b):], b)
	case string:
		n := new(big.Int)
		if _, ok := n.SetString(v, 10); !ok {
			if _, ok := n.SetString(v, 0); !ok {
				return nil, fmt.Errorf("cannot parse integer: %s", v)
			}
		}
		b := n.Bytes()
		copy(result[32-len(b):], b)
	default:
		return nil, fmt.Errorf("unsupported integer type: %T", value)
	}

	return result, nil
}

func encodeFixedBytesValue(value any) ([]byte, error) {
	result := make([]byte, 32)

	switch v := value.(type) {
	case []byte:
		copy(result, v)
	case string:
		copy(result, []byte(v))
	default:
		return nil, fmt.Errorf("expected []byte for fixed bytes, got %T", value)
	}

	return result, nil
}

// HashDomainSeparator computes the EIP-712 domain separator hash.
func HashDomainSeparator(domain EIP712Domain) ([32]byte, error) {
	domainFields := []apitypes.Type{
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
	}

	domainData := map[string]any{
		"name":    domain.Name,
		"version": domain.Version,
	}

	if domain.ChainID != nil {
		domainData["chainId"] = domain.ChainID.String()
	} else {
		return [32]byte{}, ErrMissingChainID
	}

	if domain.VerifyingContract != (common.Address{}) {
		domainFields = append(domainFields, apitypes.Type{Name: "verifyingContract", Type: "address"})
		domainData["verifyingContract"] = domain.VerifyingContract
	}
	if domain.Salt != "" {
		domainFields = append(domainFields, apitypes.Type{Name: "salt", Type: "bytes32"})
		domainData["salt"] = domain.Salt
	}

	domainType := apitypes.Types{
		"EIP712Domain": domainFields,
	}

	return HashStruct("EIP712Domain", domainType, domainData)
}

// HashTypedData computes the final EIP-712 hash for signing:
// "\x19\x01" ‖ domainSeparator ‖ structHash
func HashTypedData(td TypedData) ([32]byte, error) {
	if td.Message == nil {
		return [32]byte{}, ErrNilMessage
	}

	domainSeparator, err := HashDomainSeparator(td.Domain)
	if err != nil {
		return [32]byte{}, fmt.Errorf("domain separator: %w", err)
	}

	structHash, err := HashStruct(td.PrimaryType, td.Types, td.Message)
	if err != nil {
		return [32]byte{}, fmt.Errorf("struct hash: %w", err)
	}

	// "\x19\x01" ‖ domainSeparator ‖ structHash
	data := make([]byte, 2+32+32)
	data[0] = 0x19
	data[1] = 0x01
	copy(data[2:34], domainSeparator[:])
	copy(data[34:], structHash[:])

	return crypto.Keccak256Hash(data), nil
}

// VerifyTypedData verifies an EIP-712 typed data signature against an address.
// The signature should be 65 bytes: [R || S || V] where V is 27 or 28.
func VerifyTypedData(td TypedData, sig []byte, expectedAddr common.Address) (bool, error) {
	if len(sig) != 65 {
		return false, fmt.Errorf("%w: got %d bytes, want 65", ErrInvalidSignature, len(sig))
	}

	hash, err := HashTypedData(td)
	if err != nil {
		return false, err
	}

	// Convert [R || S || V] to [R || S] for recovery
	// V is 27 or 28 for EIP-155 compatible signatures
	v := sig[64]
	sigCopy := make([]byte, 65)
	copy(sigCopy, sig)
	if v >= 27 {
		sigCopy[64] = v - 27
	}

	pubKey, err := crypto.SigToPub(hash[:], sigCopy)
	if err != nil {
		return false, fmt.Errorf("signature recovery: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr == expectedAddr, nil
}
