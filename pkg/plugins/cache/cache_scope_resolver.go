package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
)

// GenerationReader returns the current reorg generation for a chain ("" = the
// aggregate across all chains). Implementations must read the generation
// source of truth (reorg_generations), never a second counter.
type GenerationReader func(ctx context.Context, chain string) (int64, error)

// LiveScopeResolver versions cache keys against the live reorg generation.
type LiveScopeResolver struct {
	read          GenerationReader
	schemaVersion string
}

// NewLiveScopeResolver creates a CacheScopeVersioner backed by a generation
// reader. schemaVersion must change whenever the shape of cached values
// changes; empty defaults to "1" via CacheScope.VersionToken.
func NewLiveScopeResolver(read GenerationReader, schemaVersion string) (*LiveScopeResolver, error) {
	if read == nil {
		return nil, fmt.Errorf("generation reader is required")
	}
	return &LiveScopeResolver{read: read, schemaVersion: schemaVersion}, nil
}

// VersionKey canonicalizes chain, reads its live generation, and returns
// versionedKey(chain, generation, baseKey). A generation read failure is
// returned as an error so callers bypass the cache instead of serving
// possibly-stale data under an unversioned key.
func (r *LiveScopeResolver) VersionKey(ctx context.Context, chain, baseKey string) (string, error) {
	if baseKey == "" {
		return "", fmt.Errorf("base key is required")
	}
	canonical := canonicalChainKey(chain)
	generation, err := r.read(ctx, chain)
	if err != nil {
		return "", fmt.Errorf("read reorg generation for chain %q: %w", chain, err)
	}
	scope := CacheScope{
		ChainID:       canonical,
		Generation:    generation,
		SchemaVersion: r.schemaVersion,
	}
	keys, err := NewVersionedKeyGenerator("", func() CacheScope { return scope })
	if err != nil {
		return "", fmt.Errorf("build versioned key: %w", err)
	}
	return keys.Key(baseKey), nil
}

func canonicalChainKey(chain string) string {
	trimmed := strings.TrimSpace(chain)
	if trimmed == "" {
		return ""
	}
	if id := chainid.ResolveChainID(trimmed); id > 0 {
		return strconv.Itoa(id)
	}
	return strings.ToLower(trimmed)
}

var _ portcontract.CacheScopeVersioner = (*LiveScopeResolver)(nil)
