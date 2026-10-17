package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/lib/pq"
)

// CanonicalAggregateCache stores counts/aggregates, never event pages or RPC
// proofs. Implementations must honor cancellation; their owner closes them.
type CanonicalAggregateCache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, int) error
}

// WithCanonicalAggregateCache configures an optional cache before serving.
// Missing or failed cache reads fall back to PG; a missing revision authority
// fails the query instead of reusing an unversioned result.
func (a *PostgreSQLPluginAdapter) WithCanonicalAggregateCache(cache CanonicalAggregateCache) *PostgreSQLPluginAdapter {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.aggregateCache = cache
	return a
}

// ValidateCanonicalAggregateSchema prevents enabling a cache whose mutation
// authority is missing, disabled or left dirty by an interrupted migration.
func (a *PostgreSQLPluginAdapter) ValidateCanonicalAggregateSchema(ctx context.Context) error {
	db, err := a.requireDB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var table, triggers, ledger bool
	err = db.QueryRowContext(ctx, `SELECT to_regclass('public.canonical_query_revisions') IS NOT NULL,
(SELECT count(*)=2 FROM pg_trigger WHERE tgrelid='public.blockchain_events'::regclass
 AND tgname IN ('bump_canonical_query_revision','truncate_canonical_query_revision') AND tgenabled IN ('O','A')),
COALESCE((SELECT version>=20 AND NOT dirty FROM schema_migrations LIMIT 1),false)`).Scan(&table, &triggers, &ledger)
	if err != nil {
		return fmt.Errorf("verify canonical cache migration authority: %w", err)
	}
	if !table || !triggers || !ledger {
		return fmt.Errorf("canonical aggregate cache requires clean schema20 and enabled mutation triggers")
	}
	return nil
}

