package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type GroupID string

func (g GroupID) String() string {
	return string(g)
}

type EpisodeID string

func (e EpisodeID) String() string {
	return string(e)
}

type ContentHash string

func (c ContentHash) String() string {
	return string(c)
}

func ComputeContentHash(name, groupID string, referenceTime time.Time) string {
	data := fmt.Sprintf("%s:%s:%d", name, groupID, referenceTime.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
