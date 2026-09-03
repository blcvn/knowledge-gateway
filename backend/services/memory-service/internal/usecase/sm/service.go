// Package sm implements Supermemory usecases.
//
// Absorbed from: sm-memory, sm-document, sm-profile
// (MERGE-P2-T3)
package sm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vnp-memory/services/memory-service/internal/domain/sm"
	"vnp-memory/services/memory-service/internal/usecase/port"
)

// MemoryUseCase handles SM memory CRUD and RAG.
type MemoryUseCase struct {
	repo     port.SMMemoryRepository
	embedder port.EmbeddingService
}

// NewMemoryUseCase creates a MemoryUseCase.
func NewMemoryUseCase(repo port.SMMemoryRepository, embedder port.EmbeddingService) *MemoryUseCase {
	return &MemoryUseCase{repo: repo, embedder: embedder}
}

// CreateMemory persists a new SM memory entry.
func (uc *MemoryUseCase) CreateMemory(ctx context.Context, tenantID, content string, tags []string) (*sm.SMMemory, error) {
	mem := &sm.SMMemory{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now(),
	}
	if uc.embedder != nil {
		if emb, err := uc.embedder.Embed(ctx, content); err == nil {
			mem.Embedding = emb
		}
	}
	return mem, uc.repo.Create(ctx, mem)
}

// RAG returns retrieval-augmented context from SM memories.
func (uc *MemoryUseCase) RAG(ctx context.Context, tenantID, query string, limit int) (*sm.RAGResponse, error) {
	if limit <= 0 {
		limit = 5
	}
	var memories []*sm.SMMemory
	var err error

	if uc.embedder != nil {
		emb, embErr := uc.embedder.Embed(ctx, query)
		if embErr == nil {
			memories, err = uc.repo.SemanticSearch(ctx, tenantID, emb, limit)
		}
	}
	if err != nil || len(memories) == 0 {
		memories, err = uc.repo.List(ctx, tenantID, limit)
		if err != nil {
			return nil, fmt.Errorf("rag list: %w", err)
		}
	}

	// Build context string
	var parts []string
	for _, m := range memories {
		parts = append(parts, m.Content)
	}
	context := strings.Join(parts, "\n\n")
	tokens := len(context) / 4

	return &sm.RAGResponse{
		Context: context,
		Sources: memories,
		Tokens:  tokens,
	}, nil
}

// DocumentUseCase handles SM document CRUD.
type DocumentUseCase struct {
	repo port.SMDocumentRepository
}

// NewDocumentUseCase creates a DocumentUseCase.
func NewDocumentUseCase(repo port.SMDocumentRepository) *DocumentUseCase {
	return &DocumentUseCase{repo: repo}
}

// CreateDocument persists a new document.
func (uc *DocumentUseCase) CreateDocument(ctx context.Context, tenantID, title, content, docType, url string) (*sm.SMDocument, error) {
	doc := &sm.SMDocument{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Title:     title,
		Content:   content,
		Type:      docType,
		URL:       url,
		CreatedAt: time.Now(),
	}
	return doc, uc.repo.Create(ctx, doc)
}

// GetDocument retrieves a document by ID.
func (uc *DocumentUseCase) GetDocument(ctx context.Context, id string) (*sm.SMDocument, error) {
	return uc.repo.FindByID(ctx, id)
}
