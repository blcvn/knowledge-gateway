# Key Data Flows

> Các luồng dữ liệu quan trọng nhất trong VNP Memory — trace từ request đến storage.

---

## Flow 1 — Memory Store (F01)

```
AI Agent                  API Gateway              Memory Engine
    │                         │                        │
    │  POST /v1/memory/store   │                        │
    │  {type: "episodic",      │                        │
    │   content: "...",        │                        │
    │   user_id: "u1"}         │                        │
    │─────────────────────────►│                        │
    │                         │                        │
    │                    1. Validate JWT/API Key        │
    │                    2. Inject TenantID             │
    │                    3. Rate limit check            │
    │                         │                        │
    │                    4. type == "auto"?             │
    │                    └── YES: LLM classify          │
    │                    └── NO: use given type         │
    │                         │                        │
    │                    5. Route by type:              │
    │                    episodic → graphiti-ingestion  │
    │                    semantic → cognee-ingestion    │
    │                    conversational → zep-memory   │
    │                    profile → memobase-ingestion   │
    │                    procedural → ov-fs            │
    │                    adaptive → sm-memory           │
    │                         │                        │
    │                         │──gRPC(bufconn)─────────►│
    │                         │  IngestRequest{         │
    │                         │    tenant_id, user_id,  │
    │                         │    content, metadata}   │
    │                         │                        │
    │                         │              6. Process + store:
    │                         │              └── PostgreSQL (entities)
    │                         │              └── Neo4j (graph edges)
    │                         │              └── pgvector (embeddings)
    │                         │              └── MinIO (files, if ov-fs)
    │                         │                        │
    │                         │                   7. Publish NATS:
    │                         │                   memory.blob.inserted
    │                         │                        │
    │◄─────────────────────────│                        │
    │  HTTP 202 Accepted       │                        │
    │  {id: "mem_xxx",         │        8. NATS triggers (async):
    │   type: "episodic"}      │        └── pipeline-service consolidation check
    │                         │        └── vnp-event UserEvent log
```

**SLA:** Response < 50ms (non-blocking), storage async via background goroutine.

---

## Flow 2 — Memory Recall / Cross-Engine Search (F01, F10)

```
AI Agent              API Gateway         vnp-search-hub         Engines
    │                     │                    │                    │
    │  POST /v1/memory/recall                  │                    │
    │  {query: "Tôi làm gì hôm qua?",         │                    │
    │   user_id: "u1",                         │                    │
    │   limit: 10}                             │                    │
    │─────────────────────►│                  │                    │
    │                      │  gRPC SearchRequest                    │
    │                      │──────────────────►│                    │
    │                      │                  │                    │
    │                      │           Fan-out (parallel, 500ms timeout):
    │                      │           ┌──────┴──────┬──────────────┤
    │                      │           │             │              │
    │                      │    cognee-search  graphiti-search   zep-search
    │                      │    BM25+vec       temporal KG       graph+session
    │                      │    [results]      [results]         [results]
    │                      │           │             │              │
    │                      │           └──────┬──────┴──────────────┘
    │                      │                  │
    │                      │           RRF Fusion:
    │                      │           1. Assign RRF score = 1/(k + rank_i)
    │                      │           2. Sum scores per doc
    │                      │           3. Sort by total RRF score
    │                      │           4. Deduplicate by content hash
    │                      │           5. Truncate to limit
    │                      │                  │
    │◄─────────────────────│◄─────────────────│
    │  {results: [         │  MergedResults    │
    │    {content, score,  │                   │
    │     type, engine},   │                   │
    │    ...10 items]}     │                   │
```

**SLA:** p95 < 500ms.

---

## Flow 3 — Agent Observe Hook (F08)

> **Note**: Session must be started first via `POST /v1/observe/sessions`

```
AI Agent (with SDK)        observe-service         Storage / NATS
    │                           │                       │
    │  1. POST /v1/observe/sessions                     │
    │  {project, cwd, model, agent_id}                  │
    │─────────────────────────────────────────────────► │
    │  ← {session_id: "s_xxx"}                          │
    │                           │                       │
    │  2. POST /v1/observe/sessions/{id}/observe         │
    │  {hook_type: "prompt_submit",                      │
    │   payload: {prompt, tokens, model}}               │
    │───────────────────────────►│                       │
    │                           │                       │
    │               13-Step Pipeline:                   │
    │               ├── 1. validate   (schema check)    │
    │               ├── 2. dedup      (SHA256, 30s TTL) │
    │               ├── 3. privacy    (PII redaction)   │
    │               ├── 4. build      (RawObservation)  │
    │               ├── 5. image      (extract images)  │
    │               ├── 6. mutex      (session ordering)│
    │               ├── 7. limit      (max obs check)   │
    │               ├── 8. agentId    (normalize)       │
    │               ├── 9. persist  → PostgreSQL        │──►│
    │               ├── 10. stream  → SSE broadcast     │──►│
    │               ├── 11. session   (update metadata) │──►│
    │               ├── 12. compress  (rule-based)      │──►│
    │               └── 13. index     (BM25 update)     │──►│
    │◄───────────────────────────│                       │
    │  HTTP 200 OK               │    NATS (async):      │
    │  {hook_id: "h_xxx"}        │  agent.hook.captured → pipeline-service
    │                            │  (on EndSession: agent.session.complete)
```

