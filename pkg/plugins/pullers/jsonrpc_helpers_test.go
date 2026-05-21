package pullers

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestHexToUint64(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  uint64
	}{
		{"0x0", 0},
		{"0x1", 1},
		{"0xa", 10},
		{"0xff", 255},
		{"0x100", 256},
		{"0x7fffffffffffffff", 9223372036854775807},
		{"", 0},
		{"0x", 0},
		{"0xgg", 0},
		{"0x1234567890abcdef", 1311768467294899695},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := hexToUint64(tc.input)
			if got != tc.want {
				t.Errorf("hexToUint64(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestUint64ToHex(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input uint64
		want  string
	}{
		{0, "0x0"},
		{1, "0x1"},
		{10, "0xa"},
		{255, "0xff"},
		{256, "0x100"},
		{4294967295, "0xffffffff"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			got := uint64ToHex(tc.input)
			if got != tc.want {
				t.Errorf("uint64ToHex(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHexRoundTrip(t *testing.T) {
	t.Parallel()
	values := []uint64{0, 1, 42, 255, 65535, 18446744073709551615}
	for _, v := range values {
		t.Run(uint64ToHex(v), func(t *testing.T) {
			t.Parallel()
			hex := uint64ToHex(v)
			back := hexToUint64(hex)
			if back != v {
				t.Errorf("round trip %d -> %s -> %d", v, hex, back)
			}
		})
	}
}

func TestJSONRPCError_Fields(t *testing.T) {
	t.Parallel()
	err := &JSONRPCError{Code: -32000, Message: "msg", Data: "data"}
	if err.Code != -32000 || err.Message != "msg" {
		t.Errorf("JSONRPCError fields mismatch: %+v", err)
	}
}

func TestJSONRPCRequest_Structure(t *testing.T) {
	t.Parallel()
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []any{},
		ID:      1,
	}
	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q", req.JSONRPC)
	}
	if req.Method != "eth_blockNumber" {
		t.Errorf("Method = %q", req.Method)
	}
}

func TestJSONRPCResponse_Structure(t *testing.T) {
	t.Parallel()
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  nil,
		Error:   nil,
		ID:      1,
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q", resp.JSONRPC)
	}
}

func TestBlockHeader_Defaults(t *testing.T) {
	t.Parallel()
	h := &BlockHeader{}
	if h.Number != "" {
		t.Errorf("expected empty number")
	}
}

func TestLog_Defaults(t *testing.T) {
	t.Parallel()
	l := &Log{}
	if l.Address != "" {
		t.Errorf("expected empty address")
	}
}

func TestBaseDataPullerPlugin_Creation(t *testing.T) {
	t.Parallel()
	logger := &noopLogger{}
	metrics := &noopMetrics{}
	cfg := core.Config{
		StartBlock:   100,
		MaxRetries:   3,
		RetryBackoff: 1000,
	}
	p := NewBaseDataPullerPlugin("test-puller", "1.0", cfg, logger, metrics, nil)
	if p.Name() != "test-puller" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.Version() != "1.0" {
		t.Errorf("Version() = %q", p.Version())
	}
}

func TestBaseDataPullerPlugin_Initialize(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("p", "1.0", core.Config{}, nil, nil, nil)
	cfg := core.Config{StartBlock: 200, MaxRetries: 5, RetryBackoff: 500}
	if err := p.Initialize(context.Background(), cfg); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
}

// noopLogger satisfies core.Logger interface
type noopLogger struct{}

func (n *noopLogger) Debug(string, ...any)                 {}
func (n *noopLogger) Info(string, ...any)                  {}
func (n *noopLogger) Warn(string, ...any)                  {}
func (n *noopLogger) Error(string, ...any)                 {}
func (n *noopLogger) Fatal(string, ...any)                 {}
func (n *noopLogger) WithCorrelationID(string) core.Logger { return n }

// noopMetrics satisfies core.MetricsCollector interface
type noopMetrics struct{}

func (n *noopMetrics) RecordCounter(string, int64, map[string]string)     {}
func (n *noopMetrics) RecordGauge(string, float64, map[string]string)     {}
func (n *noopMetrics) RecordHistogram(string, float64, map[string]string) {}
func (n *noopMetrics) GetMetrics() map[string]any                         { return nil }
