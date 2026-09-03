# Solution: SOL-003 — Knowledge Processing Service (LLM + Extraction)

**CR ID:** CR-GR-003  
**Solution ID:** SOL-003  
**Priority:** Critical (Wave 1)  
**Architecture:** REBUILD `services/graphiti-knowledge/` — AI Intelligence Hub

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `graphiti-knowledge` đã trong monolith (service #6 trong Graphiti group).
- **Bifrost** — LLM gateway đã configured, multi-provider routing.
- `pkg/resilience/` (từ AgentMemory SOL-007) có `CircuitBreaker` — tái dụng.
- Redis đã có — dùng cho LLM response caching.
- NATS đã embedded — publish `graphiti.entity.resolved`, `graphiti.community.rebuilt`.

---

## 2. LLM Client — `internal/adapter/client/llm/`

### 2.1. Interface

```go
// services/graphiti-knowledge/internal/adapter/client/llm/client.go

type LLMClient interface {
    GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error)
}

type GenerateOpts struct {
    ResponseSchema interface{}  // JSON schema for structured output
    PromptName     string       // for token tracking
    ModelSize      ModelSize
    MaxTokens      int
    Temperature    float64
}

type LLMResponse struct {
    Content    json.RawMessage
    TokenUsage TokenUsage
    Cached     bool
    Provider   string
    Model      string
}

type Message struct {
    Role    string  // system | user | assistant
    Content string
}

type ModelSize int
const (
    ModelSizeMedium ModelSize = iota  // gpt-4o, claude-3-5-sonnet, gemini-2.0-flash
    ModelSizeSmall                    // gpt-4o-mini, haiku, flash-lite
)
```

### 2.2. Bifrost Adapter (Production)

```go
// services/graphiti-knowledge/internal/adapter/client/llm/bifrost.go

type BifrostLLMClient struct {
    client   *bifrost.Client
    cache    LLMCache      // Redis-backed
    tracker  *TokenTracker
    retry    RetryConfig
}

func (c *BifrostLLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
    // 1. Check cache (key = MD5(provider+model+messages_json))
    cacheKey := computeCacheKey(messages, opts)
    if cached, ok := c.cache.Get(ctx, cacheKey); ok {
        return &LLMResponse{Content: cached.Content, Cached: true, TokenUsage: TokenUsage{}}, nil
    }

    // 2. Build Bifrost request with JSON schema
    req := bifrost.ChatRequest{
        Messages:   mapMessages(messages),
        Schema:     opts.ResponseSchema,
        MaxTokens:  opts.MaxTokens,
        Temperature: opts.Temperature,
    }

    // 3. Retry with exponential backoff
    var resp *bifrost.ChatResponse
    var err error
    for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
        resp, err = c.client.Chat(ctx, req)
        if err == nil { break }
        if !isRetryable(err) { return nil, err }
        delay := min(c.retry.InitialDelay * time.Duration(attempt*attempt), c.retry.MaxDelay)
        select {
        case <-ctx.Done(): return nil, ctx.Err()
        case <-time.After(delay):
        }
    }
    if err != nil { return nil, fmt.Errorf("after %d attempts: %w", c.retry.MaxAttempts, err) }

    result := &LLMResponse{
        Content:    resp.Content,
        TokenUsage: TokenUsage{PromptTokens: resp.PromptTokens, CompletionTokens: resp.CompletionTokens},
        Provider:   "bifrost",
    }

    // 4. Track token usage
    c.tracker.Track(opts.PromptName, result.TokenUsage)

    // 5. Cache result
    c.cache.Set(ctx, cacheKey, result, c.retry.CacheTTL)

    return result, nil
}
```

### 2.3. OpenAI Adapter (Dev/Fallback)

```go
// services/graphiti-knowledge/internal/adapter/client/llm/openai.go

type OpenAILLMClient struct {
    apiKey  string
    model   string
    small   string
    cache   LLMCache
    tracker *TokenTracker
    retry   RetryConfig
}

func (c *OpenAILLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
    model := c.model
    if opts.ModelSize == ModelSizeSmall { model = c.small }

    // Similar structure to Bifrost but calls OpenAI API directly
    // Uses openai-go client with structured output (json_schema response format)
    // ...
}
```

---

## 3. Prompt Registry — `internal/adapter/prompt/`

```go
// services/graphiti-knowledge/internal/adapter/prompt/registry.go

type PromptTemplate struct {
    Name    string
    Version int
    System  string
    User    func(ctx PromptContext) string
    Schema  interface{}  // expected JSON response schema
}

type PromptContext struct {
    Chunks         []string
    PrevEpisodes   []string  // for context window
    ExistingNodes  []string  // for resolution
    EntityTypes    map[string]graph.EntityTypeSchema
    EdgeTypes      map[string]graph.EdgeTypeSchema
    ReferenceTime  time.Time
    Language       string    // multilingual support
}

var DefaultPrompts = map[string]PromptTemplate{
    "extract_nodes": {
        Name: "extract_nodes", Version: 1,
        System: `You are an expert knowledge graph builder. Extract named entities from the provided text.
Extract entities that are important and well-defined. For each entity provide:
- name: the entity's name
- label: entity type (Person, Organization, Location, Concept, etc.)
- summary: brief description from the context

Return ONLY a JSON array of entities.`,
        User: func(ctx PromptContext) string {
            var sb strings.Builder
            sb.WriteString("Text to analyze:\n")
            for _, chunk := range ctx.Chunks {
                sb.WriteString(chunk)
                sb.WriteString("\n\n")
            }
            if len(ctx.PrevEpisodes) > 0 {
                sb.WriteString("\nPrevious context:\n")
                for _, ep := range ctx.PrevEpisodes { sb.WriteString(ep + "\n") }
            }
            if len(ctx.EntityTypes) > 0 {
                sb.WriteString("\nExtract ONLY entities matching these types:\n")
                for name, schema := range ctx.EntityTypes {
                    sb.WriteString(fmt.Sprintf("- %s: %s\n", name, schema.Description))
                }
            }
            if ctx.Language != "" && ctx.Language != "en" {
                sb.WriteString(fmt.Sprintf("\nIMPORTANT: The text is in %s. Extract entities in their original language.", ctx.Language))
            }
            return sb.String()
        },
        Schema: ExtractedEntityListSchema,
    },

    "extract_edges": {
        Name: "extract_edges", Version: 1,
        System: `You are an expert knowledge graph builder. Extract relationships (facts) between entities.
For each relationship provide:
- source_entity: name of the source entity
- target_entity: name of the target entity
- relation_type: uppercase relationship type (e.g. WORKS_AT, REPORTS_TO)
- fact: natural language statement of the fact
- valid_at: when the fact became true (ISO8601, null if unknown)
- invalid_at: when the fact ceased to be true (ISO8601, null if still valid)

Return ONLY a JSON array of relationships.`,
        Schema: ExtractedEdgeListSchema,
    },

    "dedupe_nodes": {
        Name: "dedupe_nodes", Version: 1,
        System: `You are resolving whether two entity mentions refer to the same real-world entity.
Given a new entity and candidate matches from the knowledge graph, decide:
- If they ARE the same entity: return {"decision": "merge", "existing_uuid": "<uuid>"}
- If they are DIFFERENT entities: return {"decision": "new"}`,
        Schema: EntityResolutionSchema,
    },

    "dedupe_edges": {
        Name: "dedupe_edges", Version: 1,
        System: `You are resolving how a new fact relates to existing facts in the knowledge graph.
Given the new fact and similar existing facts, categorize the relationship:
- DUPLICATE: identical fact already exists
- NEW: independent fact, no conflict
- CONTRADICTION: new fact contradicts existing (provide invalid_edge_uuids)
- UPDATE: new fact updates/supersedes existing (provide invalid_edge_uuids)

Return: {"resolution": "DUPLICATE|NEW|CONTRADICTION|UPDATE", "invalidated_edge_uuids": [...]}`,
        Schema: EdgeResolutionSchema,
    },

    "summarize_nodes": {
        Name: "summarize_nodes", Version: 1,
        System: `You are summarizing entities in a knowledge graph. Given an entity's existing summary and new facts about it, provide an updated, concise summary (1-3 sentences).`,
        Schema: NodeSummarySchema,
    },

    "summarize_sagas": {
        Name: "summarize_sagas", Version: 1,
        System: `You are summarizing a sequence of related events (saga). Given the episode summaries, create a coherent narrative summary (2-5 sentences).`,
        Schema: SagaSummarySchema,
    },
}
```

---

## 4. Entity Extraction — `internal/usecase/extract_entities.go`

```go
// services/graphiti-knowledge/internal/usecase/extract_entities.go

