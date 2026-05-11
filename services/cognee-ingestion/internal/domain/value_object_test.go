package domain_test

import (
	"testing"

	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

func TestMimeType_IsSupported(t *testing.T) {
	tests := []struct {
		mime     domain.MimeType
		want     bool
	}{
		{domain.MimePDF, true},
		{domain.MimeDOCX, true},
		{domain.MimePPTX, true},
		{domain.MimeCSV, true},
		{domain.MimeHTML, true},
		{domain.MimePlainText, true},
		{domain.MimeMarkdown, true},
		{domain.MimeJSON, true},
		{domain.MimeType("application/octet-stream"), false},
		{domain.MimeType("video/mp4"), false},
		{domain.MimeType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mime), func(t *testing.T) {
			got := tt.mime.IsSupported()
			if got != tt.want {
				t.Errorf("MimeType(%q).IsSupported() = %v, want %v", tt.mime, got, tt.want)
			}
		})
	}
}

func TestDatasetStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status domain.DatasetStatus
		want   bool
	}{
		{domain.DatasetPending, false},
		{domain.DatasetCognifying, false},
		{domain.DatasetReady, true},
		{domain.DatasetError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsTerminal()
			if got != tt.want {
				t.Errorf("DatasetStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestSupportedMimeTypes(t *testing.T) {
	types := domain.SupportedMimeTypes()
	if len(types) < 8 {
		t.Errorf("expected at least 8 supported types, got %d", len(types))
	}

	// All returned types must be supported
	for _, mt := range types {
		if !mt.IsSupported() {
			t.Errorf("SupportedMimeTypes() returned unsupported type: %q", mt)
		}
	}
}

func TestDataSource_String(t *testing.T) {
	if domain.SourceFile.String() != "FILE" {
		t.Errorf("SourceFile.String() = %q", domain.SourceFile.String())
	}
	if domain.SourceText.String() != "TEXT" {
		t.Errorf("SourceText.String() = %q", domain.SourceText.String())
	}
	if domain.SourceURL.String() != "URL" {
		t.Errorf("SourceURL.String() = %q", domain.SourceURL.String())
	}
}
