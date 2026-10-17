package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/rtcdance/chainpulse/pkg/chainid"
)

// ReadCacheGeneration returns the reorg generation that query cache keys must
// embed (T11). An empty chain aggregates every row, so any chain's reorg
// invalidates globally scoped entries. For a named chain it sums the rows of
// all key forms that chain may be stored under (as-given, numeric, canonical
// name), covering both monolithic symbolic writes and durable numeric writes;
// generation rows only ever increase, so the sum is monotonic across reorgs.
func ReadCacheGeneration(ctx context.Context, db *sql.DB, chain string) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("postgres db is required")
	}
	if strings.TrimSpace(chain) == "" {
		var generation int64
		if err := db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(generation),0) FROM reorg_generations`,
		).Scan(&generation); err != nil {
			return 0, fmt.Errorf("read global reorg generation: %w", err)
		}
		return generation, nil
	}

	candidates := chainGenerationCandidates(chain)
	args := make([]any, len(candidates))
	placeholders := make([]string, len(candidates))
	for i, candidate := range candidates {
		args[i] = candidate
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf(
		`SELECT COALESCE(SUM(generation),0) FROM reorg_generations WHERE chain_id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	var generation int64
	if err := db.QueryRowContext(ctx, query, args...).Scan(&generation); err != nil {
		return 0, fmt.Errorf("read reorg generation for chain %q: %w", chain, err)
	}
	return generation, nil
}

func chainGenerationCandidates(chain string) []string {
	seen := make(map[string]struct{}, 3)
	candidates := make([]string, 0, 3)
	add := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	raw := strings.ToLower(strings.TrimSpace(chain))
	add(raw)
	if id := chainid.ResolveChainID(raw); id > 0 {
		add(strconv.Itoa(id))
		add(chainid.ResolveChainName(id))
	}
	return candidates
}
