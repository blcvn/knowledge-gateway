package knowledge

import (
\t"context"
\t"encoding/json"
\t"errors"
\t"fmt"
\t"log"

\tpb "vnp-memory/services/graphiti-knowledge/internal/adapter/grpc/pb"
)

// LLMClient defines the interface for interacting with the foundational models for parsing JSON and evaluating text.
type LLMClient interface {
\tGenerateJSON(ctx context.Context, systemPrompt, userPrompt string) ([]byte, error)
\tEvaluateSimilarity(ctx context.Context, newEntity *pb.ExtractedEntity, candidates []*pb.ExtractedEntity) (*pb.Resolution, error)
}

// GraphRepository defines the persistence interface for knowledge graphs.
type GraphRepository interface {
\tSearchSimilarEntities(ctx context.Context, groupID string, entityName string, topK int) ([]*pb.ExtractedEntity, error)
}

// KnowledgeUsecase orchestrates the core knowledge extraction and resolution logic.
type KnowledgeUsecase struct {
\tllm  LLMClient
\trepo GraphRepository
}

// NewKnowledgeUsecase creates a new knowledge usecase.
func NewKnowledgeUsecase(llm LLMClient, repo GraphRepository) *KnowledgeUsecase {
\treturn &KnowledgeUsecase{
\t\tllm:  llm,
\t\trepo: repo,
\t}
}

// ExtractEntities uses an LLM to identify structured entities from raw text.
func (uc *KnowledgeUsecase) ExtractEntities(ctx context.Context, content string, previousEpisodes []string, entityTypes []string) ([]*pb.ExtractedEntity, error) {
\tsystemPrompt := `You are an expert NLP system. Extract entities from the provided text.
Return ONLY a JSON array of objects with 'name', 'label', and 'summary' fields.
Focus specifically on these types: ` + fmt.Sprintf("%v", entityTypes)

\tuserPrompt := fmt.Sprintf("Context (Previous Episodes): %v\n\nContent to analyze: %s", previousEpisodes, content)

\t// Call the LLM (e.g., GPT-4 or Claude 3.5 Sonnet)
\tresponseBytes, err := uc.llm.GenerateJSON(ctx, systemPrompt, userPrompt)
\tif err != nil {
\t\tlog.Printf("LLM extraction failed: %v", err)
\t\treturn nil, fmt.Errorf("llm generation error: %w", err)
\t}

\t// Parse the JSON response
\tvar extracted []struct {
\t\tName    string `json:"name"`
\t\tLabel   string `json:"label"`
\t\tSummary string `json:"summary"`
\t}
\tif err := json.Unmarshal(responseBytes, &extracted); err != nil {
\t\tlog.Printf("Failed to parse LLM JSON: %v. Raw Output: %s", err, string(responseBytes))
\t\treturn nil, fmt.Errorf("failed to parse extracted entities: %w", err)
\t}

\t// Map to Protobuf Domain Model
\tvar pbEntities []*pb.ExtractedEntity
\tfor _, e := range extracted {
\t\tpbEntities = append(pbEntities, &pb.ExtractedEntity{
\t\t\tName:    e.Name,
\t\t\tLabel:   e.Label,
\t\t\tSummary: e.Summary,
\t\t})
\t}

\treturn pbEntities, nil
}

// ResolveEntities processes a list of newly extracted entities and determines whether they are new
// or should be merged with existing entities in the graph (Deduplication).
func (uc *KnowledgeUsecase) ResolveEntities(ctx context.Context, extractedEntities []*pb.ExtractedEntity, groupID string) ([]*pb.Resolution, error) {
\tif groupID == "" {
\t\treturn nil, errors.New("groupID is required for resolution context")
\t}

\tvar finalResolutions []*pb.Resolution

\t// 1. Loop through each newly extracted entity
\tfor _, newEntity := range extractedEntities {
\t\t
\t\t// 2. Query the GraphRepository (Vector Search) for semantic candidates
\t\tcandidates, err := uc.repo.SearchSimilarEntities(ctx, groupID, newEntity.Name, 5)
\t\tif err != nil {
\t\t\t// Log error but continue with other entities
\t\t\tlog.Printf("Error searching candidates for %s: %v\n", newEntity.Name, err)
\t\t\tcontinue
\t\t}

\t\t// 3. If no candidates found, it's definitely a new entity
\t\tif len(candidates) == 0 {
\t\t\tfinalResolutions = append(finalResolutions, &pb.Resolution{
\t\t\t\tExtractedEntity: newEntity,
\t\t\t\tDecision:        "CREATE_NEW",
\t\t\t\tConfidence:      1.0,
\t\t\t})
\t\t\tcontinue
\t\t}

\t\t// 4. If candidates exist, ask the LLMResolver to evaluate and decide
\t\t// The LLM will check descriptions, labels, etc. to decide between CREATE_NEW or MERGE
\t\tresolution, err := uc.llm.EvaluateSimilarity(ctx, newEntity, candidates)
\t\tif err != nil {
\t\t\tlog.Printf("LLM failed to resolve %s: %v\n", newEntity.Name, err)
\t\t\t// Fallback to CREATE_NEW if LLM fails
\t\t\tfinalResolutions = append(finalResolutions, &pb.Resolution{
\t\t\t\tExtractedEntity: newEntity,
\t\t\t\tDecision:        "CREATE_NEW",
\t\t\t\tConfidence:      0.5, // Low confidence due to fallback
\t\t\t})
\t\t\tcontinue
\t\t}

\t\tfinalResolutions = append(finalResolutions, resolution)
\t}

\treturn finalResolutions, nil
}
