package extractor

import (
\t"context"
\t"errors"

\t"vnp-memory/services/zep-graph/internal/domain/model"
)

// LLMClient represents an external language model service capable of extracting graph entities.
type LLMClient interface {
\tExtractEntities(ctx context.Context, text string) ([]*model.Node, []*model.Edge, error)
}

// ExtractorUseCase manages the extraction of knowledge graphs from unstructured text.
type ExtractorUseCase struct {
\tllm LLMClient
}

// NewExtractorUseCase returns a new instance of ExtractorUseCase.
func NewExtractorUseCase(llm LLMClient) *ExtractorUseCase {
\treturn &ExtractorUseCase{
\t\tllm: llm,
\t}
}

// ExtractKnowledge processes input text, calls the LLM, and returns the graph components.
func (uc *ExtractorUseCase) ExtractKnowledge(ctx context.Context, text string) (*model.Graph, error) {
\tif text == "" {
\t\treturn nil, errors.New("input text cannot be empty")
\t}

\tnodes, edges, err := uc.llm.ExtractEntities(ctx, text)
\tif err != nil {
\t\treturn nil, err
\t}

\tgraph := &model.Graph{
\t\tID:    "graph_ext_" + generateID(),
\t\tNodes: nodes,
\t\tEdges: edges,
\t}

\treturn graph, nil
}

// generateID is a placeholder for ID generation.
func generateID() string {
\treturn "1234"
}

// Extensions to models to compile. Usually defined in domain/model.
type modelGraphExtension struct {
\tNodes []*model.Node
\tEdges []*model.Edge
}
