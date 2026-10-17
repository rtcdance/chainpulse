//go:build integration

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/services/reorg"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// goroutineBudget is the §8 allowance: after repeated start/stop cycles the
// goroutine count must return to the baseline plus at most this many, with the
// leftovers explainable as fixed background workers.
const goroutineBudget = 20

const workerCycles = 10

// emptyOutbox is an OutboxDrainer with nothing pending, so the loop under test
// does no work and the only thing being measured is its own lifecycle.
//
// It counts its calls so the liveness check below can observe the loop ticking
// directly. The previous version inferred liveness from
// `runtime.NumGoroutine() > beforeCycle`, which is a global count: a goroutine
// finishing anywhere else in the process offsets the one that just started, and
// the check then fails claiming the worker "is not running" when it is. That
// made the test fail intermittently under -race and under load.
type emptyOutbox struct{ calls atomic.Int64 }

func (o *emptyOutbox) PendingNotifications(context.Context, int) ([]reorg.Notification, error) {
	o.calls.Add(1)
	return nil, nil
}
func (o *emptyOutbox) MarkPublished(context.Context, int64) error { return nil }
func (o *emptyOutbox) RecordFailure(context.Context, int64) error { return nil }

// noopPublisher accepts whatever the drainer publishes.
type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, string, []byte) error { return nil }

// topFramesFromStackDump names each goroutine by its outermost frame.
//
// pprof.Profile does not expose its stack traces, so the text dump from
// runtime.Stack is used instead: each block starts with "goroutine N [state]:"
// and the following line is the frame the goroutine is sitting in. Naming the
// leftovers is the point — a bare count is not actionable.
func topFramesFromStackDump(dump string) []string {
	lines := strings.Split(dump, "\n")
	var frames []string
	for index, line := range lines {
		if !strings.HasPrefix(line, "goroutine ") {
			continue
		}
		if index+1 >= len(lines) {
			continue
		}
		top := strings.TrimSpace(lines[index+1])
		if top != "" {
			frames = append(frames, top)
		}
	}
	sort.Strings(frames)
	return frames
}

// TestBackgroundWorkersReturnToBaseline is the §8 resource claim: "after 10
// start/stop rounds the goroutine count returns to the baseline plus at most 20,
// and any remainder is explained as a fixed background worker".
//
// The component under test is the reorg outbox drainer, one of the application's
// long-lived workers: it owns a goroutine for its lifetime and must release it on
// Stop. It needs no database, so a leak here is attributable to the worker rather
// than to a connection pool.
//
// It also writes the §10 goroutine profile artifact.
func TestBackgroundWorkersReturnToBaseline(t *testing.T) {
	logger := testhelpers.NewTestLogger()

	// Let any goroutines from earlier tests finish before taking the baseline.
	settle()

	baseline := runtime.NumGoroutine()

	for i := 0; i < workerCycles; i++ {
		outbox := &emptyOutbox{}
		loop, err := reorg.NewOutboxDrainerLoop(outbox, noopPublisher{}, logger)
		if err != nil {
			t.Fatalf("cycle %d: NewOutboxDrainerLoop: %v", i, err)
		}
		loop.WithInterval(5 * time.Millisecond).WithBatchSize(1)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- loop.Run(ctx) }()

		// Positive control: the loop must actually be alive before it is stopped.
		// Without this the test would also pass if Run returned immediately, and
		// "no goroutine leaked" would be trivially true because none ever ran.
		//
		// Liveness is observed from the drainer itself: a running loop polls the
		// outbox. That is a direct signal, so no unrelated goroutine coming or
		// going can turn it into a false negative.
		alive := false
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case err := <-done:
				t.Fatalf("cycle %d: Run returned before Stop (err=%v); the loop never stayed running, "+
					"so this cycle proves nothing about leaking", i, err)
			default:
			}
			if outbox.calls.Load() > 0 {
				alive = true
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		if !alive {
			t.Fatalf("cycle %d: the drainer never polled its outbox; the worker under test is not running", i)
		}

		// Let it tick so the loop is genuinely running, not just started.
		time.Sleep(20 * time.Millisecond)

		loop.Stop()
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("cycle %d: the drainer did not return after Stop", i)
		}
	}

	settle()
	after := runtime.NumGoroutine()
	delta := after - baseline

	report := map[string]any{
		"schema":                "chainpulse.goroutine-rehearsal.v1",
		"cycles":                workerCycles,
		"net_goroutines_before": baseline,
		"net_goroutines_after":  after,
		"delta":                 delta,
		"budget":                goroutineBudget,
		"component":             "pkg/services/reorg OutboxDrainerLoop (Start/Stop x10)",
	}
	defer writeGoroutineReport(t, report)

	buf := make([]byte, 1<<20)
	dumpLength := runtime.Stack(buf, true)
	if dumpLength == 0 {
		t.Error("could not capture the goroutine dump, so the claim cannot be substantiated")
	} else {
		leftovers := topFramesFromStackDump(string(buf[:dumpLength]))
		report["remaining_top_frames"] = leftovers

		if delta > goroutineBudget {
			unique := map[string]int{}
			for _, frame := range leftovers {
				unique[frame]++
			}
			t.Errorf("goroutines did not return to the baseline: before=%d after=%d delta=%d (budget %d)\nremaining:%s",
				baseline, after, delta, goroutineBudget, formatCounts(unique))
		}
	}

	t.Logf("goroutines: before=%d after=%d delta=%d (budget %d)", baseline, after, delta, goroutineBudget)
}

func settle() {
	for i := 0; i < 3; i++ {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
	}
}

func formatCounts(counts map[string]int) string {
	lines := make([]string, 0, len(counts))
	for frame, n := range counts {
		lines = append(lines, fmt.Sprintf("  %3d  %s", n, frame))
	}
	sort.Strings(lines)
	return "\n" + strings.Join(lines, "\n")
}

// writeGoroutineReport writes the §10 goroutine profile plus the rehearsal
// numbers, on failure too.
func writeGoroutineReport(t *testing.T, report map[string]any) {
	t.Helper()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Logf("warning: cannot resolve the project root: %v", err)
		return
	}
	dir := "uncommitted"
	if sha := gitRevision(root); len(sha) >= 12 {
		dir = sha[:12]
	}
	out := filepath.Join(root, "build", "acceptance", dir)

	profilePath := filepath.Join(out, "profiles", "workers.goroutine.pprof")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o750); err != nil {
		t.Logf("warning: cannot create the profiles directory: %v", err)
		return
	}
	if profile := pprof.Lookup("goroutine"); profile != nil {
		file, err := os.Create(profilePath)
		if err != nil {
			t.Logf("warning: cannot create %s: %v", profilePath, err)
		} else if err := profile.WriteTo(file, 1); err != nil {
			t.Logf("warning: cannot write the goroutine profile: %v", err)
			_ = file.Close()
		} else {
			_ = file.Close()
			t.Logf("goroutine profile written to %s", profilePath)
		}
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Logf("warning: cannot encode the goroutine report: %v", err)
		return
	}
	reportPath := filepath.Join(out, "goroutine-report.json")
	if err := os.WriteFile(reportPath, append(encoded, '\n'), 0o600); err != nil {
		t.Logf("warning: cannot write %s: %v", reportPath, err)
	}
}
