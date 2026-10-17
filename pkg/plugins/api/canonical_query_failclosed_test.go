package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"

	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type canonicalLookupStub struct{ mockDomainQueryService }

func (canonicalLookupStub) AuthoritativeQuery() bool { return true }

func TestCanonicalDetailNeverFallsBackOnEmptyOrError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{{"empty", nil, http.StatusNotFound}, {"outage", errors.New("database unavailable"), http.StatusServiceUnavailable}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := NewEventQueryHandler(nil, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector())
			handler.initialized = true
			handler.SetDomainQueryService(&canonicalLookupStub{mockDomainQueryService{queryByHash: func(context.Context, string) (*blockchain.BlockchainEvent, error) { return nil, tc.err }}})
			recorder := httptest.NewRecorder()
			handler.HandleGetEventByID(recorder, httptest.NewRequest("GET", "/events/id", nil), "id")
			require.Equal(t, tc.status, recorder.Code, "a legacy fallback would dereference the absent retrieval service")
		})
	}
}

type unavailableCanonicalStats struct{ mockEventReader }

func (unavailableCanonicalStats) CanonicalStatsEnabled() bool { return true }
func (unavailableCanonicalStats) CanonicalEventStats(context.Context) (int64, map[string]int64, map[string]int64, int64, error) {
	return 0, nil, nil, 0, errors.New("database unavailable")
}

func TestCanonicalStatsNeverReportsEmptySuccessOnFailure(t *testing.T) {
	t.Parallel()
	handler := NewStatsHandler(testhelpers.NewTestLogger(), &unavailableCanonicalStats{})
	recorder := httptest.NewRecorder()
	handler.HandleGetStats(recorder, httptest.NewRequest("GET", "/events/stats", nil))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
