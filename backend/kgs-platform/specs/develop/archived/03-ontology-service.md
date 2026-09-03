# ontology-service — Ontology Management Service (EXTRACT)

> **Strategy:** 📤 EXTRACT  
> **Source:** `kgs-platform/internal/biz/ontology.go` + `ontology_validator.go` + `ontology_sync.go`  
> **Target:** `kgs-platform/cmd/ontology/`  
> **Priority:** P1 — Cần có trước khi graph-service dùng remote validation

---

## 1. Phân Tích Code Hiện Tại

### 1.1 Files Cần Tách Từ Monolith

| File | Size | Mô tả |
|------|------|-------|
| `biz/ontology.go` | 1.8KB | OntologyUsecase — CRUD operations |
| `biz/ontology_validator.go` | 5.5KB | Schema validation logic (gojsonschema) |
| `biz/ontology_sync.go` | ~1KB | Neo4j constraint sync |
| `data/ontology.go` | 2.75KB | PostgreSQL data layer |

**Nhận xét:** Code ontology trong monolith khá nhỏ (~11KB tổng) và focused. Có thể extract trực tiếp.

### 1.2 OntologyValidator (Đã Có — Quan Trọng)

```go
// kgs-platform/internal/biz/ontology_validator.go
type OntologyValidator struct {
    repo      OntologyRepo
    redisCli  *redis.Client
    schemaTTL time.Duration
    log       *log.Helper
}

// ValidateEntity validates node properties against registered schema
func (v *OntologyValidator) ValidateEntity(ctx context.Context, appID, entityType string, properties map[string]any) error {
    // 1. Load schema from Redis cache or PostgreSQL
    // 2. Compile JSON Schema (LRU cache of compiled schemas)
    // 3. Validate properties
    // Returns errors or nil
}

// ValidateEdge validates edge relation type against ontology whitelist
func (v *OntologyValidator) ValidateEdge(ctx context.Context, appID, tenantID, relationType, sourceNodeID, targetNodeID string) error {
    // 1. Load RelationType definition
    // 2. Check source/target entity types match whitelist
    // Returns errors or nil
}
```

---

## 2. Cấu Trúc Service Mới

```
kgs-platform/
├── cmd/
│   └── ontology/
│       └── main.go                    ← Entry point
└── internal/
    └── ontology/
        ├── biz/
        │   ├── ontology.go            ← TÁI SỬ DỤNG từ biz/ontology.go
        │   ├── validator.go           ← TÁI SỬ DỤNG từ biz/ontology_validator.go
        │   └── neo4j_sync.go          ← TÁI SỬ DỤNG từ biz/ontology_sync.go
        ├── data/
        │   ├── ontology_pg.go         ← TÁI SỬ DỤNG từ data/ontology.go
        │   └── models.go              ← EntityType, RelationType models
        └── server/
            └── grpc.go                ← MỚI: gRPC server
```

---

## 3. Data Models (Cần Thêm Mới)

Hiện tại trong monolith, ontology dùng chung PostgreSQL với graph. Cần tách ra:

```go
// internal/ontology/data/models.go

type EntityType struct {
    ID             uint           `gorm:"primaryKey"`
    AppID          string         `gorm:"type:varchar(50);uniqueIndex:idx_app_entity"`
    Name           string         `gorm:"type:varchar(100);uniqueIndex:idx_app_entity"`
    DisplayName    string         `gorm:"type:varchar(200)"`
    Description    string         `gorm:"type:text"`
    IDProperty     string         `gorm:"type:varchar(100)"`  // "req_id"
    Schema         datatypes.JSON `gorm:"type:jsonb;not null"`
    SearchableProps datatypes.JSON `gorm:"type:jsonb"`
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type RelationType struct {
    ID          uint           `gorm:"primaryKey"`
    AppID       string         `gorm:"type:varchar(50);uniqueIndex:idx_app_relation"`
    Name        string         `gorm:"type:varchar(100);uniqueIndex:idx_app_relation"`
    Description string         `gorm:"type:text"`
    Properties  datatypes.JSON `gorm:"type:jsonb"`
    SourceTypes datatypes.JSON `gorm:"type:jsonb;not null"` // ["Requirement"]
    TargetTypes datatypes.JSON `gorm:"type:jsonb;not null"` // ["UseCase"]
    Cardinality string         `gorm:"type:varchar(20)"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

---

## 4. gRPC API

### 4.1 Proto Definition

```protobuf
// api/ontology/v1/ontology.proto
syntax = "proto3";
package ontology.v1;

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

  // Validation (called by graph-service)
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
}
```

### 4.2 Server Implementation

