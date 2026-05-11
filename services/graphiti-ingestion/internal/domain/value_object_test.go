package domain

import (
	"testing"
)

func TestContentHashGeneration(t *testing.T) {
	body1 := "Hello World"
	hash1 := GenerateContentHash(body1)

	body2 := "Hello World"
	hash2 := GenerateContentHash(body2)

	if hash1 != hash2 {
		t.Errorf("Expected identical hashes for identical content, got %s and %s", hash1, hash2)
	}

	body3 := "Goodbye World"
	hash3 := GenerateContentHash(body3)

	if hash1 == hash3 {
		t.Errorf("Expected different hashes for different content, got %s for both", hash1)
	}
}
