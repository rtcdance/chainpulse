package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

type statsFailureLog struct {
	ports.Logger
	fields []any
}

func (l *statsFailureLog) Error(_ string, fields ...any) { l.fields = append([]any(nil), fields...) }

type canonicalStatsFailure struct{ mockEventReader }

func (*canonicalStatsFailure) CanonicalStatsEnabled() bool { return true }
func (*canonicalStatsFailure) CanonicalEventStats(context.Context) (int64, map[string]int64, map[string]int64, int64, error) {
	return 0, nil, nil, 0, context.DeadlineExceeded
}

func TestCanonicalStatsFailureLogsCauseWithoutExposingIt(t *testing.T) {
	t.Parallel()
	logger := &statsFailureLog{}
	handler := NewStatsHandler(logger, &canonicalStatsFailure{})
	w := httptest.NewRecorder()
	handler.HandleGetStats(w, httptest.NewRequest(http.MethodGet, "/events/stats", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "STATS_FAILED") || strings.Contains(w.Body.String(), context.DeadlineExceeded.Error()) {
		t.Fatalf("stats error contract changed: status=%d body=%s", w.Code, w.Body.String())
	}
	if len(logger.fields) != 2 || logger.fields[0] != "error" || logger.fields[1] != context.DeadlineExceeded.Error() {
		t.Fatalf("operator log must retain a JSON-readable cause: %#v", logger.fields)
	}
}

type mockEventReader struct {
	countEventsErr error
}

func (m *mockEventReader) GetEvent(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*blockchain.BlockchainEvent, bool, error) {
	return nil, false, nil
}

func (m *mockEventReader) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventReader) CountEvents(ctx context.Context) (int64, error) {
	if m.countEventsErr != nil {
		return 0, m.countEventsErr
	}
	return 0, nil
}

func (m *mockEventReader) GetEventStats(ctx context.Context) (map[string]int64, map[string]int64, int64, error) {
	return nil, nil, 0, nil
}

func (m *mockEventReader) Health(ctx context.Context) *core.HealthStatus {
	return nil
}

func TestNewStatsHandler(t *testing.T) {
	t.Parallel()

	h := NewStatsHandler(nil, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestStatsHandler_HandleGetStats_CountEventsError(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	reader := &mockEventReader{countEventsErr: context.DeadlineExceeded}
	h := NewStatsHandler(logger, reader)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	h.HandleGetStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestStatsHandler_HandleGetStats_Empty(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	reader := &mockEventReader{}
	h := NewStatsHandler(logger, reader)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	h.HandleGetStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
