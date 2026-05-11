package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

func TestNewDataset_Valid(t *testing.T) {
	ds, err := domain.NewDataset("tenant-123", "my-dataset", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ds.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if ds.TenantID != "tenant-123" {
		t.Errorf("tenant_id = %q, want %q", ds.TenantID, "tenant-123")
	}
	if ds.Name != "my-dataset" {
		t.Errorf("name = %q, want %q", ds.Name, "my-dataset")
	}
	if ds.Status != domain.DatasetPending {
		t.Errorf("status = %q, want %q", ds.Status, domain.DatasetPending)
	}
	if ds.FileCount != 0 {
		t.Errorf("file_count = %d, want 0", ds.FileCount)
	}
}

func TestNewDataset_MissingTenantID(t *testing.T) {
	_, err := domain.NewDataset("", "ds", "")
	if err != domain.ErrMissingTenantID {
		t.Errorf("err = %v, want ErrMissingTenantID", err)
	}
}

func TestNewDataset_MissingName(t *testing.T) {
	_, err := domain.NewDataset("t1", "", "")
	if err != domain.ErrMissingDatasetName {
		t.Errorf("err = %v, want ErrMissingDatasetName", err)
	}
}

func TestDataset_IncrementItems(t *testing.T) {
	ds, _ := domain.NewDataset("t1", "ds1", "")
	ds.IncrementItems(1024)
	ds.IncrementItems(2048)

	if ds.FileCount != 2 {
		t.Errorf("file_count = %d, want 2", ds.FileCount)
	}
	if ds.TotalSizeBytes != 3072 {
		t.Errorf("total_size = %d, want 3072", ds.TotalSizeBytes)
	}
}

func TestDataset_StatusTransitions(t *testing.T) {
	ds, _ := domain.NewDataset("t1", "ds1", "")

	ds.MarkReady()
	if ds.Status != domain.DatasetReady {
		t.Errorf("status = %q, want READY", ds.Status)
	}

	ds.MarkCognifying()
	if ds.Status != domain.DatasetCognifying {
		t.Errorf("status = %q, want COGNIFYING", ds.Status)
	}

	ds.MarkError()
	if ds.Status != domain.DatasetError {
		t.Errorf("status = %q, want ERROR", ds.Status)
	}
}

func TestNewDataItem_Valid(t *testing.T) {
	dsID := uuid.New()
	item, err := domain.NewDataItem(dsID, "tenant-1", domain.SourceFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if item.DatasetID != dsID {
		t.Errorf("dataset_id mismatch")
	}
	if item.Source != domain.SourceFile {
		t.Errorf("source = %q, want FILE", item.Source)
	}
}

func TestNewDataItem_MissingDatasetID(t *testing.T) {
	_, err := domain.NewDataItem(uuid.Nil, "t1", domain.SourceFile)
	if err != domain.ErrMissingDatasetID {
		t.Errorf("err = %v, want ErrMissingDatasetID", err)
	}
}

func TestNewDataItem_MissingTenantID(t *testing.T) {
	_, err := domain.NewDataItem(uuid.New(), "", domain.SourceFile)
	if err != domain.ErrMissingTenantID {
		t.Errorf("err = %v, want ErrMissingTenantID", err)
	}
}

func TestDataItem_StorageKey(t *testing.T) {
	dsID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	item, _ := domain.NewDataItem(dsID, "tenant-a", domain.SourceFile)

	key := item.StorageKey()
	expected := "tenant-a/11111111-1111-1111-1111-111111111111/" + item.ID.String()
	if key != expected {
		t.Errorf("storage_key = %q, want %q", key, expected)
	}
}

func TestDataItem_WithFile(t *testing.T) {
	item, _ := domain.NewDataItem(uuid.New(), "t1", domain.SourceFile)
	item.WithFile("test.pdf", domain.MimePDF, 5000, "abc123", "/path/to/file")

	if item.Filename != "test.pdf" {
		t.Errorf("filename = %q", item.Filename)
	}
	if item.MimeType != domain.MimePDF {
		t.Errorf("mime = %q", item.MimeType)
	}
	if item.SizeBytes != 5000 {
		t.Errorf("size = %d", item.SizeBytes)
	}
	if item.FileHash != "abc123" {
		t.Errorf("hash = %q", item.FileHash)
	}
}

func TestDataItem_WithText(t *testing.T) {
	item, _ := domain.NewDataItem(uuid.New(), "t1", domain.SourceText)
	item.WithText("hello world", "note.txt")

	if item.RawText != "hello world" {
		t.Errorf("raw_text = %q", item.RawText)
	}
	if item.Filename != "note.txt" {
		t.Errorf("filename = %q", item.Filename)
	}
	if item.SizeBytes != 11 {
		t.Errorf("size = %d, want 11", item.SizeBytes)
	}
}
