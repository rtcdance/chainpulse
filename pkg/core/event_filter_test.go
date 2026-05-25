package core

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestEventFilter_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		filter  EventFilter
		wantErr bool
		errMsg  string
	}{
		{
			"valid_minimal",
			EventFilter{Network: "ethereum"},
			false, "",
		},
		{
			"empty_network",
			EventFilter{},
			true, "network is required",
		},
		{
			"invalid_network_chars",
			EventFilter{Network: "eth; DROP TABLE"},
			true, "invalid characters",
		},
		{
			"invalid_status_chars",
			EventFilter{Network: "ethereum", Status: []string{"bad;status"}},
			true, "invalid characters",
		},
		{
			"from_block_gt_to_block",
			EventFilter{Network: "ethereum", FromBlock: 100, ToBlock: 50},
			true, "from_block must be <= to_block",
		},
		{
			"from_ts_gt_to_ts",
			EventFilter{Network: "ethereum", FromTimestamp: 200, ToTimestamp: 100},
			true, "from_timestamp must be <= to_timestamp",
		},
		{
			"negative_limit",
			EventFilter{Network: "ethereum", Limit: -1},
			true, "limit must be non-negative",
		},
		{
			"negative_offset",
			EventFilter{Network: "ethereum", Offset: -5},
			true, "offset must be non-negative",
		},
		{
			"min_value_gt_max_value",
			EventFilter{Network: "ethereum", MinValue: big.NewInt(200), MaxValue: big.NewInt(100)},
			true, "min_value must be <= max_value",
		},
		{
			"zero_to_block_ok",
			EventFilter{Network: "ethereum", FromBlock: 100, ToBlock: 0},
			false, "",
		},
		{
			"zero_to_ts_ok",
			EventFilter{Network: "ethereum", FromTimestamp: 100, ToTimestamp: 0},
			false, "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.filter.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && err != nil && !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error = %q, want containing %q", err.Error(), tc.errMsg)
			}
		})
	}
}

func TestEventFilter_ToQuery(t *testing.T) {
	t.Parallel()

	t.Run("full_filter", func(t *testing.T) {
		t.Parallel()
		addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
		sig := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

		ef := EventFilter{
			Network:         "ethereum",
			ContractAddress: []common.Address{addr},
			EventSignature:  []common.Hash{sig},
			FromBlock:       100,
			ToBlock:         200,
			FromTimestamp:   1600000000,
			ToTimestamp:     1700000000,
			Status:          []string{"confirmed"},
			MinValue:        big.NewInt(1000),
			MaxValue:        big.NewInt(9999),
			Limit:           50,
			Offset:          10,
		}

		query, args := ef.ToQuery()

		if !strings.Contains(query, "WHERE") {
			t.Error("expected WHERE clause")
		}
		if !strings.Contains(query, "LIMIT") {
			t.Error("expected LIMIT clause")
		}
		if !strings.Contains(query, "OFFSET") {
			t.Error("expected OFFSET clause")
		}
		if !strings.Contains(query, "ORDER BY") {
			t.Error("expected ORDER BY clause")
		}

		if len(args) == 0 {
			t.Error("expected non-empty args")
		}
	})

	t.Run("invalid_network_safety", func(t *testing.T) {
		t.Parallel()
		ef := EventFilter{Network: "eth;DROP"}
		query, args := ef.ToQuery()
		if !strings.Contains(query, "WARNING") {
			t.Errorf("expected safety warning, got: %q", query)
		}
		if args != nil {
			t.Error("expected nil args for unsafe network")
		}
	})

	t.Run("minimal_filter", func(t *testing.T) {
		t.Parallel()
		ef := EventFilter{Network: "ethereum", Limit: 0, Offset: 0}
		query, _ := ef.ToQuery()
		if strings.Contains(query, "LIMIT") {
			t.Error("expected no LIMIT when Limit=0")
		}
	})

	t.Run("multiple_addresses", func(t *testing.T) {
		t.Parallel()
		addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
		addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
		ef := EventFilter{
			Network:         "ethereum",
			ContractAddress: []common.Address{addr1, addr2},
		}
		query, _ := ef.ToQuery()
		if !strings.Contains(query, "contract_address IN") {
			t.Error("expected contract_address IN clause")
		}
	})

	t.Run("no_conditions", func(t *testing.T) {
		t.Parallel()
		ef := EventFilter{Network: ""}
		query, _ := ef.ToQuery()
		if strings.Contains(query, "WHERE") {
			t.Error("expected no WHERE clause")
		}
	})
}

