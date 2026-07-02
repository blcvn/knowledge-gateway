package vectorstore

import (
	"context"
	"math"
	"slices"
	"strings"
	"sync"
)

type InMemoryVectorAdapter struct {
	mu   sync.RWMutex
	docs map[string]VectorDocument
}

func NewInMemoryVectorAdapter() *InMemoryVectorAdapter {
	return &InMemoryVectorAdapter{docs: map[string]VectorDocument{}}
}

func (a *InMemoryVectorAdapter) Upsert(_ context.Context, doc VectorDocument) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if existing, ok := a.docs[doc.NodeID]; ok && existing.SyncVersion > doc.SyncVersion {
		return nil
	}
	if doc.DomainProps == nil {
		doc.DomainProps = map[string]any{}
	}
	doc.DomainProps["_kg_sync_version"] = doc.SyncVersion
	a.docs[doc.NodeID] = doc
	return nil
}

func (a *InMemoryVectorAdapter) UpsertBatch(ctx context.Context, docs []VectorDocument) error {
	for _, doc := range docs {
		if err := a.Upsert(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func (a *InMemoryVectorAdapter) Delete(_ context.Context, nodeID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.docs, nodeID)
	return nil
}

func (a *InMemoryVectorAdapter) DeleteBatch(ctx context.Context, docs []VectorDocument) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, doc := range docs {
		existing, ok := a.docs[doc.NodeID]
		if ok && existing.SyncVersion > doc.SyncVersion {
			continue
		}
		delete(a.docs, doc.NodeID)
	}
	return nil
}

func (a *InMemoryVectorAdapter) SnapshotDocuments() map[string]VectorDocument {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]VectorDocument, len(a.docs))
	for id, doc := range a.docs {
		result[id] = doc
	}
	return result
}

func (a *InMemoryVectorAdapter) Snapshot(_ context.Context) ([]VectorDocument, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]VectorDocument, 0, len(a.docs))
	for _, doc := range a.docs {
		result = append(result, doc)
	}
	return result, nil
}

func (a *InMemoryVectorAdapter) ReadSyncVersion(_ context.Context, entityID string) (int64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if doc, ok := a.docs[entityID]; ok {
		return doc.SyncVersion, nil
	}
	return 0, nil
}

func (a *InMemoryVectorAdapter) ANN(_ context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	results := make([]VectorResult, 0, len(a.docs))
	for _, doc := range a.docs {
		if doc.IsDeleted {
			continue
		}
		if len(filter.DomainIDs) > 0 && !contains(filter.DomainIDs, doc.DomainID) {
			continue
		}
		if len(filter.NodeTypes) > 0 && !contains(filter.NodeTypes, doc.NodeType) {
			continue
		}
		if len(filter.OwnerTenantIDs) > 0 && !contains(filter.OwnerTenantIDs, doc.OwnerTenantID) {
			continue
		}
		if len(filter.OwnerAppIDs) > 0 && !contains(filter.OwnerAppIDs, doc.OwnerAppID) {
			continue
		}
		if len(filter.ACLVisibleTo) > 0 && !intersects(filter.ACLVisibleTo, doc.ACLVisibleTo) {
			continue
		}
		score := cosine(query, doc.Embedding)
		if opts.MinScore > 0 && score < opts.MinScore {
			continue
		}
		results = append(results, VectorResult{Document: doc, Score: score})
	}
	slices.SortFunc(results, func(a, b VectorResult) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		return strings.Compare(a.Document.NodeID, b.Document.NodeID)
	})
	if opts.TopK > 0 && len(results) > opts.TopK {
		results = results[:opts.TopK]
	}
	return results, nil
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	dot := 0.0
	magA := 0.0
	magB := 0.0
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for _, left := range a {
		for _, right := range b {
			if left == right {
				return true
			}
		}
	}
	return false
}
