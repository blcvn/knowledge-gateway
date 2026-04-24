# Huong Dan Tich Hop Service Ben Ngoai Voi ai-kg-service (kgs-platform)

## 1. Muc tieu tai lieu

Tai lieu nay mo ta cach tich hop mot service ben ngoai (backend khac, workflow engine, worker, hoac API gateway noi bo) voi `ai-kg-service/kgs-platform` theo luong thuc te tu code:

- Entry point va wiring: `cmd/server/*`
- API + middleware + xu ly request: `internal/server/*`, `internal/service/*`
- Business va persistence: `internal/biz/*`, `internal/data/*`

Tai lieu tap trung vao:

1. Onboarding app + API key.
2. Header/auth/tenant/namespace dung cach.
3. Luong ghi/doc/query/search graph.
4. Overlay/version/projection cho session write nang cao.
5. Checklist van hanh + troubleshooting.

## 2. Ban do code can nam

| Khu vuc | File/chuc nang chinh | Y nghia khi tich hop |
| --- | --- | --- |
| Khoi dong service | `cmd/server/main.go` | Load config, khoi tao Kratos app HTTP + gRPC + worker |
| Dependency wiring | `cmd/server/wire_gen.go` | Cho thay full dependency graph (Postgres, Neo4j, Redis, Qdrant, NATS, OPA, projection, overlay...) |
| HTTP/gRPC server | `internal/server/http.go`, `internal/server/grpc.go` | Dang ky cac service API, middleware chain |
| Middleware | `internal/server/middleware/auth.go`, `namespace.go`, `ratelimit.go` | Auth API key, tenant/org context, namespace check, quota/rate limit |
| Graph API | `api/graph/v1/graph.proto`, `internal/service/graph.go` | CRUD graph, traversal, search, overlay, version, analytics, projection views |
| Registry API | `api/registry/v1/registry.proto`, `internal/service/registry.go` | Tao app, issue/revoke API key, lay quota |
| Ontology API | `api/ontology/v1/ontology.proto`, `internal/service/ontology.go` | Dinh nghia entity/relation type |
| Batch ingest | `internal/batch/*` | Batch upsert, dedup, vector index |
| Overlay/version | `internal/overlay/*`, `internal/version/*` | Session-scoped write, commit/discard, diff/rollback |
| Search | `internal/search/*` | Hybrid search (semantic + text + centrality) |
| Health/metrics | `internal/service/health.go`, `internal/observability/*` | `/healthz`, `/readyz`, `/metrics` |

## 3. Kien truc tich hop tong quan

Luong request tu service ben ngoai:

1. Goi HTTP (gRPC-gateway) hoac gRPC vao KGS.
2. Middleware xu ly theo thu tu:
   - tracing
   - metrics
   - recovery
   - auth (API key -> app context)
   - namespace validation (neu co `X-KG-Namespace`)
   - rate limit theo `app_id`
3. Service handler (`internal/service/*`) lay `AppContext` tu middleware.
4. Biz layer (`internal/biz/*`) enforce validation/guardrail/policy.
5. Data layer (`internal/data/*`) doc/ghi Postgres + Neo4j (+ Redis/Qdrant/NATS).

## 4. Chuan bi truoc khi tich hop

## 4.1. Ha tang phu thuoc

Theo wiring va `configs/config.yaml`, can chuan bi:

- Postgres
- Neo4j
- Redis
- OPA (neu ban co bat policy check cho write flow)
- Qdrant (neu dung semantic dedup/search)
- NATS (neu dung event-driven overlay/session)

## 4.2. Cau hinh

File mau: `configs/config.yaml`.

Diem can luu y:

- HTTP default: `0.0.0.0:8000`
- gRPC default: `0.0.0.0:9000`
- Ontology validation co the bat/tat o `data.ontology.*`
- Embedding provider: `deterministic | openai | ai-proxy`

Luu y quan trong ve OPA:

- Connectivity check luc startup dung `data.opa.url` trong config.
- Graph write policy client doc bien moi truong `OPA_URL`.
- Nen set `OPA_URL` dong bo voi `data.opa.url` de tranh mismatch.

## 4.3. Chay service

```bash
cd services/ai-kg-service/kgs-platform
go run ./cmd/server -conf ./configs
```

Smoke check:

```bash
curl -s http://localhost:8000/healthz
curl -s http://localhost:8000/readyz
curl -s http://localhost:8000/metrics | head
```

## 5. Hop dong auth, tenant, namespace

## 5.1. Header auth

Chap nhan mot trong hai:

- `Authorization: Bearer <api_key>`
- `X-API-Key: <api_key>`

## 5.2. Cac endpoint khong can auth

Theo middleware hien tai, cac API duoc skip auth:

