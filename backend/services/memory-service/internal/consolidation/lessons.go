package consolidation

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/consolidation/port"

	"github.com/google/uuid"
)

type LessonExtractor struct {
	summaryRepo  port.ISessionSummaryRepo
	lessonRepo   port.ILessonRepo
	insightRepo  port.IInsightRepo
	llm          port.ILLMProvider
	halfLifeDays int
}

func (e *LessonExtractor) ExtractAll(ctx context.Context) {
	summaries, _ := e.summaryRepo.ListRecent(ctx, 100)
	for _, s := range summaries {
		if len(s.KeyDecisions) == 0 {
			continue
		}
		lesson := agentmemory.Lesson{
			ID:         uuid.New().String(),
			Content:    s.KeyDecisions[0],
			Source:     s.SessionID,
			Confidence: 0.7,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if e.llm != nil {
			e.enrichLesson(ctx, &lesson, s)
		}
		e.lessonRepo.Save(ctx, lesson)
	}
}

func (e *LessonExtractor) ApplyDecay(ctx context.Context) {
	lessons, _ := e.lessonRepo.ListAll(ctx)
	for _, l := range lessons {
		hoursSince := time.Since(l.UpdatedAt).Hours()
		factor := math.Exp(-hoursSince / (float64(e.halfLifeDays) * 24))
		l.Confidence *= factor
		e.lessonRepo.UpdateConfidence(ctx, l.ID, l.Confidence)
	}
}

func (e *LessonExtractor) SynthesizeInsights(ctx context.Context) {
	lessons, _ := e.lessonRepo.ListHighConfidence(ctx, 0.7, 50)
	if len(lessons) < 3 {
		return
	}

	prompt := "Synthesize these lessons into 1-3 key insights:\n"
	ids := make([]string, len(lessons))
	for i, l := range lessons {
		prompt += "- " + l.Content + "\n"
		ids[i] = l.ID
	}

	if e.llm == nil {
		return
	}
	resp, err := e.llm.Chat(ctx, "Synthesize software engineering insights.", prompt)
	if err != nil {
		return
	}

	var result struct {
		Insights []string `json:"insights"`
	}
	if json.Unmarshal([]byte(resp), &result) != nil {
		return
	}

	for _, insight := range result.Insights {
		e.insightRepo.Save(ctx, agentmemory.Insight{
			ID:         uuid.New().String(),
			Content:    insight,
			LessonIDs:  ids,
			Confidence: 0.6,
			CreatedAt:  time.Now(),
		})
	}
}

func (e *LessonExtractor) enrichLesson(ctx context.Context, l *agentmemory.Lesson, s agentmemory.SessionSummary) {
	// Categorize using LLM
	prompt := "Categorize this lesson: " + l.Content + ". Return JSON: {\"categories\": [\"...\"]}"
	resp, _ := e.llm.Chat(ctx, "You categorize software lessons.", prompt)
	var r struct {
		Categories []string `json:"categories"`
	}
	json.Unmarshal([]byte(resp), &r)
	l.Categories = r.Categories
}
