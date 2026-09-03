# Change Request: CR-ZEP-003 — Temporal Knowledge Graph (Graph Service)

**CR ID:** CR-ZEP-003  
**Component:** `services/graph-service` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Zep PRD §6.1 F3, SRS §3.2, specs/services/05-graph-service.md  
**Benchmark:** #1 LoCoMo, LongMemEval (temporal reasoning capability)

---

## 1. Mô tả

Xây dựng **Graph Service (zep-graph)** — lõi Temporal Knowledge Graph của VNP Memory, tích hợp với Graphiti engine:

1. **Async Entity Extraction**: Consume NATS messages → LLM extraction via Graphiti → Neo4j upsert (10-20s).
2. **Temporal Fact Management**: Facts với `valid_at`/`invalid_at`/`expired_at` — biết khi nào thông tin đúng.
3. **9-Node Ontology**: User, Assistant, Preference, Organization, Event, Location, Document, Topic, Object.
4. **Custom Ontology**: Developer tự định nghĩa entity/edge types domain-specific.
5. **Graph Data Ingestion**: Thêm text/JSON trực tiếp vào user's knowledge graph (ngoài messages).
6. **Fact CRUD**: Get, delete (invalidate) facts.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có **Temporal Knowledge Graph** — không track khi nào thông tin đúng/sai.
- Không có **entity extraction** tự động từ messages.
- Thiếu **Neo4j integration** cho graph storage.
- Chưa hỗ trợ **custom ontology** cho domain-specific entity types.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/graph-service/` (Port gRPC: 9044)

Cần infrastructure mới: **Neo4j 5.22+** và **Graphiti service**.

### 3.2. Node Ontology (Priority Hierarchy)

```go
const (
    NodeTypeUser         NodeType = "User"         // Priority 1 — singleton per conversation
    NodeTypeAssistant    NodeType = "Assistant"     // Priority 1 — singleton
    NodeTypePreference   NodeType = "Preference"    // Priority 2 — low extraction threshold
    NodeTypeOrganization NodeType = "Organization"  // Priority 3
    NodeTypeEvent        NodeType = "Event"         // Priority 3
    NodeTypeLocation     NodeType = "Location"      // Priority 4
    NodeTypeDocument     NodeType = "Document"      // Priority 4
    NodeTypeTopic        NodeType = "Topic"         // Priority 5
    NodeTypeObject       NodeType = "Object"        // Priority 6 (last resort)
)
```

### 3.3. Temporal Annotations

```go
type TemporalAnnotation struct {
    ValidAt   *time.Time  // khi fact trở nên đúng
    InvalidAt *time.Time  // khi fact không còn đúng nữa
    ExpiredAt *time.Time  // khi fact bị superseded bởi fact mới hơn
}

// Ví dụ về temporal reasoning:
// Fact 1: "Alice worked at Acme" → ValidAt: 2020-01-01, InvalidAt: 2023-06-30
// Fact 2: "Alice works at Beta"  → ValidAt: 2023-07-01, InvalidAt: nil (hiện tại)
// Hệ thống tự biết tháng 2022 thì Alice ở Acme, tháng 2024 thì ở Beta
```

### 3.4. Async Extraction Pipeline (NATS Consumer)

```
NATS: "memory.messages.ingested" event
  │
  └── ExtractEntitiesUseCase (10-20s)
      ├── 1. Graphiti.PutMemory(sessionID, messages) — extract vào session graph
      ├── 2. Graphiti.PutMemory(userID, messages)    — extract vào user graph (nếu có user)
      ├── 3. LLM classify entities theo NodeOntology
      ├── 4. Generate TemporalAnnotations
      ├── 5. Upsert nodes/edges vào Neo4j
      └── 6. Publish "graph.extraction.completed" → Search Service invalidate cache
```

### 3.5. Graph Data Ingestion (Beyond Messages)

Developer có thể thêm data phi hội thoại vào graph:
```go
// POST /api/v2/graph/data
type AddGraphDataRequest struct {
    UserID  string  // target user's graph
    GraphID string  // alternative: specific graph scope
    Data    string  // raw text hoặc JSON string
    Type    string  // "text" | "json"
}
// Ví dụ: thêm CRM data, product catalog, user telemetry events
```

### 3.6. Custom Ontology API

```go
// POST /api/v2/graph/ontology
type SetOntologyRequest struct {
    GraphID  string
    Entities map[string]EntityDefinition  // custom node types
    Edges    map[string]EdgeDefinition    // custom edge types
}

// Ví dụ:
// "Product": {description: "Product being discussed", fields: ["name", "category"]}
// "PURCHASED": {description: "Customer purchased product"}
```

### 3.7. Fact API

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v2/facts/:uuid` | Lấy chi tiết một fact (temporal edge) |
| `DELETE` | `/api/v2/facts/:uuid` | Xóa (invalidate) fact |

### 3.8. Infrastructure mới cần thêm

- **Neo4j 5.22+**: Graph database cho temporal knowledge graph
- **Graphiti service**: Python service xử lý LLM extraction (external service, read-only integration)
- **NATS JetStream**: Async messaging cho graph extraction pipeline

### 3.9. GroupID Strategy

```go
// GroupID namespace = session_id hoặc user_id
// Episode prefix: "{groupID}-{messageUUID}" để avoid UUID collision giữa groups
func (g GroupID) WithPrefix(messageUUID string) string {
    return string(g) + "-" + messageUUID
}
```

---

## 4. Acceptance Criteria

- [ ] Gửi "Alice worked at Acme until last June, now at Beta" → graph có 2 facts với temporal annotations chính xác.
- [ ] Query graph cho Alice tháng 2022 → trả về "worked at Acme"; tháng 2024 → trả về "works at Beta".
- [ ] `POST /graph/data` với JSON product catalog → data xuất hiện trong graph với đúng node types.
- [ ] Custom ontology: định nghĩa "Product" entity → sau khi set ontology, "MacBook Pro" được extract đúng thành `NodeTypeProduct`.
- [ ] `DELETE /facts/:uuid` → fact bị invalidate (invalid_at = now), không còn xuất hiện trong search.
- [ ] Extraction latency 10-20 giây là chấp nhận được (async, documented).
