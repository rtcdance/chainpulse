package pullers

import (
	"errors"
	"math"
	"testing"
)

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
			got := parseNodeURLs(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("len(parseNodeURLs(%q)) = %d, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("parseNodeURLs(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
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

	got := defaultSolanaProgramFilters()
	if len(got) == 0 {
		t.Error("expected non-empty program filters")
	}
	if _, ok := got["TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"]; !ok {
		t.Error("expected Token program in filters")
	}
	if _, ok := got["TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"]; !ok {
		t.Error("expected Token-2022 program in filters")
	}
}

func TestDeriveSolanaWSUrl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https", "https://api.mainnet-beta.solana.com", "wss://api.mainnet-beta.solana.com"},
		{"http", "http://localhost:8899", "ws://localhost:8899"},
		{"no_scheme", "localhost:8899", "localhost:8899"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deriveSolanaWSUrl(tc.url)
			if got != tc.want {
				t.Errorf("deriveSolanaWSUrl(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}
