package search

import "testing"

func TestApplyFilters(t *testing.T) {
	items := []Result{
		{
			ID:    "n1",
			Label: "Requirement",
			Properties: map[string]any{
				"domain":          "payments",
				"confidence":      0.9,
				"provenance_type": "document",
			},
			Score: 1,
		},
		{
			ID:    "n2",
			Label: "UseCase",
			Properties: map[string]any{
				"domain":          "hr",
				"confidence":      0.4,
				"provenance_type": "manual",
			},
			Score: 0.5,
		},
	}

	got := ApplyFilters(items, Options{
		EntityTypes:     []string{"Requirement"},
		Domains:         []string{"payments"},
		MinConfidence:   0.8,
		ProvenanceTypes: []string{"document"},
	})
	if len(got) != 1 || got[0].ID != "n1" {
		t.Fatalf("unexpected filtered result: %#v", got)
	}

	got = ApplyFilters(items, Options{})
	if len(got) != 2 {
		t.Fatalf("empty filter should keep all, got=%d", len(got))
	}

	got = ApplyFilters(items, Options{Domains: []string{"unknown"}})
	if len(got) != 0 {
		t.Fatalf("all-excluded filter should be empty, got=%d", len(got))
	}
}
