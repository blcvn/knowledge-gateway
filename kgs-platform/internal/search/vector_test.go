package search

import "testing"

func TestResolveResultID_PrefersPayloadID(t *testing.T) {
	got := resolveResultID(
		"4f70a9a6-0f7f-4d9c-9a93-b18625067b33",
		map[string]any{"id": "doc-1_US-01"},
	)

	if got != "doc-1_US-01" {
		t.Fatalf("expected payload id, got %q", got)
	}
}

func TestResolveResultID_FallbackToPointID(t *testing.T) {
	pointID := "4f70a9a6-0f7f-4d9c-9a93-b18625067b33"
	got := resolveResultID(pointID, map[string]any{})

	if got != pointID {
		t.Fatalf("expected fallback point id, got %q", got)
	}
}
