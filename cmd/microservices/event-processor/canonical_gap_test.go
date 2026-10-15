package main

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/require"
)

type gapRuntime struct{ result error }

func (r gapRuntime) ProcessCompleteBlock(context.Context, appindexing.CompletedBlock) error {
	return r.result
}

func (gapRuntime) Health(context.Context) (core.HealthStatus, error) {
	return core.HealthStatus{Status: "healthy"}, nil
}
func (gapRuntime) Status() appindexing.RuntimeStatus { return appindexing.RuntimeStatus{} }

type gapAuditStore struct {
	calls  int
	result error
}

func (s *gapAuditStore) RecordOrphanedBlock(context.Context, string, string, []byte) error {
	s.calls++
	return s.result
}
func (*gapAuditStore) CompleteReorgsThrough(context.Context, string, uint64) error { return nil }

func TestCanonicalGapOrphanIsAuditedButCanonicalGapNeverAcknowledges(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"orphan-gap", "canonical-gap", "rpc-unavailable", "audit-unavailable", "storage-failure"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			frame := appindexing.CompletedBlock{Schema: appindexing.CompletedBlockSchema, ChainID: "31337", Header: &types.Header{Number: big.NewInt(12), Difficulty: big.NewInt(0)}}
			audit := &gapAuditStore{}
			gap := &appindexing.BlockGapError{Block: 12, Checkpoint: 9}
			var processErr error = gap
			if name == "storage-failure" {
				processErr = errors.New("database commit failed")
			}
			if name == "audit-unavailable" {
				audit.result = errors.New("audit unavailable")
			}
			probes := 0
			processor := durableProcessor{runtime: gapRuntime{result: processErr}, store: audit, validateCanonical: func(context.Context, string, *blockchain.Block) error {
				probes++
				if name == "rpc-unavailable" {
					return errors.New("canonical RPC unavailable")
				}
				if name == "canonical-gap" {
					return nil
				}
				return &bootstrap.NonCanonicalBlockError{ChainID: frame.ChainID, Hash: frame.Header.Hash()}
			}}
			err := processor.ProcessCompleteBlock(context.Background(), frame)
			var orphan *appindexing.OrphanedBlockError
			if name == "orphan-gap" {
				require.ErrorAs(t, err, &orphan)
				require.Equal(t, 1, audit.calls)
			} else {
				require.Error(t, err)
				require.False(t, errors.As(err, &orphan))
				if name == "audit-unavailable" {
					require.Equal(t, 1, audit.calls)
				} else {
					require.Zero(t, audit.calls)
				}
			}
			if name == "storage-failure" {
				require.Zero(t, probes)
			} else {
				require.Equal(t, 1, probes)
			}
			if name == "canonical-gap" {
				require.ErrorIs(t, err, gap)
			}
		})
	}
}
