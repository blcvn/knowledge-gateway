package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
