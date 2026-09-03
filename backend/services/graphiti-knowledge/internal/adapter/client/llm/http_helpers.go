package llm

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

// defaultHTTPClient returns a shared HTTP client with timeout
func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

// newHTTPRequest creates an http.Request with context and body
func newHTTPRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
}
