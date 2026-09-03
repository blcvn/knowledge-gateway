package outbox

import (
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
)

func TestDiffStaleEntitiesDetectsMissingAndStale(t *testing.T) {
	pg := map[string]int{
		"e1": 2,
		"e2": 1,
		"e3": 3,
	}
	neo := map[string]int{
		"e1": 1,
		"e2": 1,
	}
	stale := diffStaleEntities(pg, neo)
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale entities, got %d (%v)", len(stale), stale)
	}
	seen := map[string]bool{}
	for _, id := range stale {
		seen[id] = true
	}
	if !seen["e1"] || !seen["e3"] {
		t.Fatalf("expected stale entities e1 and e3, got %v", stale)
	}
}

func TestDiffStaleEntitiesNoDrift(t *testing.T) {
	pg := map[string]int{"e1": 2, "e2": 1}
	neo := map[string]int{"e1": 2, "e2": 1}
	stale := diffStaleEntities(pg, neo)
	if len(stale) != 0 {
		t.Fatalf("expected no stale entities, got %v", stale)
	}
}

func TestEntityToPayload(t *testing.T) {
	entity := &data.KGEntity{
		EntityID:   "e1",
		AppID:      "app-1",
		TenantID:   "tenant-1",
		EntityType: "Requirement",
		Name:       "FR-001",
		Version:    2,
	}
	payload, err := entityToPayload(entity)
	if err != nil {
		t.Fatalf("entityToPayload error: %v", err)
	}
	if payload["EntityID"] != "e1" {
		t.Fatalf("unexpected payload EntityID: %#v", payload["EntityID"])
	}
}
