package model

import (
	"time"
)

type MemoryCategory string

const (
	MemoryCategoryFact       MemoryCategory = "fact"
	MemoryCategoryPreference MemoryCategory = "preference"
	MemoryCategorySkill      MemoryCategory = "skill"
	MemoryCategoryProcedure  MemoryCategory = "procedure"
	MemoryCategoryToolSkill  MemoryCategory = "tool_skill"
)

type DedupAction string

const (
	DedupActionCreate  DedupAction = "CREATE"
	DedupActionMerge   DedupAction = "MERGE"
	DedupActionSkip    DedupAction = "SKIP"
	DedupActionArchive DedupAction = "ARCHIVE"
)

type CandidateMemory struct {
	ID          string
	SessionID   string
	AccountID   string
	Category    MemoryCategory
	Content     string
	Confidence  float64
	DedupAction DedupAction
	FSPath      string
	CreatedAt   time.Time
}
