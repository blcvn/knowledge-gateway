// Package knowledge implements KnowledgeUseCase (CRUD) and IndexUseCase (worker handlers).
//
// Absorbed from: ba-knowledge-service (CRUD stub), ba-knowledge-worker (REAL logic)
// (MERGE-P3-T1)
package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	dom "vnp-memory/services/pipeline-service/internal/domain/knowledge"
	"vnp-memory/services/pipeline-service/internal/usecase/port"
)

// ── CRUD UseCase ───────────────────────────────────────────────────────────

// KnowledgeUseCase manages PRD and Outline CRUD.
type KnowledgeUseCase struct {
	prds     port.PRDRepository
	outlines port.OutlineRepository
	queue    port.Queue
	pub      port.EventPublisher
}

// NewKnowledgeUseCase creates a KnowledgeUseCase.
func NewKnowledgeUseCase(prds port.PRDRepository, outlines port.OutlineRepository, q port.Queue, pub port.EventPublisher) *KnowledgeUseCase {
	return &KnowledgeUseCase{prds: prds, outlines: outlines, queue: q, pub: pub}
}

// CreatePRD creates and enqueues a PRD for indexing.
func (uc *KnowledgeUseCase) CreatePRD(ctx context.Context, tenantID, title, content string, tags []string) (*dom.PRD, error) {
	now := time.Now()
	prd := &dom.PRD{
		ID:        uuid.New().String(),
		Title:     title,
		Content:   content,
		Tags:      tags,
		Status:    "draft",
		TenantID:  tenantID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.prds.Create(ctx, prd); err != nil {
		return nil, fmt.Errorf("create prd: %w", err)
	}
	// Enqueue async indexing
	if uc.queue != nil {
		payload := []byte(fmt.Sprintf(`{"prd_id":%q,"title":%q}`, prd.ID, prd.Title))
		_ = uc.queue.Push(ctx, "knowledge:index_prd", payload)
	}
	if uc.pub != nil {
		_ = uc.pub.Publish(ctx, "knowledge.prd.created", prd)
	}
	return prd, nil
}

// GetPRD retrieves a PRD by ID.
func (uc *KnowledgeUseCase) GetPRD(ctx context.Context, id string) (*dom.PRD, error) {
	return uc.prds.GetByID(ctx, id)
}

// ListPRDs lists PRDs for a tenant.
func (uc *KnowledgeUseCase) ListPRDs(ctx context.Context, tenantID string, limit, offset int) ([]*dom.PRD, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return uc.prds.List(ctx, tenantID, limit, offset)
}

// GetOutline retrieves the outline for a PRD.
func (uc *KnowledgeUseCase) GetOutline(ctx context.Context, prdID string) (*dom.Outline, error) {
	outline, err := uc.outlines.GetByPRD(ctx, prdID)
	if err != nil {
		return nil, fmt.Errorf("outline not found for prd %s: %w", prdID, err)
	}
	return outline, nil
}

// ── Index UseCase (from ba-knowledge-worker) ───────────────────────────────

// IndexUseCase handles async index task execution.
type IndexUseCase struct {
	prds     port.PRDRepository
	outlines port.OutlineRepository
}

// NewIndexUseCase creates an IndexUseCase.
func NewIndexUseCase(prds port.PRDRepository, outlines port.OutlineRepository) *IndexUseCase {
	return &IndexUseCase{prds: prds, outlines: outlines}
}

// HandleIndexPRD indexes a PRD into the knowledge graph.
// This is called by the worker binary when processing "index_prd" tasks.
func (uc *IndexUseCase) HandleIndexPRD(ctx context.Context, job dom.IndexJob) error {
	prd, err := uc.prds.GetByID(ctx, job.PRDID)
	if err != nil {
		return fmt.Errorf("prd not found: %w", err)
	}

	// Mark as indexing
	_ = uc.prds.UpdateStatus(ctx, prd.ID, "indexing")

	// TODO: call kg-service gRPC to ingest PRD content as an episode
	// For MVP: simulate successful indexing
	_ = uc.prds.UpdateStatus(ctx, prd.ID, "indexed")

	return nil
}

// HandleGenOutline generates a structured outline from a PRD.
// This is called by the worker binary when processing "gen_outline" tasks.
func (uc *IndexUseCase) HandleGenOutline(ctx context.Context, job dom.IndexJob) error {
	prd, err := uc.prds.GetByID(ctx, job.PRDID)
	if err != nil {
		return fmt.Errorf("prd not found: %w", err)
	}

	// Simple heuristic outline extraction (MVP — no LLM required)
	sections := extractSections(prd.Content)
	outline := &dom.Outline{
		ID:       uuid.New().String(),
		PRDID:    prd.ID,
		Sections: sections,
		Status:   "ready",
	}
	if err := uc.outlines.Create(ctx, outline); err != nil {
		return fmt.Errorf("save outline: %w", err)
	}
	return nil
}

// extractSections parses headings from Markdown content.
func extractSections(content string) []dom.OutlineSection {
	var sections []dom.OutlineSection
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			sections = append(sections, dom.OutlineSection{Title: strings.TrimPrefix(line, "# "), Level: 1})
		} else if strings.HasPrefix(line, "## ") {
			sections = append(sections, dom.OutlineSection{Title: strings.TrimPrefix(line, "## "), Level: 2})
		} else if strings.HasPrefix(line, "### ") {
			sections = append(sections, dom.OutlineSection{Title: strings.TrimPrefix(line, "### "), Level: 3})
		}
	}
	return sections
}