> **Auth note**: TenantID is resolved at Gateway middleware — observe-service receives TenantID in gRPC metadata, not in hook payload.

---

## Flow 4 — Memory Consolidation (F12, "Sleep" mode)

```
NATS Event Trigger                pipeline-service           Storage
    │                                  │                       │
    │  Event: agent.session.complete    │                       │
    │  {session_id: "s1",              │                       │
    │   hook_count: 145,               │                       │
    │   agent_id: "a1"}               │                       │
    │──────────────────────────────────►│                       │
    │                                  │                       │
    │                         Load raw hooks (session s1)      │
    │                                  │◄──────────────────────│
    │                                  │                       │
    │                         TIER 1 — LLM Compression:        │
    │                         Group hooks by 5min window       │
    │                         LLM: compress batch → summary    │
    │                         Save compressed_blobs → postgres  │──►│
    │                         Result: 145 hooks → 12 summaries  │
    │                                  │                       │
    │                         TIER 2 — Session Summary:        │
    │                         LLM: "What happened in session?" │
    │                         Extract: attempted/succeeded/    │
    │                         failed/decisions/entities        │
    │                         Save session_summary → postgres   │──►│
    │                                  │                       │
    │                         TIER 3 — Procedure Extraction:   │
    │                         Multi-session patterns           │
    │                         LLM: extract generic procedures  │
    │                         Save → ov-fs (OpenViking L1)     │──►│
    │                                  │                       │
    │                         TIER 4 — Cross-session Insights: │
    │                         (weekly / N sessions batch)      │
    │                         Save insights → sm-memory        │──►│
    │                                  │                       │
    │                         Publish: consolidation.done      │
    │                                  │──────────────────────►│
```

---

## Flow 5 — MCP Tool Call (F13, IDE Plugin)

```
Claude Code / IDE              MCP Server (:8082)          Internal Services
    │                               │                           │
    │  POST /mcp/message            │                           │
    │  {method: "tools/call",        │                           │
    │   params: {                    │                           │
    │     name: "memory_recall",     │                           │
    │     arguments: {               │                           │
    │       query: "auth middleware" │                           │
    │       scope: "project"}}}      │                           │
    │──────────────────────────────►│                           │
    │                               │                           │
    │                          Translate:                        │
    │                          memory_recall → POST /v1/memory/recall
    │                          scope: project → filter by       │
    │                          current project context          │
    │                               │──────────────────────────►│
    │                               │                           │
    │                               │             Cross-engine search
    │                               │             ov-search (L0/L1/L2)
    │                               │             cognee-search (semantic)
    │                               │◄──────────────────────────│
    │                               │                           │
    │  {result: {content: [         │                           │
    │    {type: "text",             │                           │
    │     text: "auth middleware... │                           │
    │     uses JWT RS256..."}]}}    │                           │
    │◄──────────────────────────────│                           │
```

---

## Flow 6 — GDPR Forget (F14, F22)

```
Enterprise Admin           API Gateway         All Engines
    │                          │                   │
    │  POST /v1/admin/forget   │                   │
    │  {user_id: "u1",         │                   │
    │   tenant_id: "t1",       │                   │
    │   reason: "gdpr_request"}│                   │
    │─────────────────────────►│                   │
    │                          │                   │
    │                     Auth: admin role only    │
    │                     Audit log: forget request│
    │                          │                   │
    │                     Fan-out (parallel):      │
    │                     ┌────┴────────────────── │
    │                     │ cognee-ingestion.Delete │
    │                     │ graphiti-store.Delete   │
    │                     │ memobase-admin.Delete   │
    │                     │ zep-admin.Delete        │
    │                     │ ov-admin.Delete         │
    │                     │ sm-engine.Delete        │
    │                     │ observe-service.Delete  │
    │                     │ vnp-event.Delete        │
    │                     └────┬─────────────────── │
    │                          │ Each engine:       │
    │                          │ DELETE WHERE       │
    │                          │ tenant_id=$1 AND   │
    │                          │ user_id=$2         │
    │                          │ (also Neo4j nodes, │
    │                          │  MinIO objects,    │
    │                          │  pgvector rows)    │
    │                          │                   │
    │                     Audit log: forget complete│
    │                     (immutable, cannot delete)│
    │◄─────────────────────────│                   │
    │  HTTP 200 OK             │                   │
    │  {deleted_from: [        │                   │
    │    "cognee", "graphiti", │                   │
    │    "memobase", "zep",    │                   │
    │    "openviking",         │                   │
    │    "supermemory",        │                   │
    │    "observe", "events"], │                   │
    │   duration_ms: 2145}     │                   │
```

**SLA:** Cascading forget hoàn tất < 3 giây, audit log immutable.

---

*[← Deployment](./deployment.md) | [→ README](./README.md)*
