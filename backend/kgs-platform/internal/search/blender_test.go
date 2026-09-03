package search

import "testing"

func TestBlend(t *testing.T) {
	semantic := []Result{
		{ID: "n1", Score: 0.9},
		{ID: "n2", Score: 0.7},
	}
	text := []Result{
		{ID: "n1", Score: 0.2},
		{ID: "n3", Score: 0.8},
	}

	alphaZero := Blend(semantic, text, 0)
	for _, item := range alphaZero {
		if item.ID == "n2" && item.Score != 0 {
			t.Fatalf("alpha=0 should suppress semantic-only score, got %f", item.Score)
		}
	}

	alphaOne := Blend(semantic, text, 1)
	for _, item := range alphaOne {
		if item.ID == "n3" && item.Score != 0 {
			t.Fatalf("alpha=1 should suppress text-only score, got %f", item.Score)
		}
	}

	mixed := Blend(semantic, text, 0.5)
	if len(mixed) != 3 {
		t.Fatalf("expected 3 results, got %d", len(mixed))
	}
	found := false
	for _, item := range mixed {
		if item.ID == "n1" {
			found = true
			if item.Score <= 0.5 || item.Score >= 0.6 {
				t.Fatalf("unexpected blended score for n1: %f", item.Score)
			}
		}
	}
	if !found {
		t.Fatalf("expected n1 in blended result")
	}
}
