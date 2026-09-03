package consolidation

import (
	"context"
	"log"
	"time"

	"vnp-memory/services/memory-service/internal/consolidation/port"

	"github.com/google/uuid"
)

type ConsolidationConfig struct {
	IntervalHours      int // default 2
	DecayDays          int // default 30
	MaxMemoriesProject int // default 1000
	MinProcedureFreq   int // default 3
	LessonHalfLifeDays int // default 90
	AutoCompress       bool
}

type ConsolidationPipeline struct {
	observeRepo    port.IObservationRepo
	memRepo        port.IAgentMemoryRepo
	summaryRepo    port.ISessionSummaryRepo
	proceduralRepo port.IProceduralRepo
	lessonRepo     port.ILessonRepo
	compressor     *Compressor
	summarizer     *Summarizer
	procedural     *ProceduralExtractor
	lessons        *LessonExtractor
	config         ConsolidationConfig
}

func (p *ConsolidationPipeline) Start(ctx context.Context) {
	interval := time.Duration(p.config.IntervalHours) * time.Hour
	if interval == 0 {
		interval = 2 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (p *ConsolidationPipeline) run(ctx context.Context) {
	log.Println("[consolidation] pipeline starting")
	p.step1WorkingToEpisodic(ctx)
	p.step2EpisodicToSemantic(ctx)
	p.step3SemanticToProcedural(ctx)
	p.step4DecayAndEvict(ctx)
	log.Println("[consolidation] pipeline completed")
}

func (p *ConsolidationPipeline) step1WorkingToEpisodic(ctx context.Context) {
	sessions, _ := p.observeRepo.ListSessionsNeedingCompression(ctx)
	for _, sess := range sessions {
		rawObs, _ := p.observeRepo.ListRawUncompressed(ctx, sess.ID)
		for _, raw := range rawObs {
			comp := p.compressor.Compress(ctx, raw)
			comp.ID = uuid.New().String()
			p.observeRepo.SaveCompressed(ctx, comp)
		}
	}
}

func (p *ConsolidationPipeline) step2EpisodicToSemantic(ctx context.Context) {
	sessions, _ := p.observeRepo.ListCompletedSessionsWithoutSummary(ctx)
	for _, sess := range sessions {
		summary := p.summarizer.Summarize(ctx, sess.ID)
		if summary != nil {
			p.summaryRepo.Save(ctx, *summary)
		}
	}
}

func (p *ConsolidationPipeline) step3SemanticToProcedural(ctx context.Context) {
	p.procedural.ExtractAll(ctx)
	p.lessons.ExtractAll(ctx)
	p.lessons.SynthesizeInsights(ctx)
}

func (p *ConsolidationPipeline) step4DecayAndEvict(ctx context.Context) {
	p.lessons.ApplyDecay(ctx)
	// Evict memories over max limit
	p.evictIfNeeded(ctx)
}

// SummarizeNow is called immediately from NATS consumer
func (p *ConsolidationPipeline) SummarizeNow(ctx context.Context, sessionID string) {
	summary := p.summarizer.Summarize(ctx, sessionID)
	if summary != nil {
		p.summaryRepo.Save(ctx, *summary)
	}
}

func (p *ConsolidationPipeline) evictIfNeeded(ctx context.Context) {}
