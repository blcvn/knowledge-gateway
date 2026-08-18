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

func (p *countingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, 0, len(texts))
	for range texts {
		vec, err := p.Embed(ctx, "")
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vec)
	}
	return vectors, nil
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

// The E3/E4 cases from impl-02: an embedding failure that a human must fix has to be told apart
// from one that fixes itself, and neither may be retried the wrong number of times.

func TestRetryDoesNotRepeatARejectedCredential(t *testing.T) {
	inner := &countingProvider{dimensions: 3, modelID: "m", err: statusError(401, "token expired")}
	slept := 0
	provider := &RetryProvider{Inner: inner, MaxAttempts: 3, Sleep: func(time.Duration) { slept++ }}

	_, err := provider.Embed(context.Background(), "text")
	if !errors.Is(err, ErrEmbeddingUnauthorized) {
		t.Fatalf("want ErrEmbeddingUnauthorized, got %v", err)
	}
	// One attempt, not three: a rotated token answers the same however often it is asked, and the
	// two extra requests would only delay the alert an operator is waiting for.
	if inner.calls != 1 {
		t.Fatalf("want a single attempt for a terminal failure, got %d", inner.calls)
	}
	if slept != 0 {
		t.Fatalf("want no backoff before a terminal failure, slept %d times", slept)
	}
}

func TestRetryDoesNotRepeatRejectedInput(t *testing.T) {
	inner := &countingProvider{dimensions: 3, modelID: "m", err: statusError(400, "input exceeds context window")}
	provider := &RetryProvider{Inner: inner, MaxAttempts: 3, Sleep: func(time.Duration) {}}

	_, err := provider.Embed(context.Background(), "text")
	if !errors.Is(err, ErrEmbeddingRejectedInput) {
		t.Fatalf("want ErrEmbeddingRejectedInput, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("want a single attempt, got %d", inner.calls)
	}
}

func TestRetryKeepsRetryingRateLimits(t *testing.T) {
	// 429 is the one 4xx worth repeating: backing off is precisely the right answer to it, so it
	// must not be swept in with the terminal statuses.
	inner := &countingProvider{dimensions: 3, modelID: "m", err: statusError(429, "slow down")}
	provider := &RetryProvider{Inner: inner, MaxAttempts: 3, Sleep: func(time.Duration) {}}

	if _, err := provider.Embed(context.Background(), "text"); err == nil {
		t.Fatal("want an error after the attempts are exhausted")
	}
	if inner.calls != 3 {
		t.Fatalf("want 3 attempts for a rate limit, got %d", inner.calls)
	}
	if IsTerminal(statusError(429, "slow down")) {
		t.Fatal("a rate limit must not be classified as terminal")
	}
}

func TestRetryGivesUpAfterThreeAttemptsOnTimeout(t *testing.T) {
	// E4: a timeout is transient, so it earns the full three attempts and then stops. Asserting the
	// count rather than the elapsed time keeps this deterministic — the backoff is real sleeping in
	// production and a duration assertion here would only measure the test's own fake clock.
	inner := &countingProvider{dimensions: 3, modelID: "m", err: context.DeadlineExceeded}
	slept := 0
	provider := &RetryProvider{Inner: inner, MaxAttempts: 3, Sleep: func(time.Duration) { slept++ }}

	_, err := provider.Embed(context.Background(), "text")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want the timeout returned to the caller, got %v", err)
	}
	if inner.calls != 3 {
		t.Fatalf("want 3 attempts, got %d", inner.calls)
	}
	if slept != 2 {
		t.Fatalf("want backoff between attempts only, slept %d times", slept)
	}
}

func TestRetryStopsWhenTheCallerHasGivenUp(t *testing.T) {
	// Sleeping through two more attempts against a cancelled context delays an error the caller is
	// already waiting for, and cannot produce a usable vector either way.
	inner := &countingProvider{dimensions: 3, modelID: "m", err: errors.New("boom")}
	provider := &RetryProvider{Inner: inner, MaxAttempts: 3, Sleep: func(time.Duration) {}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Embed(ctx, "text"); err == nil {
		t.Fatal("want an error")
	}
	if inner.calls != 1 {
		t.Fatalf("want a single attempt once the context is done, got %d", inner.calls)
	}
}

func TestClassifyStatusNamesOnlyWhatCannotSucceed(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{
		{401, ErrEmbeddingUnauthorized},
		{403, ErrEmbeddingUnauthorized},
		{400, ErrEmbeddingRejectedInput},
		{413, ErrEmbeddingRejectedInput},
		{429, nil},
		{500, nil},
		{503, nil},
	} {
		if got := classifyStatus(tc.status); !errors.Is(got, tc.want) {
			t.Errorf("status %d: want %v, got %v", tc.status, tc.want, got)
		}
	}
}
