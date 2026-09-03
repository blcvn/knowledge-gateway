package ingestion

import (
	"errors"
	"strings"
)

type GroupID string

func (g GroupID) String() string {
	return string(g)
}

func (g GroupID) Validate() error {
	if strings.TrimSpace(string(g)) == "" {
		return errors.New("group_id cannot be empty")
	}
	return nil
}

type EpisodeID string

func (e EpisodeID) String() string {
	return string(e)
}

func (e EpisodeID) Validate() error {
	if strings.TrimSpace(string(e)) == "" {
		return errors.New("episode_id cannot be empty")
	}
	return nil
}

type ContentHash string

func (c ContentHash) String() string {
	return string(c)
}

func (c ContentHash) Validate() error {
	if strings.TrimSpace(string(c)) == "" {
		return errors.New("content_hash cannot be empty")
	}
	return nil
}
