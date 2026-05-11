package hash_test

import (
	"io"
	"strings"
	"testing"

	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/adapter/hash"
)

func TestSHA256Computer_Compute(t *testing.T) {
	comp := hash.NewSHA256Computer()

	input := "hello world"
	h, replay, err := comp.ComputeSHA256(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Known SHA-256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if h != expected {
		t.Errorf("hash = %q, want %q", h, expected)
	}

	// Verify replay reader contains the same bytes
	replayData, err := io.ReadAll(replay)
	if err != nil {
		t.Fatalf("read replay: %v", err)
	}
	if string(replayData) != input {
		t.Errorf("replay = %q, want %q", string(replayData), input)
	}
}

func TestSHA256Computer_DifferentInputs(t *testing.T) {
	comp := hash.NewSHA256Computer()

	h1, _, _ := comp.ComputeSHA256(strings.NewReader("input1"))
	h2, _, _ := comp.ComputeSHA256(strings.NewReader("input2"))

	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestSHA256Computer_SameInputSameHash(t *testing.T) {
	comp := hash.NewSHA256Computer()

	h1, _, _ := comp.ComputeSHA256(strings.NewReader("consistent"))
	h2, _, _ := comp.ComputeSHA256(strings.NewReader("consistent"))

	if h1 != h2 {
		t.Errorf("same input should produce same hash: %q != %q", h1, h2)
	}
}

func TestSHA256Computer_EmptyInput(t *testing.T) {
	comp := hash.NewSHA256Computer()

	h, _, err := comp.ComputeSHA256(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SHA-256 of empty string
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if h != expected {
		t.Errorf("hash = %q, want %q", h, expected)
	}
}
