package siwe

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryNonceStoreConsumeIsSingleUse(t *testing.T) {
	t.Parallel()

	store := NewMemoryNonceStore()
	ctx := context.Background()
	address := "0xAbC0000000000000000000000000000000000001"

	require.NoError(t, store.Issue(ctx, address, "nonce-1", time.Minute))

	ok, err := store.Consume(ctx, address, "nonce-1")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = store.Consume(ctx, address, "nonce-1")
	require.NoError(t, err)
	assert.False(t, ok, "a nonce is single-use")
}

func TestMemoryNonceStoreBindsNonceToAddress(t *testing.T) {
	t.Parallel()

	store := NewMemoryNonceStore()
	ctx := context.Background()

	require.NoError(t, store.Issue(ctx, "0xaaaa", "nonce-2", time.Minute))

	ok, err := store.Consume(ctx, "0xbbbb", "nonce-2")
	require.NoError(t, err)
	assert.False(t, ok, "a nonce issued to one address must not be consumable by another")

	// The failed attempt burns it, so the right address cannot retry with a
	// nonce an attacker may have probed.
	ok, err = store.Consume(ctx, "0xaaaa", "nonce-2")
	require.NoError(t, err)
	assert.False(t, ok, "a wrong-address attempt must burn the nonce")
}

func TestMemoryNonceStoreExpires(t *testing.T) {
	t.Parallel()

	store := NewMemoryNonceStore()
	ctx := context.Background()

	require.NoError(t, store.Issue(ctx, "0xaaaa", "nonce-3", time.Millisecond))
	time.Sleep(5 * time.Millisecond)

	ok, err := store.Consume(ctx, "0xaaaa", "nonce-3")
	require.NoError(t, err)
	assert.False(t, ok, "an expired nonce must not be consumable")
}

func TestMemoryNonceStoreAddressCaseInsensitive(t *testing.T) {
	t.Parallel()

	store := NewMemoryNonceStore()
	ctx := context.Background()

	require.NoError(t, store.Issue(ctx, "0xAbCd", "nonce-4", time.Minute))
	ok, err := store.Consume(ctx, "0xabcd", "nonce-4")
	require.NoError(t, err)
	assert.True(t, ok, "checksum-cased and lower-cased forms are the same address")
}

func TestMemoryNonceStoreRejectsBadInput(t *testing.T) {
	t.Parallel()

	store := NewMemoryNonceStore()
	ctx := context.Background()

	require.Error(t, store.Issue(ctx, "", "n", time.Minute))
	require.Error(t, store.Issue(ctx, "0xaaaa", "", time.Minute))
	require.Error(t, store.Issue(ctx, "0xaaaa", "n", 0))

	ok, err := store.Consume(ctx, "0xaaaa", "never-issued")
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestMemoryNonceStoreConcurrentConsumeHasOneWinner is the T08 acceptance test
// in-process: 100 goroutines race to consume one nonce and exactly one wins.
func TestMemoryNonceStoreConcurrentConsumeHasOneWinner(t *testing.T) {
	t.Parallel()

	store := NewMemoryNonceStore()
	ctx := context.Background()
	const address = "0xrace"

	require.NoError(t, store.Issue(ctx, address, "nonce-race", time.Minute))

	var wins atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if ok, err := store.Consume(ctx, address, "nonce-race"); err == nil && ok {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), wins.Load(), "exactly one concurrent consumer may win")
}

// --------------------------------------------------------------------------- //
// Redis (shared store)
// --------------------------------------------------------------------------- //

// redisNonceClient returns a client when the Redis integration gate is on,
// otherwise nil so the test skips.
func redisNonceClient(t *testing.T) redis.UniversalClient {
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
	require.NoError(t, client.Ping(context.Background()).Err(), "redis must be reachable")
	return client
}

// TestRedisNonceStoreTwoInstancesOneWinner is the T08 acceptance test across
// instances: two stores sharing one Redis race 100 consumers, and exactly one
// wins. Separate processes do not share memory, so this is the property the
// in-process store cannot provide.
func TestRedisNonceStoreTwoInstancesOneWinner(t *testing.T) {
	client := redisNonceClient(t)
	if client == nil {
		t.Skip("set CHAINPULSE_RUN_REDIS_INTEGRATION=1 and CHAINPULSE_TEST_REDIS_URL")
	}

	ctx := context.Background()
	const address = "0xredisrace"
	nonce := fmt.Sprintf("nonce-%d", time.Now().UnixNano())

	instanceA, err := NewRedisNonceStore(client)
	require.NoError(t, err)
	instanceB, err := NewRedisNonceStore(client)
	require.NoError(t, err)

	require.NoError(t, instanceA.Issue(ctx, address, nonce, time.Minute))
	t.Cleanup(func() { _ = client.Del(ctx, nonceKey(nonce)).Err() })

	stores := []NonceStore{instanceA, instanceB}

	var wins atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if ok, err := stores[i%2].Consume(ctx, address, nonce); err == nil && ok {
				wins.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), wins.Load(), "exactly one consumer across instances may win")
}

func TestRedisNonceStoreBindsNonceToAddress(t *testing.T) {
	client := redisNonceClient(t)
	if client == nil {
		t.Skip("set CHAINPULSE_RUN_REDIS_INTEGRATION=1 and CHAINPULSE_TEST_REDIS_URL")
	}

	ctx := context.Background()
	nonce := fmt.Sprintf("nonce-bind-%d", time.Now().UnixNano())

	store, err := NewRedisNonceStore(client)
	require.NoError(t, err)
	require.NoError(t, store.Issue(ctx, "0xaaaa", nonce, time.Minute))
	t.Cleanup(func() { _ = client.Del(ctx, nonceKey(nonce)).Err() })

	ok, err := store.Consume(ctx, "0xbbbb", nonce)
	require.NoError(t, err)
	assert.False(t, ok, "another address must not consume the nonce")

	// Burned by the failed attempt, exactly like the in-process store.
	ok, err = store.Consume(ctx, "0xaaaa", nonce)
	require.NoError(t, err)
	assert.False(t, ok)
}
