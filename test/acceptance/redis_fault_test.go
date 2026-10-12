//go:build integration

package acceptance

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
	"github.com/rtcdance/chainpulse/pkg/siwe"
)

func redisContainer() string {
	if name := os.Getenv("CHAINPULSE_REDIS_CONTAINER"); name != "" {
		return name
	}
	return "chainpulse-redis"
}

// redisFaultClient connects using the same env gates as the SIWE tests, so a
// missing gate skips rather than silently passing.
func redisFaultClient(t *testing.T) redis.UniversalClient {
	t.Helper()

	if os.Getenv("CHAINPULSE_RUN_REDIS_INTEGRATION") != "1" {
		return nil
	}
	url := os.Getenv("CHAINPULSE_TEST_REDIS_URL")
	if url == "" {
		return nil
	}

	opts, err := redis.ParseURL(url)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func waitForRedis(t *testing.T, client redis.UniversalClient, within time.Duration) {
	t.Helper()

	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("redis did not become reachable within %s", within)
}

// TestRedisOutageFailsClosed is the §9 E09 rehearsal for Redis.
//
// The property is fail-closed. A nonce store that reported "consumed" while its
// backend was unreachable would let the same nonce be used twice, so the
// required behaviour under an outage is:
//
//   - Issue FAILS, so a caller cannot hand out a nonce it will not be able to
//     verify later;
//   - Consume FAILS with an error, and never returns true.
//
// Recovery must then be transparent: the store works again with no state left
// in a half-written form.
func TestRedisOutageFailsClosed(t *testing.T) {
	client := redisFaultClient(t)
	if client == nil {
		t.Skip("set CHAINPULSE_RUN_REDIS_INTEGRATION=1 and CHAINPULSE_TEST_REDIS_URL")
	}
	require.NoError(t, client.Ping(context.Background()).Err(), "redis must be reachable to begin")

	store, err := siwe.NewRedisNonceStore(client)
	require.NoError(t, err)

	ctx := context.Background()
	address := "0xe09redis"
	nonce := "nonce-e09-baseline"

	// 1. Baseline: the store works while Redis is up.
	require.NoError(t, store.Issue(ctx, address, nonce, time.Minute))
	ok, err := store.Consume(ctx, address, nonce)
	require.NoError(t, err)
	require.True(t, ok, "baseline consume must succeed")

	// 2. Take Redis away.
	dockerLifecycle(t, "stop", redisContainer())
	t.Cleanup(func() {
		// Always restore, even if an assertion below fails.
		restoreRedis(redisContainer())
	})

	// 3. Issuing must fail: a nonce that cannot be verified must not be handed out.
	err = store.Issue(ctx, address, "nonce-during-outage", time.Minute)
	require.Error(t, err, "issuing a nonce must fail while Redis is unreachable")

	// 4. Consuming must fail closed: an error, and never a success.
	consumed, err := store.Consume(ctx, address, "nonce-during-outage")
	require.Error(t, err, "consuming must report the outage rather than answer")
	assert.False(t, consumed, "an outage must never be reported as a successful consume")

	// 5. Restore and confirm the store recovers with no half-written state.
	restoreRedis(redisContainer())
	waitForRedis(t, client, 60*time.Second)

	recovered := "nonce-e09-recovered"
	require.NoError(t, store.Issue(ctx, address, recovered, time.Minute),
		"the store must work again after recovery")

	ok, err = store.Consume(ctx, address, recovered)
	require.NoError(t, err)
	assert.True(t, ok, "consume must succeed again after recovery")

	// Single-use still holds: the recovered nonce is gone.
	ok, err = store.Consume(ctx, address, recovered)
	require.NoError(t, err)
	assert.False(t, ok, "single-use semantics must survive the outage")
}

// restoreRedis starts the container if it is not running, and tolerates a start
// on an already-running container so cleanup never masks the real failure.
func restoreRedis(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = exec.CommandContext(ctx, "docker", "start", name).Run()
}

// redisCachePlugin is the smallest core.CachePlugin over a raw Redis client.
//
// It exists because the production RedisCachePlugin does not satisfy
// core.CachePlugin (its Get/Set take a *CacheEntry, not (key, value, ttl)) —
// the port-conformance gap T02 recorded. Rather than paper over that here, the
// rehearsal uses a local adapter and the gap stays visible.
type redisCachePlugin struct {
	client redis.UniversalClient
}

func (c *redisCachePlugin) Name() string { return "acceptance-redis-cache" }

func (c *redisCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil // a miss, not an error
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (c *redisCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	return c.client.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err()
}

func (c *redisCachePlugin) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *redisCachePlugin) GetStats() core.CacheStats { return core.CacheStats{} }

func (c *redisCachePlugin) HealthCheck(ctx context.Context) error { return c.client.Ping(ctx).Err() }

var _ core.CachePlugin = (*redisCachePlugin)(nil)

// TestRedisOutageDegradesQueries is the other half of §9 E09.
//
// The previous rehearsal proved the SECURITY path fails closed. This one proves
// the QUERY path does the opposite: an unavailable cache must cost latency, not
// the query, so a Redis outage degrades to the source instead of failing.
//
// The two policies are deliberately different, and both are required.
func TestRedisOutageDegradesQueries(t *testing.T) {
	client := redisFaultClient(t)
	if client == nil {
		t.Skip("set CHAINPULSE_RUN_REDIS_INTEGRATION=1 and CHAINPULSE_TEST_REDIS_URL")
	}
	require.NoError(t, client.Ping(context.Background()).Err(), "redis must be reachable to begin")

	ctx := context.Background()
	reader, err := query.NewDegradingReader(
		&redisCachePlugin{client: client},
		func(context.Context, string) ([]byte, error) { return []byte("db-value"), nil },
		core.NewDefaultLogger(core.LogLevelError),
	)
	require.NoError(t, err)

	// Baseline: Redis is up and the key is absent, so this is an ordinary miss.
	require.NoError(t, client.Del(ctx, "e09:query").Err())
	result, err := reader.Read(ctx, "e09:query")
	require.NoError(t, err)
	require.Equal(t, []byte("db-value"), result.Value)
	require.False(t, result.Degraded, "a miss is not a degradation")

	// Take Redis away.
	dockerLifecycle(t, "stop", redisContainer())
	t.Cleanup(func() { restoreRedis(redisContainer()) })

	// The query must still be answered, and must say it was degraded.
	result, err = reader.Read(ctx, "e09:query")
	require.NoError(t, err, "a cache outage must not fail the query")
	assert.True(t, result.Degraded, "the degradation must be reported")
	assert.Equal(t, []byte("db-value"), result.Value, "the answer must still be correct")

	// Restore and confirm the cache is used again.
	restoreRedis(redisContainer())
	waitForRedis(t, client, 60*time.Second)

	require.NoError(t, client.Set(ctx, "e09:query", []byte("cached-value"), time.Minute).Err())
	result, err = reader.Read(ctx, "e09:query")
	require.NoError(t, err)
	assert.False(t, result.Degraded, "after recovery the cache must be used again")
	assert.Equal(t, []byte("cached-value"), result.Value)
}
