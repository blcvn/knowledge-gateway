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
	ValidAt   *time.Time `json:"valid_at"`   // when fact became true
	InvalidAt *time.Time `json:"invalid_at"` // when fact was superseded/invalidated (nil = still valid)
	ExpiredAt *time.Time `json:"expired_at"` // system time when invalidation was recorded
	GroupID   string     `json:"group_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// IsValid reports whether the edge is currently valid (not invalidated)
func (e *EntityEdge) IsValid() bool {
	return e.InvalidAt == nil
}

// IsValidAt reports whether the edge was valid at the given point in time
func (e *EntityEdge) IsValidAt(t time.Time) bool {
	if e.ValidAt != nil && e.ValidAt.After(t) {
		return false // not yet valid
	}
	if e.InvalidAt != nil && !e.InvalidAt.After(t) {
		return false // already invalidated
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