- `POST /v1/apps` (CreateApp)
- `GET /v1/apps` (ListApps)
- `GET /v1/apps/{app_id}` (GetApp)
- `POST /v1/apps/{app_id}/keys` (IssueApiKey)
- `/healthz`, `/readyz`, `/metrics`

Luu y: `DELETE /v1/keys/{key_hash}` (RevokeApiKey) khong nam trong skip list, nen can auth.

## 5.3. Tenant va org

- `X-Tenant-ID` la cach on dinh nhat de dat tenant.
- Neu khong co `X-Tenant-ID`, middleware thu parse `tenant_id` tu JWT-like token.
- Neu khong parse duoc thi fallback `default`.
- `X-Org-ID` la optional.

## 5.4. Namespace

Cong thuc namespace:

- Khong org: `graph/{appId}/{tenantId}`
- Co org: `graph/{orgId}/{appId}/{tenantId}`

`X-KG-Namespace`:

- Neu khong gui, middleware bo qua check namespace.
- Neu gui, middleware bat buoc namespace phai khop voi app context tu API key + tenant/org headers.

Khuyen nghi: luon gui `X-KG-Namespace` de tranh ghi/doi context sai.

## 5.5. Rate limit

- Sliding window theo phut, key theo `app_id`.
- Default fallback: `1000 request/phut` neu khong co quota trong DB.
- Vuot han muc tra ve `429` va header `Retry-After: 60`.

## 6. Quy trinh tich hop de xuat (E2E)

## 6.1. Buoc 1 - Tao app

```bash
curl -X POST http://localhost:8000/v1/apps \
  -H "Content-Type: application/json" \
  -d '{
    "app_name":"external-integration-demo",
    "description":"Service X integration",
    "owner":"team-x"
  }'
```

Lay `app_id` tu response.

## 6.2. Buoc 2 - Issue API key

```bash
curl -X POST "http://localhost:8000/v1/apps/${APP_ID}/keys" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"service-x-prod",
    "scopes":"all",
    "ttl_seconds":2592000
  }'
```

Luu `api_key` (chi tra ve 1 lan).

## 6.3. Buoc 3 - Tao ontology (nen lam truoc khi ingest)

Tao entity type:

```bash
curl -X POST http://localhost:8000/v1/ontology/entities \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Requirement",
    "description":"Business requirement node",
    "schema":"{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"string\"},\"name\":{\"type\":\"string\"}}}"
  }'
```

Tao relation type:

```bash
curl -X POST http://localhost:8000/v1/ontology/relations \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"DEPENDS_ON",
    "description":"Requirement dependency",
    "properties_schema":"{\"type\":\"object\"}",
    "source_types":["Requirement"],
    "target_types":["Requirement"]
  }'
```

## 6.4. Buoc 4 - Ghi du lieu graph

### Cach A: Create node/edge don le

Create node:

```bash
curl -X POST http://localhost:8000/v1/graph/nodes \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "label":"Requirement",
    "properties_json":"{\"id\":\"REQ-001\",\"name\":\"Login must support SSO\",\"priority\":\"P1\"}"
  }'
```

Create edge:

```bash
curl -X POST http://localhost:8000/v1/graph/edges \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "source_node_id":"REQ-001",
    "target_node_id":"REQ-002",
    "relation_type":"DEPENDS_ON",
    "properties_json":"{\"strength\":0.9}"
  }'
```

### Cach B: Batch upsert (khuyen nghi cho ingest lon)

```bash
curl -X POST http://localhost:8000/v1/graph/entities/batch \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "entities":[
      {
        "label":"Requirement",
        "properties_json":"{\"id\":\"REQ-100\",\"name\":\"Support MFA\",\"domain\":\"auth\"}"
      },
      {
        "label":"Requirement",
        "properties_json":"{\"id\":\"REQ-101\",\"name\":\"Password reset\",\"domain\":\"auth\"}"
      }
    ]
  }'
```

## 6.5. Buoc 5 - Query/traversal/search

Lay context:

```bash
curl -X GET "http://localhost:8000/v1/graph/nodes/REQ-001/context?depth=2&page_size=100" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default"
```

Hybrid search:

```bash
curl -X POST http://localhost:8000/v1/graph/search/hybrid \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "query":"mfa login requirement",
    "top_k":20,
    "alpha":0.7,
    "beta":0.2,
    "entity_types":["Requirement"]
  }'
```

Lay full graph (co check scope mismatch):

```bash
curl -X POST http://localhost:8000/v1/graph/full \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d "{
    \"app_id\":\"${APP_ID}\",
    \"tenant_id\":\"default\",
    \"node_limit\":500,
    \"node_offset\":0
  }"
```

## 6.6. Buoc 6 - Overlay + Versioning (tuỳ chon)

Tao overlay:

```bash
curl -X POST http://localhost:8000/v1/graph/overlays \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"sess-001","base_version":"current"}'
```

Khi ghi node/edge vao overlay, truyen `overlay_id` trong `properties_json`:

