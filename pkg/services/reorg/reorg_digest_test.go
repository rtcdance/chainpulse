package reorg

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"sync"
	"testing"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/test/acceptance/tools/digest"
)

const (
	simChainKey = "ethereum"
	simHead     = uint64(20)
)

// staticHashProvider serves a fixed canonical view. A zero hash means the
// ancestor is unknown at that height.
type staticHashProvider struct {
	hashes map[uint64]common.Hash
}

func (p staticHashProvider) GetBlockHash(_ context.Context, n uint64) (common.Hash, error) {
	return p.hashes[n], nil
}

func simHash(parts ...string) common.Hash {
	h := sha256.New()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return common.BytesToHash(h.Sum(nil))
}

type simBlock struct {
	height uint64
	hash   common.Hash
	event  digest.Event
}

type simSegment struct {
	blocks []simBlock
}

func buildSimSegment(seed string, from, to uint64) simSegment {
	var seg simSegment
	for h := from; h <= to; h++ {
		height := strconv.FormatUint(h, 10)
		seg.blocks = append(seg.blocks, simBlock{
			height: h,
			hash:   simHash("block", seed, height),
			event: digest.Event{
				Chain:     simChainKey,
				BlockHash: simHash("block", seed, height).Hex(),
				TxHash:    simHash("tx", seed, height).Hex(),
				LogIndex:  0,
				Topics:    []string{simHash("topic", seed).Hex()},
				Data:      "0x" + height,
			},
		})
	}
	return seg
}

func (seg simSegment) concat(other simSegment) simSegment {
	return simSegment{blocks: append(append([]simBlock(nil), seg.blocks...), other.blocks...)}
}

func (seg simSegment) hashes() map[uint64]common.Hash {
	out := make(map[uint64]common.Hash, len(seg.blocks))
	for _, b := range seg.blocks {
		out[b.height] = b.hash
	}
	return out
}

// simStore is an in-memory canonical chain + event store: it implements the
// DatabasePlugin surface the reorg handler touches plus the repair runner's
// EventRollerBacker and CheckpointRewinder ports. Events are soft-deleted
// (marked reorged) exactly like the PostgreSQL adapter.
type simStore struct {
	mu         sync.Mutex
	blocks     map[uint64]*blockchain.Block
	latest     uint64
	events     []*simEvent
	checkpoint map[string]uint64
}

type simEvent struct {
	ev      digest.Event
	height  uint64
	reorged bool
}

func newSimStore() *simStore {
	return &simStore{
		blocks:     make(map[uint64]*blockchain.Block),
		checkpoint: make(map[string]uint64),
	}
}

func (s *simStore) index(seg simSegment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range seg.blocks {
		s.blocks[b.height] = &blockchain.Block{Number: b.height, Hash: b.hash}
		if b.height > s.latest {
			s.latest = b.height
		}
		s.events = append(s.events, &simEvent{ev: b.event, height: b.height})
	}
}

// digestState is the canonical digest (I02 events + I03 blocks) of everything
// not soft-deleted. This is what the reorg pipeline must converge to.
func (s *simStore) digestState(t *testing.T) string {
	t.Helper()
	s.mu.Lock()
	var evs []digest.Event
	for _, e := range s.events {
		if !e.reorged {
			evs = append(evs, e.ev)
		}
	}
	var blks []digest.Block
	for h, b := range s.blocks {
		blks = append(blks, digest.Block{Chain: simChainKey, Height: h, BlockHash: b.Hash.Hex()})
	}
	s.mu.Unlock()

	eventsDigest, err := digest.CanonicalEvents(evs)
	require.NoError(t, err)
	blocksDigest, err := digest.CanonicalBlocks(blks)
	require.NoError(t, err)
	return eventsDigest + "|" + blocksDigest
}

func expectedDigest(t *testing.T, seg simSegment) string {
	t.Helper()
	evs := make([]digest.Event, 0, len(seg.blocks))
	blks := make([]digest.Block, 0, len(seg.blocks))
	for _, b := range seg.blocks {
		evs = append(evs, b.event)
		blks = append(blks, digest.Block{Chain: simChainKey, Height: b.height, BlockHash: b.hash.Hex()})
	}
	eventsDigest, err := digest.CanonicalEvents(evs)
	require.NoError(t, err)
	blocksDigest, err := digest.CanonicalBlocks(blks)
	require.NoError(t, err)
	return eventsDigest + "|" + blocksDigest
}

// DatabasePlugin surface (the subset the reorg handler touches).

