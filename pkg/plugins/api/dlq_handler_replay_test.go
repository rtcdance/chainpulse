package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Minimal in-memory SQL driver so the DLQ replay path can be exercised
// (SELECT succeeds, UPDATE fails) without a real database or an external
// mocking dependency.

var errFakeExec = errors.New("fake update failure")

func init() { sql.Register("fake-dlq", fakeDLQDriver{}) }

type fakeDLQDriver struct{}

func (fakeDLQDriver) Open(string) (driver.Conn, error) { return fakeDLQConn{}, nil }

type fakeDLQConn struct{}

func (fakeDLQConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (fakeDLQConn) Close() error              { return nil }
func (fakeDLQConn) Begin() (driver.Tx, error) { return nil, errors.New("begin unsupported") }

func (fakeDLQConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return &fakeDLQRows{
		cols: []string{"id", "chain_id", "original_event_id", "error_message", "retry_count", "status", "created_at", "updated_at"},
		rows: [][]driver.Value{{
			"dlq-1", "1", "evt-1", "boom", int64(0), "pending",
			"2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
		}},
	}, nil
}

// ExecContext always fails, simulating the UPDATE not being applied.
func (fakeDLQConn) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return nil, errFakeExec
}

type fakeDLQRows struct {
	cols []string
	rows [][]driver.Value
	i    int
}

func (r *fakeDLQRows) Columns() []string { return r.cols }
func (r *fakeDLQRows) Close() error      { return nil }
func (r *fakeDLQRows) Next(dest []driver.Value) error {
	if r.i >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.i])
	r.i++
	return nil
}

type fakePublisher struct{ published int }

func (p *fakePublisher) Publish(context.Context, string, []byte) error {
	p.published++
	return nil
}

// TestDLQHandler_HandleReplayDLQEvents_UpdateFailureDoesNotCount is the
// regression test for the inflated `replayed` counter: when the UPDATE that
// marks an event as replayed fails, the event must be counted as failed, not
// as replayed.
func TestDLQHandler_HandleReplayDLQEvents_UpdateFailureDoesNotCount(t *testing.T) {
	db, err := sql.Open("fake-dlq", "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	defer func() { _ = db.Close() }()

	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	pub := &fakePublisher{}
	h := NewDLQHandler(db, pub, logger, metrics)

	req := httptest.NewRequest(http.MethodPost, "/dlq/replay", strings.NewReader(`{"all": true}`))
	rec := httptest.NewRecorder()
	h.HandleReplayDLQEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body["replayed"]; got != float64(0) {
		t.Errorf("replayed: expected 0 when the UPDATE fails, got %v", got)
	}
	if got := body["failed"]; got != float64(1) {
		t.Errorf("failed: expected 1 when the UPDATE fails, got %v", got)
	}
	if pub.published != 1 {
		t.Errorf("expected 1 publish attempt, got %d", pub.published)
	}
}