type ExtractEntitiesUseCase struct {
    llm     LLMClient
    embedder EmbedderClient
    prompts  *PromptRegistry
    tracker  *TokenTracker
}

type ExtractedEntity struct {
    Name          string
    Label         string
    Summary       string
    NameEmbedding []float32
}

type ExtractEntitiesReq struct {
    Chunks      []string
    PrevEpisodes []string
    EntityTypes  map[string]graph.EntityTypeSchema
    Source      graph.EpisodeType
    Language    string
}

func (uc *ExtractEntitiesUseCase) Execute(ctx context.Context, req ExtractEntitiesReq) ([]ExtractedEntity, TokenUsage, error) {
    // Select prompt based on source type
    promptName := "extract_nodes"
    if req.Source == graph.EpisodeTypeMessage {
        promptName = "extract_nodes"  // same but system prompt adapts
    }

    prompt := uc.prompts.Get(promptName)
    userMsg := prompt.User(PromptContext{
        Chunks:      req.Chunks,
        PrevEpisodes: req.PrevEpisodes,
        EntityTypes: req.EntityTypes,
        Language:    req.Language,
    })

    resp, err := uc.llm.GenerateResponse(ctx, []Message{
        {Role: "system", Content: sanitize(prompt.System)},
        {Role: "user",   Content: sanitize(userMsg)},
    }, GenerateOpts{
        ResponseSchema: prompt.Schema,
        PromptName:     promptName,
        ModelSize:      ModelSizeMedium,
        Temperature:    0.0,
    })
    if err != nil { return nil, TokenUsage{}, err }

    var rawEntities []struct {
        Name    string `json:"name"`
        Label   string `json:"label"`
        Summary string `json:"summary"`
    }
    json.Unmarshal(resp.Content, &rawEntities)

    // Validate + Filter + dedupe by exact name
    seen := make(map[string]bool)
    var entities []ExtractedEntity
    for _, e := range rawEntities {
        if e.Name == "" { continue }
        key := strings.ToLower(e.Name)
        if seen[key] { continue }
        seen[key] = true

        // Validate against ontology if prescribed
        if len(req.EntityTypes) > 0 {
            if _, ok := req.EntityTypes[e.Label]; !ok {
                continue  // Filter entities not in prescribed types
            }
        }

        // Generate name embedding
        emb, _ := uc.embedder.Create(ctx, e.Name)
        entities = append(entities, ExtractedEntity{
            Name:          e.Name,
            Label:         e.Label,
            Summary:       e.Summary,
            NameEmbedding: emb,
        })
    }

    return entities, resp.TokenUsage, nil
}

