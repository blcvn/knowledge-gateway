package graph_test

import (
	"testing"
	"time"

	"vnp-memory/pkg/graph"
)

func TestEntityEdge_IsValid(t *testing.T) {
	e := graph.EntityEdge{UUID: "e1"}
	if !e.IsValid() {
		t.Error("edge with nil InvalidAt should be valid")
	}

	now := time.Now()
	e.InvalidAt = &now
	if e.IsValid() {
		t.Error("edge with InvalidAt set should not be valid")
	}
}

func TestEntityEdge_IsValidAt(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	present := time.Now()
	future := time.Now().Add(2 * time.Hour)

	validAt := time.Now().Add(-1 * time.Hour)
	invalidAt := time.Now().Add(1 * time.Hour)

	e := graph.EntityEdge{
		ValidAt:   &validAt,
		InvalidAt: &invalidAt,
	}

	// Before valid_at — should NOT be valid
	if e.IsValidAt(past) {
		t.Error("edge should not be valid before valid_at")
	}

	// Between valid_at and invalid_at — SHOULD be valid
	if !e.IsValidAt(present) {
		t.Error("edge should be valid between valid_at and invalid_at")
	}

	// After invalid_at — should NOT be valid
	if e.IsValidAt(future) {
		t.Error("edge should not be valid after invalid_at")
	}
}

func TestOntologyRegistry_IsPrescribed(t *testing.T) {
	var nilReg *graph.OntologyRegistry
	if nilReg.IsPrescribed() {
		t.Error("nil registry should not be prescribed")
	}

	empty := &graph.OntologyRegistry{}
	if empty.IsPrescribed() {
		t.Error("empty registry should not be prescribed")
	}

	prescribed := &graph.OntologyRegistry{
		EntityTypes: map[string]graph.EntityTypeSchema{
			"Person": {Name: "Person"},
		},
	}
	if !prescribed.IsPrescribed() {
		t.Error("registry with entity types should be prescribed")
	}
}
