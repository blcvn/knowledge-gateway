package write

import (
	"context"
	"testing"
	"time"
)

// archiveFixture seeds one graph identity with a run of sealed versions, newest first by version
// number, all created at a controllable age.
func archiveFixture(t *testing.T, count int, age time.Duration, headVersionNumber int64) (*MemoryStore, []GraphVersionRecord) {
	t.Helper()
	store := NewMemoryStore()
	identifierID := "identity-1"
	created := time.Now().UTC().Add(-age)

	versions := make([]GraphVersionRecord, 0, count)
	for i := 1; i <= count; i++ {
		versions = append(versions, GraphVersionRecord{
			VersionID:     "version-" + string(rune('a'+i-1)),
			IdentifierID:  identifierID,
			VersionNumber: int64(i),
			ReferenceID:   "ref-" + string(rune('a'+i-1)),
			StorageClass:  "ONLINE",
			VersionStatus: "SEALED",
			CreatedAt:     created,
			SealedAt:      created,
		})
	}

	head := ""
	for _, version := range versions {
		if version.VersionNumber == headVersionNumber {
			head = version.VersionID
		}
		store.graphVersionEntities[version.VersionID] = []GraphVersionEntityRecord{
			{VersionID: version.VersionID, EntityKind: "node", EntityID: "n-1", ChangeKind: "UPSERT"},
		}
	}
	store.graphVersions[identifierID] = versions
	store.graphIdentities["tenant|app|scope"] = GraphIdentityRecord{
		IdentifierID:      identifierID,
		OwnerTenantID:     "tenant",
		OwnerAppID:        "app",
		GraphScope:        "bas:kg:doc",
		HeadVersionNumber: headVersionNumber,
		HeadVersionID:     head,
	}
	return store, versions
}

func storedVersion(store *MemoryStore, versionID string) (GraphVersionRecord, bool) {
	for _, versions := range store.graphVersions {
		for _, version := range versions {
			if version.VersionID == versionID {
				return version, true
			}
		}
	}
	return GraphVersionRecord{}, false
}

// TestArchiveGraphVersionsKeepsRecentAndHead is the safety property: a graph can never be pruned
// back to nothing, because the keep-count and the head are both protected.
func TestArchiveGraphVersionsKeepsRecentAndHead(t *testing.T) {
	store, _ := archiveFixture(t, 10, 1000*time.Hour, 1)

	archived, err := store.ArchiveGraphVersions(context.Background(), 3, time.Now().UTC())
	if err != nil {
		t.Fatalf("ArchiveGraphVersions() error = %v", err)
	}

	// Versions 10, 9, 8 are within the keep-count; version 1 is the head. That leaves 7..2.
	if len(archived) != 6 {
		t.Fatalf("archived = %d (%v), want 6", len(archived), archived)
	}
	for _, keep := range []string{"version-j", "version-i", "version-h", "version-a"} {
		version, ok := storedVersion(store, keep)
		if !ok {
			t.Fatalf("version %s vanished", keep)
		}
		if version.StorageClass != "ONLINE" {
			t.Errorf("version %s storage_class = %s, want ONLINE", keep, version.StorageClass)
		}
		if len(store.graphVersionEntities[keep]) == 0 {
			t.Errorf("version %s lost its manifest but should have been kept", keep)
		}
	}
}

// TestArchiveGraphVersionsDropsManifests is the point of the job: the version rows are small
// history, but the per-entity manifests are what grow without bound.
func TestArchiveGraphVersionsDropsManifests(t *testing.T) {
	store, _ := archiveFixture(t, 5, 1000*time.Hour, 5)

	archived, err := store.ArchiveGraphVersions(context.Background(), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("ArchiveGraphVersions() error = %v", err)
	}
	if len(archived) == 0 {
		t.Fatal("nothing archived")
	}
	for _, versionID := range archived {
		if len(store.graphVersionEntities[versionID]) != 0 {
			t.Errorf("archived version %s kept its manifest", versionID)
		}
		version, ok := storedVersion(store, versionID)
		if !ok {
			t.Fatalf("archived version %s was deleted; it should be retained as history", versionID)
		}
		if version.StorageClass != "OFFLINE" {
			t.Errorf("archived version %s storage_class = %s, want OFFLINE", versionID, version.StorageClass)
		}
	}
}

// TestArchiveGraphVersionsRespectsRetentionWindow proves age alone protects a version: a busy graph
// that blew past the keep-count in the last hour keeps all of it.
func TestArchiveGraphVersionsRespectsRetentionWindow(t *testing.T) {
	store, _ := archiveFixture(t, 10, time.Minute, 10)

	archived, err := store.ArchiveGraphVersions(context.Background(), 3, time.Now().UTC().Add(-720*time.Hour))
	if err != nil {
		t.Fatalf("ArchiveGraphVersions() error = %v", err)
	}
	if len(archived) != 0 {
		t.Fatalf("archived = %v, want none: every version is inside the retention window", archived)
	}
}

// TestArchiveGraphVersionsSkipsPendingAndAbandoned keeps the job away from sessions that are still
// in flight, which the stale-session sweep owns instead.
func TestArchiveGraphVersionsSkipsPendingAndAbandoned(t *testing.T) {
	store, versions := archiveFixture(t, 5, 1000*time.Hour, 5)
	for i := range versions {
		if versions[i].VersionNumber == 2 {
			versions[i].VersionStatus = "PENDING_ENTITIES"
		}
		if versions[i].VersionNumber == 3 {
			versions[i].VersionStatus = "ABANDONED"
		}
	}
	store.graphVersions["identity-1"] = versions

	archived, err := store.ArchiveGraphVersions(context.Background(), 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("ArchiveGraphVersions() error = %v", err)
	}
	for _, versionID := range archived {
		version, _ := storedVersion(store, versionID)
		if version.VersionStatus != "SEALED" {
			t.Errorf("archived a %s version (%s); only SEALED versions may be archived", version.VersionStatus, versionID)
		}
	}
}

// TestArchiveGraphVersionsIsIdempotent: a second sweep finds nothing left to do, because archived
// versions are no longer ONLINE.
func TestArchiveGraphVersionsIsIdempotent(t *testing.T) {
	store, _ := archiveFixture(t, 6, 1000*time.Hour, 6)

	first, err := store.ArchiveGraphVersions(context.Background(), 2, time.Now().UTC())
	if err != nil {
		t.Fatalf("first ArchiveGraphVersions() error = %v", err)
	}
	if len(first) == 0 {
		t.Fatal("first sweep archived nothing")
	}
	second, err := store.ArchiveGraphVersions(context.Background(), 2, time.Now().UTC())
	if err != nil {
		t.Fatalf("second ArchiveGraphVersions() error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second sweep archived %v, want nothing", second)
	}
}
