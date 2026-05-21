package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
	"go.mongodb.org/mongo-driver/mongo"
)

type mockHealthDatabaseManager struct{}

type mockRolloutReportProducer struct {
	report *RolloutReportDetails
}

var _ database.DatabaseManager = (*mockHealthDatabaseManager)(nil)

func (m *mockHealthDatabaseManager) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockHealthDatabaseManager) GetMongoClient(ctx context.Context) (any, error) {
	return nil, nil
}

func (m *mockHealthDatabaseManager) GetMongoDatabase(name string) *mongo.Database {
	return nil
}

func (m *mockHealthDatabaseManager) GetPostgresDB(ctx context.Context) (any, error) {
	return nil, nil
}

func (m *mockHealthDatabaseManager) CheckMongoHealth(ctx context.Context) error {
	return nil
}

func (m *mockHealthDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	return nil
}

func (m *mockHealthDatabaseManager) Health(ctx context.Context) any {
	return map[string]any{"status": "healthy"}
}

func (m *mockHealthDatabaseManager) Close(ctx context.Context) error {
	return nil
}

func (m *mockRolloutReportProducer) BuildRolloutReport(ctx context.Context) *RolloutReportDetails {
	return m.report
}

func TestHealthCheckHandlerIncludesRuntimeComponent(t *testing.T) {
	t.Parallel()
	handler := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	handler.SetRuntimeComponentProvider(func(ctx context.Context) *ComponentStatus {
		return &ComponentStatus{
			Name:      "Indexing Runtime",
			Status:    "healthy",
			Timestamp: 123,
			Details: map[string]any{
				"ownership_mode":      "shadow",
				"shadow_owned_events": int64(4),
				"legacy_owned_events": int64(7),
				"ownership_chains":    2,
			},
		}
	})

	response := handler.performHealthCheck(context.Background())
	component, ok := response.Components["indexing_runtime"]
	if !ok {
		t.Fatalf("expected indexing_runtime component in response")
	}
	if component.Status != "healthy" {
		t.Fatalf("expected healthy runtime component, got %s", component.Status)
	}
	if got := component.Details["ownership_mode"]; got != "shadow" {
		t.Fatalf("expected ownership_mode shadow, got %v", got)
	}
	if got := component.Details["shadow_owned_events"]; got != int64(4) {
		t.Fatalf("expected shadow_owned_events 4, got %v", got)
	}
	if got := component.Details["legacy_owned_events"]; got != int64(7) {
		t.Fatalf("expected legacy_owned_events 7, got %v", got)
	}
	if got := component.Details["ownership_chains"]; got != 2 {
		t.Fatalf("expected ownership_chains 2, got %v", got)
	}
}

func TestHealthCheckHandlerRuntimeComponentProviderInvalidatesCache(t *testing.T) {
	t.Parallel()
	handler := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	handler.SetRuntimeComponentProvider(func(ctx context.Context) *ComponentStatus {
		return &ComponentStatus{
			Name:      "Indexing Runtime",
			Status:    "healthy",
			Timestamp: 1,
			Details: map[string]any{
				"ownership_mode": "legacy-only",
			},
		}
	})
	first := handler.performHealthCheck(context.Background())
	if got := first.Components["indexing_runtime"].Details["ownership_mode"]; got != "legacy-only" {
		t.Fatalf("expected initial ownership mode legacy-only, got %v", got)
	}

	handler.SetRuntimeComponentProvider(func(ctx context.Context) *ComponentStatus {
		return &ComponentStatus{
			Name:      "Indexing Runtime",
			Status:    "healthy",
			Timestamp: 2,
			Details: map[string]any{
				"ownership_mode": "runtime-owned",
			},
		}
	})
	second := handler.performHealthCheck(context.Background())
	if got := second.Components["indexing_runtime"].Details["ownership_mode"]; got != "runtime-owned" {
		t.Fatalf("expected refreshed ownership mode runtime-owned, got %v", got)
	}
}

