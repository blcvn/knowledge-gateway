package domain

import (
	"testing"
	"time"
)

func TestSearchQuery_Validate(t *testing.T) {
	validTemporal := &TemporalWindow{From: time.Now(), To: time.Now().Add(time.Hour)}
	invalidTemporal := &TemporalWindow{From: time.Now().Add(time.Hour), To: time.Now()}

	tests := []struct {
		name    string
		q       SearchQuery
		wantErr bool
	}{
		{
			name: "valid query",
			q: SearchQuery{
				Query:   "test",
				Methods: []SearchMethod{MethodCosine},
				Limit:   10,
			},
			wantErr: false,
		},
		{
			name: "empty query",
			q: SearchQuery{
				Query:   "",
				Methods: []SearchMethod{MethodCosine},
				Limit:   10,
			},
			wantErr: true,
		},
		{
			name: "no methods",
			q: SearchQuery{
				Query:   "test",
				Methods: []SearchMethod{},
				Limit:   10,
			},
			wantErr: true,
		},
		{
			name: "invalid limit",
			q: SearchQuery{
				Query:   "test",
				Methods: []SearchMethod{MethodCosine},
				Limit:   0,
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			q: SearchQuery{
				Query:   "test",
				Methods: []SearchMethod{"invalid"},
				Limit:   10,
			},
			wantErr: true,
		},
		{
			name: "invalid reranker",
			q: SearchQuery{
				Query:     "test",
				Methods:   []SearchMethod{MethodCosine},
				Rerankers: []RerankerType{"invalid"},
				Limit:     10,
			},
			wantErr: true,
		},
		{
			name: "invalid temporal filter",
			q: SearchQuery{
				Query:          "test",
				Methods:        []SearchMethod{MethodCosine},
				Limit:          10,
				TemporalFilter: invalidTemporal,
			},
			wantErr: true,
		},
		{
			name: "valid temporal filter",
			q: SearchQuery{
				Query:          "test",
				Methods:        []SearchMethod{MethodCosine},
				Limit:          10,
				TemporalFilter: validTemporal,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.q.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("SearchQuery.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
