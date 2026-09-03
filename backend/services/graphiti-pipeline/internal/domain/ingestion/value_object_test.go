package ingestion

import (
	"testing"
)

func TestGroupID_Validate(t *testing.T) {
	tests := []struct {
		name    string
		groupID GroupID
		wantErr bool
	}{
		{"valid", "group-1", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.groupID.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("GroupID.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEpisodeID_Validate(t *testing.T) {
	tests := []struct {
		name      string
		episodeID EpisodeID
		wantErr   bool
	}{
		{"valid", "ep-1", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.episodeID.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("EpisodeID.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContentHash_Validate(t *testing.T) {
	tests := []struct {
		name string
		hash ContentHash
		wantErr bool
	}{
		{"valid", "hash-1", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.hash.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ContentHash.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
