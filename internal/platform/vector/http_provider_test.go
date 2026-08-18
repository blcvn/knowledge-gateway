package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPEmbeddingProviderEmbed(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("Method = %s, want POST", req.Method)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode(request body) error = %v", err)
		}
		input, ok := payload["input"].([]any)
		if !ok || len(input) != 1 || strings.TrimSpace(input[0].(string)) != "hello" {
			t.Fatalf("payload input = %#v, want [hello]", payload["input"])
		}
		if payload["model"] != "test-model" {
			t.Fatalf("payload model = %#v, want test-model", payload["model"])
		}
		body := json.RawMessage(`{"embedding":[0.1,0.2,0.3]}`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	provider := HTTPEmbeddingProvider{URL: "http://example.invalid", Model: "test-model", APIKey: "secret", HTTPClient: client}
	vec, err := provider.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("embedding len = %d, want 3", len(vec))
	}
}

func TestHTTPEmbeddingProviderEmbedSupportsDataEnvelope(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := json.RawMessage(`{"data":[{"embedding":[0.4,0.5]}]}`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	provider := HTTPEmbeddingProvider{URL: "http://example.invalid", HTTPClient: client}
	vec, err := provider.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vec) != 2 || vec[0] != 0.4 || vec[1] != 0.5 {
		t.Fatalf("embedding = %#v, want [0.4 0.5]", vec)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Embedding models reject input past their context window instead of truncating it, and the
// rejection is not local: it aborts the projection run the node belongs to, so every node after it
// goes unembedded too. Observed on real data, where a rebuild stopped after 290 of ~6,500 nodes on
// one 42,735-character node. These tests pin the guard that prevents it.

func TestOversizedInputIsTruncatedNotRejected(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if len(payload.Input) > 0 {
			received = payload.Input[0]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"embedding": []float64{0.1, 0.2}}},
		})
	}))
	defer server.Close()

	provider := HTTPEmbeddingProvider{URL: server.URL, Model: "v_search", MaxInputChars: 100}
	if _, err := provider.Embed(context.Background(), strings.Repeat("a", 5000)); err != nil {
		t.Fatalf("Embed() error = %v, want the oversized input to be truncated and sent", err)
	}
	if len(received) != 100 {
		t.Fatalf("sent %d characters, want the input capped at 100", len(received))
	}
}

func TestInputInsideTheLimitIsSentWhole(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if len(payload.Input) > 0 {
			received = payload.Input[0]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"embedding": []float64{0.1}}},
		})
	}))
	defer server.Close()

	provider := HTTPEmbeddingProvider{URL: server.URL, MaxInputChars: 100}
	if _, err := provider.Embed(context.Background(), "vừa đủ ngắn"); err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if received != "vừa đủ ngắn" {
		t.Fatalf("sent %q, want the text unchanged", received)
	}
}

// Truncation must cut on a rune boundary: a payload ending mid-rune is invalid UTF-8, and the
// provider would reject the request for a reason that has nothing to do with length.
func TestTruncationKeepsValidUTF8(t *testing.T) {
	text := strings.Repeat("ữ", 100) // three bytes per rune
	for limit := 1; limit <= 20; limit++ {
		got, truncated := truncateForModel(text, limit)
		if !truncated {
			t.Fatalf("limit %d: expected truncation", limit)
		}
		if !utf8ValidString(got) {
			t.Fatalf("limit %d produced invalid UTF-8", limit)
		}
		if len(got) > limit {
			t.Fatalf("limit %d: got %d bytes", limit, len(got))
		}
	}
}

func utf8ValidString(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}
	return true
}

// A zero limit must fall back to the default rather than truncating everything to nothing.
func TestZeroLimitUsesTheDefault(t *testing.T) {
	got, truncated := truncateForModel("ngắn", 0)
	if truncated || got != "ngắn" {
		t.Fatalf("got %q truncated=%v, want the text untouched under the default cap", got, truncated)
	}
}
