package domain

import (
	"testing"
	"time"
)

func TestSearchMethod_IsValid(t *testing.T) {
	tests := []struct {
		name string
		m    SearchMethod
		want bool
	}{
		{"valid cosine", MethodCosine, true},
		{"valid bm25", MethodBM25, true},
		{"valid bfs", MethodBFS, true},
		{"invalid method", SearchMethod("invalid"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsValid(); got != tt.want {
				t.Errorf("SearchMethod.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRerankerType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		r    RerankerType
		want bool
	}{
		{"valid rrf", RerankerRRF, true},
		{"valid mmr", RerankerMMR, true},
		{"valid cross encoder", RerankerCrossEncoder, true},
		{"valid node distance", RerankerNodeDistance, true},
		{"valid episode mentions", RerankerEpisodeMentions, true},
		{"invalid type", RerankerType("invalid"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.IsValid(); got != tt.want {
				t.Errorf("RerankerType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemporalWindow_Validate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		t       TemporalWindow
		wantErr bool
	}{
		{
			name: "valid window",
			t: TemporalWindow{
				From: now,
				To:   now.Add(1 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "invalid window (from > to)",
			t: TemporalWindow{
				From: now.Add(1 * time.Hour),
				To:   now,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.t.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("TemporalWindow.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
