package finality

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
)

func TestParseStrategy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		raw     string
		want    Strategy
		wantErr bool
	}{
		{name: "confirmations", raw: "12", want: Strategy{Mode: StrategyConfirmations, Confirmations: 12}},
		{name: "zero confirmations", raw: "0", want: Strategy{Mode: StrategyConfirmations, Confirmations: 0}},
		{name: "latest", raw: "latest", want: Strategy{Mode: StrategyLatest}},
		{name: "tag safe", raw: "tag:safe", want: Strategy{Mode: StrategyTag, Tag: "safe"}},
		{name: "tag finalized", raw: "tag:finalized", want: Strategy{Mode: StrategyTag, Tag: "finalized"}},
		{name: "empty", raw: "", wantErr: true},
		{name: "unknown tag", raw: "tag:unsafe", wantErr: true},
		{name: "garbage", raw: "abc", wantErr: true},
		{name: "tag without value", raw: "tag:", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseStrategy(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseStrategy(%q) = %+v, want error", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStrategy(%q): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("ParseStrategy(%q) = %+v, want %+v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestStrategyCapHeight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		strategy Strategy
		latest   uint64
		want     uint64
	}{
		{name: "confirmations subtracts", strategy: Strategy{Mode: StrategyConfirmations, Confirmations: 12}, latest: 100, want: 88},
		{name: "confirmations saturates at zero", strategy: Strategy{Mode: StrategyConfirmations, Confirmations: 200}, latest: 100, want: 0},
		{name: "zero confirmations passes head", strategy: Strategy{Mode: StrategyConfirmations, Confirmations: 0}, latest: 100, want: 100},
		{name: "latest mode passes head", strategy: Strategy{Mode: StrategyLatest}, latest: 100, want: 100},
		{name: "tag mode passes head (resolved by puller)", strategy: Strategy{Mode: StrategyTag, Tag: "safe"}, latest: 100, want: 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.strategy.CapHeight(tc.latest); got != tc.want {
				t.Fatalf("CapHeight(%d) = %d, want %d", tc.latest, got, tc.want)
			}
		})
	}
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	registry, err := NewRegistry(
		Strategy{Mode: StrategyConfirmations, Confirmations: 12},
		map[string]Strategy{
			"polygon": {Mode: StrategyConfirmations, Confirmations: 128},
			"base":    {Mode: StrategyTag, Tag: "safe"},
		},
	)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if got := registry.StrategyFor("polygon"); got.Confirmations != 128 {
		t.Fatalf("StrategyFor(polygon) = %+v, want 128 confirmations", got)
	}
	if got := registry.StrategyFor("base"); got.Mode != StrategyTag || got.Tag != "safe" {
		t.Fatalf("StrategyFor(base) = %+v, want tag:safe", got)
	}
	if got := registry.StrategyFor("ethereum"); got.Mode != StrategyConfirmations || got.Confirmations != 12 {
		t.Fatalf("StrategyFor(ethereum) = %+v, want default 12 confirmations", got)
	}
}

func TestNewRegistryRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		def      Strategy
		perChain map[string]Strategy
	}{
		{name: "invalid default", def: Strategy{Mode: StrategyTag, Tag: "unsafe"}},
		{name: "invalid per-chain", def: Strategy{Mode: StrategyLatest}, perChain: map[string]Strategy{"eth": {Mode: "bogus"}}},
		{name: "empty chain id", def: Strategy{Mode: StrategyLatest}, perChain: map[string]Strategy{"": {Mode: StrategyLatest}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewRegistry(tc.def, tc.perChain); err == nil {
				t.Fatal("NewRegistry = nil error, want error")
			}
		})
	}
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()
	registry := DefaultRegistry()
	got := registry.StrategyFor("ethereum")
	if got.Mode != StrategyConfirmations || got.Confirmations != 12 {
		t.Fatalf("DefaultRegistry strategy = %+v, want 12 confirmations", got)
	}
}

func TestNewRegistryFromConfig(t *testing.T) {
	t.Parallel()
	registry, err := NewRegistryFromConfig(configmodel.FinalityConfig{
		DefaultConfirmations: 12,
		Strategies: map[string]string{
			"polygon": "128",
			"base":    "tag:safe",
			"anvil":   "latest",
		},
	})
	if err != nil {
		t.Fatalf("NewRegistryFromConfig: %v", err)
	}
	if got := registry.StrategyFor("polygon"); got.Confirmations != 128 {
		t.Fatalf("StrategyFor(polygon) = %+v, want 128 confirmations", got)
	}
	if got := registry.StrategyFor("base"); got.Mode != StrategyTag || got.Tag != "safe" {
		t.Fatalf("StrategyFor(base) = %+v, want tag:safe", got)
	}
	if got := registry.StrategyFor("anvil"); got.Mode != StrategyLatest {
		t.Fatalf("StrategyFor(anvil) = %+v, want latest", got)
	}
	if got := registry.StrategyFor("ethereum"); got.Confirmations != 12 {
		t.Fatalf("StrategyFor(ethereum) = %+v, want default 12", got)
	}
}

func TestNewRegistryFromConfigRejectsBadSpec(t *testing.T) {
	t.Parallel()
	if _, err := NewRegistryFromConfig(configmodel.FinalityConfig{
		DefaultConfirmations: 12,
		Strategies:           map[string]string{"eth": "nope"},
	}); err == nil {
		t.Fatal("NewRegistryFromConfig = nil error, want error for invalid spec")
	}
}
