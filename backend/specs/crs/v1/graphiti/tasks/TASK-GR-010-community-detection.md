# TASK-GR-010 — Community Detection (LPA + LLM Summarization)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-010 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-knowledge/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §7 |
| **Priority** | High |
| **Depends On** | TASK-GR-009 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-pipeline community detection (Leiden/Louvain)  
---

## Context

Implement `BuildCommunitiesUseCase` — chạy Label Propagation Algorithm (LPA) để phát hiện entity clusters, sau đó dùng LLM để đặt tên và tóm tắt từng community. Được trigger bởi `graphiti-admin` hoặc tự động sau batch ingestion.

---

## Goal

- `BuildCommunitiesUseCase` — orchestrate: get clusters → LLM name/summarize → save CommunityNode + CommunityEdge
- LPA implementation (in-memory) sử dụng cluster data từ `maintenanceRepo.GetCommunityClusters`
- LLM summarization cho mỗi community (sử dụng entity facts từ edges)
- Community name embedding để enable community-level search

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-knowledge/internal/usecase/build_communities.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/lpa.go` |

---

## Implementation

### File 1: `services/graphiti-knowledge/internal/usecase/lpa.go`

```go
package usecase

import (
    "math/rand"
    "sort"
)

// LPAConfig controls the LPA convergence behavior
type LPAConfig struct {
    MaxIterations int     // default: 10
    MinClusterSize int    // minimum nodes in a community, default: 3
}

var DefaultLPAConfig = LPAConfig{
    MaxIterations:  10,
    MinClusterSize: 3,
}

// LabelPropagation runs the LPA algorithm on an adjacency graph.
// Returns list of node UUID clusters (each cluster = one community).
// Input: adj[nodeUUID] = []neighborUUIDs
func LabelPropagation(adj map[string][]string, cfg LPAConfig) [][]string {
    if len(adj) == 0 { return nil }

    // Initialize: each node has its own label (its UUID)
    labels := make(map[string]string, len(adj))
    for node := range adj { labels[node] = node }

    // Build set of all nodes
    allNodes := make([]string, 0, len(adj))
    for node := range adj { allNodes = append(allNodes, node) }

    // Iterate until convergence or max iterations
    for iter := 0; iter < cfg.MaxIterations; iter++ {
        changed := false

        // Process nodes in random order (required for LPA)
        shuffled := make([]string, len(allNodes))
        copy(shuffled, allNodes)
        rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

        for _, node := range shuffled {
            neighbors := adj[node]
            if len(neighbors) == 0 { continue }

            // Count neighbor labels
            labelCount := make(map[string]int)
            for _, neighbor := range neighbors {
                if label, ok := labels[neighbor]; ok { labelCount[label]++ }
            }
            if len(labelCount) == 0 { continue }

            // Find max-frequency label (break ties randomly)
            maxCount := 0
            for _, count := range labelCount { if count > maxCount { maxCount = count } }

            var maxLabels []string
            for label, count := range labelCount {
                if count == maxCount { maxLabels = append(maxLabels, label) }
            }
            sort.Strings(maxLabels)
            newLabel := maxLabels[rand.Intn(len(maxLabels))]

            if labels[node] != newLabel {
                labels[node] = newLabel
                changed = true
            }
        }

        if !changed { break }
    }

    // Group nodes by label → communities
    clusters := make(map[string][]string)
    for node, label := range labels {
        clusters[label] = append(clusters[label], node)
    }

    // Filter out clusters smaller than MinClusterSize
    var result [][]string
    for _, cluster := range clusters {
        if len(cluster) >= cfg.MinClusterSize {
            sort.Strings(cluster)
            result = append(result, cluster)
        }
    }
    return result
}

// BuildAdjFromNodeUUIDs builds adjacency map from edge-based clusters
// (clusters come from Neo4j BFS connected components via maintenanceRepo)
func BuildAdjFromClusters(clusters [][]string) map[string][]string {
    adj := make(map[string][]string)
    for _, cluster := range clusters {
        for _, node := range cluster {
            // Connect each node to all others in the same cluster (fully connected)
            for _, other := range cluster {
                if other != node { adj[node] = append(adj[node], other) }
            }
        }
    }
    return adj
}
```

### File 2: `services/graphiti-knowledge/internal/usecase/build_communities.go`

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type BuildCommunitiesUseCase struct {
    store   port.StorePort
    llm     port.LLMPort
    prompts *prompt.PromptRegistry
    lpaConf LPAConfig
}

func NewBuildCommunitiesUseCase(store port.StorePort, llm port.LLMPort, prompts *prompt.PromptRegistry) *BuildCommunitiesUseCase {
    return &BuildCommunitiesUseCase{
        store:   store,
        llm:     llm,
        prompts: prompts,
        lpaConf: DefaultLPAConfig,
    }
}

type BuildCommunitiesReq struct {
    GroupID string
}

type BuildCommunitiesResult struct {
    CommunitiesBuilt int
    TokenUsage       llm.TokenUsage
}

// Execute runs the full community detection pipeline for a group_id.
// Steps: get clusters → LPA refinement → LLM summarize each cluster → save
func (uc *BuildCommunitiesUseCase) Execute(ctx context.Context, req BuildCommunitiesReq) (*BuildCommunitiesResult, error) {
    var totalTokens llm.TokenUsage

    // ─── Step 1: Get connected components from store ───────────────────────
    rawClusters, err := uc.store.GetCommunityClusters(ctx, req.GroupID)
    if err != nil { return nil, fmt.Errorf("get clusters: %w", err) }
    if len(rawClusters) == 0 { return &BuildCommunitiesResult{}, nil }

    // ─── Step 2: Run LPA to refine clusters ───────────────────────────────
    adj := BuildAdjFromClusters(rawClusters)
    refinedClusters := LabelPropagation(adj, uc.lpaConf)
    if len(refinedClusters) == 0 { refinedClusters = rawClusters }

    // ─── Step 3: LLM summarize each cluster ───────────────────────────────
    tmpl := uc.prompts.MustGet("summarize_nodes")  // reuse node summarization prompt

    communitySummaryPrompt := &prompt.PromptTemplate{
        Name: "summarize_community",
        SystemPrompt: `You are naming and summarizing a cluster of related entities in a knowledge graph.

