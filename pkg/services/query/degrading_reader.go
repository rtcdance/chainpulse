package query

import (
	"context"
	"fmt"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
)

// SourceLoader reads a value from the authoritative source (the database).
type SourceLoader func(ctx context.Context, key string) ([]byte, error)

// ReadResult is a value plus how it was obtained.
type ReadResult struct {
	Value []byte
	// Degraded is true when the cache could not be consulted, so the answer
	// came straight from the source. The answer is still correct; it is only
	// the latency and the load on the source that changed.
	Degraded bool
	// Reason explains a degraded read, for diagnostics and for alerting on how
	// long a degradation lasted.
	Reason string
}

// DegradingReader answers a read from the cache when it can, and from the
// authoritative source when it cannot.
//
// This is the E09 "Redis degraded query" behaviour, and it is deliberately the
// OPPOSITE of the security boundary. Two different failure policies are needed:
//
//   - A READ degrades. Losing the cache costs latency, not correctness, so
//     failing the query would turn a cache outage into a service outage.
//   - A SECURITY decision fails closed. A nonce store that reported success
//     while its backend was down would let a nonce be reused (see
//     siwe.RedisNonceStore).
//
// The distinction that matters here: a cache ERROR and a cache MISS are handled
// differently. A miss is a normal path; an error is a degradation, and it is
// reported as such rather than silently swallowed.
type DegradingReader struct {
	cache  portcontract.CachePlugin
	load   SourceLoader
	logger portcontract.Logger
}

// NewDegradingReader creates the reader.
func NewDegradingReader(cache portcontract.CachePlugin, load SourceLoader, logger portcontract.Logger) (*DegradingReader, error) {
	if cache == nil {
		return nil, fmt.Errorf("cache is required")
	}
	if load == nil {
		return nil, fmt.Errorf("source loader is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &DegradingReader{cache: cache, load: load, logger: logger}, nil
}

// Read returns the value for key.
//
// It only returns an error when the answer cannot be produced at all — that is,
// when the source fails too. A cache failure alone is a degraded success.
func (r *DegradingReader) Read(ctx context.Context, key string) (ReadResult, error) {
	if key == "" {
		return ReadResult{}, fmt.Errorf("key is required")
	}

	cached, cacheErr := r.cache.Get(ctx, key)
	if cacheErr == nil && cached != nil {
		return ReadResult{Value: cached}, nil
	}

	if cacheErr != nil {
		// Degrade: the cache is unavailable, the query is not.
		value, sourceErr := r.load(ctx, key)
		if sourceErr != nil {
			return ReadResult{}, fmt.Errorf("cache unavailable (%w) and source failed: %w", cacheErr, sourceErr)
		}

		reason := "cache error: " + cacheErr.Error()
		r.logger.Warn("serving a degraded read", "key", key, "reason", reason)
		return ReadResult{Value: value, Degraded: true, Reason: reason}, nil
	}

	// A cache miss is a normal path, not a degradation.
	value, err := r.load(ctx, key)
	if err != nil {
		return ReadResult{}, fmt.Errorf("source failed after cache miss: %w", err)
	}
	return ReadResult{Value: value}, nil
}
