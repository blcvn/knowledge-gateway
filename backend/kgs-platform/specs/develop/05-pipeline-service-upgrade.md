# pipeline-service Upgrade — KGS Rule Engine + Policy

> **Strategy:** 🔄 UPGRADE existing `services/pipeline-service/`  
> **Absorbs:** rule-engine-service + policy-service  
> **Effort:** 5 ngày  
> **Priority:** P2

---

## 1. Tại Sao pipeline-service Là Đúng Chỗ

`pipeline-service` đã có:
- `PipelineUseCase` với orchestration + background processing → tương tự rule execution
- `KnowledgeUseCase` pipeline → có thể trigger từ rules
- Background goroutine pattern trong main.go → tương tự RuleRunner + EventRunner
- NATS integration → đã có subscribe/publish infrastructure

**Mapping tương đồng:**
```
PipelineUseCase.Execute()   ≈   RuleUsecase.TriggerRule()  (manual execution)
PipelineRunner goroutine    ≈   RuleRunner goroutine       (scheduled execution)
KnowledgePipeline           ≈   ON_WRITE rule              (event-triggered)
```

---

## 2. Cấu Trúc Sau Upgrade

```
services/pipeline-service/
├── cmd/server/
│   └── main.go              [MODIFY] Thêm RuleRunner + EventRunner + PolicySyncRunner goroutines
│
├── internal/
│   ├── usecase/
│   │   ├── pipeline/        [UNCHANGED] Existing pipeline orchestration
│   │   ├── knowledge/       [UNCHANGED] Knowledge pipeline
│   │   └── kgs/             [NEW PACKAGE]
│   │       ├── rules.go     ← Từ kgs-platform/internal/biz/rules.go
│   │       ├── rule_runner.go ← Từ biz/rule_runner.go (cron scheduler)
│   │       ├── event_runner.go ← Từ biz/event_runner.go (NATS subscriber)
│   │       ├── opa_client.go ← Từ biz/opa_client.go
│   │       ├── policy.go    ← Từ biz/policy.go + policy_sync.go
│   │       └── graph_client.go ← NEW: HTTP client gọi kg-service
│   │
│   ├── domain/
│   │   ├── pipeline/        [UNCHANGED]
│   │   ├── knowledge/       [UNCHANGED]
│   │   └── kgs/             [NEW]
│   │       ├── rule.go      ← Rule, RuleExecution models
│   │       └── policy.go    ← Policy model
│   │
│   ├── adapter/
│   │   └── grpc/
│   │       ├── router.go        [MODIFY] Thêm KGS routes
│   │       ├── kgs_rules.go     [NEW] /v1/rules/** handlers
│   │       └── kgs_policy.go    [NEW] /v1/policies/** handlers
│   │
│   └── infra/
│       ├── postgres/        [EXTEND]
│       │   ├── rules_pg.go      ← Từ data/rules.go
│       │   └── policy_pg.go     ← Từ data/policy.go
│       └── nats/            [EXTEND] subscribe graph events
│
└── migrations/
    └── kgs/
        ├── 001_rules.sql
        └── 002_policies.sql
```

---

## 3. Routes Mới Thêm Vào router.go

