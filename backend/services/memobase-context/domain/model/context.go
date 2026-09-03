package model

import "strings"

type ContextResult struct {
	Context      string `json:"context"`
	ProfileCount int32  `json:"profile_count"`
	EventCount   int32  `json:"event_count"`
	TotalTokens  int32  `json:"total_tokens"`
}

type PromptTemplate struct {
	Template string
}

func (pt *PromptTemplate) Assemble(profileSection, eventSection string) string {
	res := pt.Template
	res = strings.ReplaceAll(res, "{profile_section}", profileSection)
	res = strings.ReplaceAll(res, "{event_section}", eventSection)
	return res
}

func DefaultPromptTemplate() PromptTemplate {
	return PromptTemplate{
		Template: "# Memory\nUnless the user has relevant queries, do not actively mention those memories.\n## User Background:\n{profile_section}\n## Latest Events:\n{event_section}",
	}
}

type TruncationPolicy struct {
	PreferTopics    []string
	OnlyTopics      []string
	MaxTokenSize    int32
	MaxSubtopicSize int32
}