```go
// internal/ontology/server/grpc.go

type OntologyServer struct {
    ontologypb.UnimplementedOntologyServiceServer
    validator *biz.OntologyValidator  // TÁI SỬ DỤNG
    usecase   *biz.OntologyUsecase    // TÁI SỬ DỤNG
    neo4jSync *biz.Neo4jOntologySync  // TÁI SỬ DỤNG
}

func (s *OntologyServer) ValidateNodeProperties(ctx context.Context, req *ontologypb.ValidateNodePropertiesRequest) (*ontologypb.ValidationResult, error) {
    var properties map[string]any
    json.Unmarshal(req.PropertiesJson, &properties)
    
    // TÁI SỬ DỤNG validator logic hoàn toàn
    if err := s.validator.ValidateEntity(ctx, req.AppId, req.EntityTypeName, properties); err != nil {
        return &ontologypb.ValidationResult{
            Valid:  false,
            Errors: []string{err.Error()},
        }, nil
    }
    
    return &ontologypb.ValidationResult{Valid: true}, nil
}

func (s *OntologyServer) CreateEntityType(ctx context.Context, req *ontologypb.CreateEntityTypeRequest) (*ontologypb.EntityType, error) {
    appID, _, err := extractAppContext(ctx)
    if err != nil {
        return nil, err
    }
    
    et := &biz.EntityType{
        AppID:       appID,
        Name:        req.Name,
        DisplayName: req.DisplayName,
        Schema:      req.Schema,
        IDProperty:  req.IdProperty,
    }
    
    created, err := s.usecase.CreateEntityType(ctx, et)
    if err != nil {
        return nil, toGRPCStatus(err)
    }
    
    // Sync Neo4j constraint async
    go s.neo4jSync.CreateConstraint(context.Background(), appID, created.Name, created.IDProperty)
    
    return toProtoEntityType(created), nil
}
```

---

## 5. Cache Strategy (Tái Sử Dụng)

OntologyValidator đã có Redis cache. Giữ nguyên:

```go
// Tái sử dụng từ biz/ontology_validator.go
const (
    schemaCacheTTL    = 5 * time.Minute
    fullOntologyCacheTTL = 2 * time.Minute
)

func (v *OntologyValidator) cacheKey(appID, typeName string) string {
    return fmt.Sprintf("ontology:%s:%s", appID, typeName)
}
```

---

## 6. NATS Events

| Event | Topic | Trigger |
|-------|-------|---------| 
| EntityTypeCreated | `ontology.entity_type.created` | CreateEntityType |
| EntityTypeUpdated | `ontology.entity_type.updated` | UpdateEntityType (invalidate cache) |
| EntityTypeDeleted | `ontology.entity_type.deleted` | DeleteEntityType |
| RelationTypeCreated | `ontology.relation_type.created` | CreateRelationType |
| RelationTypeDeleted | `ontology.relation_type.deleted` | DeleteRelationType |

---

## 7. Database Migrations

```sql
-- migrations/001_ontology_init.sql

CREATE TABLE IF NOT EXISTS entity_types (
    id              SERIAL PRIMARY KEY,
    app_id          VARCHAR(50) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    display_name    VARCHAR(200),
    description     TEXT,
    id_property     VARCHAR(100),
    schema_json     JSONB NOT NULL DEFAULT '{}',
    searchable_props JSONB DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE(app_id, name)
);

CREATE TABLE IF NOT EXISTS relation_types (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    properties   JSONB DEFAULT '{}',
    source_types JSONB NOT NULL DEFAULT '[]',
    target_types JSONB NOT NULL DEFAULT '[]',
    cardinality  VARCHAR(20) DEFAULT 'MANY_TO_MANY',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    UNIQUE(app_id, name)
);
```

---

## 8. Ước Tính Effort

| Task | Effort |
|------|--------|
| Tách biz/ontology*.go → internal/ontology/biz/ | 0.5 ngày |
| Tách data/ontology.go → internal/ontology/data/ | 0.5 ngày |
| Proto definition + code gen | 0.5 ngày |
| gRPC server implementation | 1 ngày |
| Database migrations | 0.5 ngày |
| NATS events | 0.5 ngày |
| Unit tests | 1 ngày |
| **Total** | **4.5 ngày** |

---

## 9. Lý Do Phải Tách Riêng

1. **Graph-service cần gọi remote** — Không thể dùng local OntologyValidator khi graph-service là service riêng
2. **Schema ownership** — Ontology là domain riêng, không nên mix với graph operations
3. **Scale độc lập** — Schema validation write thường ít hơn graph writes → không cần scale chung
4. **Schema evolution** — Khi cần thay đổi JSON Schema validation library, chỉ cần update ontology-service
5. **API access** — Clients cần đọc/write ontology qua Gateway, cần endpoint riêng
