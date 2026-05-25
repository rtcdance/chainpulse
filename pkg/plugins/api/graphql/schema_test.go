package graphql

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

func makeEvent(chainID string, contractAddr string, eventName string, status core.EventStatus, blockNumber uint64, logIndex uint64, removed bool, blockTimestamp int64) *core.BlockchainEvent {
	return &core.BlockchainEvent{
		ChainID:         chainID,
		ContractAddress: common.HexToAddress(contractAddr),
		EventName:       eventName,
		Status:          status,
		BlockNumber:     blockNumber,
		LogIndex:        logIndex,
		Removed:         removed,
		BlockTimestamp:  blockTimestamp,
	}
}

func TestApplyEventFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		events   []*core.BlockchainEvent
		filter   map[string]any
		wantIDs  []string
	}{
		{
			name: "filter by status matches one",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0xB", "Approve", core.EventStatusPending, 101, 0, false, 1001),
			},
			filter: map[string]any{"status": "confirmed"},
			wantIDs: []string{"confirmed"},
		},
		{
			name: "filter by chain_id matches one",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("137", "0xB", "Approve", core.EventStatusConfirmed, 101, 0, false, 1001),
			},
			filter: map[string]any{"chainId": "137"},
			wantIDs: []string{"137"},
		},
		{
			name: "filter by contract_address matches one",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0x1111111111111111111111111111111111111111", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0x2222222222222222222222222222222222222222", "Approve", core.EventStatusConfirmed, 101, 0, false, 1001),
			},
			filter: map[string]any{"contractAddress": common.HexToAddress("0x2222222222222222222222222222222222222222").Hex()},
			wantIDs: []string{"matched_addr"},
		},
		{
			name: "filter by event_name matches one",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0xA", "Swap", core.EventStatusConfirmed, 101, 0, false, 1001),
			},
			filter: map[string]any{"eventName": "Swap"},
			wantIDs: []string{"Swap"},
		},
		{
			name: "filter by blockNumberGte",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 200, 0, false, 1001),
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 300, 0, false, 1002),
			},
			filter: map[string]any{"blockNumberGte": 200},
			wantIDs: []string{"gte200", "gte300"},
		},
		{
			name: "filter by blockNumberLte",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 200, 0, false, 1001),
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 300, 0, false, 1002),
			},
			filter: map[string]any{"blockNumberLte": 200},
			wantIDs: []string{"lte100", "lte200"},
		},
		{
			name: "filter by removed flag",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, true, 1000),
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 101, 0, false, 1001),
			},
			filter: map[string]any{"removed": true},
			wantIDs: []string{"removed"},
		},
		{
			name: "combined filters match only one",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0xA", "Transfer", core.EventStatusPending, 100, 0, false, 1001),
				makeEvent("137", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1002),
			},
			filter: map[string]any{"chainId": "1", "status": "confirmed"},
			wantIDs: []string{"combined"},
		},
		{
			name: "no filter returns all",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
				makeEvent("1", "0xB", "Approve", core.EventStatusConfirmed, 101, 0, false, 1001),
			},
			filter:  map[string]any{},
			wantIDs: []string{"all_0", "all_1"},
		},
		{
			name: "nil filter raw returns all",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
			},
			filter:  nil,
			wantIDs: []string{"nil_filter"},
		},
		{
			name: "empty string filter criteria does not filter out",
			events: []*core.BlockchainEvent{
				makeEvent("1", "0xA", "Transfer", core.EventStatusConfirmed, 100, 0, false, 1000),
			},
			filter:  map[string]any{"chainId": "", "contractAddress": "", "eventName": "", "status": ""},
			wantIDs: []string{"empty_strings"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyEventFilter(tt.events, tt.filter)
			if len(result) != len(tt.wantIDs) {
				t.Errorf("expected %d events, got %d", len(tt.wantIDs), len(result))
			}
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	t.Parallel()

	event := &core.BlockchainEvent{
		ChainID:         "1",
		ContractAddress: common.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
		EventName:       "Transfer",
		Status:          core.EventStatusConfirmed,
		BlockNumber:     150,
		Removed:         false,
	}

	tests := []struct {
		name   string
		filter map[string]any
		want   bool
	}{
		{
			name:   "exact chain_id match",
			filter: map[string]any{"chainId": "1"},
			want:   true,
		},
		{
			name:   "chain_id mismatch",
			filter: map[string]any{"chainId": "137"},
			want:   false,
		},
		{
			name:   "exact contract address match",
			filter: map[string]any{"contractAddress": "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"},
			want:   true,
		},
		{
			name:   "contract address mismatch",
			filter: map[string]any{"contractAddress": "0x0000000000000000000000000000000000000000"},
			want:   false,
		},
		{
			name:   "exact event name match",
			filter: map[string]any{"eventName": "Transfer"},
			want:   true,
		},
		{
			name:   "event name mismatch",
			filter: map[string]any{"eventName": "Swap"},
			want:   false,
		},
		{
			name:   "exact status match",
			filter: map[string]any{"status": "confirmed"},
			want:   true,
		},
		{
			name:   "status mismatch",
			filter: map[string]any{"status": "pending"},
			want:   false,
		},
		{
			name:   "blockNumberGte passes when above threshold",
			filter: map[string]any{"blockNumberGte": 150},
			want:   true,
		},
		{
			name:   "blockNumberGte fails when below threshold",
			filter: map[string]any{"blockNumberGte": 200},
			want:   false,
		},
		{
			name:   "blockNumberLte passes when below threshold",
			filter: map[string]any{"blockNumberLte": 150},
			want:   true,
		},
		{
			name:   "blockNumberLte fails when above threshold",
			filter: map[string]any{"blockNumberLte": 100},
			want:   false,
		},
		{
			name:   "removed matches false",
			filter: map[string]any{"removed": false},
			want:   true,
		},
		{
			name:   "removed mismatch",
			filter: map[string]any{"removed": true},
			want:   false,
		},
		{
			name:   "empty filter matches all",
			filter: map[string]any{},
			want:   true,
		},
		{
			name:   "empty string criteria do not filter",
			filter: map[string]any{"chainId": "", "eventName": ""},
			want:   true,
		},
		{
			name: "all criteria match",
			filter: map[string]any{
				"chainId":         "1",
				"contractAddress": "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
				"eventName":       "Transfer",
				"status":          "confirmed",
				"blockNumberGte":  100,
				"blockNumberLte":  200,
				"removed":         false,
			},
			want: true,
		},
		{
			name: "one mismatch in combined filter",
			filter: map[string]any{
				"chainId":         "1",
				"contractAddress": "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
				"eventName":       "Swap",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := matchesFilter(event, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyEventSort(t *testing.T) {
	t.Parallel()

	events := []*core.BlockchainEvent{
		makeEvent("1", "0xA", "Zeta", core.EventStatusConfirmed, 300, 2, false, 3000),
		makeEvent("1", "0xA", "Alpha", core.EventStatusConfirmed, 100, 0, false, 1000),
		makeEvent("1", "0xA", "Beta", core.EventStatusConfirmed, 200, 1, false, 2000),
	}

	tests := []struct {
		name     string
		sort     map[string]any
		wantIDs  []uint64
	}{
		{
			name:    "sort by blockNumber ascending",
			sort:    map[string]any{"field": "blockNumber", "order": "ASC"},
			wantIDs: []uint64{100, 200, 300},
		},
		{
			name:    "sort by blockNumber descending (default)",
			sort:    map[string]any{"field": "blockNumber"},
			wantIDs: []uint64{300, 200, 100},
		},
		{
			name:    "sort by blockNumber descending (explicit)",
			sort:    map[string]any{"field": "blockNumber", "order": "DESC"},
			wantIDs: []uint64{300, 200, 100},
		},
		{
			name:    "sort by blockTimestamp ascending",
			sort:    map[string]any{"field": "blockTimestamp", "order": "ASC"},
			wantIDs: []uint64{100, 200, 300},
		},
		{
			name:    "sort by blockTimestamp descending",
			sort:    map[string]any{"field": "blockTimestamp", "order": "DESC"},
			wantIDs: []uint64{300, 200, 100},
		},
		{
			name:    "sort by logIndex ascending",
			sort:    map[string]any{"field": "logIndex", "order": "ASC"},
			wantIDs: []uint64{100, 200, 300},
		},
		{
			name:    "sort by logIndex descending",
			sort:    map[string]any{"field": "logIndex", "order": "DESC"},
			wantIDs: []uint64{300, 200, 100},
		},
		{
			name:    "sort by eventName ascending",
			sort:    map[string]any{"field": "eventName", "order": "ASC"},
			wantIDs: []uint64{100, 200, 300},
		},
		{
			name:    "sort by eventName descending",
			sort:    map[string]any{"field": "eventName", "order": "DESC"},
			wantIDs: []uint64{300, 200, 100},
		},
		{
			name:    "nil sort input returns original order",
			sort:    nil,
			wantIDs: []uint64{300, 100, 200},
		},
		{
			name:    "empty field returns original order",
			sort:    map[string]any{"field": "", "order": "ASC"},
			wantIDs: []uint64{300, 100, 200},
		},
		{
			name:    "unknown field defaults to blockNumber descending",
			sort:    map[string]any{"field": "unknownField", "order": "ASC"},
			wantIDs: []uint64{100, 200, 300},
		},
		{
			name:    "case insensitive ASC",
			sort:    map[string]any{"field": "blockNumber", "order": "asc"},
			wantIDs: []uint64{100, 200, 300},
		},
		{
			name:    "case insensitive DESC",
			sort:    map[string]any{"field": "blockNumber", "order": "desc"},
			wantIDs: []uint64{300, 200, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := applyEventSort(events, tt.sort)
			if len(result) != len(tt.wantIDs) {
				t.Fatalf("expected %d events, got %d", len(tt.wantIDs), len(result))
			}
			for i, wantBN := range tt.wantIDs {
				if result[i].BlockNumber != wantBN {
					t.Errorf("position %d: expected blockNumber %d, got %d", i, wantBN, result[i].BlockNumber)
				}
			}
		})
	}
}

func TestMapToSubscriptionPayload(t *testing.T) {
	t.Parallel()

	payload := &EventSubscriptionPayload{
		Type:            "created",
		EventID:         "evt-test-001",
		ChainID:         "1",
		ContractAddress: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		EventName:       "Transfer",
		BlockNumber:     123456,
		Timestamp:       1710000000,
	}

	result := mapToSubscriptionPayload(payload)

	tests := []struct {
		field string
		want  any
	}{
		{"type", "created"},
		{"eventId", "evt-test-001"},
		{"chainId", "1"},
		{"contractAddress", "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"},
		{"eventName", "Transfer"},
		{"blockNumber", int64(123456)},
		{"timestamp", int64(1710000000)},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			t.Parallel()
			if result[tt.field] != tt.want {
				t.Errorf("field %q: got %v, want %v", tt.field, result[tt.field], tt.want)
			}
		})
	}
}

func TestMapToSubscriptionPayloadEmptyFields(t *testing.T) {
	t.Parallel()

	payload := &EventSubscriptionPayload{
		Type:      "failed",
		EventID:   "evt-fail",
		Timestamp: 0,
	}

	result := mapToSubscriptionPayload(payload)

	if result["type"] != "failed" {
		t.Errorf("type: got %v, want failed", result["type"])
	}
	if result["eventId"] != "evt-fail" {
		t.Errorf("eventId: got %v, want evt-fail", result["eventId"])
	}
	if result["chainId"] != "" {
		t.Errorf("chainId: got %v, want empty", result["chainId"])
	}
	if result["blockNumber"] != int64(0) {
		t.Errorf("blockNumber: got %v, want 0", result["blockNumber"])
	}
}