// sanitize removes Unicode control chars and normalizes text
func sanitize(text string) string {
    return strings.Map(func(r rune) rune {
        if r < 32 && r != '\n' && r != '\t' { return -1 }
        return r
    }, text)
}
```

---

## 5. Entity Resolution — `internal/usecase/resolve_entities.go`

```go
// services/graphiti-knowledge/internal/usecase/resolve_entities.go

type ResolveEntityUseCase struct {
    llm     LLMClient
    prompts *PromptRegistry
    tracker *TokenTracker
}

type ResolveEntityReq struct {
    Entity     ExtractedEntity
    Candidates []*graph.EntityNode  // from store.NodeSimilaritySearch
}

type EntityResolution struct {
    ExistingUUID string  // empty = new entity
    Decision     string  // "merge" | "new"
}

func (uc *ResolveEntityUseCase) Execute(ctx context.Context, req ResolveEntityReq) (*EntityResolution, TokenUsage, error) {
    if len(req.Candidates) == 0 {
        return &EntityResolution{Decision: "new"}, TokenUsage{}, nil
    }

    prompt := uc.prompts.Get("dedupe_nodes")
    userMsg := buildDedupePrompt(req.Entity, req.Candidates)

    resp, err := uc.llm.GenerateResponse(ctx, []Message{
        {Role: "system", Content: prompt.System},
        {Role: "user",   Content: userMsg},
    }, GenerateOpts{
        ResponseSchema: prompt.Schema,
        PromptName:     "dedupe_nodes",
        ModelSize:      ModelSizeSmall,  // use cheaper model for resolution
        Temperature:    0.0,
    })
    if err != nil { return &EntityResolution{Decision: "new"}, TokenUsage{}, nil }

    var decision struct {
        Decision     string `json:"decision"`
        ExistingUUID string `json:"existing_uuid"`
    }
    json.Unmarshal(resp.Content, &decision)
    return &EntityResolution{Decision: decision.Decision, ExistingUUID: decision.ExistingUUID}, resp.TokenUsage, nil
}
```

---

## 6. Edge Resolution — `internal/usecase/resolve_edges.go`

```go
// services/graphiti-knowledge/internal/usecase/resolve_edges.go