```go
// ── KGS Rule Engine (NEW) ──
router.Handle("POST",   "/v1/rules",                    h.adapt(h.KGSCreateRule))
router.Handle("GET",    "/v1/rules",                    h.adapt(h.KGSListRules))
router.Handle("GET",    "/v1/rules/{id}",               h.adapt(h.KGSGetRule))
router.Handle("PUT",    "/v1/rules/{id}",               h.adapt(h.KGSUpdateRule))
router.Handle("DELETE", "/v1/rules/{id}",               h.adapt(h.KGSDeleteRule))
router.Handle("POST",   "/v1/rules/{id}/activate",      h.adapt(h.KGSActivateRule))
router.Handle("POST",   "/v1/rules/{id}/deactivate",    h.adapt(h.KGSDeactivateRule))
router.Handle("POST",   "/v1/rules/{id}/trigger",       h.adapt(h.KGSTriggerRule))
router.Handle("GET",    "/v1/rules/{id}/executions",    h.adapt(h.KGSListExecutions))
router.Handle("GET",    "/v1/rules/executions/{exec_id}", h.adapt(h.KGSGetExecution))

// ── KGS Policy (NEW) ──
router.Handle("POST",   "/v1/policies",                 h.adapt(h.KGSCreatePolicy))
router.Handle("GET",    "/v1/policies",                 h.adapt(h.KGSListPolicies))
router.Handle("GET",    "/v1/policies/{id}",            h.adapt(h.KGSGetPolicy))
router.Handle("PUT",    "/v1/policies/{id}",            h.adapt(h.KGSUpdatePolicy))
router.Handle("DELETE", "/v1/policies/{id}",            h.adapt(h.KGSDeletePolicy))
router.Handle("POST",   "/v1/policies/{id}/activate",   h.adapt(h.KGSActivatePolicy))
router.Handle("POST",   "/v1/policies/{id}/deactivate", h.adapt(h.KGSDeactivatePolicy))
router.Handle("POST",   "/v1/policies/evaluate",        h.adapt(h.KGSEvaluatePolicy))
```

---

## 4. GraphClient — Bridge sang kg-service

Rule engine cần execute Cypher queries trên graph. Thay vì gọi Neo4j trực tiếp, gọi qua kg-service:

```go
// internal/usecase/kgs/graph_client.go

type KGSGraphClient struct {
    baseURL    string
    httpClient *http.Client
}

// ExecuteRuleQuery executes a safe read-only Cypher query
// kg-service validates and applies namespace injection
func (c *KGSGraphClient) ExecuteRuleQuery(ctx context.Context, appID, cypher string, params map[string]any) (map[string]any, error) {
    reqBody, _ := json.Marshal(map[string]any{
        "cypher": cypher,
        "params": params,
    })
    
    req, _ := http.NewRequestWithContext(ctx, "POST",
        c.baseURL+"/v1/query",
        bytes.NewReader(reqBody),
    )
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-App-ID", appID)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]any
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}
```

---

## 5. RuleRunner (Tái Sử Dụng Trực Tiếp)

```go
// internal/usecase/kgs/rule_runner.go
// Copy từ kgs-platform/internal/biz/rule_runner.go
// Chỉ thay đổi: dùng KGSGraphClient thay vì local graph repo

type RuleRunner struct {
    repo      RuleRepo
    graph     GraphExecutor   // ← interface, implement bằng KGSGraphClient
    scheduler *cron.Cron
    log       *slog.Logger
}

func (r *RuleRunner) Start(ctx context.Context) error {
    // Load tất cả SCHEDULED rules active từ postgres
    rules, _ := r.repo.ListActive(ctx, "SCHEDULED")
    
    for _, rule := range rules {
        rule := rule
        r.scheduler.AddFunc(rule.Cron, func() {
            r.executeRule(ctx, &rule, "SCHEDULED", nil)
        })
    }
    
    r.scheduler.Start()
    <-ctx.Done()
    r.scheduler.Stop()
    return nil
}
```

---

## 6. EventRunner (NATS Subscriber)

```go
// internal/usecase/kgs/event_runner.go
// Copy từ kgs-platform/internal/biz/event_runner.go

type EventRunner struct {
    repo  RuleRepo
    graph GraphExecutor
    nats  *nats.Conn
    log   *slog.Logger
}

func (r *EventRunner) Start(ctx context.Context) error {
    // Subscribe to graph events từ kg-service
    subs := []string{
        "graph.node.created",
        "graph.node.updated",
        "graph.edge.created",
    }
    
    for _, subject := range subs {
        r.nats.Subscribe(subject, func(msg *nats.Msg) {
            var event struct {
                AppID      string `json:"app_id"`
                EntityType string `json:"entity_type"`
                EntityID   string `json:"entity_id"`
                Operation  string `json:"operation"`
            }
            json.Unmarshal(msg.Data, &event)
            
            // Query matching ON_WRITE rules
            rules, _ := r.repo.FindMatchingRules(ctx, event.AppID, "ON_WRITE", event.EntityType, event.Operation)
            
            // Execute each matched rule async
            for _, rule := range rules {
                go r.executeRule(ctx, &rule, "ON_WRITE", &event)
            }
        })
    }
    
    <-ctx.Done()
    return nil
}
```

