package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func FuzzEncodeType(f *testing.F) {
	seeds := []struct {
		name string
	}{
		{"Mail"},
		{""},
		{"EIP712Domain"},
		{"a"},
		{"TypeWithNumbers123"},
		{"Type/With/Special:Chars!"},
	}
	for _, s := range seeds {
		f.Add(s.name)
	}

	f.Fuzz(func(t *testing.T, typeName string) {
		// Build a simple types map with the fuzzed type name
		types := apitypes.Types{
			typeName: {
				{Name: "value", Type: "uint256"},
			},
		}

		_, err := EncodeType(typeName, types)
		// Should never panic
		_ = err
	})
}

func FuzzHashStruct(f *testing.F) {
	seeds := []struct {
		typeName string
		val      int64
	}{
		{"Mail", 100},
		{"", 0},
		{"EIP712Domain", 1},
		{"Test", -1},
	}
	for _, s := range seeds {
		f.Add(s.typeName, s.val)
	}

	f.Fuzz(func(t *testing.T, typeName string, val int64) {
		types := apitypes.Types{
			typeName: {
				{Name: "value", Type: "uint256"},
			},
		}

		data := map[string]any{
			"value": val,
		}

		_, err := HashStruct(typeName, types, data)
		// Should never panic
		_ = err
	})
}
