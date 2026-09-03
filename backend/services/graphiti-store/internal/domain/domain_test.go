package domain_test

import (
	"testing"
	"time"

	"vnp-memory/services/graphiti-store/domain"
)

func TestEntityNode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    domain.EntityNode
		wantErr bool
	}{
		{
			name:    "valid entity",
			node:    domain.EntityNode{UUID: "abc-123", Name: "Alice", GroupID: "tenant-1"},
			wantErr: false,
		},
		{
			name:    "missing uuid",
			node:    domain.EntityNode{Name: "Alice", GroupID: "tenant-1"},
			wantErr: true,
		},
		{
			name:    "missing name",
			node:    domain.EntityNode{UUID: "abc-123", GroupID: "tenant-1"},
			wantErr: true,
		},
		{
			name:    "missing group_id",
			node:    domain.EntityNode{UUID: "abc-123", Name: "Alice"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodicNode_Validate(t *testing.T) {
	validTime := time.Now()
	tests := []struct {
		name    string
		node    domain.EpisodicNode
		wantErr bool
	}{
		{
			name:    "valid episodic",
			node:    domain.EpisodicNode{UUID: "ep-1", GroupID: "t1", Content: "Hello", ValidAt: validTime},
			wantErr: false,
		},
		{
			name:    "missing content",
			node:    domain.EpisodicNode{UUID: "ep-1", GroupID: "t1", ValidAt: validTime},
			wantErr: true,
		},
		{
			name:    "missing valid_at",
			node:    domain.EpisodicNode{UUID: "ep-1", GroupID: "t1", Content: "Hello"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntityEdge_Validate(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name    string
		edge    domain.EntityEdge
		wantErr bool
	}{
		{
			name: "valid edge",
			edge: domain.EntityEdge{
				UUID: "e1", SourceNodeID: "n1", TargetNodeID: "n2",
				GroupID: "g1", Fact: "Alice works at Acme", ValidAt: now,
			},
			wantErr: false,
		},
		{
			name: "valid edge with invalid_at after valid_at",
			edge: domain.EntityEdge{
				UUID: "e1", SourceNodeID: "n1", TargetNodeID: "n2",
				GroupID: "g1", Fact: "fact", ValidAt: past, InvalidAt: &future,
			},
			wantErr: false,
		},
		{
			name: "invalid_at before valid_at",
			edge: domain.EntityEdge{
				UUID: "e1", SourceNodeID: "n1", TargetNodeID: "n2",
				GroupID: "g1", Fact: "fact", ValidAt: future, InvalidAt: &past,
			},
			wantErr: true,
		},
		{
			name: "missing fact",
			edge: domain.EntityEdge{
				UUID: "e1", SourceNodeID: "n1", TargetNodeID: "n2",
				GroupID: "g1", ValidAt: now,
			},
			wantErr: true,
		},
		{
			name: "missing source node",
			edge: domain.EntityEdge{
				UUID: "e1", TargetNodeID: "n2",
				GroupID: "g1", Fact: "fact", ValidAt: now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntityEdge_IsCurrentlyValid(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	t.Run("nil invalid_at is valid", func(t *testing.T) {
		edge := domain.EntityEdge{InvalidAt: nil}
		if !edge.IsCurrentlyValid(now) {
			t.Error("expected valid when invalid_at is nil")
		}
	})

	t.Run("invalid_at in future is valid", func(t *testing.T) {
		edge := domain.EntityEdge{InvalidAt: &future}
		if !edge.IsCurrentlyValid(now) {
			t.Error("expected valid when invalid_at is in future")
		}
	})

	t.Run("invalid_at in past is invalid", func(t *testing.T) {
		edge := domain.EntityEdge{InvalidAt: &past}
		if edge.IsCurrentlyValid(now) {
			t.Error("expected invalid when invalid_at is in past")
		}
	})
}

func TestEntityEdge_OverlapsWindow(t *testing.T) {
	// Edge valid from Jan 1 to Jan 31
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jan31 := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	edge := domain.EntityEdge{ValidAt: jan1, InvalidAt: &jan31}

	t.Run("window overlaps", func(t *testing.T) {
		from := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
		if !edge.OverlapsWindow(from, to) {
			t.Error("expected overlap")
		}
	})

	t.Run("window before edge", func(t *testing.T) {
		from := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		if edge.OverlapsWindow(from, to) {
			t.Error("expected no overlap")
		}
	})

	t.Run("window after edge", func(t *testing.T) {
		from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
		if edge.OverlapsWindow(from, to) {
			t.Error("expected no overlap")
		}
	})
}

func TestEmbeddingVector_CosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		v := domain.EmbeddingVector{1.0, 0.0, 0.0}
		score, err := v.CosineSimilarity(v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if score < 0.999 {
			t.Errorf("expected ~1.0, got %f", score)
		}
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := domain.EmbeddingVector{1.0, 0.0}
		b := domain.EmbeddingVector{0.0, 1.0}
		score, err := a.CosineSimilarity(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if score > 0.001 || score < -0.001 {
			t.Errorf("expected ~0.0, got %f", score)
		}
	})

	t.Run("dimension mismatch", func(t *testing.T) {
		a := domain.EmbeddingVector{1.0, 0.0}
		b := domain.EmbeddingVector{1.0, 0.0, 0.0}
		_, err := a.CosineSimilarity(b)
		if err == nil {
			t.Error("expected error for dimension mismatch")
		}
	})
}

func TestSearchParams_Validate(t *testing.T) {
	t.Run("valid params", func(t *testing.T) {
		p := domain.SearchParams{GroupID: "g1", Limit: 10}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing group_id", func(t *testing.T) {
		p := domain.SearchParams{Limit: 10}
		if err := p.Validate(); err == nil {
			t.Error("expected error for missing group_id")
		}
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		p := domain.SearchParams{GroupID: "g1", Limit: 500}
		_ = p.Validate()
		if p.Limit != 100 {
			t.Errorf("expected limit capped at 100, got %d", p.Limit)
		}
	})
}
