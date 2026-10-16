package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
	"github.com/stretchr/testify/require"
)

type finalityReaderFunc func(context.Context, string, string) (*blockchain.FinalityEvidence, error)

func (f finalityReaderFunc) ObserveFinality(ctx context.Context, chain, tag string) (*blockchain.FinalityEvidence, error) {
	return f(ctx, chain, tag)
}

type finalityWriterFunc func(context.Context, string, string, *blockchain.FinalityEvidence) error

func (f finalityWriterFunc) RecordFinalityObservation(ctx context.Context, chain, tag string, proof *blockchain.FinalityEvidence) error {
	return f(ctx, chain, tag, proof)
}

func TestFinalityPollDegradesWithoutInventingProof(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		rpcErr, storeErr error
		nilProof         bool
		wantKnown        bool
		wantCalls        int
	}{
		{name: "known", wantKnown: true, wantCalls: 1},
		{name: "RPC unavailable", rpcErr: errors.New("RPC down"), wantCalls: 1},
		{name: "head not committed", storeErr: errors.New("head ahead of checkpoint"), wantCalls: 2},
		{name: "nil RPC result", nilProof: true, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			reader := finalityReaderFunc(func(context.Context, string, string) (*blockchain.FinalityEvidence, error) {
				if tc.nilProof {
					return nil, nil
				}
				return &blockchain.FinalityEvidence{Number: 10}, tc.rpcErr
			})
			writer := finalityWriterFunc(func(_ context.Context, chain, tag string, proof *blockchain.FinalityEvidence) error {
				require.Equal(t, "1", chain)
				require.Equal(t, "safe", tag)
				calls++
				if proof != nil {
					return tc.storeErr
				}
				require.False(t, tc.wantKnown, "known result must never be replaced with degraded")
				return nil
			})
			known, err := observeFinalityOnce(context.Background(), reader, writer, "1", "safe")
			require.Equal(t, tc.wantKnown, known)
			require.Equal(t, tc.wantCalls, calls)
			if tc.wantKnown {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestFinalityObserverLifecycleIsBoundedAndCancellable(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		var calls atomic.Int64
		go func() {
			defer close(done)
			runFinalityPollLoop(ctx, 5*time.Second, func(context.Context) { calls.Add(1) })
		}()
		synctest.Wait()
		require.Equal(t, int64(1), calls.Load())
		time.Sleep(5 * time.Second) // Virtual time inside synctest, no wall-clock sleep.
		synctest.Wait()
		require.Equal(t, int64(2), calls.Load())
		cancel()
		synctest.Wait()
		<-done
		time.Sleep(10 * time.Second)
		synctest.Wait()
		require.Equal(t, int64(2), calls.Load(), "no poll may survive cancellation")
	})
}

func TestFinalityObserverConfirmationsDoNotAssertConsensus(t *testing.T) {
	t.Parallel()
	registry, err := finality.NewRegistry(finality.Strategy{Mode: finality.StrategyConfirmations, Confirmations: 12}, nil)
	require.NoError(t, err)
	stop, err := StartFinalityObserver(context.Background(), nil, FinalityObserverOptions{Chains: []string{"1"}, Registry: registry}, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, stop)
	stop()
	stop()
	tagRegistry, err := finality.NewRegistry(finality.Strategy{Mode: finality.StrategyTag, Tag: "safe"}, nil)
	require.NoError(t, err)
	_, err = StartFinalityObserver(context.Background(), nil, FinalityObserverOptions{Chains: []string{"1"}, Registry: tagRegistry}, nil, nil)
	require.ErrorContains(t, err, "durable canonical")
}
