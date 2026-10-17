package reorg

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// statusJobStore is a consistent in-memory JobStore: CompleteJob flips the
// job's status, so PendingJobs stops returning it. The repair fakes in
// reorg_repair_test.go record calls but do not model that state transition,
// which the multi-pass Service loop needs.
type statusJobStore struct {
	mu    sync.Mutex
	jobs  map[string]*Job
	order []string
	err   error
}

func newStatusJobStore() *statusJobStore {
	return &statusJobStore{jobs: make(map[string]*Job)}
}

func (s *statusJobStore) EnqueueJob(_ context.Context, job Job, _ []Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	if _, exists := s.jobs[job.ID]; exists {
		return nil
	}
	stored := job
	s.jobs[job.ID] = &stored
	s.order = append(s.order, job.ID)
	return nil
}

func (s *statusJobStore) PendingJobs(_ context.Context, chainID string, limit int) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Job
	for _, id := range s.order {
		j := s.jobs[id]
		if j.Status == JobDone {
			continue
		}
		if chainID != "" && j.ChainID != chainID {
			continue
		}
		out = append(out, *j)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *statusJobStore) CompleteJob(_ context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[jobID]
	if !ok {
		return nil
	}
	j.Status = JobDone
	return nil
}

func (s *statusJobStore) doneCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, j := range s.jobs {
		if j.Status == JobDone {
			n++
		}
	}
	return n
}

// flakyPublisher publishes until fail is set, then errors. Toggling fail models
// a publisher outage followed by a restart with the publisher healthy again.
type flakyPublisher struct {
	mu        sync.Mutex
	fail      bool
	published []string
}

func (p *flakyPublisher) Publish(_ context.Context, topic string, _ []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fail {
		return errors.New("publisher unavailable")
	}
	p.published = append(p.published, topic)
	return nil
}

func (p *flakyPublisher) setFail(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fail = v
}

func (p *flakyPublisher) topics() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.published...)
}

// TestServiceStartRepairsJobsAndDrainsOutbox is the T06 wiring proof: the
// exported entry point must actually run BOTH loops — resume the pending
// repair job and deliver its owed notifications — not merely construct them.
func TestServiceStartRepairsJobsAndDrainsOutbox(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	jobs := newStatusJobStore()
	require.NoError(t, jobs.EnqueueJob(context.Background(), Job{
		ID:         "ethereum:100:105",
		ChainID:    "ethereum",
		Generation: 1,
		FromBlock:  100,
		ToBlock:    105,
		Status:     JobPending,
	}, nil))

	rec := &repairRecorder{}
	outbox := &fakeOutbox{pending: []Notification{
		{ID: 1, JobID: "ethereum:100:105", ChainID: "ethereum", Generation: 1, Topic: "reorg.detected", Payload: []byte(`{}`)},
		{ID: 2, JobID: "ethereum:100:105", ChainID: "ethereum", Generation: 1, Topic: "reorg.rollback", Payload: []byte(`{}`)},
	}}
	publisher := &fakePublisher{}

	repair, err := NewRepair(jobs, &fakeRollback{rec: rec, n: 1}, &fakeCheckpoints{rec: rec}, nil, logger)
	require.NoError(t, err)
	drainer, err := NewOutboxDrainerLoop(outbox, publisher, logger)
	require.NoError(t, err)
	svc, err := NewService(repair, drainer, logger)
	require.NoError(t, err)

	svc.Start(context.Background())
	require.Eventually(t, func() bool {
		return jobs.doneCount() == 1 && outbox.pendingCount() == 0
	}, 2*time.Second, 5*time.Millisecond,
		"Start must wire both the repair resume and the outbox drain")

	svc.Stop()
	publisher.mu.Lock()
	gotTopics := append([]string(nil), publisher.published...)
	publisher.mu.Unlock()
	assert.Equal(t, []string{"reorg.detected", "reorg.rollback"}, gotTopics)
}

// TestServiceStopIsGraceful pins shutdown semantics: Stop ends Run with nil,
// not an error, and is safe to call twice.
func TestServiceStopIsGraceful(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	rec := &repairRecorder{}
	repair, err := NewRepair(newStatusJobStore(), &fakeRollback{rec: rec, n: 1}, &fakeCheckpoints{rec: rec}, nil, logger)
	require.NoError(t, err)
	drainer, err := NewOutboxDrainerLoop(&fakeOutbox{}, &fakePublisher{}, logger)
	require.NoError(t, err)
	svc, err := NewService(repair, drainer, logger)
	require.NoError(t, err)
	svc.WithInterval(5 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- svc.Run(context.Background()) }()

	svc.Stop()
	svc.Stop()
	select {
	case runErr := <-done:
		assert.NoError(t, runErr, "Stop is a graceful shutdown, not an error")
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

func TestNewServiceRequiresBothLoops(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	rec := &repairRecorder{}
	repair, err := NewRepair(newStatusJobStore(), &fakeRollback{rec: rec, n: 1}, &fakeCheckpoints{rec: rec}, nil, logger)
	require.NoError(t, err)
	drainer, err := NewOutboxDrainerLoop(&fakeOutbox{}, &fakePublisher{}, logger)
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		repair  *Repair
		drainer *OutboxDrainerLoop
		logger  portcontract.Logger
	}{
		{"nil repair", nil, drainer, logger},
		{"nil drainer", repair, nil, logger},
		{"nil logger", repair, drainer, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewService(tc.repair, tc.drainer, tc.logger)
			require.Error(t, err)
		})
	}
}
