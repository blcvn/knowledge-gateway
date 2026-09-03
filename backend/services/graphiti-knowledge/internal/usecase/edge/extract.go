package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-knowledge/internal/adapter/client/embedder"
	"vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
	"vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

// ExtractedEdge — raw LLM output for an edge
type ExtractedEdge struct {
	SourceEntity string  `json:"source_entity"`
	TargetEntity string  `json:"target_entity"`
	RelationType string  `json:"relation_type"`
	Fact         string  `json:"fact"`
	ValidAt      *string `json:"valid_at"`
	InvalidAt    *string `json:"invalid_at"`
}

type ExtractedEdgeList struct {
	Edges []ExtractedEdge `json:"edges"`
}

// EdgeResolutionDecision — LLM output for edge deduplication
type EdgeResolutionDecision struct {
	Resolution           string   `json:"resolution"`           // DUPLICATE | NEW | CONTRADICTION | UPDATE
	InvalidatedEdgeUUIDs []string `json:"invalidated_edge_uuids"`
	Reasoning            string   `json:"reasoning"`
}

// ExtractEdgesRequest — input for edge extraction use case
type ExtractEdgesRequest struct {
	Episode      graph.EpisodicNode
	PrevEpisodes []graph.EpisodicNode
	Entities     []graph.EntityNode  // resolved entities from TASK-GR-008
	GroupID      string
	Ontology     *graph.OntologyRegistry
}

// ExtractEdgesResult — resolved edges ready for persistence
type ExtractEdgesResult struct {
	EntityEdges        []graph.EntityEdge
	InvalidatedEdgeIDs []string
	TokenUsage         llm.TokenUsage
}

// EdgeExtractorUseCase — extracts and resolves entity relationships from an episode
type EdgeExtractorUseCase struct {
	llmClient   llm.LLMClient
	embedClient embedder.EmbedderClient
	prompts     *prompt.PromptRegistry
	graphDriver port.EntityEdgePort
}

func NewEdgeExtractorUseCase(
	llmClient llm.LLMClient,
	embedClient embedder.EmbedderClient,
	prompts *prompt.PromptRegistry,
	graphDriver port.EntityEdgePort,
) *EdgeExtractorUseCase {
	return &EdgeExtractorUseCase{
		llmClient:   llmClient,
		embedClient: embedClient,
		prompts:     prompts,
		graphDriver: graphDriver,
	}
}

// Execute runs the full extract→embed→dedupe pipeline for edges
func (uc *EdgeExtractorUseCase) Execute(ctx context.Context, req ExtractEdgesRequest) (*ExtractEdgesResult, error) {
	var totalUsage llm.TokenUsage
	result := &ExtractEdgesResult{}

	if len(req.Entities) < 2 {
		// Need at least 2 entities to form an edge
		return result, nil
	}

	// 1. Build entity name → UUID lookup
	entityByName := make(map[string]graph.EntityNode, len(req.Entities))
	entityNames := make([]string, 0, len(req.Entities))
	for _, e := range req.Entities {
		entityByName[strings.ToLower(e.Name)] = e
		entityNames = append(entityNames, e.Name)
	}

	// 2. Call LLM to extract edges
	extractTpl := uc.prompts.MustGet("extract_edges")
	prevContents := make([]string, len(req.PrevEpisodes))
	for i, ep := range req.PrevEpisodes {
		prevContents[i] = ep.Content
	}

	pctx := prompt.PromptContext{
		Chunks:        []string{req.Episode.Content},
		PrevEpisodes:  prevContents,
		ExistingNodes: entityNames,
		ReferenceTime: req.Episode.ValidAt.Format(time.RFC3339),
	}
	if req.Ontology != nil && len(req.Ontology.EdgeTypes) > 0 {
		pctx.EdgeTypes = req.Ontology.EdgeTypes
	}

	messages := []llm.Message{
		{Role: "system", Content: extractTpl.SystemPrompt},
		{Role: "user", Content: extractTpl.BuildUser(pctx)},
	}

	resp, err := uc.llmClient.GenerateResponse(ctx, messages, llm.GenerateOpts{
		PromptName:     "extract_edges",
		ModelSize:      llm.ModelSizeMedium,
		ResponseSchema: extractTpl.Schema,
		Temperature:    0.0,
	})
	if err != nil {
		return nil, fmt.Errorf("extract edges LLM: %w", err)
	}
	totalUsage.Add(resp.TokenUsage)

	var extracted ExtractedEdgeList
	if err := json.Unmarshal(resp.Content, &extracted); err != nil {
		return nil, fmt.Errorf("parse extract_edges response: %w", err)
	}

	// 3. For each extracted edge: resolve, embed, dedupe
	for _, rawEdge := range extracted.Edges {
		srcEntity, srcOK := entityByName[strings.ToLower(rawEdge.SourceEntity)]
		tgtEntity, tgtOK := entityByName[strings.ToLower(rawEdge.TargetEntity)]
		if !srcOK || !tgtOK {
			// Entity not found — skip (LLM hallucinated an entity name)
			continue
		}

		// 3a. Embed the fact
		factVec, err := uc.embedClient.Create(ctx, rawEdge.Fact)
		if err != nil {
			return nil, fmt.Errorf("embed fact: %w", err)
		}

		// 3b. Get existing edges between this pair
		existing, err := uc.graphDriver.GetBetweenNodes(ctx, srcEntity.UUID, tgtEntity.UUID)
		if err != nil {
			return nil, fmt.Errorf("get existing edges: %w", err)
		}

		// 3c. Resolve the edge
		newEdge, decision, resolveUsage, err := uc.resolveEdge(ctx, rawEdge, srcEntity, tgtEntity, factVec, existing, req.Episode)
		if err != nil {
			return nil, err
		}
		totalUsage.Add(resolveUsage)

		switch decision.Resolution {
		case "DUPLICATE":
			// Skip — already in graph
			continue
		case "CONTRADICTION", "UPDATE":
			result.InvalidatedEdgeIDs = append(result.InvalidatedEdgeIDs, decision.InvalidatedEdgeUUIDs...)
			if newEdge != nil {
				result.EntityEdges = append(result.EntityEdges, *newEdge)
			}
		case "NEW":
			if newEdge != nil {
				result.EntityEdges = append(result.EntityEdges, *newEdge)
			}
		}
	}

	result.TokenUsage = totalUsage
	return result, nil
}

