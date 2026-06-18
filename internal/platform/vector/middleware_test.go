package vector

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type countingProvider struct {
	dimensions int
	modelID    string
	calls      int
	result     []float64
	err        error
}

func (p *countingProvider) Embed(_ context.Context, _ string) ([]float64, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return append([]float64(nil), p.result...), nil
}

func (p *countingProvider) Dimensions() int { return p.dimensions }
func (p *countingProvider) ModelID() string { return p.modelID }

func TestCachingProvider(t *testing.T) {
	now := time.Date(2026, time.June, 18, 8, 0, 0, 0, time.UTC)
	cp := &countingProvider{dimensions: 3, modelID: "m", result: []float64{1, 2, 3}}
	cache := &CachingProvider{Inner: cp, TTL: time.Minute, Now: func() time.Time { return now }}

	vec, err := cache.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if cp.calls != 1 {
		t.Fatalf("calls = %d, want 1", cp.calls)
	}
	vec[0] = 9
	vec2, err := cache.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() cached error = %v", err)
	}
	if cp.calls != 1 {
		t.Fatalf("calls after cache = %d, want 1", cp.calls)
	}
	if vec2[0] != 1 {
		t.Fatalf("cached vec[0] = %v, want 1", vec2[0])
	}
	cache.Now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := cache.Embed(context.Background(), "hello"); err != nil {
		t.Fatalf("Embed() after ttl error = %v", err)
	}
	if cp.calls != 2 {
		t.Fatalf("calls after ttl = %d, want 2", cp.calls)
	}
}

func TestRetryProvider(t *testing.T) {
	cp := &countingProvider{dimensions: 3, modelID: "m", result: []float64{1, 2, 3}, err: errors.New("boom")}
	attempts := 0
	retried := &RetryProvider{
		Inner:        cp,
		MaxAttempts:  3,
		InitialDelay: time.Nanosecond,
		Sleep: func(time.Duration) {
			attempts++
		},
	}
	if _, err := retried.Embed(context.Background(), "hello"); err == nil {
		t.Fatal("Embed() error = nil, want failure")
	}
	if cp.calls != 3 {
		t.Fatalf("calls = %d, want 3", cp.calls)
	}
	if attempts != 2 {
		t.Fatalf("sleep attempts = %d, want 2", attempts)
	}

	cp.err = nil
	cp.calls = 0
	attempts = 0
	vec, err := retried.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() success error = %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("vec len = %d, want 3", len(vec))
	}
}

func TestProxyHTTPProviderRewritesURL(t *testing.T) {
	var gotURL string
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioNopCloser(`{"embedding":[0.4,0.5]}`),
			Header:     make(http.Header),
		}, nil
	})}
	base := HTTPEmbeddingProvider{URL: "http://original.local", Model: "m", HTTPClient: client}
	proxy := ProxyHTTPProvider{Inner: base, ProxyURL: "http://proxy.local"}
	vec, err := proxy.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if gotURL != "http://proxy.local" {
		t.Fatalf("url = %q, want proxy", gotURL)
	}
	if len(vec) != 2 {
		t.Fatalf("vec len = %d, want 2", len(vec))
	}
}

func TestRouters(t *testing.T) {
	base := &countingProvider{dimensions: 2, modelID: "m", result: []float64{1, 0}}
	direct := DirectRouter{Provider: base}
	if got := direct.RouteContext("t", "d"); got == nil {
		t.Fatal("direct route returned nil")
	}
	routing := RoutingRouter{
		Default: base,
		Tenants: map[string]EmbeddingProvider{"tenant-1": &countingProvider{dimensions: 2, modelID: "tenant", result: []float64{0, 1}}},
	}
	if got := routing.RouteContext("tenant-1", "d").ModelID(); got != "tenant" {
		t.Fatalf("tenant route model = %q, want tenant", got)
	}
	if got := routing.RouteContext("other", "d").ModelID(); got != "m" {
		t.Fatalf("default route model = %q, want m", got)
	}
}

func ioNopCloser(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}
