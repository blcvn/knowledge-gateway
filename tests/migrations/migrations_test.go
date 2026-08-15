// Package migrations_test guards the on-disk migration set.
//
// golang-migrate loads the whole directory before applying anything, so a single malformed or
// duplicated filename takes the service from "one migration is broken" to "the database cannot be
// provisioned at all". These tests are cheap and run in the default `go test ./...` sweep.
package migrations_test

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// migrationsDir is relative to this test file's package directory.
const migrationsDir = "../../migrations"

// skippedVersions are version numbers deliberately left unused. Reusing one would let
// golang-migrate skip the new migration on an environment that already recorded that version.
// See docs/operations/migration-duplicate-cleanup.md.
var skippedVersions = map[int]string{
	15: "duplicate of 000011_optimize_kg_hot_fks; may already be recorded in environments " +
		"provisioned between 2026-07-07 and 2026-07-09",
}

var migrationName = regexp.MustCompile(`^(\d{6})_([a-z0-9_]+)\.(up|down)\.sql$`)

type migrationFile struct {
	version   int
	name      string
	direction string
	filename  string
}

func loadMigrations(t *testing.T) []migrationFile {
	t.Helper()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		m := migrationName.FindStringSubmatch(name)
		if m == nil {
			t.Errorf("migration %q does not match <000000>_<snake_name>.(up|down).sql", name)
			continue
		}
		version, err := strconv.Atoi(m[1])
		if err != nil {
			t.Errorf("migration %q has a non-numeric version prefix: %v", name, err)
			continue
		}
		files = append(files, migrationFile{
			version:   version,
			name:      m[2],
			direction: m[3],
			filename:  name,
		})
	}
	if len(files) == 0 {
		t.Fatalf("no migrations found in %s", migrationsDir)
	}
	return files
}

// TestMigrationVersionsAreUnique is the guard for the 000014 collision that blocked `migrate up`
// entirely. Two different migrations sharing a version prefix make golang-migrate fail at source
// load time with "duplicate migration version".
func TestMigrationVersionsAreUnique(t *testing.T) {
	byVersion := map[int]map[string]struct{}{}
	for _, f := range loadMigrations(t) {
		if byVersion[f.version] == nil {
			byVersion[f.version] = map[string]struct{}{}
		}
		byVersion[f.version][f.name] = struct{}{}
	}

	versions := make([]int, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.Ints(versions)

	for _, v := range versions {
		names := byVersion[v]
		if len(names) <= 1 {
			continue
		}
		sorted := make([]string, 0, len(names))
		for n := range names {
			sorted = append(sorted, n)
		}
		sort.Strings(sorted)
		t.Errorf("version %06d is claimed by %d migrations: %s — golang-migrate will refuse to load the directory",
			v, len(sorted), strings.Join(sorted, ", "))
	}
}

// TestEveryMigrationHasBothDirections catches a half-added migration, which makes `migrate down`
// fail at exactly the moment it is needed most.
func TestEveryMigrationHasBothDirections(t *testing.T) {
	seen := map[string]map[string]bool{}
	for _, f := range loadMigrations(t) {
		key := f.filename[:6] + "_" + f.name
		if seen[key] == nil {
			seen[key] = map[string]bool{}
		}
		seen[key][f.direction] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !seen[k]["up"] {
			t.Errorf("migration %s has no .up.sql", k)
		}
		if !seen[k]["down"] {
			t.Errorf("migration %s has no .down.sql", k)
		}
	}
}

// TestSkippedVersionsStayUnused protects the deliberate gap at 000015. Reusing that number would
// let golang-migrate skip the new migration on any environment that already recorded version 15.
func TestSkippedVersionsStayUnused(t *testing.T) {
	for _, f := range loadMigrations(t) {
		if reason, skipped := skippedVersions[f.version]; skipped {
			t.Errorf("migration %q reuses deliberately skipped version %06d: %s",
				f.filename, f.version, reason)
		}
	}
}
