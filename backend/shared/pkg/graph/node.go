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
