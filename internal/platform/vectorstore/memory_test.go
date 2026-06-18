package vectorstore

import (
	"context"
	"testing"
)

func TestInMemoryVectorAdapterANN(t *testing.T) {
	adapter := NewInMemoryVectorAdapter()
	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	must(adapter.Upsert(context.Background(), VectorDocument{NodeID: "a", DomainID: "d", ACLVisibleTo: []string{"t:a"}, Embedding: []float64{1, 0}}))
	must(adapter.Upsert(context.Background(), VectorDocument{NodeID: "b", DomainID: "d", ACLVisibleTo: []string{"t:a"}, Embedding: []float64{0, 1}}))

	results, err := adapter.ANN(context.Background(), []float64{1, 0}, VectorFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, ANNOptions{TopK: 1})
	if err != nil {
		t.Fatalf("ANN() error = %v", err)
	}
	if len(results) != 1 || results[0].Document.NodeID != "a" {
		t.Fatalf("results = %#v, want a", results)
	}
}

func TestInMemoryVectorAdapterDelete(t *testing.T) {
	adapter := NewInMemoryVectorAdapter()
	_ = adapter.Upsert(context.Background(), VectorDocument{NodeID: "a", DomainID: "d", ACLVisibleTo: []string{"t:a"}, Embedding: []float64{1, 0}})
	_ = adapter.Delete(context.Background(), "a")
	results, err := adapter.ANN(context.Background(), []float64{1, 0}, VectorFilter{DomainIDs: []string{"d"}}, ANNOptions{TopK: 10})
	if err != nil {
		t.Fatalf("ANN() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %#v, want none", results)
	}
}
