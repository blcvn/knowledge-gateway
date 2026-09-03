# rule-engine-service — Rule Engine Service

> **Role:** Quản lý Business Rules và thực thi chúng tự động theo schedule (cron) hoặc event-driven (on-write triggers), mà không block Graph API.

---

## 1. Trách Nhiệm (Single Responsibility)

`rule-engine-service` chịu trách nhiệm **duy nhất** cho:
- **Rule CRUD**: Tạo, đọc, cập nhật, xóa business rules per app
- **Scheduled Execution**: Chạy rules theo cron schedule
- **Event-Driven Execution**: Trigger rules khi có write event từ `graph-service`
- **Execution History**: Ghi lại lịch sử thực thi, status, errors
- **Rule Actions**: Thực hiện actions sau khi rule match (webhook, push notification, emit event)

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       rule-engine-service                                │
│                                                                         │
│  gRPC Server (port 9005)                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  RuleEngineServiceServer                          │   │
│  │                                                                  │   │
│  │  CreateRule()    GetRule()    ListRules()                        │   │
│  │  UpdateRule()    DeleteRule() ActivateRule()  DeactivateRule()   │   │
│  │  TriggerRule()                [manual trigger for testing]       │   │
│  │  ListExecutions()  GetExecution()                                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Background Workers                              │   │
│  │                                                                  │   │
│  │  ┌────────────────┐          ┌──────────────────────────────┐   │   │
│  │  │  RuleRunner    │          │      EventRunner              │   │   │
│  │  │  (Scheduler)   │          │  (NATS subscriber)            │   │   │
│  │  │                │          │                               │   │   │
│  │  │  - Load SCHEDULED rules   │  - Listen: graph.node.created │   │   │
│  │  │  - Cron-based trigger │   │  - Listen: graph.edge.created │   │   │
│  │  │  - Execute Cypher  │      │  - Match ON_WRITE rules       │   │   │
│  │  │  - Record history  │      │  - Execute matched rules      │   │   │
│  │  └────────────────┘          └──────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │              Data Layer                                           │   │
│  │  PostgreSQL: rules, rule_executions                              │   │
│  │  NATS:       subscribe (graph events), publish (rule events)     │   │
│  │  graph-service: gRPC call to execute Cypher queries/mutations    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Data Models

### 3.1 Rule

```go
type Rule struct {
    ID          uint           `gorm:"primaryKey"`
    AppID       string         `gorm:"type:varchar(50);not null;index:idx_app_rule"`
    Name        string         `gorm:"type:varchar(100);not null"`
    Description string         `gorm:"type:text"`

    TriggerType string `gorm:"type:varchar(20);not null"`
    // SCHEDULED   — chạy theo cron
    // ON_WRITE    — trigger khi có node/edge được tạo/cập nhật
    // MANUAL      — chỉ chạy khi được gọi thủ công

    // For SCHEDULED triggers:
    Cron string `gorm:"type:varchar(50)"`
    // Standard cron: "0 0 * * *" (midnight daily), "*/5 * * * *" (every 5 min)

    // For ON_WRITE triggers:
    WatchEntityTypes datatypes.JSON `gorm:"type:jsonb"`
    // ["Requirement", "UseCase"] — chỉ trigger với entity types này
    WatchOperations  datatypes.JSON `gorm:"type:jsonb"`
    // ["CREATE", "UPDATE"] — chỉ trigger với operations này

    // Rule logic — Cypher query template (namespaced automatically)
    CypherQuery string `gorm:"type:text;not null"`
    // Không cho raw Cypher từ app; phải dùng safe template syntax
    // Ví dụ: "MATCH (r:Requirement {status: 'DRAFT'}) RETURN r LIMIT 100"
    // Namespace được inject tự động khi execute

    // Action after rule matches
    Action  string         `gorm:"type:varchar(50)"` // webhook | nats_event | email | none
    Payload datatypes.JSON `gorm:"type:jsonb"`
    // For webhook: { "url": "https://...", "method": "POST", "headers": {...} }
    // For nats_event: { "subject": "custom.rule.matched", "data": {...} }

    IsActive  bool           `gorm:"default:true"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 3.2 RuleExecution

