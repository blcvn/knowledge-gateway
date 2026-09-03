# ontology-service — Ontology Management Service

> **Role:** Quản lý schema graph (ontology) của từng tenant — định nghĩa EntityTypes, RelationTypes, và enforce schema validation.

---

## 1. Trách Nhiệm (Single Responsibility)

`ontology-service` là **nguồn sự thật duy nhất** cho schema graph của mỗi app:
- **EntityType CRUD**: Định nghĩa các loại node (Requirement, UseCase, ...)
- **RelationType CRUD**: Định nghĩa các loại edge (HAS_USECASE, TRANSFER_TO, ...)
- **Schema Validation**: Validate properties của node/edge theo JSON Schema
- **Relation Whitelist**: Enforce `source_type → target_type` pairs hợp lệ
- **Neo4j Constraint Sync**: Tự động tạo/xóa constraints trong Neo4j khi ontology thay đổi
- **Schema Cache**: Cache ontology trong Redis để graph-service validate nhanh

---

## 2. Kiến Trúc Nội Tại

```
┌──────────────────────────────────────────────────────────────────┐
│                      ontology-service                             │
│                                                                  │
│  gRPC Server (port 9002)                                         │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                  OntologyServiceServer                      │  │
│  │                                                            │  │
│  │  CreateEntityType()    GetEntityType()    ListEntityTypes() │  │
│  │  UpdateEntityType()    DeleteEntityType()                  │  │
│  │  CreateRelationType()  GetRelationType()  ListRelationTypes()│  │
│  │  UpdateRelationType()  DeleteRelationType()                │  │
│  │  GetFullOntology()                                         │  │
│  │  ValidateNodeProperties()   [called by graph-service]      │  │
│  │  ValidateEdgeRelation()     [called by graph-service]      │  │
│  └────────────────────────────────────────────────────────────┘  │
│                               │                                   │
│  ┌────────────────────────────▼──────────────────────────────┐   │
│  │                   Business Logic                            │   │
│  │                                                            │   │
│  │  OntologyUsecase                                           │   │
│  │  ├── JSON Schema compiler & validator (gojsonschema)       │   │
│  │  ├── Relation source/target type checker                   │   │
│  │  ├── Cache invalidation on schema change                   │   │
│  │  └── OntologyConstraintSyncer → Neo4j constraints          │   │
│  └────────────────────────────────────────────────────────────┘   │
│                               │                                    │
│  ┌────────────────────────────▼──────────────────────────────┐    │
│  │         Storage Layer                                       │    │
│  │  PostgreSQL: entity_types, relation_types                  │    │
│  │  Redis:      cache(ontology:{app_id}:{type_name}, TTL=5min)│    │
│  │  Neo4j:      constraints sync (write-only)                 │    │
│  └────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. Data Models

### 3.1 EntityType

```go
type EntityType struct {
    ID          uint           `gorm:"primaryKey"`
    AppID       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_entity"`
    Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_entity"`
    // Example: "Requirement", "UseCase", "TestCase"
    DisplayName string         `gorm:"type:varchar(200)"`
    Description string         `gorm:"type:text"`
    IDProperty  string         `gorm:"type:varchar(100)"` // Property used as unique ID, e.g. "req_id"
    Schema      datatypes.JSON `gorm:"type:jsonb;not null"`
    // JSON Schema definition:
    // {
    //   "required": ["req_id", "title"],
    //   "properties": {
    //     "req_id": { "type": "string", "pattern": "^REQ-..." },
    //     "status": { "type": "string", "enum": ["DRAFT", "APPROVED"] }
    //   }
    // }
    SearchableProps datatypes.JSON `gorm:"type:jsonb"` // ["title", "description"]
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      gorm.DeletedAt `gorm:"index"`
}
```

### 3.2 RelationType

```go
type RelationType struct {
    ID          uint           `gorm:"primaryKey"`
    AppID       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_relation"`
    Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_relation"`
    // Example: "HAS_USECASE", "TRANSFER_TO", "IMPLEMENTS"
    Description string         `gorm:"type:text"`
    Properties  datatypes.JSON `gorm:"type:jsonb"`
    // JSON Schema for edge properties
    SourceTypes datatypes.JSON `gorm:"type:jsonb;not null"`
    // ["Requirement"] — list of valid source EntityType names
    TargetTypes datatypes.JSON `gorm:"type:jsonb;not null"`
    // ["UseCase"] — list of valid target EntityType names
    Cardinality string         `gorm:"type:varchar(20)"`
    // ONE_TO_ONE | ONE_TO_MANY | MANY_TO_MANY
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

---

## 4. gRPC API

```protobuf
service OntologyService {
  // EntityType CRUD
  rpc CreateEntityType(CreateEntityTypeRequest) returns (EntityType);
  rpc GetEntityType(GetEntityTypeRequest) returns (EntityType);
  rpc ListEntityTypes(ListEntityTypesRequest) returns (ListEntityTypesResponse);
  rpc UpdateEntityType(UpdateEntityTypeRequest) returns (EntityType);
  rpc DeleteEntityType(DeleteEntityTypeRequest) returns (google.protobuf.Empty);

  // RelationType CRUD
  rpc CreateRelationType(CreateRelationTypeRequest) returns (RelationType);
  rpc GetRelationType(GetRelationTypeRequest) returns (RelationType);
  rpc ListRelationTypes(ListRelationTypesRequest) returns (ListRelationTypesResponse);
  rpc UpdateRelationType(UpdateRelationTypeRequest) returns (RelationType);
  rpc DeleteRelationType(DeleteRelationTypeRequest) returns (google.protobuf.Empty);

  // Full Ontology
  rpc GetFullOntology(GetFullOntologyRequest) returns (FullOntologyResponse);
  // Returns: {entity_types: [...], relation_types: [...]}

  // Validation (called internally by graph-service)
  rpc ValidateNodeProperties(ValidateNodePropertiesRequest) returns (ValidationResult);
  rpc ValidateEdgeRelation(ValidateEdgeRelationRequest) returns (ValidationResult);
}

message ValidateNodePropertiesRequest {
  string app_id = 1;
  string entity_type_name = 2;
  bytes properties_json = 3;
}

message ValidateEdgeRelationRequest {
  string app_id = 1;
  string relation_type_name = 2;
  string source_entity_type = 3;
  string target_entity_type = 4;
  bytes properties_json = 5;
}

message ValidationResult {
  bool valid = 1;
  repeated string errors = 2;
  // e.g. "req_id: required property missing"
  // e.g. "source type 'Contract' not allowed for relation HAS_USECASE"
}
```

---

## 5. Schema Validation Pipeline

Khi `graph-service` cần validate một write operation, nó gọi `ontology-service`:

```
graph-service receives CreateNode(app_id="ba_agent", label="Requirement", props={...})
    │
    ▼
Call: ontology-service.ValidateNodeProperties(app_id, "Requirement", props)
    │
    ▼
ontology-service:
  1. Cache lookup: Redis key "ontology:ba_agent:Requirement" (TTL=5min)
     └─ Cache miss → Load from PostgreSQL, store in cache
  2. JSON Schema Validation:
     - Compile schema if not compiled (LRU cache of compiled schemas)
     - Validate props against schema
     - Return errors if fail
  3. Return: ValidationResult{valid: true/false, errors: [...]}
    │
    ▼
graph-service:
  - If invalid → return HTTP 422 Unprocessable Entity
  - If valid → proceed to write
```

---

## 6. Ontology Validation Rules

### EntityType Constraints

| Rule | Mô tả | Error |
|------|-------|-------|
| Unique name per app | Không tạo 2 EntityType cùng tên | `ENTITY_TYPE_EXISTS` |
| Required properties | `id_property` phải có trong schema | `SCHEMA_MISSING_ID_PROPERTY` |
| Schema valid JSON Schema | Schema phải là JSON Schema hợp lệ | `INVALID_JSON_SCHEMA` |
| No deletion nếu còn nodes | Không xóa EntityType nếu còn nodes trong graph | `ENTITY_TYPE_HAS_NODES` |

### RelationType Constraints

| Rule | Mô tả | Error |
|------|-------|-------|
| Valid source/target types | source_types và target_types phải là EntityType names đã tồn tại | `INVALID_ENTITY_TYPE_REF` |
| Unique name per app | Không tạo 2 RelationType cùng tên | `RELATION_TYPE_EXISTS` |
| No deletion nếu còn edges | Không xóa RelationType nếu còn edges trong graph | `RELATION_TYPE_HAS_EDGES` |

---

## 7. Neo4j Constraint Sync

Khi EntityType được tạo/xóa, `ontology-service` sync constraints xuống Neo4j:

```go
// Khi tạo EntityType "Requirement" cho app "ba_agent":
// 1. Tạo Neo4j uniqueness constraint:
CREATE CONSTRAINT IF NOT EXISTS
FOR (n:ba_agent__Requirement)
REQUIRE n.req_id IS UNIQUE

// 2. Tạo Neo4j node existence constraint (nếu có required props):
CREATE CONSTRAINT IF NOT EXISTS
FOR (n:ba_agent__Requirement)
REQUIRE n.title IS NOT NULL
```

---

## 8. Cache Strategy

| Cache Key | Value | TTL | Invalidation |
|-----------|-------|-----|-------------|
| `ontology:{app_id}:{entity_type_name}` | EntityType JSON | 5 min | Khi EntityType được update/delete |
| `ontology:{app_id}:relations:{relation_type_name}` | RelationType JSON | 5 min | Khi RelationType được update/delete |
| `ontology:{app_id}:full` | Full ontology JSON | 2 min | Khi bất kỳ schema nào thay đổi |

---

## 9. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope |
|--------|------|-------|
| POST | `/v1/ontology/entity-types` | `ontology:write` |
| GET | `/v1/ontology/entity-types` | `ontology:read` |
| GET | `/v1/ontology/entity-types/:name` | `ontology:read` |
| PUT | `/v1/ontology/entity-types/:name` | `ontology:write` |
| DELETE | `/v1/ontology/entity-types/:name` | `ontology:write` |
| POST | `/v1/ontology/relation-types` | `ontology:write` |
| GET | `/v1/ontology/relation-types` | `ontology:read` |
| GET | `/v1/ontology/relation-types/:name` | `ontology:read` |
| PUT | `/v1/ontology/relation-types/:name` | `ontology:write` |
| DELETE | `/v1/ontology/relation-types/:name` | `ontology:write` |
| GET | `/v1/ontology` | `ontology:read` |

---

## 10. Ví Dụ: Đăng Ký Ontology BA Agent System

```json
// POST /v1/ontology/entity-types
// x-app-id: ba_agent (injected by gateway)
{
  "name": "Requirement",
  "display_name": "Software Requirement",
  "id_property": "req_id",
  "description": "IEEE 830 software requirement",
  "searchable_props": ["title", "description"],
  "schema": {
    "required": ["req_id", "title", "type", "priority", "status", "version"],
    "properties": {
      "req_id":   { "type": "string", "pattern": "^REQ-[A-Z]+-[0-9]{3}$" },
      "title":    { "type": "string", "maxLength": 200 },
      "type":     { "type": "string", "enum": ["FUNCTIONAL", "NON_FUNCTIONAL", "CONSTRAINT"] },
      "priority": { "type": "string", "enum": ["MUST", "SHOULD", "COULD", "WONT"] },
      "status":   { "type": "string", "enum": ["DRAFT", "APPROVED", "DEPRECATED"] },
      "version":  { "type": "string" }
    }
  }
}

// POST /v1/ontology/relation-types
{
  "name": "HAS_USECASE",
  "description": "Links Requirement to its derived UseCases",
  "source_types": ["Requirement"],
  "target_types": ["UseCase"],
  "cardinality": "ONE_TO_MANY",
  "properties": {
    "properties": {
      "confidence":    { "type": "number", "minimum": 0, "maximum": 1 },
      "impact_weight": { "type": "number", "minimum": 0, "maximum": 1 },
      "source":        { "type": "string", "enum": ["manual", "agent", "rule_engine"] }
    }
  }
}
```

---

## 11. Configuration

```yaml
# configs/ontology.yaml
ontology_service:
  grpc_port: 9002

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_ontology"

  redis:
    addr: redis:6379
    schema_cache_ttl: 5m

  neo4j:
    uri: bolt://neo4j:7687
    username: neo4j
    password: secret
    constraint_sync_enabled: true

  observability:
    metrics_port: 9092
```

---

## 12. NATS Events Published

| Event | Topic | Trigger |
|-------|-------|---------|
| EntityTypeCreated | `ontology.entity_type.created` | Sau khi tạo thành công |
| EntityTypeUpdated | `ontology.entity_type.updated` | Sau khi update schema |
| EntityTypeDeleted | `ontology.entity_type.deleted` | Sau khi xóa |
| RelationTypeCreated | `ontology.relation_type.created` | Sau khi tạo thành công |
| RelationTypeDeleted | `ontology.relation_type.deleted` | Sau khi xóa |
