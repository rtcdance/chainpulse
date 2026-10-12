package query

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// fakeCache is a core.CachePlugin whose Get can be made to hit, miss or fail.
type fakeCache struct {
	value []byte
	err   error
	gets  int
}

func (c *fakeCache) Name() string { return "fake-cache" }
func (c *fakeCache) Get(_ context.Context, _ string) ([]byte, error) {
	c.gets++
	return c.value, c.err
}
func (c *fakeCache) Set(context.Context, string, []byte, int) error { return nil }
func (c *fakeCache) Delete(context.Context, string) error           { return nil }
func (c *fakeCache) GetStats() core.CacheStats                      { return core.CacheStats{} }
func (c *fakeCache) HealthCheck(context.Context) error              { return nil }

var _ core.CachePlugin = (*fakeCache)(nil)

type fakeSource struct {
	value []byte
	err   error
	loads int
}

func (s *fakeSource) load(context.Context, string) ([]byte, error) {
	s.loads++
	return s.value, s.err
}

func newReader(t *testing.T, cache *fakeCache, source *fakeSource) *DegradingReader {
	t.Helper()
	r, err := NewDegradingReader(cache, source.load, core.NewDefaultLogger(core.LogLevelError))
	require.NoError(t, err)
	return r
}

func TestDegradingReaderServesFromCacheOnHit(t *testing.T) {
	t.Parallel()

	cache := &fakeCache{value: []byte("cached")}
	source := &fakeSource{value: []byte("source")}
	r := newReader(t, cache, source)

	result, err := r.Read(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("cached"), result.Value)
	assert.False(t, result.Degraded, "a cache hit is not a degradation")
	assert.Zero(t, source.loads, "the source must not be consulted on a hit")
}

func TestDegradingReaderFallsThroughOnMiss(t *testing.T) {
	t.Parallel()

	cache := &fakeCache{} // nil value, nil error == miss
	source := &fakeSource{value: []byte("source")}
	r := newReader(t, cache, source)

	result, err := r.Read(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("source"), result.Value)
	assert.False(t, result.Degraded, "a miss is a normal path, not a degradation")
	assert.Equal(t, 1, source.loads)
}

// TestDegradingReaderDegradesOnCacheError is the E09 "Redis degraded query"
// property: an unavailable cache must cost latency, not the query.
func TestDegradingReaderDegradesOnCacheError(t *testing.T) {
	t.Parallel()

	cache := &fakeCache{err: errors.New("connection refused")}
	source := &fakeSource{value: []byte("from-source")}
	r := newReader(t, cache, source)

	result, err := r.Read(context.Background(), "k")
	require.NoError(t, err, "a cache outage must NOT fail the query")
	assert.Equal(t, []byte("from-source"), result.Value)
	assert.True(t, result.Degraded, "the degradation must be reported, not silent")
	assert.Contains(t, result.Reason, "connection refused", "the reason must name the cause")
	assert.Equal(t, 1, source.loads)
}

// TestDegradingReaderFailsOnlyWhenBothFail: the reader may only fail when the
// answer genuinely cannot be produced.
func TestDegradingReaderFailsOnlyWhenBothFail(t *testing.T) {
	t.Parallel()

	cache := &fakeCache{err: errors.New("connection refused")}
	source := &fakeSource{err: errors.New("database down")}
	r := newReader(t, cache, source)

	_, err := r.Read(context.Background(), "k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache unavailable")
	assert.Contains(t, err.Error(), "database down", "both causes must be reported")
}

func TestDegradingReaderSourceFailureAfterMissIsAnError(t *testing.T) {
	t.Parallel()

	cache := &fakeCache{}
	source := &fakeSource{err: errors.New("database down")}
	r := newReader(t, cache, source)

	_, err := r.Read(context.Background(), "k")
	require.Error(t, err, "a miss with a failing source cannot be answered")
}

func TestDegradingReaderRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	r := newReader(t, &fakeCache{}, &fakeSource{})
	_, err := r.Read(context.Background(), "")
	require.Error(t, err)
}

func TestDegradingReaderRequiresDependencies(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	load := func(context.Context, string) ([]byte, error) { return nil, nil }

	_, err := NewDegradingReader(nil, load, logger)
	require.Error(t, err)

	_, err = NewDegradingReader(&fakeCache{}, nil, logger)
	require.Error(t, err)

	_, err = NewDegradingReader(&fakeCache{}, load, nil)
	require.Error(t, err)
}
