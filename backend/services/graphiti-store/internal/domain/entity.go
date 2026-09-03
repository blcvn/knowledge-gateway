package domain

import (
	"fmt"
	"time"
)

// EntityNode represents a named entity extracted from episodes.
// It is the primary node type in the temporal knowledge graph.
type EntityNode struct {
	UUID          string            `json:"uuid"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id"`
	Summary       string            `json:"summary,omitempty"`
	NameEmbedding []float32         `json:"name_embedding,omitempty"`
	Labels        []string          `json:"labels,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Validate checks that the EntityNode has all required fields.
func (n *EntityNode) Validate() error {
	if n.UUID == "" {
		return ErrMissingUUID
	}
	if n.Name == "" {
		return ErrMissingName
	}
	if n.GroupID == "" {
		return ErrMissingGroupID
	}
	return nil
}

// EpisodicNode represents a source episode content node.
// Each episode is a unit of information ingested into the graph.
type EpisodicNode struct {
	UUID        string    `json:"uuid"`
	Name        string    `json:"name"`
	GroupID     string    `json:"group_id"`
	Content     string    `json:"content"`
	Source      string    `json:"source"` // EpisodeType: message, json, text, fact_triple
	SourceDesc  string    `json:"source_desc,omitempty"`
	ValidAt     time.Time `json:"valid_at"`
	EntityEdges []string  `json:"entity_edges,omitempty"` // UUIDs of related entity edges
	CreatedAt   time.Time `json:"created_at"`
}

// Validate checks that the EpisodicNode has all required fields.
func (n *EpisodicNode) Validate() error {
	if n.UUID == "" {
		return ErrMissingUUID
	}
	if n.GroupID == "" {
		return ErrMissingGroupID
	}
	if n.Content == "" {
		return ErrEmptyContent
	}
	if n.ValidAt.IsZero() {
		return ErrMissingValidAt
	}
	return nil
}

// CommunityNode represents a cluster of related entities
// detected via label propagation, with an LLM-generated summary.
type CommunityNode struct {
	UUID          string    `json:"uuid"`
	Name          string    `json:"name"`
	GroupID       string    `json:"group_id"`
	Summary       string    `json:"summary,omitempty"`
	NameEmbedding []float32 `json:"name_embedding,omitempty"`
	Level         int       `json:"level"` // Community hierarchy level
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate checks that the CommunityNode has all required fields.
func (n *CommunityNode) Validate() error {
	if n.UUID == "" {
		return ErrMissingUUID
	}
	if n.GroupID == "" {
		return ErrMissingGroupID
	}
	return nil
}

// SagaNode represents an episode grouping for conversations.
// It links multiple episodes in temporal order.
type SagaNode struct {
	UUID             string    `json:"uuid"`
	Name             string    `json:"name"`
	GroupID          string    `json:"group_id"`
	Summary          string    `json:"summary,omitempty"`
	FirstEpisodeUUID string    `json:"first_episode_uuid,omitempty"`
	LastEpisodeUUID  string    `json:"last_episode_uuid,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Validate checks that the SagaNode has all required fields.
func (n *SagaNode) Validate() error {
	if n.UUID == "" {
		return ErrMissingUUID
	}
	if n.GroupID == "" {
		return ErrMissingGroupID
	}
	return nil
}

// String returns a human-readable representation of each node type.

func (n *EntityNode) String() string {
	return fmt.Sprintf("Entity{uuid=%s, name=%s, group=%s}", n.UUID, n.Name, n.GroupID)
}

func (n *EpisodicNode) String() string {
	return fmt.Sprintf("Episodic{uuid=%s, name=%s, group=%s}", n.UUID, n.Name, n.GroupID)
}

func (n *CommunityNode) String() string {
	return fmt.Sprintf("Community{uuid=%s, name=%s, group=%s, level=%d}", n.UUID, n.Name, n.GroupID, n.Level)
}

func (n *SagaNode) String() string {
	return fmt.Sprintf("Saga{uuid=%s, name=%s, group=%s}", n.UUID, n.Name, n.GroupID)
}
