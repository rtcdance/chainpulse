package api

import "testing"

func TestBuildEventQueryMetaFromInputDerivesSharedFields(t *testing.T) {
	meta := buildEventQueryMetaFromInput(eventQueryMetaInput{
		Source:                "event-retrieval",
		QueryPath:             "retrieval-list",
		MetadataCompleteness:  "partial",
		MetadataAttachedCount: 1,
		MetadataMissingCount:  1,
		ResultCount:           2,
	})

	if meta == nil {
		t.Fatal("expected meta")
	}
	if meta.QuerySourcePosture != "retrieval-service" {
		t.Fatalf("expected retrieval-service, got %q", meta.QuerySourcePosture)
	}
	if meta.MetadataCoveragePosture != "coverage-partial" {
		t.Fatalf("expected coverage-partial, got %q", meta.MetadataCoveragePosture)
	}
	if meta.ConsistencyPosture != "retrieval-partial" {
		t.Fatalf("expected retrieval-partial, got %q", meta.ConsistencyPosture)
	}
	if meta.QueryReliabilityHint != "served with partial metadata coverage; verify metadata completeness before relying on full event context" {
		t.Fatalf("unexpected reliability hint: %q", meta.QueryReliabilityHint)
	}
}

func TestBuildEventQueryMetaFromInputPreservesExplicitSourcePosture(t *testing.T) {
	meta := buildEventQueryMetaFromInput(eventQueryMetaInput{
		Source:                "cache",
		QuerySourcePosture:    "cache-hit",
		QueryPath:             "domain-list",
		MetadataCompleteness:  "none",
		MetadataAttachedCount: 0,
		MetadataMissingCount:  3,
		ResultCount:           3,
	})

	if meta == nil {
		t.Fatal("expected meta")
	}
	if meta.QuerySourcePosture != "cache-hit" {
		t.Fatalf("expected explicit source posture cache-hit, got %q", meta.QuerySourcePosture)
	}
	if meta.ConsistencyPosture != "query-service-direct" {
		t.Fatalf("expected query-service-direct, got %q", meta.ConsistencyPosture)
	}
	if meta.QueryReliabilityHint != "served from query-service cache; verify freshness expectations before treating as latest" {
		t.Fatalf("unexpected reliability hint: %q", meta.QueryReliabilityHint)
	}
}