---

## 7. main.go Extension

```go
// cmd/server/main.go — EXTEND

func main() {
    // ── Existing (UNCHANGED) ──
    db := setupPostgres(conf)
    natsConn := setupNATS(conf)
    // ... existing pipeline setup

    // ── NEW: KGS Rule Engine ──
    ruleRepo := postgres.NewRuleRepo(db)
    graphClient := kgs.NewKGSGraphClient(conf.KGServiceURL)
    
    ruleUC     := kgs.NewRuleUsecase(ruleRepo)
    ruleRunner := kgs.NewRuleRunner(ruleRepo, graphClient, natsConn, logger)
    eventRunner := kgs.NewEventRunner(ruleRepo, graphClient, natsConn, logger)

    if conf.Features.RuleEngineEnabled {
        go ruleRunner.Start(ctx)
        go eventRunner.Start(ctx)
    }

    // ── NEW: KGS Policy ──
    policyRepo := postgres.NewPolicyRepo(db)
    opaClient  := kgs.NewOPAClient(conf.OPA.ServerURL, redisClient)
    policyUC   := kgs.NewPolicyUsecase(policyRepo, opaClient)
    policySyncRunner := kgs.NewPolicySyncRunner(policyRepo, opaClient, logger)

    if conf.Features.PolicyEnabled {
        go policySyncRunner.Start(ctx)
    }

    // ── Existing router setup (EXTEND) ──
    h := buildHandler(existingUsecases, ruleUC, policyUC)
    router := forward.NewRouter()
    RegisterRoutes(router, h)
    srv.ListenAndServe()
}
```

---

## 8. Database Migrations

```sql
-- migrations/kgs/001_rules.sql
CREATE TABLE IF NOT EXISTS kgs.rules (
    id               SERIAL PRIMARY KEY,
    app_id           VARCHAR(50) NOT NULL,
    name             VARCHAR(100) NOT NULL,
    description      TEXT,
    trigger_type     VARCHAR(20) NOT NULL,  -- SCHEDULED | ON_WRITE | MANUAL
    cron             VARCHAR(50),
    watch_entity_types JSONB DEFAULT '[]',
    watch_operations   JSONB DEFAULT '[]',
    cypher_query     TEXT NOT NULL,
    action           VARCHAR(50),           -- webhook | nats_event | none
    payload          JSONB,
    is_active        BOOLEAN DEFAULT TRUE,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);
CREATE INDEX idx_kgs_rules_app ON kgs.rules(app_id, trigger_type, is_active);

CREATE TABLE IF NOT EXISTS kgs.rule_executions (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    rule_id      INT NOT NULL REFERENCES kgs.rules(id),
    trigger_type VARCHAR(20) NOT NULL,
    trigger_data TEXT,
    status       VARCHAR(20) NOT NULL,   -- PENDING|RUNNING|SUCCESS|FAILED|TIMEOUT
    result_count INT,
    message      TEXT,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at     TIMESTAMPTZ,
    duration_ms  INT
);
CREATE INDEX idx_kgs_exec_rule ON kgs.rule_executions(rule_id, started_at DESC);

-- migrations/kgs/002_policies.sql
CREATE TABLE IF NOT EXISTS kgs.policies (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    rego_content TEXT NOT NULL,
    is_active    BOOLEAN DEFAULT TRUE,
    version      INT DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    UNIQUE(app_id, name)
);
```

---

## 9. Effort Breakdown

| Task | Ngày |
|------|------|
| Rules + RuleExecution postgres infra | 0.5 |
| Rule usecase (copy từ biz/rules.go) | 0.5 |
| RuleRunner goroutine (copy từ biz/rule_runner.go) | 0.5 |
| EventRunner goroutine (copy từ biz/event_runner.go) | 0.5 |
| KGSGraphClient HTTP bridge | 0.5 |
| Policy + OPA client (copy từ biz/policy*.go + opa_client.go) | 0.5 |
| HTTP handlers (kgs_rules.go, kgs_policy.go) | 1 |
| main.go goroutine integration | 0.25 |
| Migrations | 0.25 |
| Tests | 0.5 |
| **Total** | **5 ngày** |
