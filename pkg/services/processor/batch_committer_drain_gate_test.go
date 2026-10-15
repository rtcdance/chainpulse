package processor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"

	"github.com/rtcdance/chainpulse/pkg/services/processor"
)

var errLeaseHeld = errors.New("lease is held by another owner")

type drainGateTx struct{}

func (drainGateTx) StoreEvent(context.Context, any) error                        { return nil }
func (drainGateTx) SaveCheckpoint(context.Context, string, uint64, string) error { return nil }
func (drainGateTx) Commit(context.Context) error                                 { return nil }
func (drainGateTx) Rollback(context.Context) error                               { return nil }

type drainGateStore struct{ owner string }

func (s *drainGateStore) AcquireLease(_ context.Context, _, ownerToken string, _ time.Duration) (int64, error) {
	if s.owner != "" && s.owner != ownerToken {
		return 0, errLeaseHeld
	}
	s.owner = ownerToken
	return 1, nil
}

func (s *drainGateStore) ReleaseLease(_ context.Context, _, ownerToken string) error {
	if s.owner == ownerToken {
		s.owner = ""
	}
	return nil
}

func (s *drainGateStore) BeginFencedTx(_ context.Context, _, ownerToken string) (portcontract.Tx, error) {
	if s.owner != ownerToken {
		return nil, errLeaseHeld
	}
	return drainGateTx{}, nil
}

// TestBatchCommitterAcceptsDrainControllerAsGate pins the wiring: the real
// shutdown gate must satisfy the interface the committer expects. It lives in
// the external test package because an internal test importing bootstrap would
// form an import cycle (bootstrap imports processor).
func TestBatchCommitterAcceptsDrainControllerAsGate(t *testing.T) {
	t.Parallel()

	var gate processor.WorkGate = bootstrap.NewDrainController()
	c, err := processor.NewBatchCommitter(&drainGateStore{}, gate, "1", "worker-A", time.Minute)
	require.NoError(t, err)

	require.NoError(t, c.CommitBlock(context.Background(), 1, "0xa", nil))
	require.True(t, gate.(*bootstrap.DrainController).Drain(time.Second))
	require.ErrorIs(t, c.CommitBlock(context.Background(), 2, "0xb", nil), processor.ErrDraining)
}
