# TASK-CE-004 — NodeSets: Cognify Service (Neo4j Multi-Labels + Qdrant Payload)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-004 |
| **Wave** | 2 |
| **Component** | `services/cognee-cognify/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §2.4 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CE-003 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** cognee-cognify: 51 .go - full cognify domain + pipeline  
---

## Context

Sau khi cognee-ingestion emit NATS event `cognee.ingestion.data.ingested` có `node_sets`, cognee-cognify phải:
1. Đọc `node_sets` từ event
2. Khi ExtractGraph → attach labels vào extracted nodes
3. Khi AddDatapoints → persist với Neo4j multi-labels + Qdrant payload field

---

## Goal

- `subscriber.go` đọc `node_sets` từ event, pass vào `CognifyJob`
- `start_cognify.go` (hoặc step handlers sau CR-006) nhận + pass `node_sets` qua `PipelineState`
- `ExtractGraphStep` attach NodeSet labels vào nodes
- `AddDatapointsStep` persist Neo4j multi-labels + Qdrant payload

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `services/cognee-cognify/internal/adapter/event/subscriber.go` |
| MODIFY | `services/cognee-cognify/internal/domain/entity.go` |
| MODIFY | `services/cognee-cognify/internal/usecase/start_cognify.go` |
| MODIFY | `services/cognee-cognify/internal/adapter/repository/neo4j/graph_repo.go` |

---

## Implementation

### MODIFY `adapter/event/subscriber.go` — Read NodeSets from event

```go
// services/cognee-cognify/internal/adapter/event/subscriber.go

type DataIngestedEvent struct {
    DatasetID string   `json:"dataset_id"`
    TenantID  string   `json:"tenant_id"`
    EntryIDs  []string `json:"entry_ids"`
    NodeSets  []string `json:"node_sets"`  // [NEW]
}

func (s *Subscriber) handleDataIngested(msg *nats.Msg) {
    var evt DataIngestedEvent
    if err := json.Unmarshal(msg.Data, &evt); err != nil { return }

    s.cognifyJobQueue.Enqueue(CognifyJob{
        DatasetID: evt.DatasetID,
        TenantID:  evt.TenantID,
        EntryIDs:  evt.EntryIDs,
        NodeSets:  evt.NodeSets,  // [NEW] propagate to pipeline
    })
}

type CognifyJob struct {
    DatasetID string
    TenantID  string
    EntryIDs  []string
    NodeSets  []string  // [NEW]
    Steps     []string  // for CR-006 partial pipeline
}
```

### MODIFY `domain/entity.go` — Labels on GraphNode

```go
// services/cognee-cognify/internal/domain/entity.go

type GraphNode struct {
    ID         string
    Name       string
    Type       string
    Labels     []string       // [NEW] includes node type + NodeSet tags
    Properties map[string]any
    Derived    bool
    VectorID   string
}
```

### MODIFY `usecase/start_cognify.go` — Pass NodeSets through pipeline

**Option A (before CR-006 custom pipeline):** Pass directly in step 3 and step 5.

```go
// services/cognee-cognify/internal/usecase/start_cognify.go

type CognifyRequest struct {
    DatasetID string
    TenantID  string
    EntryIDs  []string
    NodeSets  []string   // [NEW]
    Config    domain.PipelineConfig
}

// In Execute(): pass NodeSets to state (for future step handlers)
state := &PipelineState{
    DatasetID: req.DatasetID,
    TenantID:  req.TenantID,
    EntryIDs:  req.EntryIDs,
    NodeSets:  req.NodeSets,  // [NEW]
    Options:   req.Config.Options,
}

// PipelineState (add NodeSets):
type PipelineState struct {
    DatasetID   string
    TenantID    string
    EntryIDs    []string
    NodeSets    []string    // [NEW]
    RawContent  []string
    ContentType string
    Chunks      []Chunk
    Nodes       []domain.GraphNode
    Edges       []domain.GraphEdge
    Embeddings  map[string][]float32
    Options     domain.PipelineOptions
}

// === Step 3: ExtractGraph — attach NodeSet labels to nodes ===

