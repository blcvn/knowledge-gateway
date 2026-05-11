package extractor_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/adapter/extractor"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

func TestRegistry_Supported(t *testing.T) {
	reg := extractor.NewRegistry()
	types := reg.Supported()

	if len(types) < 5 {
		t.Errorf("expected at least 5 supported types, got %d", len(types))
	}
}

func TestRegistry_ExtractPlainText(t *testing.T) {
	reg := extractor.NewRegistry()
	text, err := reg.Extract(context.Background(), strings.NewReader("hello world"), domain.MimePlainText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
}

func TestRegistry_ExtractMarkdown(t *testing.T) {
	reg := extractor.NewRegistry()
	input := "# Title\n\nSome **bold** text"
	text, err := reg.Extract(context.Background(), strings.NewReader(input), domain.MimeMarkdown)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != input {
		t.Errorf("text = %q, want %q", text, input)
	}
}

func TestRegistry_ExtractHTML(t *testing.T) {
	reg := extractor.NewRegistry()
	input := "<html><body><h1>Title</h1><p>Content here</p></body></html>"
	text, err := reg.Extract(context.Background(), strings.NewReader(input), domain.MimeHTML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Title") {
		t.Errorf("expected text to contain 'Title', got %q", text)
	}
	if !strings.Contains(text, "Content here") {
		t.Errorf("expected text to contain 'Content here', got %q", text)
	}
	// Should NOT contain HTML tags
	if strings.Contains(text, "<h1>") {
		t.Errorf("expected no HTML tags, got %q", text)
	}
}

func TestRegistry_ExtractCSV(t *testing.T) {
	reg := extractor.NewRegistry()
	input := "name,age\nAlice,30\nBob,25"
	text, err := reg.Extract(context.Background(), strings.NewReader(input), domain.MimeCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != input {
		t.Errorf("text = %q, want %q", text, input)
	}
}

func TestRegistry_ExtractJSON(t *testing.T) {
	reg := extractor.NewRegistry()
	input := `{"key": "value", "items": [1, 2, 3]}`
	text, err := reg.Extract(context.Background(), strings.NewReader(input), domain.MimeJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != input {
		t.Errorf("text = %q, want %q", text, input)
	}
}

func TestRegistry_UnsupportedFormat(t *testing.T) {
	reg := extractor.NewRegistry()
	_, err := reg.Extract(context.Background(), strings.NewReader("data"), domain.MimeType("video/mp4"))
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}

	var unsupported *domain.ErrUnsupportedFormat
	if !containsUnsupported(err, &unsupported) {
		t.Errorf("expected ErrUnsupportedFormat, got %T: %v", err, err)
	}
}

func containsUnsupported(err error, target **domain.ErrUnsupportedFormat) bool {
	// Simple type assertion since errors.As doesn't work well with double pointers in tests
	switch e := err.(type) {
	case *domain.ErrUnsupportedFormat:
		*target = e
		return true
	default:
		return false
	}
}
