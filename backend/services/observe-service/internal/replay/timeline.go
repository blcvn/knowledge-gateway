package replay

import (
	"sort"
	"time"

	"github.com/vnp-memory/services/observe-service/internal/domain"
)

type TimelineEntry struct {
	Index     int
	Timestamp time.Time
	HookType  string
	ToolName  string
	ObsType   string
	Title     string
	Facts     []string
	Duration  time.Duration // time since previous event
	AgentID   string
}

type Timeline struct {
	SessionID   string
	Project     string
	TotalEvents int
	Duration    time.Duration // total session duration
	Entries     []TimelineEntry
}

// BuildTimeline creates a sorted, annotated timeline from observations
func BuildTimeline(sessionID, project string, obs []domain.CompressedObservation) Timeline {
	if len(obs) == 0 {
		return Timeline{SessionID: sessionID, Project: project}
	}

	sort.Slice(obs, func(i, j int) bool { return obs[i].Timestamp.Before(obs[j].Timestamp) })

	entries := make([]TimelineEntry, len(obs))
	var prevTime time.Time
	for i, o := range obs {
		var dur time.Duration
		if i > 0 {
			dur = o.Timestamp.Sub(prevTime)
		}
		entries[i] = TimelineEntry{
			Index:     i,
			Timestamp: o.Timestamp,
			HookType:  o.ObsType,
			ObsType:   o.ObsType,
			Title:     o.Title,
			Facts:     o.Facts,
			Duration:  dur,
			AgentID:   o.AgentID,
		}
		prevTime = o.Timestamp
	}

	totalDur := obs[len(obs)-1].Timestamp.Sub(obs[0].Timestamp)
	return Timeline{
		SessionID:   sessionID,
		Project:     project,
		TotalEvents: len(obs),
		Duration:    totalDur,
		Entries:     entries,
	}
}

// Filter returns subset matching filters
func (t Timeline) Filter(hookTypes []string, fromIdx, toIdx int) Timeline {
	if len(hookTypes) == 0 && fromIdx == 0 && toIdx == 0 {
		return t
	}
	hookSet := make(map[string]bool, len(hookTypes))
	for _, h := range hookTypes {
		hookSet[h] = true
	}

	var filtered []TimelineEntry
	for _, e := range t.Entries {
		if fromIdx > 0 && e.Index < fromIdx {
			continue
		}
		if toIdx > 0 && e.Index > toIdx {
			continue
		}
		if len(hookSet) > 0 && !hookSet[e.HookType] {
			continue
		}
		filtered = append(filtered, e)
	}
	t.Entries = filtered
	t.TotalEvents = len(filtered)
	return t
}
