package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewVNPEmbeddingClient_Defaults(t *testing.T) {
	c := NewVNPEmbeddingClient("", "test-key", 0)
	if c.url != defaultVNPEmbedURL {
		t.Errorf("expected default URL %q, got %q", defaultVNPEmbedURL, c.url)
	}
	if c.apiKey != "test-key" {
		t.Errorf("expected apiKey %q, got %q", "test-key", c.apiKey)
	}
	if c.httpClient == nil {
		t.Fatal("expected httpClient to be set")
	}
}

func TestNewVNPEmbeddingClient_CustomURL(t *testing.T) {
	c := NewVNPEmbeddingClient("https://custom.api/embed", "key", 5*time.Second)
	if c.url != "https://custom.api/embed" {
		t.Errorf("expected custom URL, got %q", c.url)
	}
}

func TestVNPEmbeddingClient_EmptyText(t *testing.T) {
	c := NewVNPEmbeddingClient("", "key", 0)
	_, err := c.Embed(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if got := err.Error(); got != "vnp embed: empty text" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestVNPEmbeddingClient_Success(t *testing.T) {
	embedding := []float64{0.1, 0.2, -0.3, 0.4, 0.5}
	resp := vnpEmbedResponse{
		Data: []vnpEmbedData{
			{Index: 0, Embedding: embedding},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Verify headers
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization Bearer test-api-key, got %q", auth)
		}
		if reqID := r.Header.Get("http_x_request_id"); reqID == "" {
			t.Error("expected http_x_request_id header to be set")
		}
		// Verify request body
		var reqBody vnpEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody.Input != "hello world" {
			t.Errorf("expected input %q, got %q", "hello world", reqBody.Input)
		}
		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewVNPEmbeddingClient(srv.URL, "test-api-key", 5*time.Second)
	result, err := c.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(embedding) {
		t.Fatalf("expected %d floats, got %d", len(embedding), len(result))
	}
	for i, v := range embedding {
		if result[i] != float32(v) {
			t.Errorf("result[%d] = %f, want %f", i, result[i], float32(v))
		}
	}
}

func TestVNPEmbeddingClient_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	c := NewVNPEmbeddingClient(srv.URL, "key", 5*time.Second)
	_, err := c.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if got := err.Error(); got != `vnp embed: unexpected status 500: {"error":"internal server error"}` {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestVNPEmbeddingClient_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vnpEmbedResponse{Data: []vnpEmbedData{}})
	}))
	defer srv.Close()

	c := NewVNPEmbeddingClient(srv.URL, "key", 5*time.Second)
	_, err := c.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty data")
	}
	if got := err.Error(); got != "vnp embed: empty embedding in response" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestVNPEmbeddingClient_EmptyEmbedding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vnpEmbedResponse{
			Data: []vnpEmbedData{{Index: 0, Embedding: []float64{}}},
		})
	}))
	defer srv.Close()

	c := NewVNPEmbeddingClient(srv.URL, "key", 5*time.Second)
	_, err := c.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty embedding")
	}
	if got := err.Error(); got != "vnp embed: empty embedding in response" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestVNPEmbeddingClient_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewVNPEmbeddingClient(srv.URL, "key", 5*time.Second)
	_, err := c.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestVNPEmbeddingClient_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewVNPEmbeddingClient(srv.URL, "key", 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Embed(ctx, "test")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestVNPEmbeddingClient_ImplementsInterface(t *testing.T) {
	var _ EmbeddingClient = (*VNPEmbeddingClient)(nil)
}
