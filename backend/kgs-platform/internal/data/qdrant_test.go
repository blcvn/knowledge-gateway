package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQdrantClientCRUD(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/collections/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method for collection: %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"status":"ok","result":true}`))
	})
	mux.HandleFunc("/collections/test/points", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method for upsert: %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"status":"ok","result":{"operation_id":1}}`))
	})
	mux.HandleFunc("/collections/test/points/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method for delete: %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"status":"ok","result":{"operation_id":2}}`))
	})
	mux.HandleFunc("/collections/test/points/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","result":[{"id":"n1","score":0.99,"payload":{"label":"Requirement"}}]}`))
	})
	mux.HandleFunc("/collections/test/points/search/batch", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","result":[[{"id":"n1","score":0.99,"payload":{"label":"Requirement"}}],[{"id":2,"score":0.88,"payload":{"label":"UseCase"}}]]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &QdrantClient{
		baseURL:           server.URL,
		defaultCollection: "test",
		defaultVectorSize: 3,
		httpClient:        server.Client(),
	}

	ctx := context.Background()
	if err := client.EnsureCollection(ctx, "test", 3); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	if err := client.UpsertVectors(ctx, "test", []VectorPoint{
		{ID: "n1", Vector: []float32{0.1, 0.2, 0.3}, Payload: map[string]any{"label": "Requirement"}},
	}); err != nil {
		t.Fatalf("UpsertVectors: %v", err)
	}

	searchResults, err := client.SearchVectors(ctx, "test", []float32{0.1, 0.2, 0.3}, 5, 0.9)
	if err != nil {
		t.Fatalf("SearchVectors: %v", err)
	}
	if len(searchResults) != 1 || searchResults[0].ID != "n1" {
		t.Fatalf("unexpected search results: %#v", searchResults)
	}

	batchResults, err := client.BatchSearch(ctx, "test", [][]float32{
		{0.1, 0.2, 0.3},
		{0.3, 0.2, 0.1},
	}, 2)
	if err != nil {
		t.Fatalf("BatchSearch: %v", err)
	}
	if len(batchResults) != 2 || len(batchResults[1]) != 1 || batchResults[1][0].ID != "2" {
		buf, _ := json.Marshal(batchResults)
		t.Fatalf("unexpected batch search results: %s", string(buf))
	}

	if err := client.DeleteVectors(ctx, "test", []string{"n1"}); err != nil {
		t.Fatalf("DeleteVectors: %v", err)
	}
}
