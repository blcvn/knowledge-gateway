package usecase

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/services/sm-memory/domain/model"
)

// KnowledgeEvolutionUseCase detects contradictions between memories
// and resolves them intelligently: supersede, merge, or flag.
// SOL-INTEL-003 / TASK-INTEL-005
type KnowledgeEvolutionUseCase struct {
	memStore  MemoryStore
	llmClient LLMClient
}

// MemoryStore is the port for reading/writing memories.
type MemoryStore interface {
	GetByID(ctx context.Context, id string) (*model.Memory, error)
	ListByUser(ctx context.Context, tenantID, userID string, limit int) ([]*model.Memory, error)
	Update(ctx context.Context, m *model.Memory) error
	Create(ctx context.Context, m *model.Memory) error
	MarkSuperseded(ctx context.Context, oldID, newID string) error
}

// LLMClient is the port for LLM contradiction resolution.
type LLMClient interface {
	ResolvContradiction(ctx context.Context, oldMemory, newMemory string) (ContradictionResult, error)
}

// ContradictionResult represents the LLM resolution decision.
type ContradictionResult struct {
	Action  string // "supersede" | "merge" | "coexist"
	Merged  string // merged content if Action == "merge"
	Reason  string
}

// EvolveRequest is the input for knowledge evolution.
type EvolveRequest struct {
	TenantID  string
	UserID    string
	NewMemory string
	MemoryID  string // if updating existing
}

// EvolveResult is the output of knowledge evolution.
type EvolveResult struct {
	Action     string       // "supersede" | "merge" | "coexist" | "new"
	MemoryID   string       // resulting memory ID
	SupersededIDs []string  // IDs of superseded memories
}

// NewKnowledgeEvolutionUseCase creates a new KnowledgeEvolutionUseCase.
func NewKnowledgeEvolutionUseCase(store MemoryStore, llm LLMClient) *KnowledgeEvolutionUseCase {
	return &KnowledgeEvolutionUseCase{memStore: store, llmClient: llm}
}

// Evolve processes a new memory and resolves contradictions with existing ones.
// It uses an LLM to determine whether to supersede, merge, or coexist.
func (uc *KnowledgeEvolutionUseCase) Evolve(ctx context.Context, req EvolveRequest) (*EvolveResult, error) {
	// 1. Retrieve recent memories for this user
	existing, err := uc.memStore.ListByUser(ctx, req.TenantID, req.UserID, 50)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}

	var supersededIDs []string

	// 2. Check each existing memory for potential contradiction
	for _, mem := range existing {
		if mem.ID == req.MemoryID {
			continue // skip self
		}

		// 3. Ask LLM to resolve the potential contradiction
		result, err := uc.llmClient.ResolvContradiction(ctx, mem.ID, req.NewMemory)
		if err != nil {
			continue // non-fatal: skip this memory on LLM error
		}

		switch result.Action {
		case "supersede":
			// New memory supersedes the old one
			if markErr := uc.memStore.MarkSuperseded(ctx, mem.ID, req.MemoryID); markErr == nil {
				supersededIDs = append(supersededIDs, mem.ID)
			}
		case "merge":
			// Merge old + new into a single updated memory
			merged := &model.Memory{
				ID:        mem.ID,
				CreatedAt: time.Now(),
			}
			if updateErr := uc.memStore.Update(ctx, merged); updateErr == nil {
				return &EvolveResult{
					Action:   "merge",
					MemoryID: mem.ID,
				}, nil
			}
		case "coexist":
			// Both memories can coexist — no action needed
		}
	}

	return &EvolveResult{
		Action:        "supersede",
		MemoryID:      req.MemoryID,
		SupersededIDs: supersededIDs,
	}, nil
}
