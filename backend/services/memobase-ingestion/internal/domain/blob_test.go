package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/vnp-memory/services/memobase-ingestion/internal/domain"
)

func TestChatBlobData_Validate_ValidRoles(t *testing.T) {
	d := &domain.ChatBlobData{
		Messages: []domain.ChatMessage{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "system", Content: "you are a helper"},
		},
	}
	if err := d.Validate(); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestChatBlobData_Validate_InvalidRole(t *testing.T) {
	d := &domain.ChatBlobData{
		Messages: []domain.ChatMessage{{Role: "model", Content: "hello"}},
	}
	if err := d.Validate(); err == nil {
		t.Error("expected ErrInvalidBlobRole for 'model' role")
	}
}

func TestChatBlobData_Validate_Empty(t *testing.T) {
	d := &domain.ChatBlobData{Messages: []domain.ChatMessage{}}
	if err := d.Validate(); err == nil {
		t.Error("expected ErrEmptyBlobContent for empty messages")
	}
}

func TestDocBlobData_Validate_Valid(t *testing.T) {
	d := &domain.DocBlobData{Text: "some document"}
	if err := d.Validate(); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestDocBlobData_Validate_Empty(t *testing.T) {
	d := &domain.DocBlobData{Text: ""}
	if err := d.Validate(); err == nil {
		t.Error("expected ErrEmptyBlobContent for empty text")
	}
}

func TestSummaryBlobData_Validate_Empty(t *testing.T) {
	d := &domain.SummaryBlobData{Text: ""}
	if err := d.Validate(); err == nil {
		t.Error("expected ErrEmptyBlobContent for empty summary")
	}
}

func TestBufferZone_CanFlush_Idle(t *testing.T) {
	bz := &domain.BufferZone{Status: domain.BufferStatusIdle}
	if !bz.CanFlush() {
		t.Error("idle buffer zone should be flushable")
	}
}

func TestBufferZone_CanFlush_Processing(t *testing.T) {
	bz := &domain.BufferZone{Status: domain.BufferStatusProcessing}
	if bz.CanFlush() {
		t.Error("processing buffer zone should not be flushable")
	}
}

func TestDeserializeBlobData_Chat(t *testing.T) {
	raw, _ := json.Marshal(domain.ChatBlobData{
		Messages: []domain.ChatMessage{{Role: "user", Content: "hi"}},
	})
	data, err := domain.DeserializeBlobData(raw, domain.BlobTypeChat)
	if err != nil {
		t.Fatalf("DeserializeBlobData error: %v", err)
	}
	chat, ok := data.(*domain.ChatBlobData)
	if !ok {
		t.Fatal("expected *ChatBlobData")
	}
	if len(chat.Messages) != 1 {
		t.Errorf("got %d messages, want 1", len(chat.Messages))
	}
}

func TestDeserializeBlobData_UnknownType(t *testing.T) {
	_, err := domain.DeserializeBlobData([]byte("{}"), domain.BlobType("image"))
	if err == nil {
		t.Error("expected error for unknown blob type")
	}
}
