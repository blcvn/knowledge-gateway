# API Specifications: AI Knowledge Graph Service (ai-kg-service)

> **Version:** 1.1 (DOC 4 aligned)  
> **Ngày:** 05/03/2026  
> **Base URL:** `http://localhost:8000` (local), `http://kg-service:8080` (internal K8s)  
> **Protocol:** HTTP (gRPC-Gateway) + gRPC  
> **Tham chiếu:** [LLD §6 — API Contracts](lld_ai_kg_service.md), [DOC 4 — Shared Contracts](doc4_contracts.md), [DOC 4 Compliance Review](doc4_compliance_review.md), Protobuf definitions (`api/*/v1/*.proto`)

---

## Mục lục

1. [Authentication & Headers](#1-authentication--headers)
2. [App Registry Service](#2-app-registry-service) — `registry/v1/registry.proto`
3. [Ontology Service](#3-ontology-service) — `ontology/v1/ontology.proto`
4. [Graph Service](#4-graph-service) — `graph/v1/graph.proto`
5. [Rule Engine Service](#5-rule-engine-service) — `rules/v1/rules.proto`
6. [Access Control Service](#6-access-control-service) — `accesscontrol/v1/policy.proto`
7. [Extended APIs (LLD Planned)](#7-extended-apis-lld-planned) — Batch, Search, Overlay, Projection, Analytics, Versioning
8. [NATS Event Schemas (DOC 4 §5, §8)](#8-nats-event-schemas-doc-4-5-8)

---

## 1. Authentication & Headers

### Yêu cầu chung cho mọi request

| Header           | Required/Optional                       | Mô tả                                                   |
| ---------------- | --------------------------------------- | ------------------------------------------------------- |
| `Authorization`  | **Required** (trừ `POST /v1/apps`)      | `Bearer <api_key>` — API key được phát hành từ Registry |
| `Content-Type`   | **Required** (POST/PUT)                 | `application/json`                                      |
| `Accept`         | Optional                                | `application/json` (default)                            |
| `X-API-Key`      | Optional                                | Alternative cho `Authorization` header                  |
| `X-KG-Namespace` | **Required** (cho KG APIs — §7 planned) | `graph/{appId}/{tenantId}` — namespace isolation        |
| `X-Org-ID`       | Optional                                | Organization ID (enterprise multi-org). DOC 4 §2        |

### Error Response chung

Tất cả API trả về lỗi theo format:

```json
{
  "code": 400,
  "reason": "ERR_SCHEMA_INVALID",
  "message": "invalid json schema definition: ...",
  "metadata": {}
}
```

---

## 2. App Registry Service

> **Proto:** `api/registry/v1/registry.proto`  
> **gRPC Service:** `api.registry.v1.Registry`

---

### 2.1 Create Application

Tạo một application mới trong hệ thống KGS.

|          |                      |
| -------- | -------------------- |
| **gRPC** | `Registry.CreateApp` |
| **HTTP** | `POST /v1/apps`      |
| **Auth** | Không yêu cầu        |

#### Request — `CreateAppRequest`

| Field         | Type     | Required     | Mô tả                                          |
| ------------- | -------- | ------------ | ---------------------------------------------- |
| `app_name`    | `string` | **Required** | Tên application (tối đa 200 ký tự)             |
| `description` | `string` | Optional     | Mô tả application                              |
| `owner`       | `string` | **Required** | Owner/admin của application (tối đa 100 ký tự) |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/apps \
  -H "Content-Type: application/json" \
  -d '{
    "app_name": "BA Document Intelligence",
    "description": "Knowledge graph for business requirement documents",
    "owner": "admin@company.com"
  }'
```

#### Response — `CreateAppReply`

| Field    | Type     | Mô tả                       |
| -------- | -------- | --------------------------- |
| `app_id` | `string` | UUID v4 — unique identifier |
| `status` | `string` | `ACTIVE`                    |

```json
{
  "app_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "ACTIVE"
}
```

---

### 2.2 Get Application

Lấy thông tin chi tiết của một application.

|          |                         |
| -------- | ----------------------- |
| **gRPC** | `Registry.GetApp`       |
| **HTTP** | `GET /v1/apps/{app_id}` |
| **Auth** | Không yêu cầu           |

#### Request — `GetAppRequest`

| Field    | Type     | Required     | Mô tả                             |
| -------- | -------- | ------------ | --------------------------------- |
| `app_id` | `string` | **Required** | Path param — UUID của application |

#### Request Example

```bash
curl http://localhost:8000/v1/apps/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

#### Response — `GetAppReply`

| Field         | Type     | Mô tả                               |
| ------------- | -------- | ----------------------------------- |
| `app_id`      | `string` | UUID                                |
| `app_name`    | `string` | Tên application                     |
| `description` | `string` | Mô tả                               |
| `owner`       | `string` | Owner                               |
| `status`      | `string` | `ACTIVE` / `INACTIVE` / `SUSPENDED` |

```json
{
  "app_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "app_name": "BA Document Intelligence",
  "description": "Knowledge graph for business requirement documents",
  "owner": "admin@company.com",
  "status": "ACTIVE"
}
```

---

### 2.3 List Applications

Liệt kê tất cả application đã đăng ký.

|          |                     |
| -------- | ------------------- |
| **gRPC** | `Registry.ListApps` |
| **HTTP** | `GET /v1/apps`      |
| **Auth** | Không yêu cầu       |

#### Request — `ListAppsRequest`

Không có tham số.

#### Request Example

```bash
curl http://localhost:8000/v1/apps
```

#### Response — `ListAppsReply`

| Field  | Type            | Mô tả                  |
| ------ | --------------- | ---------------------- |
| `apps` | `GetAppReply[]` | Danh sách applications |

```json
{
  "apps": [
    {
      "app_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "app_name": "BA Document Intelligence",
      "description": "Knowledge graph for business requirement documents",
      "owner": "admin@company.com",
      "status": "ACTIVE"
    }
  ]
}
```

---

### 2.4 Issue API Key

Phát hành API key cho một application. API key chỉ trả về **một lần duy nhất**.

|          |                               |
| -------- | ----------------------------- |
| **gRPC** | `Registry.IssueApiKey`        |
| **HTTP** | `POST /v1/apps/{app_id}/keys` |
| **Auth** | Không yêu cầu                 |

#### Request — `IssueApiKeyRequest`

| Field         | Type     | Required     | Mô tả                                                          |
| ------------- | -------- | ------------ | -------------------------------------------------------------- |
| `app_id`      | `string` | **Required** | Path param — UUID của application                              |
| `name`        | `string` | **Required** | Tên descriptive cho API key                                    |
| `scopes`      | `string` | **Required** | Comma-separated scopes: `read`, `write`, `all`                 |
| `ttl_seconds` | `int64`  | Optional     | Time-to-live tính bằng giây. `0` = không hết hạn. Default: `0` |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/apps/a1b2c3d4-e5f6-7890-abcd-ef1234567890/keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production API Key",
    "scopes": "read,write",
    "ttl_seconds": 86400
  }'
```

#### Response — `IssueApiKeyReply`

| Field        | Type     | Mô tả                                             |
| ------------ | -------- | ------------------------------------------------- |
| `api_key`    | `string` | Full API key — **chỉ trả về 1 lần, lưu lại ngay** |
| `key_hash`   | `string` | SHA-256 hash — dùng để revoke                     |
| `key_prefix` | `string` | Prefix (vài ký tự đầu) để nhận diện               |

```json
{
  "api_key": "kgs_ak_x7y8z9w0a1b2c3d4e5f67890abcdef12",
  "key_hash": "sha256_abc123def456789...",
  "key_prefix": "kgs_ak_x7"
}
```

> ⚠️ **Lưu ý:** `api_key` chỉ trả về một lần duy nhất trong response này. Nếu mất key, phải issue key mới.

---

### 2.5 Revoke API Key

Thu hồi một API key đã phát hành.

|          |                                   |
| -------- | --------------------------------- |
| **gRPC** | `Registry.RevokeApiKey`           |
| **HTTP** | `DELETE /v1/keys/{key_hash}`      |
| **Auth** | **Required** — `Bearer <api_key>` |

#### Request — `RevokeApiKeyRequest`

| Field      | Type     | Required     | Mô tả                                        |
| ---------- | -------- | ------------ | -------------------------------------------- |
| `key_hash` | `string` | **Required** | Path param — SHA-256 hash của key cần revoke |

#### Request Example

```bash
curl -X DELETE http://localhost:8000/v1/keys/sha256_abc123def456789 \
  -H "Authorization: Bearer kgs_ak_x7y8z9w0a1b2c3d4e5f67890abcdef12"
```

#### Response — `RevokeApiKeyReply`

| Field     | Type   | Mô tả                        |
| --------- | ------ | ---------------------------- |
| `success` | `bool` | `true` nếu revoke thành công |

```json
{
  "success": true
}
```

---

## 3. Ontology Service

> **Proto:** `api/ontology/v1/ontology.proto`  
> **gRPC Service:** `api.ontology.v1.Ontology`  
> **Mục đích:** Quản lý schema (EntityType, RelationType) của knowledge graph

---

### 3.1 Create Entity Type

Định nghĩa một loại entity mới trong Knowledge Graph.

|          |                              |
| -------- | ---------------------------- |
| **gRPC** | `Ontology.CreateEntityType`  |
| **HTTP** | `POST /v1/ontology/entities` |
| **Auth** | **Required**                 |

#### Request — `CreateEntityTypeRequest`

| Field         | Type     | Required     | Mô tả                                                                   |
| ------------- | -------- | ------------ | ----------------------------------------------------------------------- |
| `name`        | `string` | **Required** | Tên entity type (vd: `Requirement`, `UseCase`, `Actor`). Unique per app |
| `description` | `string` | Optional     | Mô tả entity type                                                       |
| `schema`      | `string` | **Required** | JSON Schema definition cho properties của entity type (raw JSON string) |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/ontology/entities \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "name": "Requirement",
    "description": "A business or functional requirement extracted from documents",
    "schema": "{\"type\":\"object\",\"properties\":{\"priority\":{\"type\":\"string\",\"enum\":[\"HIGH\",\"MEDIUM\",\"LOW\"]},\"status\":{\"type\":\"string\",\"enum\":[\"draft\",\"approved\",\"implemented\"]},\"source\":{\"type\":\"string\"}}}"
  }'
```

#### Response — `CreateEntityTypeReply`

| Field    | Type     | Mô tả                  |
| -------- | -------- | ---------------------- |
| `id`     | `uint32` | Auto-increment ID      |
| `name`   | `string` | Tên entity type đã tạo |
| `status` | `string` | `created`              |

```json
{
  "id": 1,
  "name": "Requirement",
  "status": "created"
}
```

#### Error Cases

| HTTP Status | Reason               | Khi nào                                    |
| ----------- | -------------------- | ------------------------------------------ |
| `400`       | `ERR_SCHEMA_INVALID` | `schema` không phải JSON Schema hợp lệ     |
| `409`       | `ERR_DUPLICATE`      | Entity type cùng `name` đã tồn tại cho app |

---

### 3.2 Create Relation Type

Định nghĩa một loại relation (edge) mới.

|          |                               |
| -------- | ----------------------------- |
| **gRPC** | `Ontology.CreateRelationType` |
| **HTTP** | `POST /v1/ontology/relations` |
| **Auth** | **Required**                  |

#### Request — `CreateRelationTypeRequest`

| Field               | Type       | Required     | Mô tả                                                              |
| ------------------- | ---------- | ------------ | ------------------------------------------------------------------ |
| `name`              | `string`   | **Required** | Tên relation type (vd: `DEPENDS_ON`, `IMPLEMENTS`). Unique per app |
| `description`       | `string`   | Optional     | Mô tả relation                                                     |
| `properties_schema` | `string`   | Optional     | JSON Schema cho edge properties (raw JSON string)                  |
| `source_types`      | `string[]` | **Required** | Danh sách EntityType names hợp lệ cho source node                  |
| `target_types`      | `string[]` | **Required** | Danh sách EntityType names hợp lệ cho target node                  |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/ontology/relations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "name": "DEPENDS_ON",
    "description": "Dependency relationship between requirements",
    "properties_schema": "{\"type\":\"object\",\"properties\":{\"strength\":{\"type\":\"number\",\"minimum\":0,\"maximum\":1},\"notes\":{\"type\":\"string\"}}}",
    "source_types": ["Requirement", "UseCase"],
    "target_types": ["Requirement", "UseCase", "NFR"]
  }'
```

#### Response — `CreateRelationTypeReply`

| Field    | Type     | Mô tả             |
| -------- | -------- | ----------------- |
| `id`     | `uint32` | Auto-increment ID |
| `name`   | `string` | Tên relation type |
| `status` | `string` | `created`         |

```json
{
  "id": 1,
  "name": "DEPENDS_ON",
  "status": "created"
}
```

---

### 3.3 List Entity Types

Liệt kê tất cả entity types đã đăng ký.

|          |                             |
| -------- | --------------------------- |
| **gRPC** | `Ontology.ListEntityTypes`  |
| **HTTP** | `GET /v1/ontology/entities` |
| **Auth** | **Required**                |

#### Request — `ListEntityTypesRequest`

Không có tham số.

#### Request Example

```bash
curl http://localhost:8000/v1/ontology/entities \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `ListEntityTypesReply`

| Field      | Type               | Mô tả                  |
| ---------- | ------------------ | ---------------------- |
| `entities` | `EntityTypeInfo[]` | Danh sách entity types |

**EntityTypeInfo:**

| Field    | Type     | Mô tả                  |
| -------- | -------- | ---------------------- |
| `id`     | `uint32` | ID                     |
| `name`   | `string` | Tên entity type        |
| `schema` | `string` | JSON Schema definition |

```json
{
  "entities": [
    {
      "id": 1,
      "name": "Requirement",
      "schema": "{\"type\":\"object\",\"properties\":{\"priority\":{\"type\":\"string\"},\"status\":{\"type\":\"string\"}}}"
    },
    {
      "id": 2,
      "name": "UseCase",
      "schema": "{\"type\":\"object\",\"properties\":{\"actor\":{\"type\":\"string\"},\"goal\":{\"type\":\"string\"}}}"
    }
  ]
}
```

---

### 3.4 List Relation Types

Liệt kê tất cả relation types đã đăng ký.

|          |                              |
| -------- | ---------------------------- |
| **gRPC** | `Ontology.ListRelationTypes` |
| **HTTP** | `GET /v1/ontology/relations` |
| **Auth** | **Required**                 |

#### Request — `ListRelationTypesRequest`

Không có tham số.

#### Request Example

```bash
curl http://localhost:8000/v1/ontology/relations \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `ListRelationTypesReply`

| Field       | Type                 | Mô tả                    |
| ----------- | -------------------- | ------------------------ |
| `relations` | `RelationTypeInfo[]` | Danh sách relation types |

**RelationTypeInfo:**

| Field               | Type       | Mô tả                           |
| ------------------- | ---------- | ------------------------------- |
| `id`                | `uint32`   | ID                              |
| `name`              | `string`   | Tên relation type               |
| `properties_schema` | `string`   | JSON Schema cho edge properties |
| `source_types`      | `string[]` | Valid source entity types       |
| `target_types`      | `string[]` | Valid target entity types       |

```json
{
  "relations": [
    {
      "id": 1,
      "name": "DEPENDS_ON",
      "properties_schema": "{\"type\":\"object\",\"properties\":{\"strength\":{\"type\":\"number\"}}}",
      "source_types": ["Requirement", "UseCase"],
      "target_types": ["Requirement", "UseCase", "NFR"]
    }
  ]
}
```

---

## 4. Graph Service

> **Proto:** `api/graph/v1/graph.proto`  
> **gRPC Service:** `api.graph.v1.Graph`  
> **Mục đích:** CRUD cho nodes (entities) và edges (relationships) trong Knowledge Graph

---

### 4.1 Create Node

Tạo một node mới trong Knowledge Graph.

|          |                        |
| -------- | ---------------------- |
| **gRPC** | `Graph.CreateNode`     |
| **HTTP** | `POST /v1/graph/nodes` |
| **Auth** | **Required**           |

#### Request — `CreateNodeRequest`

| Field             | Type     | Required     | Mô tả                                                                                     |
| ----------------- | -------- | ------------ | ----------------------------------------------------------------------------------------- |
| `label`           | `string` | **Required** | Label của node — phải match EntityType name đã đăng ký (vd: `Requirement`, `UseCase`)     |
| `properties_json` | `string` | **Required** | Raw JSON string chứa properties. Phải tuân thủ JSON Schema đã định nghĩa trong EntityType |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/graph/nodes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "label": "Requirement",
    "properties_json": "{\"name\":\"FR-001 Payment Gateway\",\"priority\":\"HIGH\",\"status\":\"draft\",\"source\":\"BRD_v2.pdf\"}"
  }'
```

#### Response — `CreateNodeReply`

| Field             | Type     | Mô tả                                            |
| ----------------- | -------- | ------------------------------------------------ |
| `node_id`         | `string` | ID của node trong Neo4j (internal ID hoặc UUID)  |
| `label`           | `string` | Label đã tạo                                     |
| `properties_json` | `string` | Properties đã lưu (bao gồm `app_id` được inject) |

```json
{
  "node_id": "4:abc123:0",
  "label": "Requirement",
  "properties_json": "{\"name\":\"FR-001 Payment Gateway\",\"priority\":\"HIGH\",\"status\":\"draft\",\"source\":\"BRD_v2.pdf\",\"app_id\":\"a1b2c3d4\"}"
}
```

#### Error Cases

| HTTP Status | Reason               | Khi nào                                                           |
| ----------- | -------------------- | ----------------------------------------------------------------- |
| `400`       | `ERR_SCHEMA_INVALID` | `properties_json` không hợp lệ hoặc không match EntityType schema |
| `403`       | `ERR_FORBIDDEN`      | OPA policy từ chối action `CREATE_NODE`                           |

---

### 4.2 Get Node

Lấy thông tin một node theo ID.

|          |                                 |
| -------- | ------------------------------- |
| **gRPC** | `Graph.GetNode`                 |
| **HTTP** | `GET /v1/graph/nodes/{node_id}` |
| **Auth** | **Required**                    |

#### Request — `GetNodeRequest`

| Field     | Type     | Required     | Mô tả                    |
| --------- | -------- | ------------ | ------------------------ |
| `node_id` | `string` | **Required** | Path param — ID của node |

#### Request Example

```bash
curl http://localhost:8000/v1/graph/nodes/4:abc123:0 \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `GetNodeReply`

| Field             | Type     | Mô tả                 |
| ----------------- | -------- | --------------------- |
| `node_id`         | `string` | ID                    |
| `label`           | `string` | Label                 |
| `properties_json` | `string` | Properties (raw JSON) |

```json
{
  "node_id": "4:abc123:0",
  "label": "Requirement",
  "properties_json": "{\"name\":\"FR-001 Payment Gateway\",\"priority\":\"HIGH\",\"status\":\"draft\"}"
}
```

---

### 4.3 Create Edge

Tạo một relationship giữa 2 nodes.

|          |                        |
| -------- | ---------------------- |
| **gRPC** | `Graph.CreateEdge`     |
| **HTTP** | `POST /v1/graph/edges` |
| **Auth** | **Required**           |

#### Request — `CreateEdgeRequest`

| Field             | Type     | Required     | Mô tả                                                                          |
| ----------------- | -------- | ------------ | ------------------------------------------------------------------------------ |
| `source_node_id`  | `string` | **Required** | ID của node nguồn                                                              |
| `target_node_id`  | `string` | **Required** | ID của node đích                                                               |
| `relation_type`   | `string` | **Required** | Loại relation — phải match RelationType đã đăng ký                             |
| `properties_json` | `string` | Optional     | JSON properties cho edge. Nếu có, phải tuân thủ RelationType.properties_schema |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/graph/edges \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "source_node_id": "4:abc123:0",
    "target_node_id": "4:def456:0",
    "relation_type": "DEPENDS_ON",
    "properties_json": "{\"strength\": 0.85, \"notes\": \"Payment depends on auth module\"}"
  }'
```

#### Response — `CreateEdgeReply`

| Field             | Type     | Mô tả             |
| ----------------- | -------- | ----------------- |
| `edge_id`         | `string` | ID của edge       |
| `source_node_id`  | `string` | Source node ID    |
| `target_node_id`  | `string` | Target node ID    |
| `relation_type`   | `string` | Loại relation     |
| `properties_json` | `string` | Properties đã lưu |

```json
{
  "edge_id": "5:rel789:0",
  "source_node_id": "4:abc123:0",
  "target_node_id": "4:def456:0",
  "relation_type": "DEPENDS_ON",
  "properties_json": "{\"strength\":0.85,\"notes\":\"Payment depends on auth module\"}"
}
```

---

### 4.4 Get Context (Neighborhood)

Lấy các node láng giềng (cùng relationship) xung quanh một node.

|          |                                         |
| -------- | --------------------------------------- |
| **gRPC** | `Graph.GetContext`                      |
| **HTTP** | `GET /v1/graph/nodes/{node_id}/context` |
| **Auth** | **Required**                            |

#### Request — `GetContextRequest`

| Field       | Type     | Required     | Mô tả                                                                  |
| ----------- | -------- | ------------ | ---------------------------------------------------------------------- |
| `node_id`   | `string` | **Required** | Path param — ID node trung tâm                                         |
| `depth`     | `int32`  | Optional     | Độ sâu traversal (1–10). Default: `1`. Max: `10`                       |
| `direction` | `string` | Optional     | Hướng traversal. Enum: `INCOMING`, `OUTGOING`, `BOTH`. Default: `BOTH` |

#### Request Example

```bash
# HTTP — query params cho các field không nằm trong path
curl "http://localhost:8000/v1/graph/nodes/4:abc123:0/context?depth=2&direction=BOTH" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `GraphReply`

| Field   | Type          | Mô tả                         |
| ------- | ------------- | ----------------------------- |
| `nodes` | `GraphNode[]` | Danh sách nodes trong context |
| `edges` | `GraphEdge[]` | Danh sách edges kết nối       |

**GraphNode:**

| Field             | Type     | Mô tả                 |
| ----------------- | -------- | --------------------- |
| `id`              | `string` | Node ID               |
| `label`           | `string` | Label (entity type)   |
| `properties_json` | `string` | Properties (raw JSON) |

**GraphEdge:**

| Field             | Type     | Mô tả           |
| ----------------- | -------- | --------------- |
| `id`              | `string` | Edge ID         |
| `source`          | `string` | Source node ID  |
| `target`          | `string` | Target node ID  |
| `type`            | `string` | Relation type   |
| `properties_json` | `string` | Edge properties |

```json
{
  "nodes": [
    {
      "id": "4:abc123:0",
      "label": "Requirement",
      "properties_json": "{\"name\":\"FR-001 Payment Gateway\",\"priority\":\"HIGH\"}"
    },
    {
      "id": "4:def456:0",
      "label": "Requirement",
      "properties_json": "{\"name\":\"FR-002 Auth Module\",\"priority\":\"HIGH\"}"
    }
  ],
  "edges": [
    {
      "id": "5:rel789:0",
      "source": "4:abc123:0",
      "target": "4:def456:0",
      "type": "DEPENDS_ON",
      "properties_json": "{\"strength\":0.85}"
    }
  ]
}
```

#### Error Cases

| HTTP Status | Reason               | Khi nào                        |
| ----------- | -------------------- | ------------------------------ |
| `400`       | `ERR_DEPTH_EXCEEDED` | `depth` > 10 (MaxAllowedDepth) |

---

### 4.5 Get Impact (Downstream)

Phân tích tác động downstream — tìm tất cả nodes bị ảnh hưởng theo hướng outgoing.

|          |                                        |
| -------- | -------------------------------------- |
| **gRPC** | `Graph.GetImpact`                      |
| **HTTP** | `GET /v1/graph/nodes/{node_id}/impact` |
| **Auth** | **Required**                           |

#### Request — `GetImpactRequest`

| Field       | Type     | Required     | Mô tả                                  |
| ----------- | -------- | ------------ | -------------------------------------- |
| `node_id`   | `string` | **Required** | Path param — ID node gốc               |
| `max_depth` | `int32`  | Optional     | Độ sâu tối đa. Default: `3`. Max: `10` |

#### Request Example

```bash
curl "http://localhost:8000/v1/graph/nodes/4:abc123:0/impact?max_depth=3" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `GraphReply`

Cùng format với [§4.4 GetContext Response](#response--graphreply).

```json
{
  "nodes": [
    {
      "id": "4:def456:0",
      "label": "UseCase",
      "properties_json": "{\"name\":\"UC-001\"}"
    },
    {
      "id": "4:ghi789:0",
      "label": "APIEndpoint",
      "properties_json": "{\"path\":\"/api/payment\"}"
    }
  ],
  "edges": [
    {
      "id": "5:r1",
      "source": "4:abc123:0",
      "target": "4:def456:0",
      "type": "IMPLEMENTS",
      "properties_json": "{}"
    },
    {
      "id": "5:r2",
      "source": "4:def456:0",
      "target": "4:ghi789:0",
      "type": "CALLS",
      "properties_json": "{}"
    }
  ]
}
```

---

### 4.6 Get Coverage (Upstream)

Phân tích coverage upstream — tìm tất cả nodes từ cấp trên trỏ đến node này.

|          |                                          |
| -------- | ---------------------------------------- |
| **gRPC** | `Graph.GetCoverage`                      |
| **HTTP** | `GET /v1/graph/nodes/{node_id}/coverage` |
| **Auth** | **Required**                             |

#### Request — `GetCoverageRequest`

| Field       | Type     | Required     | Mô tả                                  |
| ----------- | -------- | ------------ | -------------------------------------- |
| `node_id`   | `string` | **Required** | Path param — ID node                   |
| `max_depth` | `int32`  | Optional     | Độ sâu tối đa. Default: `3`. Max: `10` |

#### Request Example

```bash
curl "http://localhost:8000/v1/graph/nodes/4:def456:0/coverage?max_depth=5" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `GraphReply`

Cùng format với [§4.4 GetContext Response](#response--graphreply).

```json
{
  "nodes": [
    {
      "id": "4:abc123:0",
      "label": "Requirement",
      "properties_json": "{\"name\":\"FR-001\"}"
    }
  ],
  "edges": [
    {
      "id": "5:r1",
      "source": "4:abc123:0",
      "target": "4:def456:0",
      "type": "TRACED_TO",
      "properties_json": "{}"
    }
  ]
}
```

---

### 4.7 Get Subgraph

Lấy subgraph chứa tập hợp nodes và tất cả edges nối giữa chúng.

|          |                           |
| -------- | ------------------------- |
| **gRPC** | `Graph.GetSubgraph`       |
| **HTTP** | `POST /v1/graph/subgraph` |
| **Auth** | **Required**              |

#### Request — `GetSubgraphRequest`

| Field      | Type       | Required     | Mô tả                            |
| ---------- | ---------- | ------------ | -------------------------------- |
| `node_ids` | `string[]` | **Required** | Danh sách node IDs (tối đa 1000) |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/graph/subgraph \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "node_ids": ["4:abc123:0", "4:def456:0", "4:ghi789:0"]
  }'
```

#### Response — `GraphReply`

Cùng format với [§4.4 GetContext Response](#response--graphreply).

```json
{
  "nodes": [
    {
      "id": "4:abc123:0",
      "label": "Requirement",
      "properties_json": "{\"name\":\"FR-001\"}"
    },
    {
      "id": "4:def456:0",
      "label": "UseCase",
      "properties_json": "{\"name\":\"UC-001\"}"
    },
    {
      "id": "4:ghi789:0",
      "label": "APIEndpoint",
      "properties_json": "{\"path\":\"/api/payment\"}"
    }
  ],
  "edges": [
    {
      "id": "5:r1",
      "source": "4:abc123:0",
      "target": "4:def456:0",
      "type": "DEPENDS_ON",
      "properties_json": "{}"
    },
    {
      "id": "5:r2",
      "source": "4:def456:0",
      "target": "4:ghi789:0",
      "type": "CALLS",
      "properties_json": "{}"
    }
  ]
}
```

#### Error Cases

| HTTP Status | Reason               | Khi nào                      |
| ----------- | -------------------- | ---------------------------- |
| `400`       | `ERR_NODES_EXCEEDED` | `node_ids` chứa > 1000 items |

---

## 5. Rule Engine Service

> **Proto:** `api/rules/v1/rules.proto`  
> **gRPC Service:** `api.rules.v1.Rules`  
> **Mục đích:** Quản lý business rules chạy scheduled (gocron) hoặc event-driven (Redis Stream)

---

### 5.1 Create Rule

Tạo một business rule mới.

|          |                    |
| -------- | ------------------ |
| **gRPC** | `Rules.CreateRule` |
| **HTTP** | `POST /v1/rules`   |
| **Auth** | **Required**       |

#### Request — `CreateRuleRequest`

| Field          | Type     | Required        | Mô tả                                                         |
| -------------- | -------- | --------------- | ------------------------------------------------------------- |
| `name`         | `string` | **Required**    | Tên rule (tối đa 100 ký tự)                                   |
| `description`  | `string` | Optional        | Mô tả rule                                                    |
| `trigger_type` | `string` | **Required**    | Enum: `SCHEDULED` hoặc `ON_WRITE`                             |
| `cron`         | `string` | **Conditional** | Cron expression. **Required** khi `trigger_type = SCHEDULED`  |
| `cypher_query` | `string` | **Required**    | Cypher query để execute khi rule trigger                      |
| `action`       | `string` | Optional        | Loại action: `LOG`, `WEBHOOK`, `NOTIFICATION`. Default: `LOG` |
| `payload_json` | `string` | Optional        | JSON payload cho action (vd: webhook URL). Default: `{}`      |

#### Request Example — Scheduled Rule

```bash
curl -X POST http://localhost:8000/v1/rules \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "name": "Circular Dependency Detector",
    "description": "Detect circular dependencies in requirement graph every 6 hours",
    "trigger_type": "SCHEDULED",
    "cron": "0 */6 * * *",
    "cypher_query": "MATCH p=(n:Requirement)-[:DEPENDS_ON*2..5]->(n) RETURN p LIMIT 10",
    "action": "LOG",
    "payload_json": "{}"
  }'
```

#### Request Example — ON_WRITE Rule

```bash
curl -X POST http://localhost:8000/v1/rules \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "name": "Auto-tag High Priority",
    "description": "When a node is created, tag it if priority is HIGH",
    "trigger_type": "ON_WRITE",
    "cypher_query": "MATCH (n {app_id: $app_id}) WHERE n.priority = \"HIGH\" SET n.tagged = true RETURN n",
    "action": "LOG",
    "payload_json": "{}"
  }'
```

#### Response — `RuleReply`

| Field          | Type     | Mô tả                                      |
| -------------- | -------- | ------------------------------------------ |
| `id`           | `int64`  | Auto-increment ID                          |
| `name`         | `string` | Tên rule                                   |
| `description`  | `string` | Mô tả                                      |
| `trigger_type` | `string` | `SCHEDULED` hoặc `ON_WRITE`                |
| `cron`         | `string` | Cron expression (có thể rỗng nếu ON_WRITE) |
| `cypher_query` | `string` | Cypher query                               |
| `action`       | `string` | Action type                                |
| `payload_json` | `string` | Action payload                             |
| `is_active`    | `bool`   | `true` — rule đang active                  |

```json
{
  "id": 1,
  "name": "Circular Dependency Detector",
  "description": "Detect circular dependencies in requirement graph every 6 hours",
  "trigger_type": "SCHEDULED",
  "cron": "0 */6 * * *",
  "cypher_query": "MATCH p=(n:Requirement)-[:DEPENDS_ON*2..5]->(n) RETURN p LIMIT 10",
  "action": "LOG",
  "payload_json": "{}",
  "is_active": true
}
```

---

### 5.2 List Rules

Liệt kê tất cả rules của app.

|          |                   |
| -------- | ----------------- |
| **gRPC** | `Rules.ListRules` |
| **HTTP** | `GET /v1/rules`   |
| **Auth** | **Required**      |

#### Request — `ListRulesRequest`

Không có tham số.

#### Request Example

```bash
curl http://localhost:8000/v1/rules \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `ListRulesReply`

| Field   | Type          | Mô tả           |
| ------- | ------------- | --------------- |
| `rules` | `RuleReply[]` | Danh sách rules |

```json
{
  "rules": [
    {
      "id": 1,
      "name": "Circular Dependency Detector",
      "trigger_type": "SCHEDULED",
      "cron": "0 */6 * * *",
      "cypher_query": "MATCH p=(n:Requirement)-[:DEPENDS_ON*2..5]->(n) RETURN p LIMIT 10",
      "action": "LOG",
      "payload_json": "{}",
      "is_active": true
    }
  ]
}
```

---

### 5.3 Get Rule

Lấy chi tiết một rule.

|          |                      |
| -------- | -------------------- |
| **gRPC** | `Rules.GetRule`      |
| **HTTP** | `GET /v1/rules/{id}` |
| **Auth** | **Required**         |

#### Request — `GetRuleRequest`

| Field | Type    | Required     | Mô tả                |
| ----- | ------- | ------------ | -------------------- |
| `id`  | `int64` | **Required** | Path param — Rule ID |

#### Request Example

```bash
curl http://localhost:8000/v1/rules/1 \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `RuleReply`

Cùng format với [§5.1 Response](#response--rulereply).

---

## 6. Access Control Service

> **Proto:** `api/accesscontrol/v1/policy.proto`  
> **gRPC Service:** `api.accesscontrol.v1.AccessControl`  
> **Mục đích:** Quản lý OPA Rego policies cho fine-grained access control

---

### 6.1 Create Policy

Tạo một OPA policy mới. Policy sẽ được upload lên OPA sidecar tự động.

|          |                              |
| -------- | ---------------------------- |
| **gRPC** | `AccessControl.CreatePolicy` |
| **HTTP** | `POST /v1/policies`          |
| **Auth** | **Required**                 |

#### Request — `CreatePolicyRequest`

| Field          | Type     | Required     | Mô tả                                                   |
| -------------- | -------- | ------------ | ------------------------------------------------------- |
| `name`         | `string` | **Required** | Tên policy (tối đa 100 ký tự)                           |
| `description`  | `string` | Optional     | Mô tả policy                                            |
| `rego_content` | `string` | **Required** | Nội dung Rego policy. Phải bắt đầu bằng `package kgs.*` |

#### Request Example

```bash
curl -X POST http://localhost:8000/v1/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "name": "Graph Read Access",
    "description": "Allow reading graph nodes for users with read scope",
    "rego_content": "package kgs.authz\n\ndefault allow = false\n\nallow {\n    input.user.scopes[_] == \"read\"\n    input.action == \"READ_NODE\"\n}\n\nallow {\n    input.user.scopes[_] == \"all\"\n}"
  }'
```

#### Response — `PolicyReply`

| Field          | Type     | Mô tả                       |
| -------------- | -------- | --------------------------- |
| `id`           | `int64`  | Auto-increment ID           |
| `name`         | `string` | Tên policy                  |
| `description`  | `string` | Mô tả                       |
| `rego_content` | `string` | Nội dung Rego               |
| `is_active`    | `bool`   | `true` — policy đang active |

```json
{
  "id": 1,
  "name": "Graph Read Access",
  "description": "Allow reading graph nodes for users with read scope",
  "rego_content": "package kgs.authz\n\ndefault allow = false\n\nallow {\n    input.user.scopes[_] == \"read\"\n    input.action == \"READ_NODE\"\n}\n\nallow {\n    input.user.scopes[_] == \"all\"\n}",
  "is_active": true
}
```

---

### 6.2 List Policies

|          |                              |
| -------- | ---------------------------- |
| **gRPC** | `AccessControl.ListPolicies` |
| **HTTP** | `GET /v1/policies`           |
| **Auth** | **Required**                 |

#### Request — `ListPoliciesRequest`

Không có tham số.

#### Request Example

```bash
curl http://localhost:8000/v1/policies \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `ListPoliciesReply`

| Field      | Type            | Mô tả              |
| ---------- | --------------- | ------------------ |
| `policies` | `PolicyReply[]` | Danh sách policies |

```json
{
  "policies": [
    {
      "id": 1,
      "name": "Graph Read Access",
      "description": "Allow reading graph nodes for users with read scope",
      "rego_content": "package kgs.authz\n...",
      "is_active": true
    }
  ]
}
```

---

### 6.3 Get Policy

|          |                           |
| -------- | ------------------------- |
| **gRPC** | `AccessControl.GetPolicy` |
| **HTTP** | `GET /v1/policies/{id}`   |
| **Auth** | **Required**              |

#### Request — `GetPolicyRequest`

| Field | Type    | Required     | Mô tả                  |
| ----- | ------- | ------------ | ---------------------- |
| `id`  | `int64` | **Required** | Path param — Policy ID |

#### Request Example

```bash
curl http://localhost:8000/v1/policies/1 \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response — `PolicyReply`

Cùng format với [§6.1 Response](#response--policyreply).

---

## 7. Extended APIs (LLD Planned)

> **Status:** Thiết kế trong LLD §6 — chưa có proto definitions  
> **Network Policy:** Chỉ nhận traffic từ Execution Platform (internal only)

Các API dưới đây nằm trong kế hoạch triển khai theo LLD. Chúng sẽ được namespace-scoped với `{ns}` = `graph/{appId}/{tenantId}`.

---

### 7.1 Batch Upsert Entities

Bulk upsert entities với semantic dedup.

|            |                                         |
| ---------- | --------------------------------------- |
| **HTTP**   | `POST /kg/{ns}/entities/batch`          |
| **Caller** | KG Write Agent (via Execution Platform) |

#### Request

| Field                       | Type        | Required     | Mô tả                                                                                 |
| --------------------------- | ----------- | ------------ | ------------------------------------------------------------------------------------- |
| `entities`                  | `Entity[]`  | **Required** | Danh sách entities (tối đa 1000)                                                      |
| `entities[].entityId`       | `string`    | **Required** | UUID v4                                                                               |
| `entities[].entityType`     | `string`    | **Required** | Loại entity (phải match registered EntityType)                                        |
| `entities[].name`           | `string`    | **Required** | Tên entity                                                                            |
| `entities[].properties`     | `object`    | Optional     | Key-value properties                                                                  |
| `entities[].embedding`      | `float32[]` | Optional     | Vector embedding cho semantic search                                                  |
| `entities[].confidence`     | `float64`   | Optional     | Độ tin cậy [0.0–1.0]. Default: `1.0`                                                  |
| `entities[].sourceFile`     | `string`    | Optional     | File nguồn từ đó extract                                                              |
| `entities[].chunkId`        | `string`    | Optional     | Chunk ID trong file nguồn                                                             |
| `entities[].skillId`        | `string`    | Optional     | Skill ID đã extract entity                                                            |
| `entities[].provenanceType` | `string`    | Optional     | Enum: `EXTRACTED`, `GENERATED`, `MANUAL`. Default: `EXTRACTED`                        |
| `entities[].domains`        | `string[]`  | Optional     | Danh sách domain tags                                                                 |
| `overlayId`                 | `string`    | Optional     | Nếu cung cấp → ghi vào overlay thay vì base graph                                     |
| `conflictPolicy`            | `string`    | Optional     | Enum: `KEEP_OVERLAY`, `KEEP_BASE`, `MERGE`, `REQUIRE_MANUAL`. Default: `KEEP_OVERLAY` |

#### Request Example

```bash
curl -X POST http://localhost:8080/kg/graph/app-001/tenant-001/entities/batch \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "entities": [
        {
            "entityId": "550e8400-e29b-41d4-a716-446655440001",
            "entityType": "Requirement",
            "name": "FR-001 Payment Gateway Integration",
            "properties": {"priority": "HIGH", "status": "draft"},
            "embedding": [0.123, 0.456, 0.789],
            "confidence": 0.92,
            "sourceFile": "BRD_v2.pdf",
            "chunkId": "chunk-003",
            "provenanceType": "EXTRACTED",
            "domains": ["payment"]
        },
        {
            "entityId": "550e8400-e29b-41d4-a716-446655440002",
            "entityType": "UseCase",
            "name": "UC-001 Process Refund",
            "properties": {"actor": "Customer", "goal": "Request refund"},
            "confidence": 0.88,
            "sourceFile": "BRD_v2.pdf",
            "provenanceType": "EXTRACTED",
            "domains": ["payment", "customer-service"]
        }
    ],
    "overlayId": "overlay-session-abc123",
    "conflictPolicy": "KEEP_OVERLAY"
  }'
```

#### Response

| Field        | Type      | Mô tả                                       |
| ------------ | --------- | ------------------------------------------- |
| `created`    | `int`     | Số entities được tạo mới                    |
| `updated`    | `int`     | Số entities được update (merge)             |
| `skipped`    | `int`     | Số entities bị skip (trùng, không thay đổi) |
| `conflicted` | `int`     | Số entities có conflict                     |
| `errors`     | `Error[]` | Danh sách lỗi (nếu có)                      |

```json
{
  "created": 2,
  "updated": 0,
  "skipped": 0,
  "conflicted": 0,
  "errors": []
}
```

---

### 7.2 Hybrid Search

Tìm kiếm hybrid: Vector (semantic) + BM25 (text) + Graph reranking.

|            |                               |
| ---------- | ----------------------------- |
| **HTTP**   | `POST /kg/{ns}/search/hybrid` |
| **Caller** | KG Read Agent                 |

#### Request

| Field                     | Type       | Required     | Mô tả                                                              |
| ------------------------- | ---------- | ------------ | ------------------------------------------------------------------ |
| `query`                   | `string`   | **Required** | Câu query tìm kiếm                                                 |
| `topK`                    | `int`      | Optional     | Số kết quả trả về. Default: `20`. Max: `100`                       |
| `alpha`                   | `float`    | Optional     | Blend ratio semantic/text [0.0=text, 1.0=semantic]. Default: `0.5` |
| `filters`                 | `object`   | Optional     | Bộ lọc                                                             |
| `filters.entityTypes`     | `string[]` | Optional     | Lọc theo entity types                                              |
| `filters.domains`         | `string[]` | Optional     | Lọc theo domains                                                   |
| `filters.minConfidence`   | `float`    | Optional     | Confidence tối thiểu [0.0–1.0]                                     |
| `filters.provenanceTypes` | `string[]` | Optional     | Lọc theo provenance: `EXTRACTED`, `GENERATED`, `MANUAL`            |

#### Request Example

```bash
curl -X POST http://localhost:8080/kg/graph/app-001/tenant-001/search/hybrid \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "query": "payment refund process for customer",
    "topK": 10,
    "alpha": 0.6,
    "filters": {
        "entityTypes": ["Requirement", "UseCase"],
        "domains": ["payment"],
        "minConfidence": 0.65,
        "provenanceTypes": ["EXTRACTED"]
    }
  }'
```

#### Response

| Field                      | Type             | Mô tả                                     |
| -------------------------- | ---------------- | ----------------------------------------- |
| `results`                  | `SearchResult[]` | Kết quả đã sắp xếp theo `finalScore` DESC |
| `results[].entityId`       | `string`         | Entity ID                                 |
| `results[].entityType`     | `string`         | Entity type                               |
| `results[].name`           | `string`         | Tên entity                                |
| `results[].finalScore`     | `float`          | Điểm tổng hợp cuối cùng                   |
| `results[].semanticScore`  | `float`          | Điểm cosine similarity                    |
| `results[].textScore`      | `float`          | Điểm BM25                                 |
| `results[].centrality`     | `float`          | PageRank centrality                       |
| `results[].provenanceType` | `string`         | Loại provenance                           |
| `totalCandidates`          | `int`            | Tổng số candidates trước filter           |
| `searchDurationMs`         | `int`            | Thời gian search (ms)                     |

```json
{
  "results": [
    {
      "entityId": "550e8400-e29b-41d4-a716-446655440002",
      "entityType": "UseCase",
      "name": "UC-001 Process Refund",
      "finalScore": 0.87,
      "semanticScore": 0.91,
      "textScore": 0.72,
      "centrality": 0.65,
      "provenanceType": "EXTRACTED"
    }
  ],
  "totalCandidates": 156,
  "searchDurationMs": 234
}
```

---

### 7.3 Overlay — Create

Tạo overlay graph session-scoped cho temporary writes.

|            |                         |
| ---------- | ----------------------- |
| **HTTP**   | `POST /kg/{ns}/overlay` |
| **Caller** | KG Write Agent          |

#### Request

| Field          | Type     | Required     | Mô tả                                            |
| -------------- | -------- | ------------ | ------------------------------------------------ |
| `session_id`   | `string` | **Required** | Chat session ID                                  |
| `base_version` | `string` | Optional     | Version ID để base trên. Default: latest version |

#### Request Example

```bash
curl -X POST http://localhost:8080/kg/graph/app-001/tenant-001/overlay \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "session_id": "session-xyz-789",
    "base_version": "current"
  }'
```

#### Response

```json
{
  "overlayId": "overlay-abc123",
  "status": "CREATED",
  "baseVersionId": "version-v001",
  "ttl": "1h"
}
```

---

### 7.4 Overlay — Commit

Commit overlay vào base graph, tạo version delta mới.

|            |                                     |
| ---------- | ----------------------------------- |
| **HTTP**   | `POST /kg/{ns}/overlay/{id}/commit` |
| **Caller** | KG Write Agent                      |

#### Request

| Field            | Type     | Required     | Mô tả                                                                                 |
| ---------------- | -------- | ------------ | ------------------------------------------------------------------------------------- |
| `id`             | `string` | **Required** | Path param — Overlay ID                                                               |
| `conflictPolicy` | `string` | Optional     | Enum: `KEEP_OVERLAY`, `KEEP_BASE`, `MERGE`, `REQUIRE_MANUAL`. Default: `KEEP_OVERLAY` |

#### Request Example

```bash
curl -X POST http://localhost:8080/kg/graph/app-001/tenant-001/overlay/overlay-abc123/commit \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "conflictPolicy": "KEEP_OVERLAY"
  }'
```

#### Response

```json
{
  "newVersionId": "version-v002",
  "entitiesCommitted": 58,
  "edgesCommitted": 124,
  "conflictsResolved": 3
}
```

> **DOC 4 §5:** After successful commit, KG Service publishes `OVERLAY_COMMIT` event to NATS topic `overlay.committed.{tenantId}`. See [§8 NATS Event Schemas](#8-nats-event-schemas-doc-4-5-8).

#### Error Cases

| HTTP Status | Reason                   | Khi nào                                           |
| ----------- | ------------------------ | ------------------------------------------------- |
| `409`       | `ERR_OVERLAY_CONFLICT`   | `conflictPolicy = REQUIRE_MANUAL` và có conflicts |
| `400`       | `ERR_OVERLAY_NOT_ACTIVE` | Overlay đã committed hoặc discarded               |

---

### 7.5 Overlay — Discard

Hủy overlay (xóa temporary data).

|            |                                |
| ---------- | ------------------------------ |
| **HTTP**   | `DELETE /kg/{ns}/overlay/{id}` |
| **Caller** | KG Write Agent / Chat Agent    |

#### Request Example

```bash
curl -X DELETE http://localhost:8080/kg/graph/app-001/tenant-001/overlay/overlay-abc123 \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response

```json
{
  "overlayId": "overlay-abc123",
  "status": "DISCARDED"
}
```

> **DOC 4 §5:** After discard, KG Service publishes `OVERLAY_DISCARD` event to NATS topic `overlay.discarded.{tenantId}`. See [§8 NATS Event Schemas](#8-nats-event-schemas-doc-4-5-8).

---

### 7.6 Coverage Report

|            |                                  |
| ---------- | -------------------------------- |
| **HTTP**   | `GET /kg/{ns}/coverage/{domain}` |
| **Caller** | Reflect Agent                    |

#### Request

| Field    | Type     | Required     | Mô tả                                            |
| -------- | -------- | ------------ | ------------------------------------------------ |
| `domain` | `string` | **Required** | Path param — Domain để phân tích (vd: `payment`) |

#### Request Example

```bash
curl http://localhost:8080/kg/graph/app-001/tenant-001/coverage/payment \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response

```json
{
  "domain": "payment",
  "totalEntities": 45,
  "coveredEntities": 38,
  "coveragePercent": 84.4,
  "uncoveredTypes": ["NFR", "DataModel"],
  "timestamp": "2026-03-05T10:30:00Z"
}
```

---

### 7.7 Traceability Matrix

|            |                              |
| ---------- | ---------------------------- |
| **HTTP**   | `POST /kg/{ns}/traceability` |
| **Caller** | Output Agent                 |

#### Request

| Field         | Type       | Required     | Mô tả                                                |
| ------------- | ---------- | ------------ | ---------------------------------------------------- |
| `sourceTypes` | `string[]` | **Required** | Entity types nguồn (vd: `["Requirement"]`)           |
| `targetTypes` | `string[]` | **Required** | Entity types đích (vd: `["UseCase", "APIEndpoint"]`) |
| `maxHops`     | `int`      | Optional     | Số hop tối đa. Default: `5`                          |

#### Request Example

```bash
curl -X POST http://localhost:8080/kg/graph/app-001/tenant-001/traceability \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..." \
  -d '{
    "sourceTypes": ["Requirement"],
    "targetTypes": ["UseCase", "APIEndpoint"],
    "maxHops": 3
  }'
```

#### Response

```json
{
  "matrix": [
    {
      "source": { "entityId": "...", "name": "FR-001", "type": "Requirement" },
      "targets": [
        {
          "entityId": "...",
          "name": "UC-001",
          "type": "UseCase",
          "hops": 1,
          "path": ["IMPLEMENTS"]
        },
        {
          "entityId": "...",
          "name": "/api/payment",
          "type": "APIEndpoint",
          "hops": 2,
          "path": ["IMPLEMENTS", "CALLS"]
        }
      ]
    }
  ],
  "totalSources": 15,
  "totalTargets": 42,
  "computeDurationMs": 890
}
```

---

## Error Codes Reference

| Error Code                | HTTP | Retryable | Mô tả                                           |
| ------------------------- | ---- | --------- | ----------------------------------------------- |
| `ERR_UNAUTHORIZED`        | 401  | No        | Missing hoặc invalid API key                    |
| `ERR_FORBIDDEN`           | 403  | No        | OPA policy denied / namespace mismatch          |
| `ERR_SCHEMA_INVALID`      | 400  | No        | JSON Schema hoặc payload validation failed      |
| `ERR_DEPTH_EXCEEDED`      | 400  | No        | Query depth > `MaxAllowedDepth` (10)            |
| `ERR_NODES_EXCEEDED`      | 400  | No        | Node count > `MaxAllowedNodes` (1000)           |
| `ERR_DUPLICATE`           | 409  | No        | Entity/relation type name already exists        |
| `ERR_SESSION_CONFLICT`    | 409  | Yes       | Optimistic lock version mismatch                |
| `ERR_OVERLAY_CONFLICT`    | 409  | No        | Overlay commit conflict (cần manual resolution) |
| `ERR_OVERLAY_NOT_ACTIVE`  | 400  | No        | Overlay đã committed hoặc discarded             |
| `ERR_VERSION_NOT_FOUND`   | 404  | No        | Version ID không tồn tại                        |
| `ERR_NOT_FOUND`           | 404  | No        | Resource không tồn tại                          |
| `ERR_RATE_LIMIT`          | 429  | Yes       | Too many requests (per app quota)               |
| `ERR_TIMEOUT`             | 408  | Yes       | Request timeout                                 |
| `ERR_SERVICE_UNAVAILABLE` | 503  | Yes       | Neo4j / Qdrant / Redis down                     |

### Structured Error Response Format (DOC 4 §6)

```json
{
  "code": 400,
  "reason": "ERR_OVERLAY_NOT_ACTIVE",
  "message": "overlay [overlay-abc123] is already committed",
  "metadata": {
    "overlay_id": "overlay-abc123",
    "current_status": "COMMITTED"
  }
}
```

---

## 7.8 Version Management — List Versions

Liệt kê lịch sử version của knowledge graph.

|            |                                |
| ---------- | ------------------------------ |
| **HTTP**   | `GET /kg/{ns}/versions`        |
| **Caller** | Execution Platform / Dashboard |

#### Request

| Field    | Type  | Required | Mô tả                            |
| -------- | ----- | -------- | -------------------------------- |
| `limit`  | `int` | Optional | Số version trả về. Default: `20` |
| `offset` | `int` | Optional | Phân trang. Default: `0`         |

#### Request Example

```bash
curl "http://localhost:8080/kg/graph/app-001/tenant-001/versions?limit=10" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response

```json
{
  "versions": [
    {
      "versionId": "version-v002",
      "parentVersionId": "version-v001",
      "entitiesAdded": 12,
      "entitiesModified": 3,
      "edgesAdded": 18,
      "createdAt": "2026-03-05T11:30:00Z",
      "label": "session-xyz commit"
    },
    {
      "versionId": "version-v001",
      "parentVersionId": "",
      "entitiesAdded": 45,
      "entitiesModified": 0,
      "edgesAdded": 67,
      "createdAt": "2026-03-05T10:00:00Z",
      "label": "initial extraction"
    }
  ],
  "total": 2
}
```

---

### 7.9 Version Management — Diff Versions

So sánh 2 versions của knowledge graph.

|            |                                        |
| ---------- | -------------------------------------- |
| **HTTP**   | `GET /kg/{ns}/versions/{v1}/diff/{v2}` |
| **Caller** | Execution Platform / Dashboard         |

#### Request

| Field | Type     | Required     | Mô tả                     |
| ----- | -------- | ------------ | ------------------------- |
| `v1`  | `string` | **Required** | Path param — Version ID 1 |
| `v2`  | `string` | **Required** | Path param — Version ID 2 |

#### Request Example

```bash
curl "http://localhost:8080/kg/graph/app-001/tenant-001/versions/version-v001/diff/version-v002" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response

```json
{
  "fromVersion": "version-v001",
  "toVersion": "version-v002",
  "addedEntities": 12,
  "modifiedEntities": 3,
  "removedEntities": 0,
  "addedEdges": 18,
  "removedEdges": 2,
  "changes": [
    {
      "type": "ENTITY_ADDED",
      "entityId": "550e8400-...",
      "entityType": "Requirement",
      "name": "FR-010 Notification"
    },
    {
      "type": "ENTITY_MODIFIED",
      "entityId": "550e8400-...",
      "field": "priority",
      "before": "MEDIUM",
      "after": "HIGH"
    }
  ]
}
```

---

### 7.10 Version Management — Rollback

Restore graph về một version trước đó.

|            |                                        |
| ---------- | -------------------------------------- |
| **HTTP**   | `POST /kg/{ns}/versions/{id}/rollback` |
| **Caller** | Admin / Dashboard                      |

#### Request

| Field | Type     | Required     | Mô tả                          |
| ----- | -------- | ------------ | ------------------------------ |
| `id`  | `string` | **Required** | Path param — Target version ID |

#### Request Example

```bash
curl -X POST "http://localhost:8080/kg/graph/app-001/tenant-001/versions/version-v001/rollback" \
  -H "Authorization: Bearer kgs_ak_x7y8z9..."
```

#### Response

```json
{
  "newVersionId": "version-v003",
  "rolledBackTo": "version-v001",
  "entitiesRestored": 45,
  "edgesRestored": 67
}
```

---

## 8. NATS Event Schemas (DOC 4 §5, §8)

KG Service publish và subscribe events qua NATS JetStream. Các events này được định nghĩa trong DOC 4 Shared Contracts.

### 8.1 Events KG Service Publishes

| Event             | NATS Topic                     | Trigger                        | Subscriber |
| ----------------- | ------------------------------ | ------------------------------ | ---------- |
| `OVERLAY_COMMIT`  | `overlay.committed.{tenantId}` | Overlay committed successfully | Chat Agent |
| `OVERLAY_DISCARD` | `overlay.discarded.{tenantId}` | Overlay discarded              | Chat Agent |

### 8.2 Events KG Service Subscribes To

| Event           | NATS Topic                  | Publisher  | Handler                     |
| --------------- | --------------------------- | ---------- | --------------------------- |
| `SESSION_CLOSE` | `session.close.{sessionId}` | Chat Agent | Auto commit/discard overlay |
| `BUDGET_STOP`   | `budget.stop.{sessionId}`   | Chat Agent | Commit partial overlay      |

### 8.3 OVERLAY_COMMIT Event Schema

Published after successful overlay commit ([§7.4 Overlay Commit](#74-overlay--commit)).

| Field            | Type     | Required     | Mô tả                  |
| ---------------- | -------- | ------------ | ---------------------- |
| `event_id`       | `string` | **Required** | UUID v4                |
| `event_type`     | `string` | **Required** | `"OVERLAY_COMMIT"`     |
| `app_id`         | `string` | **Required** | App ID                 |
| `tenant_id`      | `string` | **Required** | Tenant ID              |
| `session_id`     | `string` | **Required** | Session ID của overlay |
| `overlay_id`     | `string` | **Required** | Overlay ID             |
| `version_id`     | `string` | **Required** | New version created    |
| `entities_added` | `int`    | **Required** | Số entities committed  |
| `edges_added`    | `int`    | **Required** | Số edges committed     |
| `timestamp`      | `string` | **Required** | ISO 8601               |

```json
{
  "event_id": "evt-oc-001",
  "event_type": "OVERLAY_COMMIT",
  "app_id": "app-001",
  "tenant_id": "tenant-001",
  "session_id": "session-xyz-789",
  "overlay_id": "overlay-abc123",
  "version_id": "version-v002",
  "entities_added": 58,
  "edges_added": 124,
  "timestamp": "2026-03-05T11:35:00Z"
}
```

### 8.4 SESSION_CLOSE Event Schema (Subscribed)

KG Service subscribes to this event and auto-handles overlay cleanup.

| Field        | Type     | Required     | Mô tả                   |
| ------------ | -------- | ------------ | ----------------------- |
| `event_type` | `string` | **Required** | `"SESSION_CLOSE"`       |
| `session_id` | `string` | **Required** | Session being closed    |
| `reason`     | `string` | Optional     | `"normal"`, `"timeout"` |
| `timestamp`  | `string` | **Required** | ISO 8601                |

**Handler Logic (DOC 4 §8):**

```
ON SESSION_CLOSE:
  1. Find overlay by session_id
  2. If no overlay → return (no-op)
  3. If overlay has temp entities/edges → commit overlay + publish OVERLAY_COMMIT
  4. If overlay is query-only → discard overlay + publish OVERLAY_DISCARD
```

```json
{
  "event_type": "SESSION_CLOSE",
  "session_id": "session-xyz-789",
  "reason": "normal",
  "timestamp": "2026-03-05T12:00:00Z"
}
```

### 8.5 BUDGET_STOP Event Schema (Subscribed)

| Field             | Type     | Required     | Mô tả                     |
| ----------------- | -------- | ------------ | ------------------------- |
| `event_type`      | `string` | **Required** | `"BUDGET_STOP"`           |
| `session_id`      | `string` | **Required** | Session to stop           |
| `reason`          | `string` | **Required** | `"hard_limit_exceeded"`   |
| `grace_period_ms` | `int`    | **Required** | Grace period before force |
| `timestamp`       | `string` | **Required** | ISO 8601                  |

**Handler Logic:**

```
ON BUDGET_STOP:
  1. Find overlay by session_id
  2. If overlay has partial data → commit with status=PARTIAL + publish OVERLAY_COMMIT
  3. Otherwise → discard overlay
```