type ResolveEdgeUseCase struct {
    llm     LLMClient
    prompts *PromptRegistry
}

type ResolveEdgeReq struct {
    NewEdge       graph.EntityEdge
    ExistingEdges []*graph.EntityEdge
    ReferenceTime time.Time
}

type EdgeResolution struct {
    Resolution           string    // DUPLICATE | NEW | CONTRADICTION | UPDATE
    InvalidatedEdgeUUIDs []string
}

func (uc *ResolveEdgeUseCase) Execute(ctx context.Context, req ResolveEdgeReq) (*EdgeResolution, TokenUsage, error) {
    if len(req.ExistingEdges) == 0 {
        return &EdgeResolution{Resolution: "NEW"}, TokenUsage{}, nil
    }

    prompt := uc.prompts.Get("dedupe_edges")
    userMsg := buildEdgeDedupePrompt(req.NewEdge, req.ExistingEdges, req.ReferenceTime)

    resp, err := uc.llm.GenerateResponse(ctx, []Message{
        {Role: "system", Content: prompt.System},
        {Role: "user",   Content: userMsg},
    }, GenerateOpts{
        ResponseSchema: prompt.Schema,
        PromptName:     "dedupe_edges",
        ModelSize:      ModelSizeSmall,
    })
    if err != nil { return &EdgeResolution{Resolution: "NEW"}, TokenUsage{}, nil }

    var res struct {
        Resolution           string   `json:"resolution"`
        InvalidatedEdgeUUIDs []string `json:"invalidated_edge_uuids"`
    }
    json.Unmarshal(resp.Content, &res)
    return &EdgeResolution{Resolution: res.Resolution, InvalidatedEdgeUUIDs: res.InvalidatedEdgeUUIDs}, resp.TokenUsage, nil
}
```

---

## 7. Community Detection — `internal/usecase/build_community.go`

```go
// services/graphiti-knowledge/internal/usecase/build_community.go

type BuildCommunityUseCase struct {
    llm       LLMClient
    embedder  EmbedderClient
    storePort port.StorePort  // to get clusters + save communities
    semaphore chan struct{}    // max 10 concurrent community summarizations
}

func (uc *BuildCommunityUseCase) Execute(ctx context.Context, groupID string) error {
    // 1. Get community clusters (adjacency lists from store)
    clusters, err := uc.storePort.GetCommunityClusters(ctx, []string{groupID})
    if err != nil { return err }

    // 2. Remove existing communities for group
    uc.storePort.RemoveCommunities(ctx, groupID)

    // 3. Label Propagation (in-memory, O(N))
    propagated := labelPropagation(clusters)

    // 4. For each community: LLM summarization with bounded concurrency
    g, gctx := errgroup.WithContext(ctx)
    for _, cluster := range propagated {
        cluster := cluster  // capture
        g.Go(func() error {
            uc.semaphore <- struct{}{}
            defer func() { <-uc.semaphore }()
            return uc.summarizeAndPersistCommunity(gctx, cluster, groupID)
        })
    }
    return g.Wait()
}

func (uc *BuildCommunityUseCase) summarizeAndPersistCommunity(ctx context.Context, nodeUUIDs []string, groupID string) error {
    // Get node summaries for LLM context
    nodes, _ := uc.storePort.GetEntityNodes(ctx, nodeUUIDs)
    summaries := extractSummaries(nodes)

    // Hierarchical LLM summarization (bottom-up if too many nodes)
    communitySummary := uc.hierarchicalSummarize(ctx, summaries)

    // Generate name embedding
    nameEmb, _ := uc.embedder.Create(ctx, communitySummary[:min(len(communitySummary), 100)])

    community := graph.CommunityNode{
        UUID:          uuid.New().String(),
        Name:          communitySummary[:min(len(communitySummary), 50)] + "...",
        Summary:       communitySummary,
        NameEmbedding: nameEmb,
        GroupID:       groupID,
        CreatedAt:     time.Now(),
    }

    // Persist CommunityNode + CommunityEdge per member
    if err := uc.storePort.SaveCommunityNode(ctx, community); err != nil { return err }
    for _, nodeUUID := range nodeUUIDs {
        uc.storePort.SaveCommunityEdge(ctx, graph.CommunityEdge{
            UUID:       uuid.New().String(),
            SourceUUID: community.UUID,
            TargetUUID: nodeUUID,
            GroupID:    groupID,
            CreatedAt:  time.Now(),
        })
    }
    return nil
}

