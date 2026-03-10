package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIEmbeddingClientEmbedSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var req openAIEmbeddingRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Fatalf("request body is not valid json: %v", err)
		}
		if req.Model != "text-embedding-3-small" {
			t.Fatalf("unexpected model: %q", req.Model)
		}
		if len(req.Input) != 1 || req.Input[0] != "hello" {
			t.Fatalf("unexpected input payload: %#v", req.Input)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1,2,3]}]}`))
	}))
	defer srv.Close()

	client := NewOpenAIEmbeddingClient(srv.URL, "test-key", "text-embedding-3-small", 3, 3*time.Second)
	vector, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("embed error: %v", err)
	}
	if len(vector) != 3 {
		t.Fatalf("vector size mismatch: got=%d want=3", len(vector))
	}
}

func TestOpenAIEmbeddingClientEmbedErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key","type":"invalid_api_key"}}`))
	}))
	defer srv.Close()

	client := NewOpenAIEmbeddingClient(srv.URL, "bad-key", "text-embedding-3-small", 3, 3*time.Second)
	if _, err := client.Embed(context.Background(), "hello"); err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}
