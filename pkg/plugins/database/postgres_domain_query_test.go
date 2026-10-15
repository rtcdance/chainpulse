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
