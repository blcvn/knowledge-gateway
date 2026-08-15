package write

import (
	"testing"
	"time"
)

// etaStore is the generic reader: it can only list everything and resolve nodes one at a time, and
// it counts both so the tests can assert on the access pattern rather than only on the result.
type etaStore struct {
	projections []ProjectionVersionRecord
	nodes       map[string]NodeRecord

	listCalls int
	nodeCalls int
}

func (s *etaStore) ListProjectionVersions() []ProjectionVersionRecord {
	s.listCalls++
	return s.projections
}

func (s *etaStore) GetNodeByID(id string) (NodeRecord, bool) {
	s.nodeCalls++
	node, ok := s.nodes[id]
	return node, ok
}

// boundedETAStore additionally answers the sample directly, as the Postgres repository does.
type boundedETAStore struct {
	etaStore

	boundedCalls int
	lastLimit    int
	recent       []ProjectionVersionRecord
}

func (s *boundedETAStore) ListRecentNodeProjectionVersionsByDomain(domainID string, limit int) []ProjectionVersionRecord {
	s.boundedCalls++
	s.lastLimit = limit
	records := make([]ProjectionVersionRecord, 0, limit)
	for _, record := range s.recent {
		node, ok := s.nodes[record.EntityID]
		if !ok || node.DomainID != domainID {
			continue
		}
		records = append(records, record)
		if len(records) == limit {
			break
		}
	}
	return records
}

// sample builds count projections whose lag is a fixed number of milliseconds, newest first.
func sample(domainID string, count int, lagMs int) (*etaStore, []ProjectionVersionRecord) {
	base := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	store := &etaStore{nodes: map[string]NodeRecord{}}
	records := make([]ProjectionVersionRecord, 0, count)
	for i := 0; i < count; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		updated := base.Add(time.Duration(i) * time.Minute)
		record := ProjectionVersionRecord{
			EntityID:          id,
			EntityKind:        "kg_node",
			SourceVersion:     1,
			SourceUpdatedAt:   updated,
			LastGraphSyncedAt: updated.Add(time.Duration(lagMs) * time.Millisecond),
		}
		store.projections = append(store.projections, record)
		store.nodes[id] = NodeRecord{ID: id, DomainID: domainID}
		records = append(records, record)
	}
	return store, records
}

// TestEstimateSyncETAPrefersTheBoundedQuery pins the fix for a write path whose cost grew with the
// size of the whole projection table: a store that can filter in the database must never be walked
// row by row.
func TestEstimateSyncETAPrefersTheBoundedQuery(t *testing.T) {
	base, records := sample("d1", 40, 200)
	store := &boundedETAStore{etaStore: *base}
	// Newest first, the order the bounded query promises.
	for i := len(records) - 1; i >= 0; i-- {
		store.recent = append(store.recent, records[i])
	}

	got := estimateSyncETA(store, "d1", 1000)

	if store.boundedCalls != 1 {
		t.Fatalf("bounded query called %d times, want exactly 1", store.boundedCalls)
	}
	if store.listCalls != 0 || store.nodeCalls != 0 {
		t.Fatalf("full scan used anyway: listCalls=%d nodeCalls=%d, want 0 and 0", store.listCalls, store.nodeCalls)
	}
	if store.lastLimit != syncETASampleSize {
		t.Fatalf("limit = %d, want the sample size %d", store.lastLimit, syncETASampleSize)
	}
	if want := 300; got != want {
		t.Fatalf("estimate = %d, want %d (median lag 200ms with the 1.5 margin)", got, want)
	}
}

// TestEstimateSyncETAMatchesAcrossStores: the two paths must agree, or the estimate would depend on
// which backend is configured rather than on the data.
func TestEstimateSyncETAMatchesAcrossStores(t *testing.T) {
	generic, records := sample("d1", 40, 200)
	bounded := &boundedETAStore{etaStore: *generic}
	for i := len(records) - 1; i >= 0; i-- {
		bounded.recent = append(bounded.recent, records[i])
	}

	if got, want := estimateSyncETA(bounded, "d1", 1000), estimateSyncETA(generic, "d1", 1000); got != want {
		t.Fatalf("bounded estimate = %d, generic estimate = %d; the two paths must agree", got, want)
	}
}

// TestEstimateSyncETAIgnoresOtherDomains guards the filter the bounded query moved into SQL.
func TestEstimateSyncETAIgnoresOtherDomains(t *testing.T) {
	store, _ := sample("other", 40, 200)

	if got, want := estimateSyncETA(store, "d1", 1234), 1234; got != want {
		t.Fatalf("estimate = %d, want the default %d when no sample belongs to the domain", got, want)
	}
}

// TestEstimateSyncETAFallsBackWithoutEnoughSamples keeps a thin sample from producing a confident
// number out of two data points.
func TestEstimateSyncETAFallsBackWithoutEnoughSamples(t *testing.T) {
	store, _ := sample("d1", 4, 200)

	if got, want := estimateSyncETA(store, "d1", 777), 777; got != want {
		t.Fatalf("estimate = %d, want the default %d with fewer than five samples", got, want)
	}
}
