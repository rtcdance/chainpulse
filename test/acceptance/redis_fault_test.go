//go:build integration

package acceptance

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
