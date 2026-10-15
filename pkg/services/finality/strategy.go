package finality

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
)

// StrategyMode selects how a chain's finality is resolved at the puller boundary.
type StrategyMode string

const (
	// StrategyConfirmations caps the pull target at latest minus N confirmations.
	StrategyConfirmations StrategyMode = "confirmations"
	// StrategyTag resolves the pull target via an RPC block tag (safe/finalized).
	StrategyTag StrategyMode = "tag"
	// StrategyLatest processes the head immediately with no cap.
	StrategyLatest StrategyMode = "latest"
)

// Strategy is the resolved finality strategy for one chain.
type Strategy struct {
	Mode          StrategyMode
	Confirmations uint64
	Tag           string
}

// Validate reports whether the strategy is internally consistent.
func (s Strategy) Validate() error {
	switch s.Mode {
	case StrategyConfirmations:
		return nil
	case StrategyLatest:
		return nil
	case StrategyTag:
		switch s.Tag {
		case "safe", "finalized":
			return nil
		default:
			return fmt.Errorf("unsupported finality tag %q (want safe or finalized)", s.Tag)
		}
	default:
		return fmt.Errorf("unsupported finality mode %q", s.Mode)
	}
}

// CapHeight returns the highest block satisfying a confirmations strategy.
// Tag-mode returns latest unchanged; the puller resolves the tag via RPC.
func (s Strategy) CapHeight(latest uint64) uint64 {
	if s.Mode != StrategyConfirmations {
		return latest
	}
	if s.Confirmations >= latest {
		return 0
	}
	return latest - s.Confirmations
}

// ParseStrategy parses a strategy spec: "12" (confirmations), "latest",
// "tag:safe" or "tag:finalized".
func ParseStrategy(raw string) (Strategy, error) {
	spec := strings.TrimSpace(raw)
	if spec == "" {
		return Strategy{}, fmt.Errorf("empty finality strategy spec")
	}
	if spec == string(StrategyLatest) {
		return Strategy{Mode: StrategyLatest}, nil
	}
	if tag, ok := strings.CutPrefix(spec, string(StrategyTag)+":"); ok {
		strategy := Strategy{Mode: StrategyTag, Tag: strings.TrimSpace(tag)}
		if err := strategy.Validate(); err != nil {
			return Strategy{}, err
		}
		return strategy, nil
	}
	n, err := strconv.ParseUint(spec, 10, 64)
	if err != nil {
		return Strategy{}, fmt.Errorf("invalid finality strategy %q: %w", raw, err)
	}
	return Strategy{Mode: StrategyConfirmations, Confirmations: n}, nil
}

// Registry maps chain IDs to finality strategies with a default fallback.
type Registry struct {
	defaultStrategy Strategy
	strategies      map[string]Strategy
}

// NewRegistry builds a registry; the default strategy must validate and
// per-chain overrides are optional.
func NewRegistry(defaultStrategy Strategy, perChain map[string]Strategy) (*Registry, error) {
	if err := defaultStrategy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid default finality strategy: %w", err)
	}
	copied := make(map[string]Strategy, len(perChain))
	for chainID, strategy := range perChain {
		if chainID == "" {
			return nil, fmt.Errorf("empty chain ID in finality strategies")
		}
		if err := strategy.Validate(); err != nil {
			return nil, fmt.Errorf("invalid finality strategy for chain %q: %w", chainID, err)
		}
		copied[chainID] = strategy
	}
	return &Registry{defaultStrategy: defaultStrategy, strategies: copied}, nil
}

// DefaultRegistry returns the EVM default: 12 confirmations everywhere.
func DefaultRegistry() *Registry {
	registry, err := NewRegistry(Strategy{Mode: StrategyConfirmations, Confirmations: 12}, nil)
	if err != nil {
		panic(err)
	}
	return registry
}

// NewRegistryFromConfig builds a registry from the configmodel finality
// section, parsing each per-chain spec string.
func NewRegistryFromConfig(cfg configmodel.FinalityConfig) (*Registry, error) {
	defaultStrategy := Strategy{Mode: StrategyConfirmations, Confirmations: cfg.DefaultConfirmations}
	perChain := make(map[string]Strategy, len(cfg.Strategies))
	for chainID, spec := range cfg.Strategies {
		strategy, err := ParseStrategy(spec)
		if err != nil {
			return nil, fmt.Errorf("chain %q: %w", chainID, err)
		}
		perChain[chainID] = strategy
	}
	return NewRegistry(defaultStrategy, perChain)
}

// StrategyFor returns the strategy for a chain, falling back to the default.
func (r *Registry) StrategyFor(chainID string) Strategy {
	if strategy, ok := r.strategies[chainID]; ok {
		return strategy
	}
	return r.defaultStrategy
}
