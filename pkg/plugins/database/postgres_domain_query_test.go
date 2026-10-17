package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalFilterIsDeterministicAndParameterized(t *testing.T) {
	t.Parallel()
	filter := map[string]any{"chainId": map[string]any{"$in": []any{1, "1", "ethereum"}}, "eventName": "Transfer' OR TRUE --", "blockNumber": map[string]any{"$gte": uint64(10), "$lte": uint64(20)}}
	var args []any
	query, err := domainPredicate(filter, &args, 0)
	require.NoError(t, err)
	require.NotContains(t, query, "Transfer")
	require.Contains(t, query, "chain_id IN")
	require.Contains(t, args, "Transfer' OR TRUE --")
	for range 10 {
		var next []any
		q, err := domainPredicate(filter, &next, 0)
		require.NoError(t, err)
		require.Equal(t, query, q)
		require.Equal(t, args, next)
	}
}

func TestCanonicalFilterRejectsUnsupportedOperators(t *testing.T) {
	t.Parallel()
	for _, filter := range []map[string]any{{"unexpected": "value"}, {"blockNumber": map[string]any{"$where": "javascript"}}, {"$or": []any{}}, {"chainId": map[string]any{"$in": []any{}}}} {
		var args []any
		_, err := domainPredicate(filter, &args, 0)
		require.Error(t, err)
	}
}

// The chain-alias expansion in event_query_handler emits three identical values
// for an unknown chain (chainID, its string form, and ResolveChainName, which
// falls back to the same string). That produced chain_id IN ('31337','31337',
// '31337'), which blocks the ordered index walk on
// idx_events_canonical_chain_page and forces a 147MB sort spill (762ms vs 6.6ms
// at 123,675 rows, measured 2026-10-16). Collapsing duplicates must yield a
// single equality; distinct values must keep IN; both must stay deterministic.
func TestCanonicalFilterDeduplicatesMembershipValues(t *testing.T) {
	t.Parallel()
	dup := map[string]any{"chainId": map[string]any{"$in": []any{31337, "31337", "31337"}}}
	var args []any
	query, err := domainPredicate(dup, &args, 0)
	require.NoError(t, err)
	require.NotContains(t, query, " IN ", "identical values must collapse to one equality")
	require.Contains(t, query, "chain_id=$1")
	require.Equal(t, []any{"31337"}, args)

	distinct := map[string]any{"chainId": map[string]any{"$in": []any{"1", "ethereum"}}}
	var distinctArgs []any
	distinctQuery, err := domainPredicate(distinct, &distinctArgs, 0)
	require.NoError(t, err)
	require.Contains(t, distinctQuery, "chain_id IN ($1,$2)")
	require.Equal(t, []any{"1", "ethereum"}, distinctArgs)

	// Determinism: the same filter must always produce the same SQL and args.
	for range 10 {
		var again []any
		q, err := domainPredicate(dup, &again, 0)
		require.NoError(t, err)
		require.Equal(t, query, q)
		require.Equal(t, args, again)
	}
}
