package processor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// TestCheckAndMark_ConcurrentSingleWinner is the regression test for the
// idempotency TOCTOU: IsDuplicate takes an RLock and MarkProcessed takes a
// separate Lock, so concurrent callers can all observe "not a duplicate" and
// all proceed to process the same event.
//
// CheckAndMark must make check-and-insert a single atomic critical section, so
// exactly one caller wins and every other caller sees a duplicate.
func TestCheckAndMark_ConcurrentSingleWinner(t *testing.T) {
	svc := NewDefaultIdempotencyService(testhelpers.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	if err := svc.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = svc.Stop() }()

	const goroutines = 64
	var winners, errs int64

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			duplicate, err := svc.CheckAndMark(context.Background(), "same-hash")
			if err != nil {
				atomic.AddInt64(&errs, 1)
				return
			}
			if !duplicate {
				atomic.AddInt64(&winners, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if errs != 0 {
		t.Fatalf("unexpected errors: %d", errs)
	}
	if winners != 1 {
		t.Errorf("expected exactly 1 non-duplicate winner, got %d — check-and-mark is not atomic", winners)
	}
}

// TestCheckAndMark_Sequential documents the non-concurrent contract.
func TestCheckAndMark_Sequential(t *testing.T) {
	svc := NewDefaultIdempotencyService(testhelpers.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	if err := svc.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = svc.Stop() }()

	dup, err := svc.CheckAndMark(context.Background(), "h1")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if dup {
		t.Error("first call must not report a duplicate")
	}

	dup, err = svc.CheckAndMark(context.Background(), "h1")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !dup {
		t.Error("second call must report a duplicate")
	}
}
