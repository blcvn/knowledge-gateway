package community

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

// BuildCommunitiesRequest — input for community building use case
type BuildCommunitiesRequest struct {
	GroupID  string
	GroupIDs []string // typically [GroupID]
	LPA      LPAConfig
}

// BuildCommunitiesResult — newly built communities ready for persistence
type BuildCommunitiesResult struct {
	Communities []graph.CommunityNode
	MemberEdges []graph.CommunityEdge // HAS_MEMBER: community → entity
	TokenUsage  llm.TokenUsage
}

// CommunityBuilderUseCase — discovers communities using LPA + LLM summarization
type CommunityBuilderUseCase struct {
	llmClient   llm.LLMClient
	embedClient embedder.EmbedderClient
	prompts     *prompt.PromptRegistry
	graphDriver port.CommunityPort
}

func NewCommunityBuilderUseCase(
	llmClient llm.LLMClient,
	embedClient embedder.EmbedderClient,
	prompts *prompt.PromptRegistry,
	graphDriver port.CommunityPort,
) *CommunityBuilderUseCase {
	return &CommunityBuilderUseCase{
		llmClient:   llmClient,
		embedClient: embedClient,
		prompts:     prompts,
		graphDriver: graphDriver,
	}
}

// Execute runs the full pipeline:
// 1. GetCommunityClusters from Neo4j (connected components)
// 2. RunLPA to partition clusters into communities
// 3. For each community: load members, LLM-summarize, embed
// 4. Return community nodes + HAS_MEMBER edges
func (uc *CommunityBuilderUseCase) Execute(ctx context.Context, req BuildCommunitiesRequest) (*BuildCommunitiesResult, error) {
	var totalUsage llm.TokenUsage
	result := &BuildCommunitiesResult{}

	// 1. Get connected clusters from graph
	clusters, err := uc.graphDriver.GetClusters(ctx, req.GroupIDs)
	if err != nil {
		return nil, fmt.Errorf("get clusters: %w", err)
	}

	if len(clusters) == 0 {
		return result, nil
	}

	// 2. Run LPA to detect communities
	lpaConfig := req.LPA
	if lpaConfig.MaxIterations == 0 {
		lpaConfig = DefaultLPAConfig
	}
	communities := RunLPA(clusters, lpaConfig)

	// 3. Load entity info for summarization
	allMemberUUIDs := make([]string, 0)
	for _, members := range communities {
		allMemberUUIDs = append(allMemberUUIDs, members...)
	}
	entityMap, err := uc.graphDriver.GetEntityByUUIDs(ctx, allMemberUUIDs)
	if err != nil {
		return nil, fmt.Errorf("load community members: %w", err)
	}

	// 4. LLM summarize each community
	for _, memberUUIDs := range communities {
		// Build description of community members for LLM
		memberDescriptions := make([]string, 0, len(memberUUIDs))
		for _, uid := range memberUUIDs {
			if entity, ok := entityMap[uid]; ok {
				memberDescriptions = append(memberDescriptions,
					fmt.Sprintf("- %s (%s): %s",
						entity.Name,
						strings.Join(entity.Labels, "/"),
						entity.Summary))
			}
		}

		if len(memberDescriptions) == 0 {
			continue
		}

		// Call LLM for community summary (reuse summarize_node prompt)
		summaryTPL := uc.prompts.MustGet("summarize_node")
		pctx := prompt.PromptContext{
			Chunks: []string{"Community members:\n" + strings.Join(memberDescriptions, "\n")},
		}
		// Override system prompt for community context
		sysPrompt := `You are a knowledge graph expert. Given a list of related entities that form a community cluster, create:
- A concise summary (2-3 sentences) describing what these entities have in common and their relationships
- A short name (3-6 words) that captures the theme of this community

Return as JSON with "summary" and "name" fields.`

		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":    map[string]interface{}{"type": "string"},
				"summary": map[string]interface{}{"type": "string"},
			},
			"required": []string{"name", "summary"},
		}

		messages := []llm.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: summaryTPL.BuildUser(pctx)},
		}

		resp, err := uc.llmClient.GenerateResponse(ctx, messages, llm.GenerateOpts{
			PromptName:     "community_summary",
			ModelSize:      llm.ModelSizeSmall,
			ResponseSchema: schema,
			Temperature:    0.0,
		})
		if err != nil {
			// Skip this community on LLM failure — don't abort the whole build
			continue
		}
		totalUsage.Add(resp.TokenUsage)

		var output struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(resp.Content, &output); err != nil {
			continue
		}

		// Embed community name
		nameVec, err := uc.embedClient.Create(ctx, output.Name)
		if err != nil {
			nameVec = make([]float32, 1536)
		}

		communityNode := graph.CommunityNode{
			UUID:          uuid.New().String(),
			Name:          output.Name,
			Summary:       output.Summary,
			NameEmbedding: nameVec,
			GroupID:       req.GroupID,
			CreatedAt:     time.Now(),
		}
		result.Communities = append(result.Communities, communityNode)

		// Build HAS_MEMBER edges
		for _, memberUUID := range memberUUIDs {
			result.MemberEdges = append(result.MemberEdges, graph.CommunityEdge{
				UUID:       uuid.New().String(),
				SourceUUID: communityNode.UUID,
				TargetUUID: memberUUID,
				GroupID:    req.GroupID,
				CreatedAt:  time.Now(),
			})
		}
	}

	result.TokenUsage = totalUsage
	return result, nil
}