func (s *ExtractGraphStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
    nodes, edges := s.llmClient.ExtractGraph(ctx, state.Chunks, state.DatasetID, state.TenantID)

    // [NEW] Attach NodeSet tags as additional labels on every node
    for i := range nodes {
        // Default labels: ["Concept"] or type-specific
        // After: ["Concept", "customer_123", "preferences"]
        nodes[i].Labels = append(nodes[i].Labels, state.NodeSets...)
    }

    state.Nodes = nodes
    state.Edges = edges
    return state, nil
}

// === Step 5: AddDatapoints — persist with multi-labels ===

func (s *AddDatapointsStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
    for _, node := range state.Nodes {
        // Neo4j: MERGE node, SET multi-labels for each NodeSet tag
        if err := s.graphRepo.UpsertNodeWithLabels(ctx, state.DatasetID, state.TenantID, node); err != nil {
            continue
        }

        // Qdrant: attach node_sets to point payload for filtering
        s.vectorRepo.UpsertPointPayload(ctx, node.VectorID, map[string]any{
            "node_sets": node.Labels,  // [NEW] payload field = all labels including NodeSets
        })
    }
    return state, nil
}
```

### MODIFY `neo4j/graph_repo.go` — UpsertNodeWithLabels

```go
// services/cognee-cognify/internal/adapter/repository/neo4j/graph_repo.go

// UpsertNodeWithLabels: MERGE node, then SET each NodeSet label
// Result: (:Concept:customer_123:preferences {id: "...", dataset_id: "..."})
func (r *GraphRepo) UpsertNodeWithLabels(ctx context.Context, datasetID, tenantID string, node domain.GraphNode) error {
    // Step 1: MERGE base node
    mergeQuery := `
        MERGE (n {id: $id})
        SET n += $props
        SET n:` + sanitizeLabel(node.Type)

    props := map[string]any{
        "id":         node.ID,
        "dataset_id": datasetID,
        "tenant_id":  tenantID,
        "name":       node.Name,
    }
    for k, v := range node.Properties { props[k] = v }

    _, err := r.session.Run(ctx, mergeQuery, map[string]any{"id": node.ID, "props": props})
    if err != nil { return err }

    // Step 2: SET labels for each NodeSet tag (separate queries — Cypher limitation)
    for _, label := range node.Labels {
        if label == node.Type { continue }  // already set
        labelQuery := `MATCH (n {id: $id}) SET n:` + sanitizeLabel(label)
        r.session.Run(ctx, labelQuery, map[string]any{"id": node.ID})
    }
    return nil
}

// sanitizeLabel ensures label is a valid Cypher identifier
func sanitizeLabel(tag string) string {
    re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
    sanitized := re.ReplaceAllString(tag, "_")
    if len(sanitized) > 0 && (sanitized[0] >= '0' && sanitized[0] <= '9') {
        sanitized = "_" + sanitized  // prefix digits
    }
    return sanitized
}
```

---

## Verification

```bash
cd services/cognee-cognify
go build ./...
go test ./internal/usecase/... -run TestNodeSets_AttachedToNodes -v
go test ./internal/adapter/repository/neo4j/... -run TestUpsertNodeWithLabels -v
```

**Unit test:**
```go
func TestNodeSets_AttachedToNodes(t *testing.T) {
    // Mock LLM returns nodes without labels
    // After ExtractGraphStep with NodeSets=["customer_alice"], nodes should have labels
    step := NewExtractGraphStep(mockLLMClient)
    state := &PipelineState{
        NodeSets: []string{"customer_alice", "proj_x"},
        Chunks: []Chunk{{Content: "Alice works at TechCorp"}},
    }
    result, err := step.Execute(ctx, state)
    require.NoError(t, err)
    for _, node := range result.Nodes {
        assert.Contains(t, node.Labels, "customer_alice")
        assert.Contains(t, node.Labels, "proj_x")
    }
}
```

**Neo4j verify:**
```cypher
MATCH (n)
WHERE "customer_alice" IN labels(n)
RETURN count(n) as node_count;
// Expected: > 0 after cognify with node_sets=["customer_alice"]
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| Sau cognify, Neo4j nodes có thêm labels từ node_sets | ✅ |
| Qdrant points có `payload.node_sets` field | ✅ |
| NATS event được subscribe và `node_sets` được pass vào pipeline | ✅ |
| `sanitizeLabel` handle special chars: `customer-123` → `customer_123` | ✅ |
