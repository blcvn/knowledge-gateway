// Package scraper implements the URLScraper port for web content extraction.
package scraper

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// HTTPScraper implements port.URLScraper using net/http.
// For production use, consider replacing with colly or rod for JS-rendered pages.
type HTTPScraper struct {
	client *http.Client
	logger *slog.Logger
}

// NewHTTPScraper creates a new HTTP-based scraper with the given timeout.
func NewHTTPScraper(timeout time.Duration, logger *slog.Logger) *HTTPScraper {
	return &HTTPScraper{
		client: &http.Client{Timeout: timeout},
		logger: logger.With("adapter", "http_scraper"),
	}
}

// Scrape fetches the URL and returns cleaned text content.
func (s *HTTPScraper) Scrape(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "VNP-Memory-Ingestion/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	// Limit reading to 10MB
	const maxSize = 10 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	text := stripHTMLTags(string(body))
	text = cleanWhitespace(text)

	s.logger.Info("url scraped", "url", url, "content_length", len(text))
	return text, nil
}

// stripHTMLTags removes HTML tags from content.
func stripHTMLTags(s string) string {
	var result []byte
	inTag := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '<':
			inTag = true
		case s[i] == '>':
			inTag = false
			result = append(result, ' ')
		case !inTag:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// cleanWhitespace normalizes whitespace sequences to single spaces and trims.
func cleanWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
