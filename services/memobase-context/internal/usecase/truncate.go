package usecase

import (
	"sort"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/model"
)

// rough token estimation (1 token ~ 4 chars for English)
func estimateTokens(text string) int32 {
	return int32(len(text) / 4)
}

func TruncateProfiles(profiles []*model.Profile, req *model.TruncationPolicy) []*model.Profile {
	// 1. Sort by updated_at DESC
	sort.SliceStable(profiles, func(i, j int) bool {
		return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
	})

	// 2. Filter only_topics
	if len(req.OnlyTopics) > 0 {
		onlyMap := make(map[string]bool)
		for _, t := range req.OnlyTopics {
			onlyMap[t] = true
		}
		var filtered []*model.Profile
		for _, p := range profiles {
			if onlyMap[p.Topic] {
				filtered = append(filtered, p)
			}
		}
		profiles = filtered
	}

	// 3. Move prefer_topics to front while keeping DESC order within groups
	if len(req.PreferTopics) > 0 {
		preferMap := make(map[string]bool)
		for _, t := range req.PreferTopics {
			preferMap[t] = true
		}
		var preferred []*model.Profile
		var others []*model.Profile
		for _, p := range profiles {
			if preferMap[p.Topic] {
				preferred = append(preferred, p)
			} else {
				others = append(others, p)
			}
		}
		profiles = append(preferred, others...)
	}

	// 4. Cap subtopics and apply token limit
	subtopicCount := make(map[string]int32)
	var result []*model.Profile
	var currentTokens int32 = 0

	for _, p := range profiles {
		// Cap subtopics
		if req.MaxSubtopicSize > 0 {
			key := p.Topic + "::" + p.SubTopic
			if subtopicCount[key] >= req.MaxSubtopicSize {
				continue
			}
			subtopicCount[key]++
		}

		// Calculate tokens
		pTokens := estimateTokens(p.Content) + estimateTokens(p.Topic) + estimateTokens(p.SubTopic)
		if req.MaxTokenSize > 0 && currentTokens+pTokens > req.MaxTokenSize {
			break
		}
		
		result = append(result, p)
		currentTokens += pTokens
	}

	return result
}
