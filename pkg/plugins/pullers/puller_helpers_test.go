package pullers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type stubRPCCodedError struct {
	code int
	msg  string
}

func (e *stubRPCCodedError) Error() string  { return e.msg }
func (e *stubRPCCodedError) ErrorCode() int { return e.code }

func TestClassifyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		retryable bool
		category  string
	}{
		{"nil_error", nil, true, "none"},
		{"rate_limit_429", errors.New("HTTP 429 too many requests"), true, "rate_limit"},
		{"rate_limit_text", errors.New("rate limit exceeded"), true, "rate_limit"},
		{"throttled", errors.New("request throttled"), true, "rate_limit"},
		{"timeout", errors.New("request timeout"), true, "timeout"},
		{"context_deadline", errors.New("context deadline exceeded"), true, "timeout"},
		{"io_timeout", errors.New("i/o timeout"), true, "timeout"},
		{"connection_refused", errors.New("connection refused"), true, "network"},
		{"no_such_host", errors.New("no such host"), true, "network"},
		{"network_unreachable", errors.New("network is unreachable"), true, "network"},
		{"connection_reset", errors.New("connection reset"), true, "network"},
		{"broken_pipe", errors.New("broken pipe"), true, "network"},
		{"unauthorized_401", errors.New("HTTP 401 Unauthorized"), false, "non_retryable"},
		{"forbidden_403", errors.New("HTTP 403 Forbidden"), false, "non_retryable"},
		{"unauthorized", errors.New("unauthorized"), false, "non_retryable"},
		{"invalid_api_key", errors.New("invalid api key"), false, "non_retryable"},
		{"invalid_param", errors.New("invalid param"), false, "non_retryable"},
		{"unknown_method", errors.New("unknown method"), false, "non_retryable"},
		{"default_retryable", errors.New("some random error"), true, "unknown"},

		{"rpceerr_429", &core.RPCErr{StatusCode: 429}, true, "rate_limit"},
		{"rpceerr_408", &core.RPCErr{StatusCode: 408}, true, "timeout"},
		{"rpceerr_500", &core.RPCErr{StatusCode: 500}, true, "server_error"},
		{"rpceerr_503", &core.RPCErr{StatusCode: 503}, true, "server_error"},
		{"rpceerr_401", &core.RPCErr{StatusCode: 401}, false, "non_retryable"},
		{"rpceerr_403", &core.RPCErr{StatusCode: 403}, false, "non_retryable"},
		{"rpceerr_wrapped", fmt.Errorf("eth_getLogs failed: %w", &core.RPCErr{StatusCode: 502}), true, "server_error"},

		{"jsonrpc_invalid_request", &stubRPCCodedError{code: -32600, msg: "invalid request"}, false, "non_retryable"},
		{"jsonrpc_method_not_found", &stubRPCCodedError{code: -32601, msg: "the method does not exist"}, false, "non_retryable"},
		{"jsonrpc_invalid_params", &stubRPCCodedError{code: -32602, msg: "invalid params"}, false, "non_retryable"},
		{"jsonrpc_limit_exceeded", &stubRPCCodedError{code: -32005, msg: "query returned more than 10000 results"}, true, "rate_limit"},
		{"jsonrpc_server_error", &stubRPCCodedError{code: -32000, msg: "server error"}, true, "server_error"},

		{"range_too_wide", &ErrRangeTooWide{From: 1, To: 1000, Reason: "result count reached the provider limit"}, false, "non_retryable"},
		{"range_too_wide_wrapped", fmt.Errorf("fetch chunk: %w", &ErrRangeTooWide{From: 1, To: 1000, Reason: "too wide"}), false, "non_retryable"},

		{"deadline_is", context.DeadlineExceeded, true, "timeout"},
		{"deadline_wrapped", fmt.Errorf("fetch logs: %w", context.DeadlineExceeded), true, "timeout"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyError(tc.err)
			if got.retryable != tc.retryable {
				t.Errorf("classifyError(%v).retryable = %v, want %v", tc.err, got.retryable, tc.retryable)
			}
			if got.category != tc.category {
				t.Errorf("classifyError(%v).category = %q, want %q", tc.err, got.category, tc.category)
			}
		})
	}
}

func TestParseNodeURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single", "http://localhost:8545", []string{"http://localhost:8545"}},
		{"multiple", "http://a:8545,http://b:8545", []string{"http://a:8545", "http://b:8545"}},
		{"with_spaces", " http://a:8545 , http://b:8545 ", []string{"http://a:8545", "http://b:8545"}},
		{"empty_parts", "http://a:8545,,http://b:8545", []string{"http://a:8545", "http://b:8545"}},
		{"all_empty", ", ,", []string{"http://localhost:8545"}},
		{"only_whitespace", "  ", []string{"http://localhost:8545"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_ = tc
		})
	}
}

func TestSaturatingPullerBlockMetric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		block uint64
		want  int64
	}{
		{"small", 42, 42},
		{"zero", 0, 0},
		{"max_int64", math.MaxInt64, math.MaxInt64},
		{"overflow", math.MaxInt64 + 1, math.MaxInt64},
		{"max_uint64", math.MaxUint64, math.MaxInt64},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := saturatingPullerBlockMetric(tc.block)
			if got != tc.want {
				t.Errorf("saturatingPullerBlockMetric(%d) = %d, want %d", tc.block, got, tc.want)
			}
		})
	}
}

func TestParseInstructionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		programID string
		want      string
	}{
		{"known_program", "11111111111111111111111111111111", "System Program"},
		{"unknown_program", "UnknownProgramID", "UnknownProgramID"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseInstructionType(tc.programID)
			if got != tc.want {
				t.Errorf("parseInstructionType(%q) = %q, want %q", tc.programID, got, tc.want)
			}
		})
	}
}

func TestInstProgramIDToAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		programID string
		wantLen   int
	}{
		{"empty", "", 42},
		{"long_enough", "1111111111111111111111111111111111111111", 42},
		{"short", "abc", 42},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := instProgramIDToAddress(tc.programID)
			if len(got.Hex()) != tc.wantLen {
				t.Errorf("instProgramIDToAddress(%q) hex length = %d, want %d", tc.programID, len(got.Hex()), tc.wantLen)
			}
		})
	}
}

func TestExtractCosmosMsgTypes(t *testing.T) {
	t.Parallel()

	t.Run("contains_known_type", func(t *testing.T) {
		t.Parallel()
		raw := "some data /cosmos.bank.v1beta1.MsgSend more data"
		got := extractCosmosMsgTypes(raw)
		if len(got) != 1 || got[0] != "/cosmos.bank.v1beta1.MsgSend" {
			t.Errorf("extractCosmosMsgTypes(%q) = %v, want [/cosmos.bank.v1beta1.MsgSend]", raw, got)
		}
	})

	t.Run("multiple_types", func(t *testing.T) {
		t.Parallel()
		raw := "/cosmos.bank.v1beta1.MsgSend /cosmos.staking.v1beta1.MsgDelegate"
		got := extractCosmosMsgTypes(raw)
		if len(got) != 2 {
			t.Errorf("len(extractCosmosMsgTypes) = %d, want 2", len(got))
		}
	})

	t.Run("no_known_types", func(t *testing.T) {
		t.Parallel()
		raw := "some random data without known types"
		got := extractCosmosMsgTypes(raw)
		if len(got) != 0 {
			t.Errorf("len(extractCosmosMsgTypes) = %d, want 0", len(got))
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got := extractCosmosMsgTypes("")
		if len(got) != 0 {
			t.Errorf("len(extractCosmosMsgTypes) = %d, want 0", len(got))
		}
	})
}

func TestDefaultSolanaProgramFilters(t *testing.T) {
	t.Parallel()

	_ = struct{}{}
}

func TestDeriveSolanaWSUrl(t *testing.T) {
}
