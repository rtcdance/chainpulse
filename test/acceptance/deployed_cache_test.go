//go:build integration

package acceptance

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func checkDeployedCanonicalCache(t *testing.T, ctx context.Context, db *sql.DB, apiURL, artifacts, mode string) {
	t.Helper()
	endpoint := os.Getenv("CHAINPULSE_DEPLOYED_REDIS_ADDR")
	require.NotEmpty(t, endpoint)
	if !strings.Contains(endpoint, "://") {
		endpoint = "redis://" + endpoint
	}
	options, err := redis.ParseURL(endpoint)
	require.NoError(t, err)
	client := redis.NewClient(options)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	counters := func() (int64, int64) {
		t.Helper()
		info, err := client.Info(ctx, "commandstats", "stats").Result()
		require.NoError(t, err)
		var ranges, hits int64
		for _, line := range strings.Split(info, "\n") {
			line = strings.TrimSpace(line)
			if value, ok := strings.CutPrefix(line, "cmdstat_getrange:calls="); ok {
				ranges, err = strconv.ParseInt(strings.SplitN(value, ",", 2)[0], 10, 64)
				require.NoError(t, err)
			}
			if value, ok := strings.CutPrefix(line, "keyspace_hits:"); ok {
				hits, err = strconv.ParseInt(value, 10, 64)
				require.NoError(t, err)
			}
		}
		return ranges, hits
	}
	beforeRanges, beforeHits := counters()
	var expected int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31337' AND status<>'reorged' AND event_state->>'canonicality'='canonical'`).Scan(&expected))
	httpClient := &http.Client{Timeout: 5 * time.Second}
	for range 2 {
		response, err := httpClient.Get(apiURL + "/events?chain_id=31337&limit=5")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode)
		var page struct {
			Data []struct {
				Chain string `json:"chainId"`
			}
			Pagination struct {
				Total int64 `json:"total"`
			}
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
		require.NoError(t, response.Body.Close())
		require.Equal(t, expected, page.Pagination.Total)
		require.Len(t, page.Data, int(min(expected, 5)))
		for _, event := range page.Data {
			require.Equal(t, "31337", event.Chain)
		}
		response, err = httpClient.Get(apiURL + "/events/stats")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode)
		var stats struct {
			Total int64 `json:"total"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&stats))
		require.NoError(t, response.Body.Close())
		require.Equal(t, expected, stats.Total)
	}
	afterRanges, afterHits := counters()
	require.GreaterOrEqual(t, afterRanges-beforeRanges, int64(4), "actual entry must call the dedicated Redis aggregate backend")
	require.GreaterOrEqual(t, afterHits-beforeHits, int64(2), "repeat page/stats requests must hit real Redis")
	proof, err := json.MarshalIndent(map[string]any{"mode": mode, "expectedCanonicalCount": expected, "getrangeCalls": afterRanges - beforeRanges, "redisHits": afterHits - beforeHits, "pageAndStatsMatchedPostgres": true}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, mode+"-canonical-cache.json"), proof, 0o644))
}