func (uc *EdgeExtractorUseCase) resolveEdge(
	ctx context.Context,
	rawEdge ExtractedEdge,
	srcEntity, tgtEntity graph.EntityNode,
	factVec []float32,
	existing []*graph.EntityEdge,
	episode graph.EpisodicNode,
) (*graph.EntityEdge, EdgeResolutionDecision, llm.TokenUsage, error) {
	var usage llm.TokenUsage
	now := time.Now()

	// Build the new edge object
	newEdge := &graph.EntityEdge{
		UUID:           uuid.New().String(),
		SourceNodeUUID: srcEntity.UUID,
		TargetNodeUUID: tgtEntity.UUID,
		Name:           rawEdge.RelationType,
		Fact:           rawEdge.Fact,
		FactEmbedding:  factVec,
		Episodes:       []string{episode.UUID},
		GroupID:        episode.GroupID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if rawEdge.ValidAt != nil && *rawEdge.ValidAt != "" {
		if t, err := time.Parse(time.RFC3339, *rawEdge.ValidAt); err == nil {
			newEdge.ValidAt = &t
		}
	}
	if rawEdge.InvalidAt != nil && *rawEdge.InvalidAt != "" {
		if t, err := time.Parse(time.RFC3339, *rawEdge.InvalidAt); err == nil {
			newEdge.InvalidAt = &t
		}
	}

	// No existing edges → always NEW
	if len(existing) == 0 {
		return newEdge, EdgeResolutionDecision{Resolution: "NEW"}, usage, nil
	}

	// Call LLM for edge deduplication
	dedupeTPL := uc.prompts.MustGet("dedupe_edges")

	newFactStr := fmt.Sprintf("%s → %s [%s]: %s",
		srcEntity.Name, tgtEntity.Name, rawEdge.RelationType, rawEdge.Fact)

	existingStrs := make([]string, 0, len(existing))
	for _, e := range existing {
		validAtStr := "null"
		if e.ValidAt != nil {
			validAtStr = e.ValidAt.Format(time.RFC3339)
		}
		invalidAtStr := "null"
		if e.InvalidAt != nil {
			invalidAtStr = e.InvalidAt.Format(time.RFC3339)
		}
		existingStrs = append(existingStrs, fmt.Sprintf("%s | %s→%s [%s] | %s | valid_at=%s | invalid_at=%s",
			e.UUID, srcEntity.Name, tgtEntity.Name, e.Name, e.Fact, validAtStr, invalidAtStr))
	}

	pctx := prompt.PromptContext{
		Chunks:        []string{newFactStr},
		ExistingNodes: existingStrs,
		ReferenceTime: now.Format(time.RFC3339),
	}
	messages := []llm.Message{
		{Role: "system", Content: dedupeTPL.SystemPrompt},
		{Role: "user", Content: dedupeTPL.BuildUser(pctx)},
	}

	resp, err := uc.llmClient.GenerateResponse(ctx, messages, llm.GenerateOpts{
		PromptName:     "dedupe_edges",
		ModelSize:      llm.ModelSizeSmall,
		ResponseSchema: dedupeTPL.Schema,
		Temperature:    0.0,
	})
	if err != nil {
		// Fallback: treat as NEW
		return newEdge, EdgeResolutionDecision{Resolution: "NEW"}, usage, nil
	}
	usage.Add(resp.TokenUsage)

	var decision EdgeResolutionDecision
	if err := json.Unmarshal(resp.Content, &decision); err != nil {
		return newEdge, EdgeResolutionDecision{Resolution: "NEW"}, usage, nil
	}

	return newEdge, decision, usage, nil
}
