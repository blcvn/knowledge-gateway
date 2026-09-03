# TASK-GR-001 — Shared Graph Types Package

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-001 |
| **Wave** | 1 (Foundation) |
| **Component** | `pkg/graph/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §2, SOL-005 §2 |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 2h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** shared/pkg/graph: 4 .go (Node, Edge, Community types)  
---

## Context

Tạo shared package `pkg/graph/` chứa domain types được dùng bởi TẤT CẢ graphiti services (ingestion, store, knowledge, search, admin). Package này là nền tảng — phải được tạo trước bất kỳ service nào.

---

## Goal

- `pkg/graph/node.go` — EntityNode, EpisodicNode, CommunityNode, SagaNode
- `pkg/graph/edge.go` — EntityEdge, EpisodicEdge, CommunityEdge, HasEpisodeEdge, NextEpisodeEdge
- `pkg/graph/ontology.go` — EntityTypeSchema, EdgeTypeSchema, OntologyRegistry, OntologyProperty

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `pkg/graph/node.go` |
| CREATE | `pkg/graph/edge.go` |
| CREATE | `pkg/graph/ontology.go` |
| CREATE | `pkg/graph/node_test.go` |
| CREATE | `pkg/graph/edge_test.go` |

---

## Implementation

### File 1: `pkg/graph/node.go`

```go
package graph

import "time"

// EpisodeType defines the format of episode content
type EpisodeType string

const (
    EpisodeTypeMessage    EpisodeType = "message"
    EpisodeTypeText       EpisodeType = "text"
    EpisodeTypeJSON       EpisodeType = "json"
    EpisodeTypeFactTriple EpisodeType = "fact_triple"
)

// EntityNode represents a named entity in the knowledge graph.
// Stored as Neo4j node with label :Entity
type EntityNode struct {
    UUID          string         `json:"uuid"`
    Name          string         `json:"name"`
    Labels        []string       `json:"labels"`        // entity type labels (e.g. ["Person"])
    Summary       string         `json:"summary"`       // LLM-generated summary
    Attributes    map[string]any `json:"attributes"`    // custom properties from ontology
    NameEmbedding []float32      `json:"name_embedding"` // 1536-dim vector
    GroupID       string         `json:"group_id"`       // multi-tenant partition key
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
}

// EpisodicNode represents a memory episode (an event or piece of content).
// Stored as Neo4j node with label :Episodic
type EpisodicNode struct {
    UUID              string         `json:"uuid"`
    Name              string         `json:"name"`
    Content           string         `json:"content"`            // raw episode text
    Source            EpisodeType    `json:"source"`
    SourceDescription string         `json:"source_description"`
    ValidAt           time.Time      `json:"valid_at"`           // when event occurred
    EntityEdges       []string       `json:"entity_edges"`       // UUIDs of MENTIONS edges
    EpisodeMetadata   map[string]any `json:"episode_metadata"`
    GroupID           string         `json:"group_id"`
    CreatedAt         time.Time      `json:"created_at"`
}

// CommunityNode represents a cluster of related entities.
// Built by Label Propagation + LLM summarization.
// Stored as Neo4j node with label :Community
type CommunityNode struct {
    UUID          string    `json:"uuid"`
    Name          string    `json:"name"`    // LLM-generated community name
    Summary       string    `json:"summary"` // LLM-generated community description
    NameEmbedding []float32 `json:"name_embedding"`
    GroupID       string    `json:"group_id"`
    CreatedAt     time.Time `json:"created_at"`
}

// SagaNode represents a narrative sequence of related episodes.
// Stored as Neo4j node with label :Saga
type SagaNode struct {
    UUID             string     `json:"uuid"`
    Name             string     `json:"name"`
    GroupID          string     `json:"group_id"`
    Summary          string     `json:"summary"`           // incremental LLM summary
    FirstEpisodeUUID string     `json:"first_episode_uuid"`
    LastEpisodeUUID  string     `json:"last_episode_uuid"`
    LastSummarizedAt *time.Time `json:"last_summarized_at"` // nil = never summarized
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}
```

### File 2: `pkg/graph/edge.go`

```go
package graph

import "time"

