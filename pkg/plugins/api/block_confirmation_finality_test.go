package api

import (
	"context"
	"errors"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
)

// fakeFinalityChecker returns a canned verdict so the tracker's handling of
// each confidence level can be asserted without an RPC node.
type fakeFinalityChecker struct {
	status *finality.FinalityResult
	err    error
	calls  int
}

func (f *fakeFinalityChecker) GetFinalizedBlockNumber(_ context.Context, _ string) (uint64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.status.FinalizedBlockNumber, nil
}

func (f *fakeFinalityChecker) GetFinalizedBlockNumberWithStatus(_ context.Context, _ string) (*finality.FinalityResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.status, nil
}

func (f *fakeFinalityChecker) IsBlockFinalized(_ context.Context, _ string, blockNumber uint64) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.status.Degraded || f.status.Confidence != finality.ConfidenceFinalized {
		return false, nil
	}
	return blockNumber <= f.status.FinalizedBlockNumber, nil
}

func (f *fakeFinalityChecker) IsBlockFinalizedWithStatus(_ context.Context, _ string, blockNumber uint64) (*finality.FinalityResult, error) {
	status, err := f.GetFinalizedBlockNumberWithStatus(context.Background(), "")
	if err != nil {
		return nil, err
	}
	status.IsFinalized = blockNumber <= status.FinalizedBlockNumber
	return status, nil
}

// TestConfirmationCutoff pins the T07 / F08 rule as a pure function: a finality
// verdict may only raise the confirmation cutoff when it carries a real
// guarantee. This is the decision that used to be made inline, from a bare
// height, so that an L2 "safe" or a "latest" fallback promoted events the chain
// had not committed to.
func TestConfirmationCutoff(t *testing.T) {
	t.Parallel()

	const depth = int64(1000)

	cases := []struct {
		name   string
		status *finality.FinalityResult
		want   int64
	}{
		{
			name:   "no verdict keeps the depth-based cutoff",
			status: nil,
			want:   depth,
		},
		{
			name: "a genuinely finalized height raises the cutoff",
			status: &finality.FinalityResult{
				Confidence:           finality.ConfidenceFinalized,
				Source:               finality.SourceFinalizedTag,
				FinalizedBlockNumber: 5000,
			},
			want: 5000,
		},
		{
			name: "an L2 safe height must not raise the cutoff",
			status: &finality.FinalityResult{
				Degraded:             true,
				Confidence:           finality.ConfidenceSafe,
				Source:               finality.SourceSafeTag,
				FinalizedBlockNumber: 5000,
			},
			want: depth,
		},
		{
			name: "a latest-tag fallback must not raise the cutoff",
			status: &finality.FinalityResult{
				Degraded:             true,
				Confidence:           finality.ConfidenceDegraded,
				Source:               finality.SourceLatestFallback,
				FinalizedBlockNumber: 5000,
			},
			want: depth,
		},
		{
			name: "an unknown confidence must not raise the cutoff",
			status: &finality.FinalityResult{
				Confidence:           finality.ConfidenceUnknown,
				FinalizedBlockNumber: 5000,
			},
			want: depth,
		},
		{
			name: "a finalized confidence that is itself degraded is still refused",
			status: &finality.FinalityResult{
				Degraded:             true,
				Confidence:           finality.ConfidenceFinalized,
				FinalizedBlockNumber: 5000,
			},
			want: depth,
		},
		{
			name: "a finalized height below the depth cutoff does not lower it",
			status: &finality.FinalityResult{
				Confidence:           finality.ConfidenceFinalized,
				Source:               finality.SourceFinalizedTag,
				FinalizedBlockNumber: 10,
			},
			want: depth,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := confirmationCutoff(depth, tc.status); got != tc.want {
				t.Errorf("confirmationCutoff(%d, %+v) = %d, want %d", depth, tc.status, got, tc.want)
			}
		})
	}
}

func newTrackerWithFinality(t *testing.T, fc finality.FinalityChecker) *BlockConfirmationTracker {
	t.Helper()

	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, NewSubscriptionHub(), nil)
	tracker.SetFinalityChecker(fc)
	return tracker
}

// TestBlockConfirmationTracker_AsksForConfidenceNotJustHeight is the wiring
// half of T07: the tracker must consume the confidence-carrying accessor. If it
// reverted to GetFinalizedBlockNumber it would be unable to tell a finalized
// height from a "safe" or "latest" one, and the pure-function test above would
// no longer protect anything. The counter here is bumped only by the
// status-carrying method, so a revert shows up as zero calls.
func TestBlockConfirmationTracker_AsksForConfidenceNotJustHeight(t *testing.T) {
	t.Parallel()

	fc := &fakeFinalityChecker{status: &finality.FinalityResult{
		Confidence: finality.ConfidenceFinalized,
		Source:     finality.SourceFinalizedTag,
		// Height 0 on purpose: head=10 with 12 confirmations gives a depth
		// cutoff of -2, and a finalized height of 0 does not raise it, so the
		// call returns before dereferencing the (nil) database.
		FinalizedBlockNumber: 0,
	}}
	tracker := newTrackerWithFinality(t, fc)

	tracker.transitionPendingEvents(context.Background(), "ethereum", 10, 12)

	if fc.calls != 1 {
		t.Errorf("expected the tracker to ask for the finality status once, got %d", fc.calls)
	}
}

// TestBlockConfirmationTracker_IgnoresDegradedFinality is the negative half:
// with a degraded verdict the cutoff must stay depth-based. head=100 with 12
// confirmations gives a depth cutoff of 88, so the call proceeds to the
// database — which is nil here. Reaching it panics, so the assertion is that a
// degraded verdict never gets that far.
func TestBlockConfirmationTracker_IgnoresDegradedFinality(t *testing.T) {
	t.Parallel()

	fc := &fakeFinalityChecker{status: &finality.FinalityResult{
		Confidence:           finality.ConfidenceSafe,
		Source:               finality.SourceSafeTag,
		Degraded:             true,
		FinalizedBlockNumber: 5000,
	}}
	tracker := newTrackerWithFinality(t, fc)

	// head=10, confirmations=12 → depth cutoff -2 → must return before the DB.
	// Before the fix the degraded height 5000 was adopted as the cutoff, so this
	// call went on to dereference the nil *sql.DB.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a degraded finality verdict was used as the cutoff: %v", r)
		}
	}()
	tracker.transitionPendingEvents(context.Background(), "ethereum", 10, 12)
}

// TestBlockConfirmationTracker_ToleratesFinalityError pins that a finality
// failure degrades to depth-based confirmation instead of blocking ingestion.
func TestBlockConfirmationTracker_ToleratesFinalityError(t *testing.T) {
	t.Parallel()

	fc := &fakeFinalityChecker{err: errors.New("rpc unreachable")}
	tracker := newTrackerWithFinality(t, fc)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a finality error must fall back to depth-based confirmation: %v", r)
		}
	}()
	tracker.transitionPendingEvents(context.Background(), "ethereum", 10, 12)

	if fc.calls != 1 {
		t.Errorf("expected one finality attempt, got %d", fc.calls)
	}
}
