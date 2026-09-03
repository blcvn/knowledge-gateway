package tokenizer_test

import (
	"strings"
	"testing"

	"github.com/vnp-memory/pkg/tokenizer"
)

func newTokenizer(t *testing.T) *tokenizer.TiktokenTokenizer {
	t.Helper()
	tok, err := tokenizer.New("gpt-4o")
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}
	return tok
}

func TestTiktokenTokenizer_New_ValidModel(t *testing.T) {
	_, err := tokenizer.New("gpt-4o")
	if err != nil {
		t.Errorf("expected no error for 'gpt-4o', got: %v", err)
	}
}

func TestTiktokenTokenizer_New_InvalidModel(t *testing.T) {
	_, err := tokenizer.New("invalid-model-xyz")
	if err == nil {
		t.Error("expected error for invalid model, got nil")
	}
}

func TestTiktokenTokenizer_Count_Empty(t *testing.T) {
	tok := newTokenizer(t)
	if got := tok.Count(""); got != 0 {
		t.Errorf("Count(\"\") = %d, want 0", got)
	}
}

func TestTiktokenTokenizer_Count_ShortText(t *testing.T) {
	tok := newTokenizer(t)
	if got := tok.Count("Hello"); got == 0 {
		t.Error("Count(\"Hello\") = 0, want > 0")
	}
}

func TestTiktokenTokenizer_CountMessages_Overhead(t *testing.T) {
	tok := newTokenizer(t)
	// Empty messages → base overhead = 3
	got := tok.CountMessages([]tokenizer.ChatMessage{})
	if got != 3 {
		t.Errorf("CountMessages([]) = %d, want 3", got)
	}
}

func TestTiktokenTokenizer_CountMessages_Single(t *testing.T) {
	tok := newTokenizer(t)
	msgs := []tokenizer.ChatMessage{{Role: "user", Content: "hi"}}
	got := tok.CountMessages(msgs)
	// 3 (base) + 4 (overhead) + Count("user") + Count("hi") >= 8
	if got < 8 {
		t.Errorf("CountMessages([1 msg]) = %d, want >= 8", got)
	}
}

func TestTiktokenTokenizer_TruncateToTokens_Under(t *testing.T) {
	tok := newTokenizer(t)
	text := "Hello world"
	result := tok.TruncateToTokens(text, 1000)
	if result != text {
		t.Errorf("TruncateToTokens short text = %q, want %q", result, text)
	}
}

func TestTiktokenTokenizer_TruncateToTokens_Over(t *testing.T) {
	tok := newTokenizer(t)
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 200)
	maxTokens := 10
	result := tok.TruncateToTokens(text, maxTokens)
	if tok.Count(result) > maxTokens {
		t.Errorf("TruncateToTokens result has %d tokens, want <= %d", tok.Count(result), maxTokens)
	}
	if len(result) >= len(text) {
		t.Error("TruncateToTokens did not truncate")
	}
}

func TestTiktokenTokenizer_TruncateToTokens_Exact(t *testing.T) {
	tok := newTokenizer(t)
	text := strings.Repeat("hello ", 50)
	maxTokens := 20
	result := tok.TruncateToTokens(text, maxTokens)
	count := tok.Count(result)
	if count > maxTokens {
		t.Errorf("truncated result has %d tokens, want <= %d", count, maxTokens)
	}
}

func TestTiktokenTokenizer_CountBlob_Chat(t *testing.T) {
	tok := newTokenizer(t)
	data := tokenizer.ChatBlobData{
		Messages: []tokenizer.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}
	got := tok.CountBlob(data, "chat")
	if got == 0 {
		t.Error("CountBlob(chat) = 0, want > 0")
	}
}

func TestTiktokenTokenizer_CountBlob_Doc(t *testing.T) {
	tok := newTokenizer(t)
	got := tok.CountBlob("This is a document.", "doc")
	if got == 0 {
		t.Error("CountBlob(doc) = 0, want > 0")
	}
}

func TestTiktokenTokenizer_CountBlob_Unknown(t *testing.T) {
	tok := newTokenizer(t)
	got := tok.CountBlob("something", "image")
	if got != 0 {
		t.Errorf("CountBlob(unknown) = %d, want 0", got)
	}
}