func (s *simStore) GetEvent(_ context.Context, _ string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (s *simStore) QueryEvents(_ context.Context, _ any) ([]any, error) { return nil, nil }

func (s *simStore) GetAllEvents(_ context.Context) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (s *simStore) GetEventsByBlockRange(_ context.Context, _, _ uint64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (s *simStore) StoreEvent(_ context.Context, _ any) error { return nil }

func (s *simStore) BatchStoreEvents(_ context.Context, _ []any) error { return nil }

func (s *simStore) DeleteEvent(_ context.Context, _ string) error { return nil }

func (s *simStore) DeleteEventsByBlockRange(_ context.Context, from, to uint64) (int64, error) {
	return s.MarkEventsAsReorged(context.Background(), from, to)
}

func (s *simStore) MarkEventsAsReorged(_ context.Context, from, to uint64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for _, e := range s.events {
		if e.height >= from && e.height <= to && !e.reorged {
			e.reorged = true
			n++
		}
	}
	return n, nil
}

func (s *simStore) GetBlock(_ context.Context, number uint64) (*blockchain.Block, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.blocks[number]
	if !ok {
		return nil, fmt.Errorf("block %d not found", number)
	}
	return b, nil
}

func (s *simStore) GetLatestBlock(_ context.Context) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest, nil
}

func (s *simStore) GetAllBlocks(_ context.Context) ([]*blockchain.Block, error) { return nil, nil }

func (s *simStore) GetReorgStats(_ context.Context) (*portcontract.ReorgStats, error) {
	return &portcontract.ReorgStats{}, nil
}

func (s *simStore) Name() string                                      { return "sim-store" }
func (s *simStore) Version() string                                   { return "test" }
func (s *simStore) Initialize(_ context.Context, _ core.Config) error { return nil }
func (s *simStore) Start(_ context.Context) error                     { return nil }
func (s *simStore) Stop(_ context.Context) error                      { return nil }
func (s *simStore) Health(_ context.Context) error                    { return nil }

// Repair-runner ports.

func (s *simStore) RollbackEvents(ctx context.Context, from, to uint64) (int64, error) {
	return s.MarkEventsAsReorged(ctx, from, to)
}

func (s *simStore) RewindCheckpoint(_ context.Context, chainID string, toBlock uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoint[chainID] = toBlock
	return nil
}

// TestReorgDigestConvergesAfterDepth is the T06 acceptance: reorgs of depth 1,
// 2, and 12 must leave chain state whose canonical digest equals that of a
// chain that never saw the orphaned blocks — including a stale replay of the
// repair job in the middle.
func TestReorgDigestConvergesAfterDepth(t *testing.T) {
	for _, depth := range []uint64{1, 2, 12} {
		t.Run(fmt.Sprintf("depth_%d", depth), func(t *testing.T) {
			t.Parallel()
			runDigestConvergence(t, depth)
		})
	}
}

func runDigestConvergence(t *testing.T, depth uint64) {
	t.Helper()
	fork := simHead - depth + 1

	shared := buildSimSegment("shared", 1, fork-1)
	orphanTail := buildSimSegment("orphan-tail", fork, simHead)
	canonicalTail := buildSimSegment("canonical-tail", fork, simHead)

	orphanChain := shared.concat(orphanTail)
	canonicalChain := shared.concat(canonicalTail)
	expected := expectedDigest(t, canonicalChain)

	store := newSimStore()
	store.index(orphanChain)

	jobs := newStatusJobStore()
	gens := NewMemGenerationStore()
	logger := core.NewDefaultLogger(core.LogLevelError)
	handler := NewReorgHandler(store, logger, 64, 128).
		WithChainID(simChainKey).
		WithJobStore(jobs).
		WithGenerationStore(gens)
	handler.SetBlockHashProvider(staticHashProvider{hashes: canonicalChain.hashes()})

	for _, b := range orphanChain.blocks {
		handler.UpdateBlockHash(b.height, b.hash)
	}

	ctx := context.Background()
	detected, reorgBlock, err := handler.DetectReorg(ctx, simHead, canonicalChain.hashes()[simHead])
	require.NoError(t, err)
	require.True(t, detected, "a replaced head must be detected as a reorg")
	require.Equal(t, fork, reorgBlock, "the divergence point must be the fork height")

	require.NoError(t, handler.HandleReorg(ctx, reorgBlock))

	rec := &repairRecorder{}
	derived := &fakeDerived{rec: rec}
	repair, err := NewRepair(jobs, store, store, derived, logger)
	require.NoError(t, err)

	repaired, err := repair.Resume(ctx, simChainKey, 10)
	require.NoError(t, err)
	require.Equal(t, 1, repaired)

	// (d) Replaying the same repair job must be idempotent: a stale pending
	// copy re-runs every step without changing the converged state.
	stale := Job{
		ID:         ReorgJobID(simChainKey, fork, simHead),
		ChainID:    simChainKey,
		Generation: 1,
		FromBlock:  fork,
		ToBlock:    simHead,
		Status:     JobPending,
	}
	require.NoError(t, repair.RepairJob(ctx, stale))
	pending, err := jobs.PendingJobs(ctx, "", 10)
	require.NoError(t, err)
	require.Empty(t, pending, "the replay must not reopen the completed job")

	// The puller re-indexes the canonical tail after the rollback notification.
	store.index(canonicalTail)

	assert.Equal(t, expected, store.digestState(t),
		"post-reorg state must digest equal to a chain that never saw the orphaned blocks")

	current, err := gens.CurrentGeneration(ctx, simChainKey)
	require.NoError(t, err)
	assert.Equal(t, int64(1), current, "the generation fence must be observable after the reorg")
	assert.Equal(t, int64(1), derived.generation, "derived invalidation must run under the new generation")
	assert.Equal(t, fmt.Sprintf("%d-%d", fork, simHead), derived.rangeSeen)
}

// TestOutboxPublishFailureThenRestartConverges: a publish outage leaves the
// notifications pending (at-least-once), and a restarted drainer delivers them;
// the repair side converges independently of delivery.
func TestOutboxPublishFailureThenRestartConverges(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	ctx := context.Background()

	jobs := newStatusJobStore()
	require.NoError(t, jobs.EnqueueJob(ctx, Job{
		ID:         "ethereum:100:105",
		ChainID:    simChainKey,
		Generation: 1,
		FromBlock:  100,
		ToBlock:    105,
		Status:     JobPending,
	}, nil))

	rec := &repairRecorder{}
	repair, err := NewRepair(jobs, &fakeRollback{rec: rec, n: 3}, &fakeCheckpoints{rec: rec}, nil, logger)
	require.NoError(t, err)

	outbox := &fakeOutbox{pending: []Notification{
		{ID: 1, JobID: "ethereum:100:105", ChainID: simChainKey, Generation: 1, Topic: "reorg.detected", Payload: []byte(`{}`)},
		{ID: 2, JobID: "ethereum:100:105", ChainID: simChainKey, Generation: 1, Topic: "reorg.rollback", Payload: []byte(`{}`)},
	}}
	publisher := &flakyPublisher{fail: true}

	drainer, err := NewOutboxDrainerLoop(outbox, publisher, logger)
	require.NoError(t, err)
	published, drainErr := drainer.DrainOnce(ctx)
	require.Error(t, drainErr, "a publish outage must be reported")
	assert.Zero(t, published)
	assert.Equal(t, 2, outbox.pendingCount(), "undelivered notifications must stay pending across the outage")

	repaired, err := repair.Resume(ctx, "", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, repaired, "the repair side must converge even while publish is down")

	publisher.setFail(false)
	restarted, err := NewOutboxDrainerLoop(outbox, publisher, logger)
	require.NoError(t, err)
	published, err = restarted.DrainOnce(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, published)
	assert.Zero(t, outbox.pendingCount(), "a restarted drainer must converge the outbox")
	assert.Equal(t, []string{"reorg.detected", "reorg.rollback"}, publisher.topics())

	repaired, err = repair.Resume(ctx, "", 10)
	require.NoError(t, err)
	assert.Zero(t, repaired, "the done job must not be repaired again after restart")
}

// TestOverDepthReorgStopsCommitting: a rollback larger than maxRollback must
// error out without committing the rollback or recording a repair job.
func TestOverDepthReorgStopsCommitting(t *testing.T) {
	t.Parallel()

	db := reorgDB(t)
	jobs := &fakeJobStore{}
	handler := NewReorgHandler(db, core.NewDefaultLogger(core.LogLevelError), 12, 5).
		WithChainID(simChainKey).
		WithJobStore(jobs)

	err := handler.HandleReorg(context.Background(), 100) // 105-100+1 = 6 > 5
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reorg too large")
	assert.False(t, db.deleteCalled, "an over-depth reorg must not commit a rollback")
	assert.Empty(t, jobs.jobs, "an over-depth reorg must not record a repair job")
}

// TestMissingAncestorStopsCommitting: when the provider cannot confirm a
// single canonical ancestor, detection must surface an error instead of
// inventing a reorg at the head that HandleReorg would then commit.
func TestMissingAncestorStopsCommitting(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(newMockReorgDatabase())
	handler.SetBlockHashProvider(staticHashProvider{hashes: nil})

	ctx := context.Background()
	if _, _, err := handler.DetectReorg(ctx, 100, common.HexToHash("0xaaa")); err != nil {
		t.Fatalf("seed call must succeed: %v", err)
	}

	reorg, reorgBlock, err := handler.DetectReorg(ctx, 100, common.HexToHash("0xbbb"))
	require.Error(t, err, "an unknown ancestor must surface an error, not a reorg")
	assert.False(t, reorg, "no canonical ancestor must never be reported as a reorg")
	assert.Zero(t, reorgBlock)
	assert.Contains(t, err.Error(), "reorg scan failed")
}