func TestHealthCheckHandlerHandleReadyIncludesReadinessDetails(t *testing.T) {
	t.Parallel()
	handler := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}
	handler.SetReadinessDetailsProvider(func(ctx context.Context) map[string]any {
		return map[string]any{
			"ownership_mode":                  "shadow",
			"rollout_ready_for_runtime_owned": false,
			"rollout_status":                  "shadow-observe",
			"rollout_reason":                  "shared runtime still coexists with legacy writes",
			"rollout_gate_decision":           "hold",
			"rollout_gate_reason":             "shared runtime still coexists with legacy writes",
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.HandleReady(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response ReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if !response.Ready {
		t.Fatalf("expected infrastructure readiness to remain true")
	}
	if got := response.Details["ownership_mode"]; got != "shadow" {
		t.Fatalf("expected ownership_mode shadow, got %v", got)
	}
	if got := response.Details["rollout_ready_for_runtime_owned"]; got != false {
		t.Fatalf("expected rollout_ready_for_runtime_owned false, got %v", got)
	}
	if got := response.Details["rollout_status"]; got != "shadow-observe" {
		t.Fatalf("expected rollout_status shadow-observe, got %v", got)
	}
	if got := response.Details["rollout_gate_decision"]; got != "hold" {
		t.Fatalf("expected rollout_gate_decision hold, got %v", got)
	}
}

func TestHealthCheckHandlerHandleRolloutIncludesReport(t *testing.T) {
	t.Parallel()
	handler := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}
	handler.SetRolloutReportProvider(func(ctx context.Context) *RolloutReportDetails {
		return &RolloutReportDetails{
			ReportID:       "monolithic-ownership-rollout-runtime",
			SchemaFamily:   "ownership-rollout-report",
			ReportVersion:  "v1",
			Service:        "monolithic",
			ReportScope:    "ownership-rollout",
			ReportSource:   "monolithic",
			ReportMode:     "runtime",
			DeploymentMode: "monolithic",
			GeneratedAt:    int64(1700000000),
			Mode:           "shadow",
			Progression: RolloutReportStateReason{
				State: "observe",
			},
			CutoverCandidate: RolloutReportCandidate{
				Eligible: false,
			},
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()

	handler.HandleRollout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response RolloutReportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if !response.Available {
		t.Fatal("expected rollout report to be available")
	}
	if got := response.Details.ReportID; got != "monolithic-ownership-rollout-runtime" {
		t.Fatalf("expected report_id monolithic-ownership-rollout-runtime, got %v", got)
	}
	if got := response.Details.SchemaFamily; got != "ownership-rollout-report" {
		t.Fatalf("expected schema_family ownership-rollout-report, got %v", got)
	}
	if got := response.Details.ReportVersion; got != "v1" {
		t.Fatalf("expected report_version v1, got %v", got)
	}
	if got := response.Details.Service; got != "monolithic" {
		t.Fatalf("expected service monolithic, got %v", got)
	}
	if got := response.Details.ReportScope; got != "ownership-rollout" {
		t.Fatalf("expected report_scope ownership-rollout, got %v", got)
	}
	if got := response.Details.ReportSource; got != "monolithic" {
		t.Fatalf("expected report_source monolithic, got %v", got)
	}
	if got := response.Details.ReportMode; got != "runtime" {
		t.Fatalf("expected report_mode runtime, got %v", got)
	}
	if got := response.Details.DeploymentMode; got != "monolithic" {
		t.Fatalf("expected deployment_mode monolithic, got %v", got)
	}
	if got := response.Details.GeneratedAt; got != int64(1700000000) {
		t.Fatalf("expected generated_at 1700000000, got %v", got)
	}
	if got := response.Details.Mode; got != "shadow" {
		t.Fatalf("expected ownership_mode shadow, got %v", got)
	}
	if got := response.Details.Progression.State; got != "observe" {
		t.Fatalf("expected rollout_effective_state observe, got %v", got)
	}
}

func TestHealthCheckHandlerHandleRolloutUnavailableWithoutProvider(t *testing.T) {
	t.Parallel()
	handler := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()

	handler.HandleRollout(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var response RolloutReportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if response.Available {
		t.Fatal("expected rollout report to be unavailable")
	}
	if response.Status != "unavailable" {
		t.Fatalf("expected status unavailable, got %s", response.Status)
	}
}

func TestHealthCheckHandlerHandleRolloutWithProducer(t *testing.T) {
	t.Parallel()
	handler := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}
	handler.SetRolloutReportProducer(&mockRolloutReportProducer{
		report: &RolloutReportDetails{
			ReportID:       "producer-report",
			SchemaFamily:   "ownership-rollout-report",
			ReportVersion:  "v1",
			Service:        "monolithic",
			ReportScope:    "ownership-rollout",
			ReportSource:   "producer",
			ReportMode:     "runtime",
			DeploymentMode: "monolithic",
			GeneratedAt:    1700000001,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()

	handler.HandleRollout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response RolloutReportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if response.Details == nil {
		t.Fatal("expected rollout details")
	}
	if got := response.Details.ReportID; got != "producer-report" {
		t.Fatalf("expected producer-report, got %q", got)
	}
	if got := response.Details.ReportSource; got != "producer" {
		t.Fatalf("expected producer source, got %q", got)
	}
}
