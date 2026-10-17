package processor

import (
	"context"
	"sync"
	"testing"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/stretchr/testify/require"
)

// recordingFencedStore wraps a fakeStore and appends every fencing operation to
// an ordered log, so a test can assert WHEN lease effects happen relative to
// each other — not just that they eventually happened.
type recordingFencedStore struct {
	inner *fakeStore

	mu  sync.Mutex
	ops []string
}

func (s *recordingFencedStore) record(op string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ops = append(s.ops, op)
}

func (s *recordingFencedStore) operations() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.ops...)
}

func (s *recordingFencedStore) AcquireLease(ctx context.Context, chainID, ownerToken string, ttl time.Duration) (int64, error) {
	s.record("acquire " + ownerToken)
	return s.inner.AcquireLease(ctx, chainID, ownerToken, ttl)
}

func (s *recordingFencedStore) ReleaseLease(ctx context.Context, chainID, ownerToken string) error {
	s.record("release " + ownerToken)
	return s.inner.ReleaseLease(ctx, chainID, ownerToken)
}

func (s *recordingFencedStore) BeginFencedTx(ctx context.Context, chainID, ownerToken string) (portcontract.Tx, error) {
	s.record("begin " + ownerToken)
	return s.inner.BeginFencedTx(ctx, chainID, ownerToken)
}

// TestBatchCommitterRevokedEffectFencesOldOwnerBeforeNextGenerationTakesOver
// pins the DB-effect half of the T05 rebalance contract at unit level (it runs
// under make test-unit / -short with no environment and no skip path):
//
//  1. The Revoked effect (Release) frees the lease and leaves the old owner
//     fenced out — a begin attempt with the revoked token fails before anyone
//     else has taken over.
//  2. RotateOwner (the next generation's Assigned effect) releases with the
//     OLD token before swapping, and only then does the next generation
//     acquire the lease and commit under the new token.
//  3. The old token stays rejected after the takeover.
//
// The transport half — the consumer running Revoked to completion before the
// next generation's Assigned is invoked — is pinned against a real coordinator
// by test/acceptance/kafka_generation_fencing_ordering_test.go; this test pins
// the effect ordering those hooks must produce once invoked.
func TestBatchCommitterRevokedEffectFencesOldOwnerBeforeNextGenerationTakesOver(t *testing.T) {
	t.Parallel()

	inner := &fakeStore{}
	store := &recordingFencedStore{inner: inner}
	committer := newCommitter(t, store, nil) // chain "1", initial owner "worker-A"
	ctx := context.Background()

	// Generation 1: Assigned rotates the initial owner in, then a fenced
	// commit persists under that token.
	require.NoError(t, committer.RotateOwner(ctx, "kafka-gen1"))
	require.NoError(t, committer.CommitBlock(ctx, 10, "0xgen1", testEvents(1)))
	require.Len(t, inner.committed, 1)
	require.Equal(t, "kafka-gen1", inner.owner)

	// Revoked: Release must hand the lease back and fence the old owner
	// immediately — not at TTL expiry, and before any next generation exists.
	require.NoError(t, committer.Release(ctx))
	_, err := store.BeginFencedTx(ctx, "1", "kafka-gen1")
	require.ErrorContains(t, err, "lease is held by another owner",
		"the revoked owner must be rejected before any takeover")
	require.Len(t, inner.committed, 1, "nothing may commit after revocation")

	// Next generation: RotateOwner releases with the OLD token first, then the
	// new token acquires the lease and commits.
	require.NoError(t, committer.RotateOwner(ctx, "kafka-gen2"))
	require.NoError(t, committer.CommitBlock(ctx, 11, "0xgen2", testEvents(1)))
	require.Len(t, inner.committed, 2, "the next generation must commit")
	require.Equal(t, "kafka-gen2", inner.owner)

	// The old token must stay fenced out after the takeover.
	_, err = store.BeginFencedTx(ctx, "1", "kafka-gen1")
	require.ErrorContains(t, err, "lease is held by another owner",
		"the old owner must stay fenced out after the takeover")

	// Exact effect ordering: every lease operation issued by the three
	// production calls above (RotateOwner, CommitBlock, Release), in order.
	// This is what makes release-before-swap and revoke-before-takeover
	// regressions fail here rather than only in the tagged acceptance lane.
	require.Equal(t, []string{
		"release worker-A",   // RotateOwner(gen1): release the initial token before swapping
		"acquire kafka-gen1", // CommitBlock → EnsureLease
		"begin kafka-gen1",   // fenced transaction under gen1
		"release kafka-gen1", // Release (Revoked effect)
		"begin kafka-gen1",   // stale begin attempt → rejected
		"release kafka-gen1", // RotateOwner(gen2): OLD token released before the swap
		"acquire kafka-gen2", // next generation's lease acquisition (revoke already done)
		"begin kafka-gen2",   // fenced transaction under gen2
		"begin kafka-gen1",   // post-takeover begin with the old token → rejected
	}, store.operations())
}
