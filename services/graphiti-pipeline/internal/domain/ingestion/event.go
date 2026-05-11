package ingestion

import "time"

type EpisodeIngested struct {
	EpisodeID  string    `json:"episode_id"`
	GroupID    string    `json:"group_id"`
	NodesCount int       `json:"nodes_count"`
	EdgesCount int       `json:"edges_count"`
	Timestamp  time.Time `json:"timestamp"`
}

type EpisodeFailed struct {
	EpisodeID string    `json:"episode_id"`
	GroupID   string    `json:"group_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}