```go
type RuleExecution struct {
    ID          uint      `gorm:"primaryKey"`
    AppID       string    `gorm:"type:varchar(50);not null;index:idx_app_exec"`
    RuleID      uint      `gorm:"not null;index"`
    TriggerType string    `gorm:"type:varchar(20);not null"` // SCHEDULED | ON_WRITE | MANUAL
    TriggerData string    `gorm:"type:text"`
    // For ON_WRITE: JSON with {entity_id, operation}
    Status      string    `gorm:"type:varchar(20);not null"`
    // PENDING | RUNNING | SUCCESS | FAILED | TIMEOUT
    ResultCount int       // Number of nodes/edges matched by rule
    Message     string    `gorm:"type:text"` // Error message if FAILED
    StartedAt   time.Time `gorm:"index"`
    EndedAt     *time.Time
    DurationMs  int
}
```

---

## 4. gRPC API

```protobuf
service RuleEngineService {
  // Rule Management
  rpc CreateRule(CreateRuleRequest) returns (Rule);
  rpc GetRule(GetRuleRequest) returns (Rule);
  rpc ListRules(ListRulesRequest) returns (ListRulesResponse);
  rpc UpdateRule(UpdateRuleRequest) returns (Rule);
  rpc DeleteRule(DeleteRuleRequest) returns (google.protobuf.Empty);
  rpc ActivateRule(ActivateRuleRequest) returns (Rule);
  rpc DeactivateRule(DeactivateRuleRequest) returns (Rule);

  // Manual Execution
  rpc TriggerRule(TriggerRuleRequest) returns (RuleExecution);

  // Execution History
  rpc ListExecutions(ListExecutionsRequest) returns (ListExecutionsResponse);
  rpc GetExecution(GetExecutionRequest) returns (RuleExecution);
}

message CreateRuleRequest {
  string name = 1;
  string description = 2;
  string trigger_type = 3;     // SCHEDULED | ON_WRITE | MANUAL
  string cron = 4;             // For SCHEDULED (e.g., "0 9 * * 1")
  repeated string watch_entity_types = 5; // For ON_WRITE
  repeated string watch_operations = 6;   // For ON_WRITE
  string cypher_query = 7;
  string action = 8;           // webhook | nats_event | none
  bytes payload_json = 9;
}
```

---

## 5. Rule Execution Flows

### 5.1 Scheduled Rule Execution

```
RuleRunner (background goroutine):
  1. Start cron scheduler (robfig/cron)
  2. At startup: Load all ACTIVE SCHEDULED rules from PostgreSQL
  3. Register each rule's Cron expression in scheduler
  4. When cron fires:
     a. Create RuleExecution record (status=RUNNING)
     b. Call graph-service.ExecuteRuleQuery(app_id, cypher_query)
        → graph-service injects namespace, executes, returns results
     c. Process results → call rule.Action (webhook/event)
     d. Update RuleExecution (status=SUCCESS|FAILED, result_count, duration_ms)
  5. On rule activate/deactivate → hot-reload scheduler (no restart needed)
```

### 5.2 Event-Driven Rule Execution

```
EventRunner (NATS subscriber):
  1. Subscribe to NATS subjects:
     - graph.node.created
     - graph.node.updated
     - graph.edge.created
  2. On receive event:
     a. Extract: {app_id, entity_type, operation, entity_id}
     b. Query PostgreSQL:
        SELECT * FROM rules
        WHERE app_id = ? AND trigger_type = 'ON_WRITE'
          AND is_active = true
          AND watch_entity_types @> '["Requirement"]'
          AND watch_operations @> '["CREATE"]'
     c. For each matched rule:
        - Execute asynchronously (goroutine pool)
        - Inject trigger_data into Cypher (e.g., $triggered_entity_id)
        - Record execution history
  3. Retry on failure (max 3 retries, exponential backoff)
```

### 5.3 Ví Dụ Rule: Tự Động Detect Requirements Chưa Có TestCase

