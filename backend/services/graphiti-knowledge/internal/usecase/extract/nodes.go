package extract

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

// ExtractedNode — raw LLM output before resolution
type ExtractedNode struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
}

// ExtractedNodeList — LLM structured response
type ExtractedNodeList struct {
	Entities []ExtractedNode `json:"entities"`
}

// ExtractNodesRequest — input for the extract-nodes use case
type ExtractNodesRequest struct {
	Episode      graph.EpisodicNode
	PrevEpisodes []graph.EpisodicNode
	GroupID      string
	Ontology     *graph.OntologyRegistry
}

// ExtractNodesResult — resolved entities ready for persistence
type ExtractNodesResult struct {
	Entities    []graph.EntityNode
	EpisodicEdges []graph.EpisodicEdge // MENTIONS: episode → entity
	TokenUsage  llm.TokenUsage
}

// NodeExtractorUseCase — extracts and resolves entities from an episode
type NodeExtractorUseCase struct {
	llmClient    llm.LLMClient
	embedClient  embedder.EmbedderClient
	prompts      *prompt.PromptRegistry
	graphDriver  port.EntityNodePort
}

func NewNodeExtractorUseCase(
	llmClient llm.LLMClient,
	embedClient embedder.EmbedderClient,
	prompts *prompt.PromptRegistry,
	graphDriver port.EntityNodePort,
) *NodeExtractorUseCase {
	return &NodeExtractorUseCase{
		llmClient:   llmClient,
		embedClient: embedClient,
		prompts:     prompts,
		graphDriver: graphDriver,
	}
}

// Execute runs the full extract→embed→resolve pipeline for entities
func (uc *NodeExtractorUseCase) Execute(ctx context.Context, req ExtractNodesRequest) (*ExtractNodesResult, error) {
	var totalUsage llm.TokenUsage
	result := &ExtractNodesResult{}

	// 1. Build extract_nodes prompt
	extractTpl := uc.prompts.MustGet("extract_nodes")
	prevContents := make([]string, len(req.PrevEpisodes))
	for i, ep := range req.PrevEpisodes {
		prevContents[i] = ep.Content
	}

	pctx := prompt.PromptContext{
		Chunks:        []string{req.Episode.Content},
		PrevEpisodes:  prevContents,
		ReferenceTime: req.Episode.ValidAt.Format(time.RFC3339),
	}
	if req.Ontology != nil && req.Ontology.IsPrescribed() {
		pctx.EntityTypes = req.Ontology.EntityTypes
	}

	messages := []llm.Message{
		{Role: "system", Content: extractTpl.SystemPrompt},
		{Role: "user", Content: extractTpl.BuildUser(pctx)},
	}

	// 2. Call LLM for entity extraction
	resp, err := uc.llmClient.GenerateResponse(ctx, messages, llm.GenerateOpts{
		PromptName:     "extract_nodes",
		ModelSize:      llm.ModelSizeMedium,
		ResponseSchema: extractTpl.Schema,
		Temperature:    0.0,
	})
	if err != nil {
		return nil, fmt.Errorf("extract nodes LLM: %w", err)
	}
	totalUsage.Add(resp.TokenUsage)

	// 3. Parse response
	var extracted ExtractedNodeList
	if err := json.Unmarshal(resp.Content, &extracted); err != nil {
		return nil, fmt.Errorf("parse extract_nodes response: %w", err)
	}

	// 4. For each extracted entity: embed, resolve against existing, create or reuse
	for _, rawNode := range extracted.Entities {
		if strings.TrimSpace(rawNode.Name) == "" {
			continue
		}

		// 4a. Search for existing entities with similar name
		nameVec, err := uc.embedClient.Create(ctx, rawNode.Name)
		if err != nil {
			return nil, fmt.Errorf("embed entity name %q: %w", rawNode.Name, err)
		}

		candidates, err := uc.graphDriver.FindCandidates(ctx, nameVec, req.GroupID)
		if err != nil {
			return nil, fmt.Errorf("find candidates: %w", err)
		}

		// 4b. Resolve: dedupe using LLM if candidates found
		resolvedNode, resolveUsage, err := uc.resolveEntity(ctx, rawNode, req.Episode, nameVec, candidates)
		if err != nil {
			return nil, err
		}
		totalUsage.Add(resolveUsage)

		result.Entities = append(result.Entities, resolvedNode)

		// 4c. Build MENTIONS edge
		epEdge := graph.EpisodicEdge{
			UUID:       uuid.New().String(),
			SourceUUID: req.Episode.UUID,
			TargetUUID: resolvedNode.UUID,
			GroupID:    req.GroupID,
			CreatedAt:  time.Now(),
		}
		result.EpisodicEdges = append(result.EpisodicEdges, epEdge)
	}

	result.TokenUsage = totalUsage
	return result, nil
}