// EntityEdge represents a fact/relationship between two EntityNodes.
// This is the primary data carrier of the temporal knowledge graph.
// Stored as Neo4j relationship :RELATES_TO with bi-temporal fields.
//
// TEMPORAL INVARIANT: EntityEdge is NEVER deleted.
// Contradicting facts are marked with invalid_at; historical queries
// can use valid_at + invalid_at for point-in-time snapshots.
type EntityEdge struct {
    UUID           string     `json:"uuid"`
    SourceNodeUUID string     `json:"source_node_uuid"`
    TargetNodeUUID string     `json:"target_node_uuid"`
    Name           string     `json:"name"`           // relation type (e.g. "WORKS_AT")
    Fact           string     `json:"fact"`           // natural language fact
    FactEmbedding  []float32  `json:"fact_embedding"` // 1536-dim vector
    Episodes       []string   `json:"episodes"`       // episode UUIDs that mentioned this fact
    // Bi-temporal fields
    ValidAt        *time.Time `json:"valid_at"`        // when fact became true
    InvalidAt      *time.Time `json:"invalid_at"`      // when fact was superseded/invalidated (nil = still valid)
    ExpiredAt      *time.Time `json:"expired_at"`      // system time when invalidation was recorded
    GroupID        string     `json:"group_id"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

// IsValid reports whether the edge is currently valid (not invalidated)
func (e *EntityEdge) IsValid() bool {
    return e.InvalidAt == nil
}

// IsValidAt reports whether the edge was valid at the given point in time
func (e *EntityEdge) IsValidAt(t time.Time) bool {
    if e.ValidAt != nil && e.ValidAt.After(t) {
        return false  // not yet valid
    }
    if e.InvalidAt != nil && !e.InvalidAt.After(t) {
        return false  // already invalidated
    }
    return true
}

// EpisodicEdge represents a MENTIONS relationship between an episode and an entity.
// episode -[MENTIONS]-> entity
type EpisodicEdge struct {
    UUID       string    `json:"uuid"`
    SourceUUID string    `json:"source_uuid"` // episode UUID
    TargetUUID string    `json:"target_uuid"` // entity UUID
    GroupID    string    `json:"group_id"`
    CreatedAt  time.Time `json:"created_at"`
}

// CommunityEdge represents HAS_MEMBER relationship: community → entity
// community -[HAS_MEMBER]-> entity
type CommunityEdge struct {
    UUID       string    `json:"uuid"`
    SourceUUID string    `json:"source_uuid"` // community UUID
    TargetUUID string    `json:"target_uuid"` // entity UUID
    GroupID    string    `json:"group_id"`
    CreatedAt  time.Time `json:"created_at"`
}

// HasEpisodeEdge represents HAS_EPISODE relationship: saga → episode
// saga -[HAS_EPISODE]-> episode
type HasEpisodeEdge struct {
    UUID       string    `json:"uuid"`
    SourceUUID string    `json:"source_uuid"` // saga UUID
    TargetUUID string    `json:"target_uuid"` // episode UUID
    GroupID    string    `json:"group_id"`
    CreatedAt  time.Time `json:"created_at"`
}

// NextEpisodeEdge represents sequential ordering within a saga
// episode_n -[NEXT_EPISODE]-> episode_n+1
type NextEpisodeEdge struct {
    UUID       string    `json:"uuid"`
    SourceUUID string    `json:"source_uuid"` // previous episode UUID
    TargetUUID string    `json:"target_uuid"` // next episode UUID
    GroupID    string    `json:"group_id"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### File 3: `pkg/graph/ontology.go`

```go
package graph

import "time"

// OntologyProperty defines a typed property within an entity/edge schema
type OntologyProperty struct {
    Name        string `json:"name"`
    Type        string `json:"type"`        // string | number | boolean | datetime
    Description string `json:"description"`
    Required    bool   `json:"required"`
}

// EntityTypeSchema — prescribed entity type definition.
// When registered for a group_id, LLM extraction is constrained to these types.
type EntityTypeSchema struct {
    Name        string             `json:"name"`
    Description string             `json:"description"`
    Properties  []OntologyProperty `json:"properties,omitempty"`
    Examples    []string           `json:"examples,omitempty"`
}

// EdgeTypeSchema — prescribed relationship type definition.
// Constrains LLM to only extract relationships of these types between valid source/target types.
type EdgeTypeSchema struct {
    Name        string             `json:"name"`
    Description string             `json:"description"`
    SourceTypes []string           `json:"source_types,omitempty"` // allowed source entity types
    TargetTypes []string           `json:"target_types,omitempty"` // allowed target entity types
    Properties  []OntologyProperty `json:"properties,omitempty"`
    Examples    []string           `json:"examples,omitempty"`
}

// OntologyRegistry — per group_id ontology configuration.
// nil/empty = "learned ontology" (LLM chooses any label freely).
// Non-empty = "prescribed ontology" (LLM constrained to defined types).
type OntologyRegistry struct {
    GroupID     string                      `json:"group_id"`
    EntityTypes map[string]EntityTypeSchema `json:"entity_types"` // key: type name
    EdgeTypes   map[string]EdgeTypeSchema   `json:"edge_types"`   // key: relation name
    CreatedAt   time.Time                   `json:"created_at"`
    UpdatedAt   time.Time                   `json:"updated_at"`
}

// IsPrescribed returns true if this registry has defined types (not learned mode)
func (r *OntologyRegistry) IsPrescribed() bool {
    return r != nil && len(r.EntityTypes) > 0
}
```

### File 4: `pkg/graph/node_test.go`

```go
package graph_test

import (
    "testing"
    "time"

    "github.com/vnp-memory/pkg/graph"
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
```

---

## Verification

```bash
cd /path/to/vnp-memory
go build ./pkg/graph/...
go test ./pkg/graph/... -v
```

**Expected output:**
```
--- PASS: TestEntityEdge_IsValid
--- PASS: TestEntityEdge_IsValidAt
--- PASS: TestOntologyRegistry_IsPrescribed
PASS
```