func TestEventFilter_GetCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		ef := EventFilter{Network: "ethereum"}
		key := ef.GetCacheKey()
		if key != "events:ethereum" {
			t.Errorf("key = %q, want events:ethereum", key)
		}
	})

	t.Run("with_address", func(t *testing.T) {
		t.Parallel()
		addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
		ef := EventFilter{
			Network:         "ethereum",
			ContractAddress: []common.Address{addr},
		}
		key := ef.GetCacheKey()
		if !strings.Contains(key, ":addr:") {
			t.Errorf("expected :addr: in key: %q", key)
		}
	})

	t.Run("with_block_range", func(t *testing.T) {
		t.Parallel()
		ef := EventFilter{
			Network:   "ethereum",
			FromBlock: 100,
			ToBlock:   200,
		}
		key := ef.GetCacheKey()
		if !strings.Contains(key, ":from:100") {
			t.Errorf("expected :from:100 in key: %q", key)
		}
		if !strings.Contains(key, ":to:200") {
			t.Errorf("expected :to:200 in key: %q", key)
		}
	})

	t.Run("full_key", func(t *testing.T) {
		t.Parallel()
		addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
		sig := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
		ef := EventFilter{
			Network:         "ethereum",
			ContractAddress: []common.Address{addr},
			EventSignature:  []common.Hash{sig},
			FromBlock:       100,
			ToBlock:         200,
			Limit:           50,
			Offset:          10,
		}
		key := ef.GetCacheKey()
		if key == "events:ethereum" {
			t.Error("expected complex key")
		}
	})
}

func TestNewEventFilterBuilder(t *testing.T) {
	t.Parallel()

	b := NewEventFilterBuilder()
	if b.filter.Limit != 100 {
		t.Errorf("default Limit = %d, want 100", b.filter.Limit)
	}
}

func TestEventFilterBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f, err := NewEventFilterBuilder().Network("ethereum").Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if f.Network != "ethereum" {
			t.Errorf("Network = %q", f.Network)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		_, err := NewEventFilterBuilder().Build()
		if err == nil {
			t.Error("expected validation error")
		}
	})
}

func TestEventFilterBuilder_AllMethods(t *testing.T) {
	t.Parallel()

	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	sig1 := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	sig2 := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	b := NewEventFilterBuilder().
		Network("polygon").
		ContractAddress(addr1, addr2).
		EventSignature(sig1, sig2).
		Topics([]common.Hash{sig1}, []common.Hash{sig2}).
		BlockRange(100, 200).
		TimeRange(time.Unix(1600000000, 0), time.Unix(1700000000, 0)).
		Status("confirmed", "finalized").
		ValueRange(big.NewInt(100), big.NewInt(1000)).
		Pagination(50, 10)

	f, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Network != "polygon" {
		t.Errorf("Network = %q", f.Network)
	}
	if len(f.ContractAddress) != 2 {
		t.Errorf("ContractAddress len = %d", len(f.ContractAddress))
	}
	if len(f.EventSignature) != 2 {
		t.Errorf("EventSignature len = %d", len(f.EventSignature))
	}
	if len(f.Topics) != 2 {
		t.Errorf("Topics len = %d", len(f.Topics))
	}
	if f.FromBlock != 100 || f.ToBlock != 200 {
		t.Errorf("BlockRange = %d-%d", f.FromBlock, f.ToBlock)
	}
	if f.FromTimestamp != 1600000000 || f.ToTimestamp != 1700000000 {
		t.Errorf("TimeRange = %d-%d", f.FromTimestamp, f.ToTimestamp)
	}
	if len(f.Status) != 2 {
		t.Errorf("Status len = %d", len(f.Status))
	}
	if f.MinValue.Cmp(big.NewInt(100)) != 0 || f.MaxValue.Cmp(big.NewInt(1000)) != 0 {
		t.Error("ValueRange mismatch")
	}
	if f.Limit != 50 || f.Offset != 10 {
		t.Errorf("Pagination = %d-%d", f.Limit, f.Offset)
	}
}