// resolveEntity calls the dedupe_nodes prompt to decide merge vs new
func (uc *NodeExtractorUseCase) resolveEntity(
	ctx context.Context,
	rawNode ExtractedNode,
	episode graph.EpisodicNode,
	nameVec []float32,
	candidates []port.EntityCandidate,
) (graph.EntityNode, llm.TokenUsage, error) {
	var usage llm.TokenUsage

	// Build candidate list for LLM
	candidateStrs := make([]string, len(candidates))
	for i, c := range candidates {
		candidateStrs[i] = fmt.Sprintf("%s | %s | %s", c.UUID, c.Name, c.Summary)
	}

	// No candidates → always create new
	if len(candidateStrs) == 0 {
		return buildNewEntityNode(rawNode, nameVec, episode.GroupID), usage, nil
	}

	dedupeTPL := uc.prompts.MustGet("dedupe_nodes")
	pctx := prompt.PromptContext{
		Chunks:        []string{fmt.Sprintf("%s (%s): %s", rawNode.Name, rawNode.Label, rawNode.Summary)},
		PrevEpisodes:  []string{episode.Content},
		ExistingNodes: candidateStrs,
	}
	messages := []llm.Message{
		{Role: "system", Content: dedupeTPL.SystemPrompt},
		{Role: "user", Content: dedupeTPL.BuildUser(pctx)},
	}

	resp, err := uc.llmClient.GenerateResponse(ctx, messages, llm.GenerateOpts{
		PromptName:     "dedupe_nodes",
		ModelSize:      llm.ModelSizeSmall, // cost optimization
		ResponseSchema: dedupeTPL.Schema,
		Temperature:    0.0,
	})
	if err != nil {
		// Fallback: create new node
		return buildNewEntityNode(rawNode, nameVec, episode.GroupID), usage, nil
	}
	usage.Add(resp.TokenUsage)

	var decision struct {
		Decision     string `json:"decision"`
		ExistingUUID string `json:"existing_uuid"`
	}
	if err := json.Unmarshal(resp.Content, &decision); err != nil {
		return buildNewEntityNode(rawNode, nameVec, episode.GroupID), usage, nil
	}

	if decision.Decision == "merge" && decision.ExistingUUID != "" {
		// Find the matching candidate
		for _, c := range candidates {
			if c.UUID == decision.ExistingUUID {
				// Return the existing node (updated summary may be generated separately)
				return graph.EntityNode{
					UUID:          c.UUID,
					Name:          c.Name,
					Labels:        c.Labels,
					Summary:       c.Summary,
					NameEmbedding: nameVec,
					GroupID:       episode.GroupID,
					CreatedAt:     c.CreatedAt,
					UpdatedAt:     time.Now(),
				}, usage, nil
			}
		}
	}

	return buildNewEntityNode(rawNode, nameVec, episode.GroupID), usage, nil
}

func buildNewEntityNode(raw ExtractedNode, nameVec []float32, groupID string) graph.EntityNode {
	return graph.EntityNode{
		UUID:          uuid.New().String(),
		Name:          raw.Name,
		Labels:        []string{raw.Label},
		Summary:       raw.Summary,
		NameEmbedding: nameVec,
		GroupID:       groupID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}