func (a *PostgreSQLPluginAdapter) aggregateKey(ctx context.Context, tx *sql.Tx, kind string, filter map[string]any, scope []string) (CanonicalAggregateCache, string, error) {
	a.mu.RLock()
	cache := a.aggregateCache
	a.mu.RUnlock()
	if cache == nil {
		return nil, "", nil
	}
	if len(scope) == 0 || len(scope) > 50 {
		return nil, "", fmt.Errorf("canonical aggregate cache requires 1..50 declared chains")
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.chain_id,COALESCE(r.revision,0),COALESCE(r.epoch,'absent')
FROM (SELECT DISTINCT unnest($1::text[]) AS chain_id) s
LEFT JOIN canonical_query_revisions r USING(chain_id) ORDER BY s.chain_id`, pq.Array(scope))
	if err != nil {
		return nil, "", fmt.Errorf("read aggregate revision snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type revision struct {
		Chain string
		Value int64
		Epoch string
	}
	versions := make([]revision, 0, len(scope))
	for rows.Next() {
		var value revision
		if err := rows.Scan(&value.Chain, &value.Value, &value.Epoch); err != nil {
			return nil, "", fmt.Errorf("scan aggregate revision: %w", err)
		}
		versions = append(versions, value)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("read aggregate revisions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, "", fmt.Errorf("close aggregate revisions: %w", err)
	}
	if len(versions) == 0 {
		return nil, "", fmt.Errorf("aggregate revision scope is empty")
	}
	encoded, err := json.Marshal(struct {
		Kind     string
		Versions []revision
		Filter   map[string]any
	}{kind, versions, filter})
	if err != nil {
		return nil, "", fmt.Errorf("encode aggregate cache key: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return cache, "chainpulse:canonical-aggregate:v1:" + hex.EncodeToString(digest[:]), nil
}

func (a *PostgreSQLPluginAdapter) cachedCount(ctx context.Context, cache CanonicalAggregateCache, key string) (int64, bool) {
	if cache == nil {
		return 0, false
	}
	cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	data, err := cache.Get(cacheCtx, key)
	count, parseErr := strconv.ParseInt(string(data), 10, 64)
	result := "miss"
	if err != nil {
		result = "error"
	} else if parseErr == nil && count >= 0 {
		result = "hit"
	}
	a.recordAggregateCache(result)
	return count, result == "hit"
}

func (a *PostgreSQLPluginAdapter) canonicalCount(ctx context.Context, tx *sql.Tx, cache CanonicalAggregateCache, key, where string, args []any) (int64, error) {
	if total, hit := a.cachedCount(ctx, cache, key); hit {
		return total, nil
	}
	payload, err := a.aggregateFlights.read(ctx, key, func() ([]byte, error) {
		// Another owner may have populated Redis after our initial lookup.
		if cache != nil {
			if total, hit := a.cachedCount(ctx, cache, key); hit {
				return []byte(strconv.FormatInt(total, 10)), nil
			}
		}
		a.recordAggregateOrigin("count")
		var total int64
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM blockchain_events WHERE "+where, args...).Scan(&total); err != nil {
			return nil, fmt.Errorf("count canonical events: %w", err)
		}
		// This transaction is read-only and the key binds the exact revision.
		// Later page/proof failure cannot change the validity of this count.
		a.storeCount(ctx, cache, key, total)
		return []byte(strconv.FormatInt(total, 10)), nil
	})
	if err != nil {
		return 0, err
	}
	total, err := strconv.ParseInt(string(payload), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decode shared canonical count: %w", err)
	}
	return total, nil
}

func (a *PostgreSQLPluginAdapter) canonicalStatsFromCache(ctx context.Context, tx *sql.Tx, cache CanonicalAggregateCache, key string, scope []string) (canonicalStats, error) {
	if stats, hit := a.cachedStats(ctx, cache, key, scope); hit {
		return stats, nil
	}
	payload, err := a.aggregateFlights.read(ctx, key, func() ([]byte, error) {
		stats, hit := a.cachedStats(ctx, cache, key, scope)
		if !hit {
			a.recordAggregateOrigin("dashboard")
			var err error
			stats, err = readCanonicalGroupedStats(ctx, tx, scope)
			if err != nil {
				return nil, err
			}
		}
		payload, err := json.Marshal(stats)
		if err != nil {
			return nil, fmt.Errorf("encode canonical aggregate: %w", err)
		}
		if !hit {
			a.storeAggregate(ctx, cache, key, payload)
		}
		return payload, nil
	})
	if err != nil {
		return canonicalStats{}, err
	}
	var stats canonicalStats
	if err := json.Unmarshal(payload, &stats); err != nil {
		return stats, fmt.Errorf("decode shared canonical aggregate: %w", err)
	}
	if !stats.valid(scope) {
		return stats, fmt.Errorf("shared canonical aggregate is inconsistent")
	}
	return stats, nil
}

func (a *PostgreSQLPluginAdapter) recordAggregateOrigin(operation string) {
	if a.metricsCollector != nil {
		a.metricsCollector.RecordCounter("canonical_aggregate_origin_read", 1, map[string]string{"operation": operation})
	}
}

func (a *PostgreSQLPluginAdapter) storeCount(ctx context.Context, cache CanonicalAggregateCache, key string, count int64) {
	a.storeAggregate(ctx, cache, key, []byte(strconv.FormatInt(count, 10)))
}

func (a *PostgreSQLPluginAdapter) storeAggregate(ctx context.Context, cache CanonicalAggregateCache, key string, data []byte) {
	if cache == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := cache.Set(cacheCtx, key, data, 30); err != nil {
		a.recordAggregateCache("write_error")
	}
}

type canonicalStats struct {
	Total   int64
	Reorged int64
	ByChain map[string]int64
	ByName  map[string]int64
}

func (a *PostgreSQLPluginAdapter) cachedStats(ctx context.Context, cache CanonicalAggregateCache, key string, scope []string) (canonicalStats, bool) {
	var stats canonicalStats
	if cache == nil {
		return stats, false
	}
	cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	data, err := cache.Get(cacheCtx, key)
	if err != nil {
		a.recordAggregateCache("error")
		return stats, false
	}
	if len(data) == 0 || len(data) > 65536 || json.Unmarshal(data, &stats) != nil || !stats.valid(scope) {
		a.recordAggregateCache("miss")
		return canonicalStats{}, false
	}
	a.recordAggregateCache("hit")
	return stats, true
}

func (s canonicalStats) valid(scope []string) bool {
	if s.Total < 0 || s.Reorged < 0 || s.ByChain == nil || s.ByName == nil || len(s.ByChain) > 50 || len(s.ByName) > 1024 {
		return false
	}
	allowed := make(map[string]bool, len(scope))
	for _, chain := range scope {
		allowed[chain] = true
	}
	for chain := range s.ByChain {
		if !allowed[chain] {
			return false
		}
	}
	for _, group := range []map[string]int64{s.ByChain, s.ByName} {
		var sum int64
		for _, count := range group {
			if count <= 0 || count > s.Total-sum {
				return false
			}
			sum += count
		}
		if sum != s.Total {
			return false
		}
	}
	return true
}

func (a *PostgreSQLPluginAdapter) recordAggregateCache(result string) {
	if a.metricsCollector != nil {
		a.metricsCollector.RecordCounter("canonical_aggregate_cache", 1, map[string]string{"result": result})
	}
}
