package api

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestParseIntParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		queryString  string
		paramName    string
		defaultValue int
		want         int
	}{
		{"returns_default_when_param_missing", "", "limit", 1000, 1000},
		{"returns_default_when_param_empty", "limit=", "limit", 1000, 1000},
		{"parses_positive_int", "limit=50", "limit", 1000, 50},
		{"parses_zero", "limit=0", "limit", 1000, 0},
		{"returns_default_on_invalid", "limit=abc", "limit", 1000, 1000},
		{"returns_default_on_negative", "limit=-5", "limit", 1000, -5},
		{"parses_large_value", "limit=99999", "limit", 1000, 99999},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			got := parseIntParam(req, tt.paramName, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseIntParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInt64Param(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		queryString  string
		paramName    string
		defaultValue int64
		want         int64
	}{
		{"returns_default_when_param_missing", "", "start_time", int64(0), 0},
		{"returns_default_when_param_empty", "start_time=", "start_time", int64(0), 0},
		{"parses_positive_int64", "start_time=1700000000", "start_time", int64(0), 1700000000},
		{"parses_zero", "start_time=0", "start_time", int64(100), 0},
		{"returns_default_on_invalid", "start_time=abc", "start_time", int64(0), 0},
		{"parses_negative", "end_time=-100", "end_time", int64(0), -100},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			got := parseInt64Param(req, tt.paramName, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseInt64Param() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseExportFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		queryString string
		want        exportFilter
	}{
		{
			name:        "default_values",
			queryString: "",
			want:        exportFilter{Format: "json", Limit: 1000, Offset: 0},
		},
		{
			name:        "format_csv",
			queryString: "format=csv",
			want:        exportFilter{Format: "csv", Limit: 1000, Offset: 0},
		},
		{
			name:        "format_uppercase_normalized",
			queryString: "format=CSV",
			want:        exportFilter{Format: "csv", Limit: 1000, Offset: 0},
		},
		{
			name:        "custom_limit_offset",
			queryString: "limit=50&offset=10",
			want:        exportFilter{Format: "json", Limit: 50, Offset: 10},
		},
		{
			name:        "chain_id_and_event_name",
			queryString: "chainId=1&eventName=Transfer",
			want:        exportFilter{Format: "json", Limit: 1000, Offset: 0, ChainID: "1", EventName: "Transfer"},
		},
		{
			name:        "chain_id_with_whitespace_trimmed",
			queryString: "chainId=%20%201%20%20",
			want:        exportFilter{Format: "json", Limit: 1000, Offset: 0, ChainID: "1"},
		},
		{
			name:        "contract_filter",
			queryString: "contract=0xabc123",
			want:        exportFilter{Format: "json", Limit: 1000, Offset: 0, Contract: "0xabc123"},
		},
		{
			name:        "time_range",
			queryString: "start_time=1700000000&end_time=1710000000",
			want:        exportFilter{Format: "json", Limit: 1000, Offset: 0, StartTime: 1700000000, EndTime: 1710000000},
		},
		{
			name:        "full_filter",
			queryString: "format=csv&limit=500&offset=20&chainId=ethereum&eventName=Swap&contract=0xdef&start_time=100&end_time=200",
			want:        exportFilter{Format: "csv", Limit: 500, Offset: 20, ChainID: "ethereum", EventName: "Swap", Contract: "0xdef", StartTime: 100, EndTime: 200},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/export?"+tt.queryString, nil)
			got := parseExportFilter(req)
			if got != tt.want {
				t.Errorf("parseExportFilter() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidateExportFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter exportFilter
		want   string
	}{
		{"valid_defaults", exportFilter{Limit: 1000, Offset: 0}, ""},
		{"valid_max_limit", exportFilter{Limit: 10000, Offset: 0}, ""},
		{"valid_min_limit", exportFilter{Limit: 1, Offset: 0}, ""},
		{"valid_with_time_range", exportFilter{Limit: 100, Offset: 0, StartTime: 100, EndTime: 200}, ""},
		{"valid_zero_offset", exportFilter{Limit: 100, Offset: 0}, ""},
		{"limit_zero", exportFilter{Limit: 0, Offset: 0}, "limit must be between 1 and 10000"},
		{"limit_negative", exportFilter{Limit: -1, Offset: 0}, "limit must be between 1 and 10000"},
		{"limit_exceeds_max", exportFilter{Limit: 10001, Offset: 0}, "limit must be between 1 and 10000"},
		{"offset_negative", exportFilter{Limit: 100, Offset: -1}, "offset must be greater than or equal to 0"},
		{"start_after_end", exportFilter{Limit: 100, Offset: 0, StartTime: 200, EndTime: 100}, "start_time must be less than or equal to end_time"},
		{"start_equal_end_valid", exportFilter{Limit: 100, Offset: 0, StartTime: 100, EndTime: 100}, ""},
		{"only_start_time_valid", exportFilter{Limit: 100, Offset: 0, StartTime: 100}, ""},
		{"only_end_time_valid", exportFilter{Limit: 100, Offset: 0, EndTime: 100}, ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := validateExportFilter(tt.filter)
			if got != tt.want {
				t.Errorf("validateExportFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventsToExportRecords(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	txHash := common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef")

	tests := []struct {
		name   string
		events []*blockchain.BlockchainEvent
		want   int
	}{
		{
			name:   "empty_slice",
			events: []*blockchain.BlockchainEvent{},
			want:   0,
		},
		{
			name:   "nil_slice",
			events: nil,
			want:   0,
		},
		{
			name: "single_event",
			events: []*blockchain.BlockchainEvent{
				{
					ID:              "evt_001",
					EventName:       "Transfer",
					ChainID:         "1",
					ContractAddress: addr,
					BlockNumber:     12345678,
					TransactionHash: txHash,
					BlockTimestamp:  1700000000,
					Status:          blockchain.EventStatusConfirmed,
				},
			},
			want: 1,
		},
		{
			name: "multiple_events",
			events: []*blockchain.BlockchainEvent{
				{
					ID:              "evt_001",
					EventName:       "Transfer",
					ChainID:         "1",
					ContractAddress: addr,
					BlockNumber:     100,
					TransactionHash: txHash,
					BlockTimestamp:  1000,
					Status:          blockchain.EventStatusConfirmed,
				},
				{
					ID:              "evt_002",
					EventName:       "Approval",
					ChainID:         "56",
					ContractAddress: common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					BlockNumber:     200,
					TransactionHash: common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
					BlockTimestamp:  2000,
					Status:          blockchain.EventStatusPending,
				},
			},
			want: 2,
		},
		{
			name: "skips_nil_event",
			events: []*blockchain.BlockchainEvent{
				nil,
				{
					ID:              "evt_001",
					EventName:       "Transfer",
					ChainID:         "1",
					ContractAddress: addr,
					BlockNumber:     100,
					TransactionHash: txHash,
					BlockTimestamp:  1000,
					Status:          blockchain.EventStatusFinalized,
				},
				nil,
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &ExportHandler{}
			got := h.eventsToExportRecords(tt.events)
			if len(got) != tt.want {
				t.Errorf("eventsToExportRecords() len = %d, want %d", len(got), tt.want)
			}
			if tt.want > 0 && len(got) > 0 {
				first := got[0]
				src := tt.events[0]
				if src != nil {
					if first.ID != src.ID {
						t.Errorf("ID = %q, want %q", first.ID, src.ID)
					}
					if first.EventName != src.EventName {
						t.Errorf("EventName = %q, want %q", first.EventName, src.EventName)
					}
					if first.BlockNumber != src.BlockNumber {
						t.Errorf("BlockNumber = %d, want %d", first.BlockNumber, src.BlockNumber)
					}
				}
			}
		})
	}
}

func TestMatchesExportFilter(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	otherAddr := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	baseEvent := &blockchain.BlockchainEvent{
		ChainID:         "1",
		EventName:       "Transfer",
		ContractAddress: addr,
		BlockTimestamp:  1700000000,
	}

	tests := []struct {
		name   string
		event  *blockchain.BlockchainEvent
		filter exportFilter
		want   bool
	}{
		{"empty_filter_matches_all", baseEvent, exportFilter{}, true},
		{"matching_chain_id_numeric", baseEvent, exportFilter{ChainID: "1"}, true},
		{"non_matching_chain_id", baseEvent, exportFilter{ChainID: "56"}, false},
		{"matching_event_name", baseEvent, exportFilter{EventName: "Transfer"}, true},
		{"non_matching_event_name", baseEvent, exportFilter{EventName: "Swap"}, false},
		{"matching_contract_case_insensitive", baseEvent, exportFilter{Contract: "0x1234567890abcdef1234567890abcdef12345678"}, true},
		{"matching_contract_case_insensitive_upper", baseEvent, exportFilter{Contract: "0X1234567890ABCDEF1234567890ABCDEF12345678"}, true},
		{"non_matching_contract", baseEvent, exportFilter{Contract: "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"}, false},
		{"event_within_time_range", baseEvent, exportFilter{StartTime: 1600000000, EndTime: 1800000000}, true},
		{"event_before_start_time", baseEvent, exportFilter{StartTime: 1800000000}, false},
		{"event_after_end_time", baseEvent, exportFilter{EndTime: 1600000000}, false},
		{"event_at_start_boundary", baseEvent, exportFilter{StartTime: 1700000000}, true},
		{"event_at_end_boundary", baseEvent, exportFilter{EndTime: 1700000000}, true},
		{"chain_id_by_name_matches", &blockchain.BlockchainEvent{ChainID: "1", EventName: "Transfer", ContractAddress: addr, BlockTimestamp: 1000}, exportFilter{ChainID: "ethereum"}, true},
		{"multiple_conditions_all_match", baseEvent, exportFilter{ChainID: "1", EventName: "Transfer", Contract: addr.Hex()}, true},
		{"multiple_conditions_one_fails", baseEvent, exportFilter{ChainID: "1", EventName: "Swap"}, false},
		{"non_matching_chain_with_other_matching", &blockchain.BlockchainEvent{ChainID: "56", EventName: "Transfer", ContractAddress: addr, BlockTimestamp: 1000}, exportFilter{ChainID: "1"}, false},
		{"contract_filter_non_matching_address", &blockchain.BlockchainEvent{ContractAddress: otherAddr, ChainID: "1", EventName: "Transfer", BlockTimestamp: 1000}, exportFilter{Contract: addr.Hex()}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &ExportHandler{}
			got := h.matchesExportFilter(tt.event, tt.filter)
			if got != tt.want {
				t.Errorf("matchesExportFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewExportHandler(t *testing.T) {
	t.Parallel()

	h := NewExportHandler(&MockLogger{}, &mockEventStore{})
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.logger == nil {
		t.Error("expected logger to be set")
	}
	if h.eventStore == nil {
		t.Error("expected eventStore to be set")
	}
}

func TestExportEventsCSV(t *testing.T) {
	t.Parallel()

	h := &ExportHandler{logger: &MockLogger{}}

	records := []exportRecord{
		{ID: "evt-1", EventName: "Transfer", ChainID: "1", ContractAddress: "0xabc", BlockNumber: 100, TransactionHash: "0xtx", Timestamp: 1234, Status: "confirmed"},
		{ID: "evt-2", EventName: "Swap", ChainID: "56", ContractAddress: "0xdef", BlockNumber: 200, TransactionHash: "0xty", Timestamp: 5678, Status: "pending"},
	}

	w := httptest.NewRecorder()
	h.ExportEventsCSV(w, records)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("expected text/csv content type, got %q", contentType)
	}

	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") {
		t.Errorf("expected attachment disposition, got %q", disposition)
	}

	reader := csv.NewReader(w.Body)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header + 2 data), got %d", len(rows))
	}
	if rows[0][0] != "id" {
		t.Errorf("expected header 'id', got %q", rows[0][0])
	}
	if rows[1][0] != "evt-1" {
		t.Errorf("expected 'evt-1', got %q", rows[1][0])
	}
}

func TestExportEventsCSV_Empty(t *testing.T) {
	t.Parallel()

	h := &ExportHandler{logger: &MockLogger{}}

	w := httptest.NewRecorder()
	h.ExportEventsCSV(w, nil)

	reader := csv.NewReader(w.Body)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (header only), got %d", len(rows))
	}
}

func TestExportEventsJSON(t *testing.T) {
	t.Parallel()

	h := &ExportHandler{logger: &MockLogger{}}

	records := []exportRecord{
		{ID: "evt-1", EventName: "Transfer", ChainID: "1", ContractAddress: "0xabc", BlockNumber: 100, TransactionHash: "0xtx", Timestamp: 1234, Status: "confirmed"},
	}

	w := httptest.NewRecorder()
	h.ExportEventsJSON(w, records)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json, got %q", contentType)
	}

	var decoded []exportRecord
	if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(decoded) != 1 {
		t.Errorf("expected 1 record, got %d", len(decoded))
	}
	if decoded[0].ID != "evt-1" {
		t.Errorf("expected 'evt-1', got %q", decoded[0].ID)
	}
}

func TestExportEventsJSON_Empty(t *testing.T) {
	t.Parallel()

	h := &ExportHandler{logger: &MockLogger{}}

	w := httptest.NewRecorder()
	h.ExportEventsJSON(w, nil)

	var decoded []exportRecord
	if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if decoded != nil {
		t.Errorf("expected null, got %v", decoded)
	}
}
