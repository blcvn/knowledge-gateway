// Package hash implements the HashComputer port for SHA-256 file deduplication.
package hash

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
)

// SHA256Computer implements port.HashComputer.
type SHA256Computer struct{}

// NewSHA256Computer creates a new SHA-256 hash computer.
func NewSHA256Computer() *SHA256Computer {
	return &SHA256Computer{}
}

// ComputeSHA256 reads the entire reader, computes the SHA-256 hash,
// and returns both the hex-encoded hash and a new reader that replays the same bytes.
func (c *SHA256Computer) ComputeSHA256(reader io.Reader) (string, io.Reader, error) {
	var buf bytes.Buffer
	h := sha256.New()

	// TeeReader: reads go to both hash and buffer
	tee := io.TeeReader(reader, &buf)
	if _, err := io.Copy(h, tee); err != nil {
		return "", nil, fmt.Errorf("compute sha256: %w", err)
	}

	hash := fmt.Sprintf("%x", h.Sum(nil))
	return hash, &buf, nil
}
