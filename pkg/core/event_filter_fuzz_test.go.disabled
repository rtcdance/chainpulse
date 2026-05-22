package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func FuzzParseEventFilter(f *testing.F) {
	// Fuzz the EventFilter.Validate method with various field combinations
	seeds := []struct {
		network string
		fromBlk uint64
		toBlk   uint64
		limit   int
		offset  int
		minVal  int64
		maxVal  int64
	}{
		{"ethereum", 0, 0, 10, 0, 0, 0},
		{"", 0, 0, 0, 0, 0, 0},
		{"polygon", 100, 50, -1, -1, 100, 50},
		{"arbitrum", 1, 1000000, 1000000, 1000000, 0, 999999999},
	}
	for _, s := range seeds {
		f.Add(s.network, s.fromBlk, s.toBlk, s.limit, s.offset, s.minVal, s.maxVal)
	}

	f.Fuzz(func(t *testing.T, network string, fromBlk, toBlk uint64, limit, offset int, minVal, maxVal int64) {
		ef := &EventFilter{
			Network:   network,
			FromBlock: fromBlk,
			ToBlock:   toBlk,
			Limit:     limit,
			Offset:    offset,
			MinValue:  big.NewInt(minVal),
			MaxValue:  big.NewInt(maxVal),
		}

		// Validate should never panic
		_ = ef.Validate()

		// ToQuery should never panic
		_ = ef.ToQuery()

		// GetCacheKey should never panic
		_ = ef.GetCacheKey()
	})
}

func FuzzEventFilterBuild(f *testing.F) {
	seeds := []struct {
		network string
		addr    string
		limit   int
		offset  int
	}{
		{"ethereum", "0x1234567890abcdef1234567890abcdef12345678", 10, 0},
		{"", "", 0, 0},
		{"polygon", "0x0000000000000000000000000000000000000000", -1, -1},
	}
	for _, s := range seeds {
		f.Add(s.network, s.addr, s.limit, s.offset)
	}

	f.Fuzz(func(t *testing.T, network, addr string, limit, offset int) {
		builder := NewEventFilterBuilder()
		builder.Network(network)

		if common.IsHexAddress(addr) {
			builder.ContractAddress(common.HexToAddress(addr))
		}

		builder.Pagination(limit, offset)

		// Build should never panic
		_, _ = builder.Build()
	})
}