// labelPropagation — standard algorithm, O(N) iterations until convergence
func labelPropagation(clusters [][]string) [][]string {
    labels := make(map[string]string)
    for _, cluster := range clusters {
        for _, node := range cluster { labels[node] = cluster[0] }
    }

    adj := buildAdjacency(clusters)
    changed := true
    for changed {
        changed = false
        for node := range labels {
            // Count neighbor labels
            counts := make(map[string]int)
            for _, neighbor := range adj[node] { counts[labels[neighbor]]++ }
            // Adopt plurality label
            bestLabel := labels[node]
            bestCount := 0
            for label, count := range counts {
                if count > bestCount { bestLabel = label; bestCount = count }
            }
            if bestLabel != labels[node] {
                labels[node] = bestLabel
                changed = true
            }
        }
    }

    // Group nodes by final label
    groups := make(map[string][]string)
    for node, label := range labels { groups[label] = append(groups[label], node) }
    var result [][]string
    for _, group := range groups {
        if len(group) > 1 { result = append(result, group) }
    }
    return result
}
```

---

## 8. Token Tracker — `internal/infra/telemetry/token_tracker.go`

```go
// services/graphiti-knowledge/internal/infra/telemetry/token_tracker.go

type TokenTracker struct {
    mu    sync.RWMutex
    usage map[string]*TokenUsageAgg   // key: prompt_type
}

type TokenUsageAgg struct {
    PromptTokens     int64
    CompletionTokens int64
    TotalTokens      int64
    CallCount        int64
}

func (t *TokenTracker) Track(promptName string, usage TokenUsage) {
    t.mu.Lock()
    defer t.mu.Unlock()
    agg, ok := t.usage[promptName]
    if !ok {
        agg = &TokenUsageAgg{}
        t.usage[promptName] = agg
    }
    agg.PromptTokens     += int64(usage.PromptTokens)
    agg.CompletionTokens += int64(usage.CompletionTokens)
    agg.TotalTokens      += int64(usage.TotalTokens)
    agg.CallCount++
}

func (t *TokenTracker) GetAll() map[string]TokenUsageAgg {
    t.mu.RLock()
    defer t.mu.RUnlock()
    result := make(map[string]TokenUsageAgg, len(t.usage))
    for k, v := range t.usage { result[k] = *v }
    return result
}

// Prometheus metric export
func (t *TokenTracker) RegisterMetrics(reg prometheus.Registerer) {
    // graphiti_knowledge_llm_tokens_total{prompt_type, token_type}
    // Updated on every Track() call
}
```

---

## 9. LLM Cache — Redis-backed

```go
// services/graphiti-knowledge/internal/adapter/cache/redis_llm_cache.go

type RedisLLMCache struct {
    client redis.Client
    ttl    time.Duration
}

func (c *RedisLLMCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
    val, err := c.client.Get(ctx, "llm:"+key).Bytes()
    if err != nil { return nil, false }
    var resp CachedResponse
    if err := json.Unmarshal(val, &resp); err != nil { return nil, false }
    return &resp, true
}

func (c *RedisLLMCache) Set(ctx context.Context, key string, resp *LLMResponse, ttl time.Duration) {
    data, _ := json.Marshal(resp)
    c.client.Set(ctx, "llm:"+key, data, ttl)
}

// Cache key = MD5(provider + model + sorted_messages_json + schema_name)
func computeCacheKey(messages []Message, opts GenerateOpts) string {
    h := md5.New()
    json.NewEncoder(h).Encode(messages)
    fmt.Fprintf(h, "|%s|%d", opts.PromptName, opts.ModelSize)
    return hex.EncodeToString(h.Sum(nil))
}
```

---

## 10. gRPC Service Handler

```go
// services/graphiti-knowledge/internal/adapter/grpc/handler.go

func (h *KnowledgeHandler) ExtractEntities(ctx context.Context, req *pb.ExtractEntitiesRequest) (*pb.ExtractEntitiesResponse, error) {
    entities, usage, err := h.extractEntitiesUC.Execute(ctx, usecase.ExtractEntitiesReq{
        Chunks:      req.Chunks,
        PrevEpisodes: req.PrevEpisodes,
        EntityTypes:  mapProtoEntityTypes(req.EntityTypes),
        Source:      graph.EpisodeType(req.Source),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "extract entities: %v", err) }
    return &pb.ExtractEntitiesResponse{
        Entities:   mapEntitiesToProto(entities),
        TokenUsage: domainTokenUsageToPB(usage),
    }, nil
}