```json
{
  "label": "Requirement",
  "properties_json": "{\"overlay_id\":\"<overlay-id>\",\"id\":\"REQ-999\",\"name\":\"Draft change\"}"
}
```

Commit overlay:

```bash
curl -X POST http://localhost:8000/v1/graph/overlays/${OVERLAY_ID}/commit \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{"conflict_policy":"KEEP_OVERLAY"}'
```

Conflict policy hop le:

- `KEEP_OVERLAY`
- `KEEP_BASE`
- `MERGE`
- `REQUIRE_MANUAL`

Version APIs:

- `GET /v1/graph/versions`
- `GET /v1/graph/versions/diff?from_version_id=...&to_version_id=...`
- `POST /v1/graph/versions/{version_id}/rollback`

## 6.7. Buoc 7 - Role-based projection (tuỳ chon)

Tao view definition:

```bash
curl -X POST http://localhost:8000/v1/graph/views \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "X-Tenant-ID: default" \
  -H "X-KG-Namespace: graph/${APP_ID}/default" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name":"reader",
    "allowed_entity_types":["Requirement"],
    "allowed_fields":["id","name","priority"],
    "pii_mask_fields":["owner_email"]
  }'
```

Khi query, gui them `X-KG-Role: reader` de nhan du lieu da duoc filter/mask.

## 7. Guardrails va gioi han can biet

- Do sau traversal toi da: `10` (`ERR_DEPTH_EXCEEDED`)
- So node cho subgraph/fullgraph toi da: `10000` (`ERR_NODES_EXCEEDED`)
- Batch upsert max entities: `1000`
- Label va relation type bat buoc regex Cypher identifier: `^[A-Za-z_][A-Za-z0-9_]*$`
- `properties_json` phai la JSON object hop le
- Pagination cho `GetContext/GetImpact/GetCoverage`:
  - request: `page_size`, `page_token`
  - response header: `X-Next-Page-Token`

## 8. Tich hop event-driven voi NATS

Neu bat NATS, `overlay.SessionCloseListener` se:

- Listen `session.close.*`
  - neu overlay co delta -> auto commit
  - neu khong co delta -> auto discard
- Listen `budget.stop.*`
  - auto `CommitPartial`

Topics phat sinh boi overlay manager:

- `overlay.committed.<tenant>`
- `overlay.discarded.<tenant>`

Payload chua metadata nhu `overlay_id`, `namespace`, `session_id`, `new_version_id`.

## 9. Tich hop gRPC

- Proto contracts nam trong `api/*/v1/*.proto`.
- HTTP va gRPC su dung cung service definitions.
- Ban co the generate client tu proto va goi truc tiep qua port gRPC (mac dinh `9000`) neu can throughput cao hon.

## 10. Observability va van hanh

## 10.1. Health endpoints

- `GET /healthz`: liveness
- `GET /readyz`: readiness check Postgres/Redis/Neo4j/Qdrant/NATS

## 10.2. Metrics endpoint

- `GET /metrics` (Prometheus)
- Metric tieu bieu:
  - `kg_request_total`
  - `kg_request_duration_ms`
  - `kg_entity_write_total`
  - `kg_search_duration_ms`
  - `kg_overlay_count_active`
  - `kg_lock_acquire_duration_ms`

## 10.3. Tool smoke test co san

```bash
cd services/ai-kg-service/kgs-platform
go run ./cmd/api-tester --base-url http://localhost:8000 --verbose
```

Tool nay chay full flow registry -> ontology -> graph -> overlay -> version.

## 11. Cac han che hien tai can luu y khi tich hop

1. `RulesService` va `PolicyService` dang hardcode `appID = "demo-app"` trong service layer.
2. `RuleRunner`/`PolicySyncRunner` cung dang van hanh theo `demo-app`.
3. Namespace org-level (`X-Org-ID`) co check o middleware, nhung phan lon graph internals dang tinh namespace theo `app_id + tenant_id`.
4. OPA config startup (`data.opa.url`) va OPA runtime client (`OPA_URL`) can dong bo bang tay.

Khuyen nghi: neu tich hop production, uu tien su dung `Registry + Ontology + Graph` va coi `Rules/Policy` la khu vuc can hardening them.

## 12. Checklist production cho service ben ngoai

1. Quan ly bi mat API key qua secret manager (khong hardcode).
2. Luon gui `X-Tenant-ID` ro rang, va `X-KG-Namespace` dung cong thuc.
3. Dung retry co backoff cho `429/5xx`.
4. Han che size batch <= 1000 va toi uu idempotency theo `properties.id`.
5. Giam sat `/readyz` + metrics + log auth/rate-limit.
6. Test flow overlay commit/discard truoc khi bat tren traffic that.
7. Neu dung semantic search, dam bao vector size giua embedding provider va Qdrant khop nhau.

