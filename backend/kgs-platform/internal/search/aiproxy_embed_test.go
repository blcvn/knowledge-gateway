package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAIProxyEmbeddingClientEmbedOpenAIStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1,2,3]}]}`))
	}))
	defer srv.Close()

	client := NewAIProxyEmbeddingClient(srv.URL, "/ai/embeddings", "", "text-embedding-3-small", 3, 3*time.Second)
	vector, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("embed error: %v", err)
	}
	if len(vector) != 3 {
		t.Fatalf("vector size mismatch: got=%d want=3", len(vector))
	}
}

func TestAIProxyEmbeddingClientEmbedCompletionTextVector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"code":200},"completion":{"text":"[1,2,3]"}}`))
	}))
	defer srv.Close()

	client := NewAIProxyEmbeddingClient(srv.URL, "/ai/complete", "", "text-embedding-3-small", 3, 3*time.Second)
	vector, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("embed error: %v", err)
	}
	if len(vector) != 3 {
		t.Fatalf("vector size mismatch: got=%d want=3", len(vector))
	}
}