func (h *KnowledgeHandler) ResolveEdge(ctx context.Context, req *pb.ResolveEdgeRequest) (*pb.ResolveEdgeResponse, error) {
    resolution, usage, err := h.resolveEdgeUC.Execute(ctx, usecase.ResolveEdgeReq{
        NewEdge:       mapProtoToEntityEdge(req.NewEdge),
        ExistingEdges: mapProtoToEntityEdges(req.ExistingEdges),
        ReferenceTime: req.ReferenceTime.AsTime(),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "resolve edge: %v", err) }
    return &pb.ResolveEdgeResponse{
        Resolution:           resolution.Resolution,
        InvalidatedEdgeUuids: resolution.InvalidatedEdgeUUIDs,
        TokenUsage:           domainTokenUsageToPB(usage),
    }, nil
}

func (h *KnowledgeHandler) GetTokenUsage(ctx context.Context, req *pb.GetTokenUsageRequest) (*pb.GetTokenUsageResponse, error) {
    all := h.tokenTracker.GetAll()
    return &pb.GetTokenUsageResponse{ByPrompt: mapTokenUsageToProto(all)}, nil
}
```

---

## 11. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/graphiti-knowledge/internal/adapter/client/llm/bifrost.go` | Bifrost LLM adapter |
| `services/graphiti-knowledge/internal/adapter/client/llm/openai.go` | OpenAI adapter |
| `services/graphiti-knowledge/internal/adapter/client/llm/anthropic.go` | Anthropic adapter |
| `services/graphiti-knowledge/internal/adapter/client/llm/generic.go` | Generic OpenAI-compat |
| `services/graphiti-knowledge/internal/adapter/client/embedder/client.go` | EmbedderClient interface |
| `services/graphiti-knowledge/internal/adapter/client/embedder/openai.go` | OpenAI embedder |
| `services/graphiti-knowledge/internal/adapter/client/reranker/client.go` | CrossEncoder interface |
| `services/graphiti-knowledge/internal/adapter/prompt/registry.go` | 6 prompt templates |
| `services/graphiti-knowledge/internal/adapter/cache/redis_llm_cache.go` | Redis LLM cache |
| `services/graphiti-knowledge/internal/usecase/extract_entities.go` | Entity extraction |
| `services/graphiti-knowledge/internal/usecase/resolve_entities.go` | Two-phase resolution |
| `services/graphiti-knowledge/internal/usecase/extract_edges.go` | Edge extraction |
| `services/graphiti-knowledge/internal/usecase/resolve_edges.go` | Temporal resolution |
| `services/graphiti-knowledge/internal/usecase/extract_attributes.go` | Attribute extraction |
| `services/graphiti-knowledge/internal/usecase/build_community.go` | Label Propagation + LLM |
| `services/graphiti-knowledge/internal/usecase/summarize_saga.go` | Incremental saga summary |
| `services/graphiti-knowledge/internal/infra/telemetry/token_tracker.go` | Per-prompt token tracking |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/graphiti-knowledge/internal/adapter/grpc/handler.go` | Implement all 14 RPCs |
| `api/proto/graphiti/knowledge/v1/knowledge.proto` | Full gRPC contract |
| `apps/memory/internal/bootstrap/graphiti.go` | Init LLM client + all knowledge usecases |

---

## 12. Acceptance Criteria Mapping

| AC từ CR-GR-003 | Covered by |
|----------------|-----------|
| ExtractEntities "Alice joined engineering" → entities | extractEntitiesUC with extract_nodes prompt |
| ResolveEntity "Alice" (exists) → existing_uuid | Phase 1 cosine fast path |
| ResolveEdge contradiction → CONTRADICTION + invalidated_uuids | resolveEdgeUC + dedupe_edges prompt |
| BuildCommunities 10 entities → CommunityNode | buildCommunityUC label propagation + LLM |
| GenerateEmbedding → []float32 dim 1536 | embedder.Create() |
| Rerank passages → scored array | rerankerClient.Rank() |
| Token usage 10 calls → GetTokenUsage total_tokens > 0 | TokenTracker + GetTokenUsage RPC |
| Cache hit: same prompt 2x → cached=true, 0 tokens | RedisLLMCache.Get() |
| Custom ontology entity_types → only extract matching | ExtractEntities ontology filter |
| SummarizeSaga 5 episodes → text summary | summarizeSagaUC |
