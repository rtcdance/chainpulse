package api

import "testing"

func TestBuildSingleEventQueryResponse(t *testing.T) {
	meta := &QueryMeta{Source: "domain-query", QueryPath: "domain-first"}
	response := buildSingleEventQueryResponse(map[string]interface{}{"id": "evt-1"}, meta)
	if response == nil {
		t.Fatal("expected response")
	}
	if response.Pagination != nil {
		t.Fatalf("expected no pagination for single event response, got %+v", response.Pagination)
	}
	if response.Meta != meta {
		t.Fatalf("expected meta pointer to be preserved")
	}
	if response.Timestamp <= 0 {
		t.Fatalf("expected positive timestamp, got %d", response.Timestamp)
	}
}

func TestBuildPaginatedEventQueryResponse(t *testing.T) {
	meta := &QueryMeta{Source: "event-retrieval", QueryPath: "retrieval-list"}
	data := []interface{}{"a", "b"}
	response := buildPaginatedEventQueryResponse(data, 20, 5, 42, meta)
	if response == nil {
		t.Fatal("expected response")
	}
	if response.Meta != meta {
		t.Fatalf("expected meta pointer to be preserved")
	}
	if response.Pagination == nil {
		t.Fatal("expected pagination")
	}
	if response.Pagination.Limit != 20 || response.Pagination.Offset != 5 || response.Pagination.Total != 42 {
		t.Fatalf("unexpected pagination: %+v", response.Pagination)
	}
	if response.Timestamp <= 0 {
		t.Fatalf("expected positive timestamp, got %d", response.Timestamp)
	}
}