Given entity names and their relationships, provide:
- summary: A 2-3 sentence description of what these entities have in common
- title: A 3-7 word name for this community (e.g. "Engineering Team", "Sales Operations", "Product Roadmap")`,
        Schema: SagaSummarySchema,
    }

    var communities []graph.CommunityNode
    var communityEdges []graph.CommunityEdge

    for _, cluster := range refinedClusters {
        if len(cluster) < uc.lpaConf.MinClusterSize { continue }

        // Get entity details
        nodes, err := uc.store.GetEntityNodes(ctx, cluster)
        if err != nil || len(nodes) == 0 { continue }

        // Build cluster description for LLM
        entityDescs := make([]string, 0, len(nodes))
        for _, n := range nodes {
            desc := n.Name
            if n.Summary != "" { desc += ": " + n.Summary }
            entityDescs = append(entityDescs, desc)
        }

        userMsg := fmt.Sprintf(
            "Entities in this cluster:\n%s",
            strings.Join(entityDescs, "\n"),
        )

        resp, err := uc.llm.Generate(ctx, "summarize_community", []llm.Message{
            {Role: "system", Content: communitySummaryPrompt.SystemPrompt},
            {Role: "user",   Content: userMsg},
        }, llm.GenerateOpts{
            PromptName:  "summarize_community",
            ModelSize:   llm.ModelSizeSmall,
            Temperature: 0.1,
            Schema:      communitySummaryPrompt.Schema,
        })
        if err != nil {
            // Fallback: use first entity name as community name
            community := graph.CommunityNode{
                UUID:      uuid.New().String(),
                Name:      "Cluster: " + nodes[0].Name,
                Summary:   "",
                GroupID:   req.GroupID,
                CreatedAt: time.Now(),
            }
            communities = append(communities, community)
            for _, node := range nodes {
                communityEdges = append(communityEdges, graph.CommunityEdge{
                    UUID:       uuid.New().String(),
                    SourceUUID: community.UUID,
                    TargetUUID: node.UUID,
                    GroupID:    req.GroupID,
                    CreatedAt:  time.Now(),
                })
            }
            continue
        }

        totalTokens.Add(resp.TokenUsage)

        var out struct {
            Summary string `json:"summary"`
            Title   string `json:"title"`
        }
        json.Unmarshal(resp.Content, &out)

        // Generate community name embedding
        communityName := out.Title
        if communityName == "" { communityName = "Cluster: " + nodes[0].Name }

        nameEmb, _ := uc.llm.Embed(ctx, communityName)

        community := graph.CommunityNode{
            UUID:          uuid.New().String(),
            Name:          communityName,
            Summary:       out.Summary,
            NameEmbedding: nameEmb,
            GroupID:       req.GroupID,
            CreatedAt:     time.Now(),
        }
        communities = append(communities, community)

        for _, node := range nodes {
            communityEdges = append(communityEdges, graph.CommunityEdge{
                UUID:       uuid.New().String(),
                SourceUUID: community.UUID,
                TargetUUID: node.UUID,
                GroupID:    req.GroupID,
                CreatedAt:  time.Now(),
            })
        }
    }

    // ─── Step 4: Save to store ─────────────────────────────────────────────
    for _, community := range communities {
        if err := uc.store.SaveCommunityNode(ctx, community); err != nil {
            return nil, fmt.Errorf("save community %s: %w", community.UUID, err)
        }
    }
    for _, edge := range communityEdges {
        if err := uc.store.SaveCommunityEdge(ctx, edge); err != nil {
            return nil, fmt.Errorf("save community edge: %w", err)
        }
    }

    return &BuildCommunitiesResult{
        CommunitiesBuilt: len(communities),
        TokenUsage:       totalTokens,
    }, nil
}

// Use SagaSummarySchema from prompt package
var SagaSummarySchema = map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "summary": map[string]interface{}{"type": "string"},
        "title":   map[string]interface{}{"type": "string"},
    },
    "required": []string{"summary"},
}
```

---

## Verification

```bash
cd services/graphiti-knowledge
go build ./internal/usecase/...
go test ./internal/usecase/... -run TestLabelPropagation -v
```

**Unit test for LPA:**
```go
func TestLabelPropagation_Basic(t *testing.T) {
    // Two disconnected clusters: {A,B,C} and {X,Y,Z}
    adj := map[string][]string{
        "A": {"B", "C"}, "B": {"A", "C"}, "C": {"A", "B"},
        "X": {"Y", "Z"}, "Y": {"X", "Z"}, "Z": {"X", "Y"},
    }
    clusters := LabelPropagation(adj, LPAConfig{MaxIterations: 10, MinClusterSize: 2})
    if len(clusters) != 2 {
        t.Errorf("expected 2 communities, got %d", len(clusters))
    }
}

func TestLabelPropagation_SingleCluster(t *testing.T) {
    adj := map[string][]string{
        "A": {"B"}, "B": {"A", "C"}, "C": {"B"},
    }
    clusters := LabelPropagation(adj, LPAConfig{MaxIterations: 10, MinClusterSize: 2})
    if len(clusters) == 0 {
        t.Error("expected at least 1 cluster")
    }
}
```

**Expected:** 2 clusters returned for 2 disconnected groups.
