package domain

import (
	"github.com/vnp-memory/services/ov-session/domain/model"
)

type SessionCommitted struct {
	SessionID   string `json:"session_id"`
	AccountID   string `json:"account_id"`
	ArchivePath string `json:"archive_path"`
}

type MemoryExtracted struct {
	SessionID string                  `json:"session_id"`
	Memories  []model.CandidateMemory `json:"memories"`
	FSPaths   []string                `json:"fs_paths"`
}