```json
// POST /v1/rules
{
  "name": "detect_untested_requirements",
  "description": "Tìm Requirements APPROVED nhưng chưa có TestCase sau 7 ngày",
  "trigger_type": "SCHEDULED",
  "cron": "0 9 * * 1",           // Thứ 2 hàng tuần lúc 9am
  "cypher_query": "MATCH (r:Requirement {status: 'APPROVED'}) WHERE NOT (r)-[:VALIDATES]-(:TestCase) AND r.created_at < datetime() - duration('P7D') RETURN r",
  "action": "webhook",
  "payload": {
    "url": "https://ba-agent.example.com/hooks/untested-requirements",
    "method": "POST"
  }
}
```

### 5.4 Ví Dụ Rule: Auto-Tag khi Requirement được APPROVE

```json
{
  "name": "auto_notify_on_requirement_approve",
  "trigger_type": "ON_WRITE",
  "watch_entity_types": ["Requirement"],
  "watch_operations": ["UPDATE"],
  "cypher_query": "MATCH (r:Requirement {id: $triggered_entity_id, status: 'APPROVED'}) RETURN r",
  "action": "nats_event",
  "payload": {
    "subject": "ba_agent.requirement.approved",
    "data": { "template": "{{ .node_id }}" }
  }
}
```

---

## 6. Rule Safety (Cypher Template Restrictions)

Để đảm bảo an toàn, `rule-engine-service` **không cho phép** raw Cypher từ app. Thay vào đó, dùng template syntax bị kiểm soát:

| Allowed | Not Allowed |
|---------|------------|
| `MATCH`, `RETURN`, `WHERE` | `CREATE`, `MERGE`, `DELETE`, `SET` (write ops) |
| Filter by property | Cross-namespace label access (kiểm tra prefix) |
| `$triggered_entity_id` variable | Raw label without `__` namespace prefix |
| `LIMIT` clause | Unbounded queries (enforced: default LIMIT 1000) |

> Rules chỉ được **đọc** data từ graph. Nếu cần write, rule action phải gọi webhook → app tự quyết định write hay không.

---

## 7. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope | Mô tả |
|--------|------|-------|-------|
| POST | `/v1/rules` | `rules:write` | Tạo rule mới |
| GET | `/v1/rules` | `rules:read` | List rules của app |
| GET | `/v1/rules/:id` | `rules:read` | Chi tiết rule |
| PUT | `/v1/rules/:id` | `rules:write` | Update rule |
| DELETE | `/v1/rules/:id` | `rules:write` | Xóa rule |
| POST | `/v1/rules/:id/activate` | `rules:write` | Activate rule |
| POST | `/v1/rules/:id/deactivate` | `rules:write` | Deactivate rule |
| POST | `/v1/rules/:id/trigger` | `rules:write` | Manual trigger (testing) |
| GET | `/v1/rules/:id/executions` | `rules:read` | Lịch sử execution |
| GET | `/v1/rules/executions/:exec_id` | `rules:read` | Chi tiết execution |

---

## 8. Configuration

```yaml
# configs/rule-engine.yaml
rule_engine_service:
  grpc_port: 9005

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_rules"

  nats:
    addr: nats:4222
    subjects:
      subscribe:
        - graph.node.created
        - graph.node.updated
        - graph.edge.created

  dependencies:
    graph_service: graph-service:9003

  scheduler:
    max_concurrent_rules: 50
    execution_timeout: 60s

  event_runner:
    worker_pool_size: 20
    max_retry: 3
    retry_backoff: 5s

  observability:
    metrics_port: 9095
```

---

## 9. Observability

| Metric | Mô tả |
|--------|-------|
| `rule_engine_executions_total{app_id, trigger_type, status}` | Số lần execute theo status |
| `rule_engine_execution_duration_seconds{app_id}` | Thời gian execute |
| `rule_engine_active_rules_total{app_id}` | Số rules đang active |
| `rule_engine_action_calls_total{action_type, result}` | Số lần gọi action (webhook, event) |
| `rule_engine_scheduler_jobs_total` | Số cron jobs đang chạy |
