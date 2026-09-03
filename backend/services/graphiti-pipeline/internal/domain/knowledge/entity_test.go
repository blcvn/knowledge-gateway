package knowledge

import (
	"testing"
	"time"
)

func TestExtractedEdge_Validate(t *testing.T) {
	now := time.Now()
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)

	tests := []struct {
		name    string
		edge    ExtractedEdge
		wantErr error
	}{
		{
			name: "valid edge without invalid_at",
			edge: ExtractedEdge{ValidAt: now},
			wantErr: nil,
		},
		{
			name: "valid edge with invalid_at after valid_at",
			edge: ExtractedEdge{ValidAt: before, InvalidAt: &now},
			wantErr: nil,
		},
		{
			name: "invalid edge missing valid_at",
			edge: ExtractedEdge{},
			wantErr: ErrInvalidEdgeValidAt,
		},
		{
			name: "invalid edge valid_at after invalid_at",
			edge: ExtractedEdge{ValidAt: after, InvalidAt: &now},
			wantErr: ErrInvalidEdgeTimeWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()
			if err != tt.wantErr {
				t.Errorf("ExtractedEdge.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbeddingVector_Validate(t *testing.T) {
	vec := EmbeddingVector{1.0, 2.0, 3.0}
	if err := vec.Validate(3); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if err := vec.Validate(4); err == nil {
		t.Errorf("Expected error for invalid dimension")
	}
}

func TestPromptTemplate_Validate(t *testing.T) {
	valid := PromptTemplate{Name: "test", Template: "{{.Input}}"}
	if err := valid.Validate(); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	invalid := PromptTemplate{Name: "", Template: "test"}
	if err := invalid.Validate(); err == nil {
		t.Errorf("Expected error for missing name")
	}
}
