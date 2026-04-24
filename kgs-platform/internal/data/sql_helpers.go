package data

import (
	"strings"

	"github.com/google/uuid"
)

// nullableUUID returns nil for empty UUID strings so Postgres uuid columns
// receive NULL instead of invalid empty text.
func nullableUUID(v string) any {
	cleaned := strings.TrimSpace(v)
	if cleaned == "" {
		return nil
	}
	switch strings.ToLower(cleaned) {
	case "<nil>", "nil", "null":
		return nil
	}
	if _, err := uuid.Parse(cleaned); err != nil {
		return nil
	}
	return cleaned
}
