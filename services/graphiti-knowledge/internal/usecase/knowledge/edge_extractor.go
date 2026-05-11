package knowledge

import (
\t"context"
\t"encoding/json"
\t"fmt"
\t"log"

\tpb "vnp-memory/services/graphiti-knowledge/internal/adapter/grpc/pb"
)

// ExtractEdges evaluates the extracted entities against the original content to find relationships (edges) between them.
func (uc *KnowledgeUsecase) ExtractEdges(ctx context.Context, content string, resolvedEntities []*pb.ExtractedEntity, previousEpisodes []string) ([]*pb.ExtractedEdge, error) {
\tif len(resolvedEntities) < 2 {
\t\t// Cannot form a relationship without at least two entities
\t\treturn []*pb.ExtractedEdge{}, nil
\t}

\t// 1. Prepare the entity context to guide the LLM
\tvar entityList string
\tfor _, e := range resolvedEntities {
\t\tentityList += fmt.Sprintf("- [%s] %s\n", e.Label, e.Name)
\t}

\t// 2. Build the System Prompt
\tsystemPrompt := `You are an expert Knowledge Graph Relationship Extractor. 
Analyze the provided text and identify relationships (edges) between the listed entities.
Return ONLY a JSON array of objects with the following fields:
- "source": the EXACT name of the source entity
- "target": the EXACT name of the target entity
- "relation": a concise verb or phrase describing the relationship (e.g., "FOUNDED", "WORKS_FOR", "LOCATED_IN")
- "fact": the specific sentence or clause from the text proving this relationship
- "temporal": an array of time/date strings (if any) relevant to this relationship.

Strictly adhere to JSON format.`

\t// 3. Build the User Prompt
\tuserPrompt := fmt.Sprintf(
\t\t"Entities available for relation extraction:\n%s\nContext (Previous Episodes):\n%v\n\nContent to analyze:\n%s",
\t\tentityList,
\t\tpreviousEpisodes,
\t\tcontent,
\t)

\t// 4. Call LLM
\tresponseBytes, err := uc.llm.GenerateJSON(ctx, systemPrompt, userPrompt)
\tif err != nil {
\t\tlog.Printf("LLM edge extraction failed: %v", err)
\t\treturn nil, fmt.Errorf("llm generation error for edges: %w", err)
\t}

\t// 5. Parse the Response
\tvar extracted []struct {
\t\tSource   string   `json:"source"`
\t\tTarget   string   `json:"target"`
\t\tRelation string   `json:"relation"`
\t\tFact     string   `json:"fact"`
\t\tTemporal []string `json:"temporal"`
\t}
\t
\tif err := json.Unmarshal(responseBytes, &extracted); err != nil {
\t\tlog.Printf("Failed to parse LLM JSON for edges: %v. Raw Output: %s", err, string(responseBytes))
\t\treturn nil, fmt.Errorf("failed to parse extracted edges: %w", err)
\t}

\t// 6. Map to Protobuf and Validate
\tvar pbEdges []*pb.ExtractedEdge
\t
\t// Create a lookup map to validate that LLM didn't hallucinate entity names
\tvalidEntities := make(map[string]bool)
\tfor _, e := range resolvedEntities {
\t\tvalidEntities[e.Name] = true
\t}

\tfor _, edge := range extracted {
\t\t// Validate source and target exist in our known entities
\t\tif !validEntities[edge.Source] || !validEntities[edge.Target] {
\t\t\tlog.Printf("Warning: LLM hallucinated entity relation: %s -> %s", edge.Source, edge.Target)
\t\t\tcontinue
\t\t}

\t\tpbEdges = append(pbEdges, &pb.ExtractedEdge{
\t\t\tSource:   edge.Source,
\t\t\tTarget:   edge.Target,
\t\t\tRelation: edge.Relation,
\t\t\tFact:     edge.Fact,
\t\t\tTemporal: edge.Temporal,
\t\t})
\t}

\treturn pbEdges, nil
}
