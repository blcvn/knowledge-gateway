package data

import "strings"

// nullableUUID returns nil for empty UUID strings so Postgres uuid columns
// receive NULL instead of invalid empty text.
func nullableUUID(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}
