package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

func newParamValidationHandler(t *testing.T) *EventQueryHandler {
	t.Helper()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}
	handler := NewEventQueryHandler(retrieval, logger, metrics)
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}
	return handler
}

func TestEventQueryHandlerRejectsMalformedNumericParams(t *testing.T) {
	t.Parallel()
	handler := newParamValidationHandler(t)

	for _, tc := range []struct{ name, url string }{
		{"limit_not_a_number", "/events?limit=abc"},
		{"limit_overflow", "/events?limit=99999999999999999999999"},
		{"offset_not_a_number", "/events?offset=xyz"},
		{"from_block_not_a_number", "/events?from_block=abc"},
		{"to_block_fractional", "/events?to_block=1.5"},
		{"from_time_not_a_number", "/events?from_time=abc"},
		{"to_time_not_a_number", "/events?to_time=abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			handler.HandleGetAllEvents(rr, httptest.NewRequest(http.MethodGet, tc.url, nil))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for %q, got %d body=%s", tc.url, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestEventQueryHandlerCorrelatedRejectsMalformedNumericParams(t *testing.T) {
	t.Parallel()
	handler := newParamValidationHandler(t)

	for _, tc := range []struct{ name, url string }{
		{"limit_not_a_number", "/events/correlated/corr-1?limit=abc"},
		{"offset_not_a_number", "/events/correlated/corr-1?offset=xyz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			handler.HandleGetCorrelatedEvents(rr, httptest.NewRequest(http.MethodGet, tc.url, nil), "corr-1")
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for %q, got %d body=%s", tc.url, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestEventQueryHandlerAcceptsValidNumericParams(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}
	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			return &domainquery.Result{
				Events: []blockchain.BlockchainEvent{
					{
						ID:             "param-ok-1",
						BlockNumber:    1,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "cache",
			}, nil
		},
	})
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.HandleGetAllEvents(rr, httptest.NewRequest(http.MethodGet, "/events?limit=25&offset=0&from_block=1&to_block=100", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for valid numeric params, got %d body=%s", rr.Code, rr.Body.String())
	}
}
