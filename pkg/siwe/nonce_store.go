package siwe

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// NonceStore issues and consumes SIWE nonces.
//
// Consumption is the security boundary, so it MUST be atomic AND bound to the
// address. The T08 acceptance is "same nonce, two instances, 100 concurrent →
// exactly one success", which a check-then-delete store cannot satisfy: two
// requests can both observe the nonce before either deletes it, and both then
// believe they consumed it.
type NonceStore interface {
	// Issue binds a nonce to an address for ttl.
	Issue(ctx context.Context, address, nonce string, ttl time.Duration) error

	// Consume atomically removes the nonce IF AND ONLY IF it is still bound to
	// this address.
	//
	// It returns false — not an error — for an unknown, already-consumed,
	// expired or differently-bound nonce, so a caller cannot use the error to
	// tell "wrong address" from "already used".
	Consume(ctx context.Context, address, nonce string) (bool, error)
}

// normalizeAddress lower-cases so checksum-cased and lower-cased forms of the
// same address are one binding.
func normalizeAddress(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

// nonceKey namespaces the binding in a shared store.
func nonceKey(nonce string) string {
	return "siwe:nonce:" + nonce
}

// --------------------------------------------------------------------------- //
// In-process store (single instance)
// --------------------------------------------------------------------------- //

// MemoryNonceStore is an atomic in-process store. It is correct for ONE
// instance; a deployment with several instances needs RedisNonceStore, because
// separate processes do not share this map.
//
// Memory is reclaimed opportunistically by Issue rather than by a background
// goroutine: a goroutine would need a stop interface, and a store that is
// dropped without being stopped leaks it. An issued-but-never-redeemed nonce
// is removed the next time the map has grown past its sweep threshold.
type MemoryNonceStore struct {
	mu     sync.Mutex
	nonces map[string]memoryNonce
	// nextSweep is the map size at which Issue sweeps expired entries. It
	// doubles after each sweep, so sweeping stays amortised O(1) per Issue
	// even when every entry is still live.
	nextSweep int
}

type memoryNonce struct {
	address   string
	expiresAt time.Time
}

// initialSweepThreshold is the smallest map size that triggers a sweep. Below
// it the waste is bounded by a few thousand small structs, which is cheaper
// than scanning on every Issue.
const initialSweepThreshold = 1024

// NewMemoryNonceStore creates an empty in-process store.
func NewMemoryNonceStore() *MemoryNonceStore {
	return &MemoryNonceStore{
		nonces:    make(map[string]memoryNonce),
		nextSweep: initialSweepThreshold,
	}
}

// Issue implements NonceStore.
func (s *MemoryNonceStore) Issue(_ context.Context, address, nonce string, ttl time.Duration) error {
	if address == "" || nonce == "" {
		return fmt.Errorf("address and nonce are required")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.nonces) >= s.nextSweep {
		s.sweepLocked()
	}

	s.nonces[nonce] = memoryNonce{
		address:   normalizeAddress(address),
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// sweepLocked drops expired entries. The caller must hold mu.
func (s *MemoryNonceStore) sweepLocked() {
	now := time.Now()
	for nonce, entry := range s.nonces {
		if now.After(entry.expiresAt) {
			delete(s.nonces, nonce)
		}
	}
	s.nextSweep = len(s.nonces) * 2
	if s.nextSweep < initialSweepThreshold {
		s.nextSweep = initialSweepThreshold
	}
}

// Len reports how many bindings are held, expired ones included. It exists so
// tests can assert that the store is bounded.
func (s *MemoryNonceStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.nonces)
}

// Consume implements NonceStore. The delete happens BEFORE the comparison, so a
// nonce is single-use whatever the outcome: a wrong-address attempt cannot be
// retried with the right address.
func (s *MemoryNonceStore) Consume(_ context.Context, address, nonce string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.nonces[nonce]
	if !ok {
		return false, nil
	}
	delete(s.nonces, nonce)

	if time.Now().After(entry.expiresAt) {
		return false, nil
	}
	return entry.address == normalizeAddress(address), nil
}

// --------------------------------------------------------------------------- //
// Redis store (shared across instances)
// --------------------------------------------------------------------------- //

// RedisNonceStore shares nonce bindings across instances.
type RedisNonceStore struct {
	client redis.UniversalClient
}

// consumeNonceScript compares the stored address with the presented one and
// deletes the key in the same round trip. Doing the compare and the delete
// separately is exactly the check-then-act window this closes.
//
// KEYS[1] = nonce key, ARGV[1] = normalized address.
const consumeNonceScript = `
local stored = redis.call('GET', KEYS[1])
if stored == false then
  return 0
end
redis.call('DEL', KEYS[1])
if stored == ARGV[1] then
  return 1
end
return 0
`

// NewRedisNonceStore wraps a Redis client.
func NewRedisNonceStore(client redis.UniversalClient) (*RedisNonceStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	return &RedisNonceStore{client: client}, nil
}

// Issue implements NonceStore.
func (s *RedisNonceStore) Issue(ctx context.Context, address, nonce string, ttl time.Duration) error {
	if address == "" || nonce == "" {
		return fmt.Errorf("address and nonce are required")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}

	if err := s.client.Set(ctx, nonceKey(nonce), normalizeAddress(address), ttl).Err(); err != nil {
		return fmt.Errorf("issue nonce: %w", err)
	}
	return nil
}

// Consume implements NonceStore using the atomic Lua compare-and-delete.
func (s *RedisNonceStore) Consume(ctx context.Context, address, nonce string) (bool, error) {
	if nonce == "" {
		return false, nil
	}

	result, err := s.client.Eval(ctx, consumeNonceScript, []string{nonceKey(nonce)}, normalizeAddress(address)).Int64()
	if err != nil {
		return false, fmt.Errorf("consume nonce: %w", err)
	}
	return result == 1, nil
}
