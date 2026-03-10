# Phương án tích hợp Ontology vào ai-kg-service (kgs-platform)

> **Ngày:** 10/03/2026
> **Tham chiếu:**
> - [Tài liệu Ontology](ai_kg_service_ontology.md)
> - [Báo cáo đối chiếu Ontology](repo_ontology_ai_kg_service.md)
> - Source code `kgs-platform/`
> **Mục tiêu:** Chuyển Ontology từ passive registry thành active enforcement — wire đầy đủ 5 vai trò: Schema Registry, Validation Gate, Constraint Enforcer, Projection Rules Store, Vocabulary Provider.

---

## 1. Phân tích hiện trạng

### 1.1 Gaps xác định từ báo cáo đối chiếu

| Gap | Mức | Mô tả |
|-----|-----|-------|
| **G1: OntologyService in-memory** | P0 | `service/ontology.go` lưu map trong bộ nhớ, mất khi restart. Không dùng GORM models `EntityType`/`RelationType` đã có |
| **G2: Validation Gate không wire** | P0 | `biz/graph.go:CreateNode` KHÔNG gọi ontology validate trước khi write Neo4j |
| **G3: Constraint Enforcer không wire** | P0 | `biz/graph.go:CreateEdge` có `TODO: Validate relation whitelist` (line 174) nhưng chưa implement |
| **G4: JSON Schema validation thiếu** | P1 | `EntityType.Schema` (jsonb) tồn tại nhưng không dùng để validate properties |
| **G5: Projection không liên kết Ontology** | P1 | `ViewDefinition` lưu `AllowedEntityTypes` bằng tay, không sync từ Ontology |
| **G6: ontologyRepo không wire** | P0 | `data/ontology.go` có `GetEntityType/GetRelationType` (Redis cache + Postgres) nhưng không inject vào bất kỳ usecase nào |
| **G7: ViewResolver không dùng** | P2 | `wire_gen.go:82-83`: `viewResolver := biz.NewViewResolver(engine2)` rồi `_ = viewResolver` |

### 1.2 Assets có sẵn (có thể tái sử dụng)

| Asset | File | Trạng thái | Tái sử dụng |
|-------|------|-----------|-------------|
| `EntityType` GORM model | `biz/ontology.go:11-25` | Đầy đủ: ID, AppID, TenantID, Name, Schema (jsonb) | 100% |
| `RelationType` GORM model | `biz/ontology.go:28-44` | Đầy đủ: SourceTypes, TargetTypes (jsonb arrays) | 100% |
| `ontologyRepo` (data layer) | `data/ontology.go` | GetEntityType + GetRelationType với Redis cache 5 phút | 100% |
| `ErrSchemaInvalid` error | `biz/errors.go:21-23` | Đã định nghĩa, sẵn sàng dùng | 100% |
| `OntologySyncManager` | `biz/ontology_sync.go` | Stub — chỉ log, không làm gì | Thay thế |
| `OntologyService` (service) | `service/ontology.go` | In-memory, cần chuyển sang Postgres | Rewrite |
| `ViewResolver` | `biz/view_resolver.go` | Hoạt động nhưng `_ = viewResolver` trong wire | Wire lại |

---

## 2. Kiến trúc tổng thể sau tích hợp

### 2.1 Luồng Write (sau tích hợp)

```
Client → CreateNode(label="Requirement", properties={...})
     │
     ▼
GraphService.CreateNode()
     │
     ▼
GraphUsecase.CreateNode()
     │
     ├── 1. Overlay check (giữ nguyên)
     │
     ├── 2. Lock (giữ nguyên)
     │
     ├── 3. OPA Policy Check (giữ nguyên)
     │
     ├── 4. ★ NEW: Ontology Validation ★
     │   │
     │   ├── ontologyValidator.ValidateEntity(appID, label, properties)
     │   │   ├── Lookup EntityType bằng ontologyRepo.GetEntityType(appID, label)
     │   │   │   (Redis cache → Postgres fallback)
     │   │   ├── Nếu không tìm thấy → 400 ERR_SCHEMA_INVALID "unknown entity type"
     │   │   ├── Nếu EntityType.Schema != nil → validate properties against JSON Schema
     │   │   └── Nếu schema mismatch → 400 ERR_SCHEMA_INVALID "properties validation failed"
     │   │
     │   └── Return nil (passed)
     │
     ├── 5. Data Persistence → Neo4j (giữ nguyên)
     │
     └── 6. Event + Observability (giữ nguyên)
```

```
Client → CreateEdge(relationType="DEPENDS_ON", sourceID, targetID)
     │
     ▼
GraphUsecase.CreateEdge()
     │
     ├── 1. Overlay check (giữ nguyên)
     │
     ├── 2. Dual lock (giữ nguyên)
     │
     ├── 3. ★ NEW: Ontology Constraint Validation ★
     │   │
     │   ├── ontologyValidator.ValidateEdge(appID, tenantID, relationType, sourceID, targetID)
     │   │   ├── Lookup RelationType bằng ontologyRepo.GetRelationType(appID, relationType)
     │   │   ├── Nếu không tìm thấy → 400 ERR_SCHEMA_INVALID "unknown relation type"
     │   │   ├── Lookup source node label từ repo.GetNode(sourceID) → extract label
     │   │   ├── Lookup target node label từ repo.GetNode(targetID) → extract label
     │   │   ├── Check sourceLabel ∈ RelationType.SourceTypes
     │   │   ├── Check targetLabel ∈ RelationType.TargetTypes
     │   │   └── Nếu mismatch → 400 ERR_SCHEMA_INVALID "source/target type not allowed"
     │   │
     │   └── Return nil (passed)
     │
     └── 4. Data Persistence → Neo4j (giữ nguyên)
```

### 2.2 Luồng Projection (sau tích hợp)

```
Client → GetContext(role="BA", ...)
     │
     ▼
GraphService → query result from Neo4j
     │
     ├── Lookup ViewDefinition for role "BA"
     │   │
     │   ├── Nếu có ViewDefinition → dùng ViewDefinition (ưu tiên)
     │   │
     │   └── Nếu không có → ★ NEW: fallback sang Ontology ★
     │       └── Lấy tất cả EntityTypes cho appID
     │       └── Build danh sách AllowedEntityTypes từ Ontology
     │
     └── Apply Projection → filter nodes/edges + PII mask
```

---

## 3. Nguyên tắc thiết kế

| # | Nguyên tắc | Mô tả |
|---|-----------|-------|
| 1 | **Additive-only** | KHÔNG sửa logic hiện có trong `CreateNode`/`CreateEdge`. Thêm validation step mới vào giữa |
| 2 | **Feature flag** | `ontology.validation_enabled` (default `false`). Khi `false`, validation bị skip |
| 3 | **Soft mode trước** | Giai đoạn đầu: validation chỉ log warning, KHÔNG reject. Chuyển sang hard reject khi ổn định |
| 4 | **Tái sử dụng tối đa** | Dùng `ontologyRepo`, `EntityType`, `RelationType`, `ErrSchemaInvalid` đã có |
| 5 | **Cache-first** | Ontology lookup qua Redis cache (5 phút TTL) — không tạo thêm latency cho write path |
| 6 | **Graceful degradation** | Nếu ontology lookup fail (Redis/Postgres down), bypass validation — không block write |

---

## 4. Chi tiết thiết kế

### 4.1 Config — Feature Flags

```protobuf
// internal/conf/conf.proto — THÊM message Ontology trong Data

message Data {
  // ... existing fields KHÔNG ĐỔI ...

  message OntologyConfig {
    bool validation_enabled = 1;     // master switch — default false
    bool strict_mode = 2;            // true = reject invalid, false = log warning only
    bool schema_validation = 3;      // enable JSON Schema validation for properties
    bool edge_constraint_check = 4;  // enable source/target type checking for edges
    bool sync_projection = 5;        // auto-sync ViewDefinitions from Ontology
  }
  OntologyConfig ontology = 9;  // next available field number
}
```

```yaml
# configs/config.yaml
data:
  ontology:
    validation_enabled: false    # default OFF — opt-in
    strict_mode: false           # log-only ban đầu
    schema_validation: true
    edge_constraint_check: true
    sync_projection: false       # Phase 2
```

### 4.2 OntologyValidator — Biz Layer (NEW)

```go
// internal/biz/ontology_validator.go — NEW

package biz

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/santhosh-tekuri/jsonschema/v5"
)

// OntologyRepo defines the interface for ontology data access
type OntologyRepo interface {
    GetEntityType(ctx context.Context, appID, name string) (*EntityType, error)
    GetRelationType(ctx context.Context, appID, name string) (*RelationType, error)
}

// OntologyValidatorConfig holds feature flags for ontology validation
type OntologyValidatorConfig struct {
    Enabled             bool
    StrictMode          bool // true = reject, false = log warning only
    SchemaValidation    bool
    EdgeConstraintCheck bool
}

// OntologyValidator validates graph writes against ontology definitions
type OntologyValidator struct {
    repo   OntologyRepo
    graph  GraphRepo  // for GetNode — lookup source/target labels
    config OntologyValidatorConfig
    log    *log.Helper
}

func NewOntologyValidator(
    repo OntologyRepo,
    graph GraphRepo,
    config OntologyValidatorConfig,
    logger log.Logger,
) *OntologyValidator {
    return &OntologyValidator{
        repo:   repo,
        graph:  graph,
        config: config,
        log:    log.NewHelper(logger),
    }
}

// ValidateEntity checks:
// 1. EntityType exists in ontology registry
// 2. Properties match JSON Schema (if schema_validation enabled)
func (v *OntologyValidator) ValidateEntity(ctx context.Context, appID, label string, properties map[string]any) error {
    if v == nil || !v.config.Enabled {
        return nil // validation disabled — pass through
    }
    if v.repo == nil {
        return nil // no ontology repo — graceful degradation
    }

    entityType, err := v.repo.GetEntityType(ctx, appID, label)
    if err != nil {
        // Ontology lookup failed — graceful degradation
        v.log.Warnf("ontology lookup failed for entity type %q: %v (bypassing validation)", label, err)
        return nil
    }
    if entityType == nil {
        return v.handleViolation(ctx, "unknown entity type",
            map[string]string{"label": label, "app_id": appID})
    }

    // JSON Schema validation
    if v.config.SchemaValidation && len(entityType.Schema) > 0 {
        if err := v.validateJSONSchema(entityType.Schema, properties); err != nil {
            return v.handleViolation(ctx, fmt.Sprintf("properties validation failed: %v", err),
                map[string]string{"label": label, "app_id": appID})
        }
    }

    return nil
}

// ValidateEdge checks:
// 1. RelationType exists in ontology registry
// 2. Source node label ∈ RelationType.SourceTypes
// 3. Target node label ∈ RelationType.TargetTypes
func (v *OntologyValidator) ValidateEdge(
    ctx context.Context,
    appID, tenantID, relationType, sourceNodeID, targetNodeID string,
) error {
    if v == nil || !v.config.Enabled {
        return nil
    }
    if v.repo == nil {
        return nil
    }

    // 1. Lookup RelationType
    relType, err := v.repo.GetRelationType(ctx, appID, relationType)
    if err != nil {
        v.log.Warnf("ontology lookup failed for relation type %q: %v (bypassing validation)", relationType, err)
        return nil
    }
    if relType == nil {
        return v.handleViolation(ctx, "unknown relation type",
            map[string]string{"relation_type": relationType, "app_id": appID})
    }

    // 2. Check source/target constraints (if enabled)
    if !v.config.EdgeConstraintCheck {
        return nil
    }

    sourceTypes := decodeJSONArray(relType.SourceTypes)
    targetTypes := decodeJSONArray(relType.TargetTypes)

    // If SourceTypes/TargetTypes are empty, allow any — no constraint defined
    if len(sourceTypes) == 0 && len(targetTypes) == 0 {
        return nil
    }

    // Lookup source node to get its label
    if len(sourceTypes) > 0 {
        sourceNode, err := v.graph.GetNode(ctx, appID, tenantID, sourceNodeID)
        if err != nil {
            v.log.Warnf("cannot lookup source node %s for edge validation: %v", sourceNodeID, err)
            return nil // graceful degradation
        }
        sourceLabel, _ := sourceNode["label"].(string)
        if sourceLabel == "" {
            sourceLabel = extractLabelFromNode(sourceNode)
        }
        if !containsIgnoreCase(sourceTypes, sourceLabel) {
            return v.handleViolation(ctx,
                fmt.Sprintf("source node type %q not allowed for relation %q (allowed: %v)",
                    sourceLabel, relationType, sourceTypes),
                map[string]string{
                    "relation_type": relationType,
                    "source_label":  sourceLabel,
                    "source_id":     sourceNodeID,
                })
        }
    }

    // Lookup target node to get its label
    if len(targetTypes) > 0 {
        targetNode, err := v.graph.GetNode(ctx, appID, tenantID, targetNodeID)
        if err != nil {
            v.log.Warnf("cannot lookup target node %s for edge validation: %v", targetNodeID, err)
            return nil
        }
        targetLabel, _ := targetNode["label"].(string)
        if targetLabel == "" {
            targetLabel = extractLabelFromNode(targetNode)
        }
        if !containsIgnoreCase(targetTypes, targetLabel) {
            return v.handleViolation(ctx,
                fmt.Sprintf("target node type %q not allowed for relation %q (allowed: %v)",
                    targetLabel, relationType, targetTypes),
                map[string]string{
                    "relation_type": relationType,
                    "target_label":  targetLabel,
                    "target_id":     targetNodeID,
                })
        }
    }

    return nil
}

// handleViolation either rejects (strict mode) or logs a warning (soft mode)
func (v *OntologyValidator) handleViolation(ctx context.Context, message string, metadata map[string]string) error {
    if v.config.StrictMode {
        return ErrSchemaInvalid(message, metadata)
    }
    // Soft mode — log warning, do NOT reject
    v.log.Warnf("[ontology-soft] %s metadata=%v", message, metadata)
    return nil
}

// validateJSONSchema validates properties against a JSON Schema stored in EntityType.Schema
func (v *OntologyValidator) validateJSONSchema(schema json.RawMessage, properties map[string]any) error {
    if len(schema) == 0 || string(schema) == "{}" || string(schema) == "null" {
        return nil // no schema defined — skip
    }

    compiler := jsonschema.NewCompiler()
    if err := compiler.AddResource("entity_schema.json", strings.NewReader(string(schema))); err != nil {
        v.log.Warnf("failed to compile entity JSON Schema: %v", err)
        return nil // schema parse failure → graceful degradation
    }

    compiled, err := compiler.Compile("entity_schema.json")
    if err != nil {
        v.log.Warnf("failed to compile entity JSON Schema: %v", err)
        return nil
    }

    if err := compiled.Validate(properties); err != nil {
        return err
    }
    return nil
}

// --- helpers ---

func decodeJSONArray(raw json.RawMessage) []string {
    if len(raw) == 0 {
        return nil
    }
    var out []string
    _ = json.Unmarshal(raw, &out)
    return out
}

func containsIgnoreCase(list []string, value string) bool {
    lower := strings.ToLower(strings.TrimSpace(value))
    for _, item := range list {
        if strings.ToLower(strings.TrimSpace(item)) == lower {
            return true
        }
    }
    return false
}

func extractLabelFromNode(node map[string]any) string {
    // Neo4j nodes often store labels in "labels" array
    if labels, ok := node["labels"].([]any); ok && len(labels) > 0 {
        for _, l := range labels {
            if s, ok := l.(string); ok && s != "Entity" {
                return s
            }
        }
    }
    return ""
}
```

### 4.3 OntologyService — Chuyển sang Postgres persistence

```go
// internal/service/ontology.go — REWRITE (giữ nguyên API contract)

package service

import (
    "context"

    pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/ontology/v1"
    "github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"

    "gorm.io/datatypes"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

type OntologyService struct {
    pb.UnimplementedOntologyServer
    db *gorm.DB
}

func NewOntologyService(db *gorm.DB) *OntologyService {
    return &OntologyService{db: db}
}

func (s *OntologyService) CreateEntityType(ctx context.Context, req *pb.CreateEntityTypeRequest) (*pb.CreateEntityTypeReply, error) {
    appCtx, err := getAppContext(ctx)
    if err != nil {
        return nil, err
    }
    if req.GetName() == "" {
        return &pb.CreateEntityTypeReply{Status: "INVALID"}, nil
    }

    entity := biz.EntityType{
        AppID:    appCtx.AppID,
        TenantID: appCtx.TenantID,
        Name:     req.GetName(),
        Description: req.GetDescription(),
        Schema:   datatypes.JSON(req.GetSchema()),
    }

    // Upsert — create or update
    result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "app_id"}, {Name: "tenant_id"}, {Name: "name"}},
        DoUpdates: clause.AssignmentColumns([]string{"description", "schema", "updated_at"}),
    }).Create(&entity)

    if result.Error != nil {
        return nil, result.Error
    }

    status := "CREATED"
    if result.RowsAffected == 0 {
        status = "EXISTS"
    }

    return &pb.CreateEntityTypeReply{
        Id:     uint32(entity.ID),
        Name:   entity.Name,
        Status: status,
    }, nil
}

func (s *OntologyService) CreateRelationType(ctx context.Context, req *pb.CreateRelationTypeRequest) (*pb.CreateRelationTypeReply, error) {
    appCtx, err := getAppContext(ctx)
    if err != nil {
        return nil, err
    }
    if req.GetName() == "" {
        return &pb.CreateRelationTypeReply{Status: "INVALID"}, nil
    }

    sourceTypesJSON, _ := json.Marshal(req.GetSourceTypes())
    targetTypesJSON, _ := json.Marshal(req.GetTargetTypes())

    relation := biz.RelationType{
        AppID:       appCtx.AppID,
        TenantID:    appCtx.TenantID,
        Name:        req.GetName(),
        Description: req.GetDescription(),
        Properties:  datatypes.JSON(req.GetPropertiesSchema()),
        SourceTypes: datatypes.JSON(sourceTypesJSON),
        TargetTypes: datatypes.JSON(targetTypesJSON),
    }

    result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "app_id"}, {Name: "tenant_id"}, {Name: "name"}},
        DoUpdates: clause.AssignmentColumns([]string{"description", "properties", "source_types", "target_types", "updated_at"}),
    }).Create(&relation)

    if result.Error != nil {
        return nil, result.Error
    }

    status := "CREATED"
    if result.RowsAffected == 0 {
        status = "EXISTS"
    }

    return &pb.CreateRelationTypeReply{
        Id:     uint32(relation.ID),
        Name:   relation.Name,
        Status: status,
    }, nil
}

func (s *OntologyService) ListEntityTypes(ctx context.Context, req *pb.ListEntityTypesRequest) (*pb.ListEntityTypesReply, error) {
    appCtx, err := getAppContext(ctx)
    if err != nil {
        return nil, err
    }

    var entities []biz.EntityType
    if err := s.db.WithContext(ctx).
        Where("app_id = ?", appCtx.AppID).
        Order("name ASC").
        Find(&entities).Error; err != nil {
        return nil, err
    }

    out := make([]*pb.EntityTypeInfo, 0, len(entities))
    for _, e := range entities {
        out = append(out, &pb.EntityTypeInfo{
            Id:     uint32(e.ID),
            Name:   e.Name,
            Schema: string(e.Schema),
        })
    }
    return &pb.ListEntityTypesReply{Entities: out}, nil
}

func (s *OntologyService) ListRelationTypes(ctx context.Context, req *pb.ListRelationTypesRequest) (*pb.ListRelationTypesReply, error) {
    appCtx, err := getAppContext(ctx)
    if err != nil {
        return nil, err
    }

    var relations []biz.RelationType
    if err := s.db.WithContext(ctx).
        Where("app_id = ?", appCtx.AppID).
        Order("name ASC").
        Find(&relations).Error; err != nil {
        return nil, err
    }

    out := make([]*pb.RelationTypeInfo, 0, len(relations))
    for _, r := range relations {
        out = append(out, &pb.RelationTypeInfo{
            Id:               uint32(r.ID),
            Name:             r.Name,
            PropertiesSchema: string(r.Properties),
            SourceTypes:      decodeJSONStringSlice(r.SourceTypes),
            TargetTypes:      decodeJSONStringSlice(r.TargetTypes),
        })
    }
    return &pb.ListRelationTypesReply{Relations: out}, nil
}

func decodeJSONStringSlice(raw datatypes.JSON) []string {
    if len(raw) == 0 {
        return nil
    }
    var out []string
    _ = json.Unmarshal(raw, &out)
    return out
}
```

### 4.4 Tích hợp vào GraphUsecase — Wire Validation

```go
// internal/biz/graph.go — MODIFIED (thêm ontologyValidator, KHÔNG sửa logic cũ)

type GraphUsecase struct {
    repo        GraphRepo
    ontology    *OntologySyncManager     // existing — kept
    validator   *OntologyValidator       // ★ NEW — ontology validation
    planner     *QueryPlanner
    opa         *OPAClient
    redisCli    *redis.Client
    lockMgr     lock.LockManager
    nodeLockTTL time.Duration
    overlay     OverlayDeltaWriter
    log         *log.Helper
}

func NewGraphUsecase(
    repo GraphRepo,
    planner *QueryPlanner,
    opa *OPAClient,
    redisCli *redis.Client,
    lockMgr lock.LockManager,
    overlay OverlayDeltaWriter,
    validator *OntologyValidator,  // ★ NEW parameter
    logger log.Logger,
) *GraphUsecase {
    return &GraphUsecase{
        repo:        repo,
        planner:     planner,
        opa:         opa,
        redisCli:    redisCli,
        lockMgr:     lockMgr,
        nodeLockTTL: lockTTLFromEnv(),
        overlay:     overlay,
        validator:   validator,   // ★ NEW
        log:         log.NewHelper(logger),
    }
}
```

#### CreateNode — thêm validation step

```go
func (uc *GraphUsecase) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
    // ... existing: nil check, id generation, overlay check (lines 72-88 KHÔNG ĐỔI) ...

    // ... existing: lock (lines 90-95 KHÔNG ĐỔI) ...

    // 1. OPA Policy Check (KHÔNG ĐỔI — lines 97-111)
    // ...

    // ★ NEW — 1.5. Ontology Validation (giữa OPA và Data Persistence)
    if err := uc.validator.ValidateEntity(lockCtx, appID, label, properties); err != nil {
        observability.ObserveEntityWrite("create_node", err)
        return nil, err
    }

    // 2. Data Persistence (KHÔNG ĐỔI — lines 113-118)
    // ...

    // 3. Trigger Event (KHÔNG ĐỔI — lines 120-131)
    // ...
}
```

#### CreateEdge — thay thế TODO bằng validation thực

```go
func (uc *GraphUsecase) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
    // ... existing: overlay check (lines 140-150 KHÔNG ĐỔI) ...

    // ... existing: dual lock (lines 152-172 KHÔNG ĐỔI) ...

    // ★ REPLACE TODO (line 174) with actual validation:
    if err := uc.validator.ValidateEdge(lockCtx, appID, tenantID, relationType, sourceNodeID, targetNodeID); err != nil {
        observability.ObserveEntityWrite("create_edge", err)
        return nil, err
    }

    // Data Persistence (KHÔNG ĐỔI — lines 175-177)
    result, err := uc.repo.CreateEdge(lockCtx, appID, tenantID, relationType, sourceNodeID, targetNodeID, properties)
    observability.ObserveEntityWrite("create_edge", err)
    return result, err
}
```

### 4.5 Tích hợp vào Batch — Validate trước khi bulk write

```go
// internal/batch/batch.go — MODIFIED (thêm validator)

type Usecase struct {
    writer    Writer
    deduper   Deduper
    indexer   VectorIndexer
    validator EntityValidator  // ★ NEW
}

// EntityValidator interface cho batch validation
type EntityValidator interface {
    ValidateEntity(ctx context.Context, appID, label string, properties map[string]any) error
}

func (u *Usecase) Execute(ctx context.Context, req BatchUpsertRequest) (*BatchUpsertResult, error) {
    // ... existing: size check, dedup (KHÔNG ĐỔI) ...

    for i := range unique {
        // ... existing: label check, id generation (KHÔNG ĐỔI) ...

        // ★ NEW — validate each entity against ontology
        if u.validator != nil {
            if err := u.validator.ValidateEntity(ctx, req.AppID, unique[i].Label, unique[i].Properties); err != nil {
                return nil, fmt.Errorf("entity[%d] ontology validation failed: %w", i, err)
            }
        }
    }

    // ... existing: bulk write, indexer (KHÔNG ĐỔI) ...
}
```

### 4.6 ontologyRepo interface — Export để inject

```go
// internal/data/ontology.go — MODIFIED (export struct)

// OntologyRepo thay vì ontologyRepo (unexported → exported)
type OntologyRepo struct {  // ★ RENAMED from ontologyRepo
    data *Data
    log  *log.Helper
}

func NewOntologyRepo(data *Data, logger log.Logger) *OntologyRepo {
    return &OntologyRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

// GetEntityType — KHÔNG ĐỔI logic, chỉ rename receiver
func (r *OntologyRepo) GetEntityType(ctx context.Context, appID, name string) (*biz.EntityType, error) {
    // ... existing code KHÔNG ĐỔI ...
}

// GetRelationType — KHÔNG ĐỔI logic
func (r *OntologyRepo) GetRelationType(ctx context.Context, appID, name string) (*biz.RelationType, error) {
    // ... existing code KHÔNG ĐỔI ...
}
```

### 4.7 Wire DI — Kết nối tất cả

```go
// cmd/server/wire_gen.go — MODIFIED

func wireApp(confServer *conf.Server, confData *conf.Data, logger log.Logger) (*kratos.App, func(), error) {
    // ... existing: data, greeter, registry (KHÔNG ĐỔI) ...

    // ★ NEW — Ontology persistence
    db := data.NewGormDB(dataData)
    ontologyService := service.NewOntologyService(db)  // ★ CHANGED: inject db

    // ★ NEW — Ontology Repo + Validator
    ontologyRepo := data.NewOntologyRepo(dataData, logger)
    graphRepo := data.NewGraphRepo(dataData, logger)

    ontologyConfig := biz.OntologyValidatorConfig{
        Enabled:             confData.Ontology.GetValidationEnabled(),
        StrictMode:          confData.Ontology.GetStrictMode(),
        SchemaValidation:    confData.Ontology.GetSchemaValidation(),
        EdgeConstraintCheck: confData.Ontology.GetEdgeConstraintCheck(),
    }
    ontologyValidator := biz.NewOntologyValidator(ontologyRepo, graphRepo, ontologyConfig, logger)

    // ★ MODIFIED — inject validator into GraphUsecase
    graphUsecase := biz.NewGraphUsecase(
        graphRepo, queryPlanner, opaClient, client, redisLockManager,
        overlayManager,
        ontologyValidator,  // ★ NEW parameter
        logger,
    )

    // ... rest KHÔNG ĐỔI ...

    // ★ REMOVED: `_ = viewResolver` → wire vào GraphService nếu cần
}
```

### 4.8 Projection — Liên kết ViewDefinition với Ontology

```go
// internal/projection/ontology_sync.go — NEW

package projection

import (
    "context"

    "github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"
)

// OntologyProjectionSync syncs Ontology EntityTypes → ViewDefinition AllowedEntityTypes
type OntologyProjectionSync struct {
    db  *gorm.DB
    log *log.Helper
}

func NewOntologyProjectionSync(db *gorm.DB, logger log.Logger) *OntologyProjectionSync {
    return &OntologyProjectionSync{
        db:  db,
        log: log.NewHelper(logger),
    }
}

// SyncRoleView ensures ViewDefinition.AllowedEntityTypes includes all known EntityTypes
// from the Ontology for a given app/tenant/role
func (s *OntologyProjectionSync) SyncRoleView(ctx context.Context, appID, tenantID, roleName string) error {
    // 1. Fetch all EntityTypes for this app
    var entityTypes []biz.EntityType
    if err := s.db.WithContext(ctx).
        Where("app_id = ?", appID).
        Find(&entityTypes).Error; err != nil {
        return err
    }

    allTypes := make([]string, 0, len(entityTypes))
    for _, et := range entityTypes {
        allTypes = append(allTypes, et.Name)
    }

    // 2. Find or create ViewDefinition for this role
    var record ViewDefinitionRecord
    err := s.db.WithContext(ctx).
        Where("app_id = ? AND tenant_id = ? AND role_name = ?", appID, tenantID, roleName).
        Take(&record).Error

    if err == gorm.ErrRecordNotFound {
        // No ViewDefinition for this role — skip (don't auto-create)
        s.log.Infof("no ViewDefinition for role %q — skipping ontology sync", roleName)
        return nil
    }
    if err != nil {
        return err
    }

    // 3. Merge existing AllowedEntityTypes with Ontology types
    existing := decodeJSONStringArray(record.AllowedEntityTypesJSON)
    merged := mergeStringSlices(existing, allTypes)
    record.AllowedEntityTypesJSON = encodeJSONStringArray(merged)

    return s.db.WithContext(ctx).Save(&record).Error
}

func mergeStringSlices(a, b []string) []string {
    seen := make(map[string]struct{}, len(a)+len(b))
    for _, v := range a {
        seen[v] = struct{}{}
    }
    for _, v := range b {
        seen[v] = struct{}{}
    }
    result := make([]string, 0, len(seen))
    for v := range seen {
        result = append(result, v)
    }
    return result
}
```

### 4.9 Redis Cache Invalidation

```go
// internal/data/ontology.go — THÊM method InvalidateCache

// InvalidateEntityType removes cached entity type after ontology update
func (r *OntologyRepo) InvalidateEntityType(ctx context.Context, appID, name string) {
    cacheKey := fmt.Sprintf("%s%s:%s", entityCachePrefix, appID, name)
    r.data.rc.Del(ctx, cacheKey)
}

// InvalidateRelationType removes cached relation type after ontology update
func (r *OntologyRepo) InvalidateRelationType(ctx context.Context, appID, name string) {
    cacheKey := fmt.Sprintf("%s%s:%s", relationCachePrefix, appID, name)
    r.data.rc.Del(ctx, cacheKey)
}
```

Gọi invalidation khi `OntologyService.CreateEntityType/CreateRelationType` thành công:

```go
// service/ontology.go — sau khi upsert thành công
if s.ontologyRepo != nil {
    s.ontologyRepo.InvalidateEntityType(ctx, appCtx.AppID, req.GetName())
}
```

---

## 5. Package Structure sau tích hợp

```
kgs-platform/internal/
├── biz/
│   ├── graph.go                 # MODIFIED — thêm validator field + inject vào constructor
│   ├── ontology.go              # KHÔNG ĐỔI — GORM models (EntityType, RelationType)
│   ├── ontology_validator.go    # ★ NEW — validation logic (ValidateEntity, ValidateEdge)
│   ├── ontology_sync.go         # REMOVED hoặc DEPRECATED — thay bởi ontology_validator.go
│   ├── view_resolver.go         # KHÔNG ĐỔI
│   ├── errors.go                # KHÔNG ĐỔI — ErrSchemaInvalid đã sẵn sàng
│   └── ...
│
├── data/
│   ├── ontology.go              # MODIFIED — export struct OntologyRepo + thêm InvalidateCache
│   └── ...
│
├── service/
│   ├── ontology.go              # REWRITE — Postgres persistence (thay thế in-memory)
│   └── ...
│
├── batch/
│   ├── batch.go                 # MODIFIED — thêm EntityValidator interface + validate trước bulk
│   └── ...
│
├── projection/
│   ├── projection.go            # KHÔNG ĐỔI
│   ├── ontology_sync.go         # ★ NEW — sync Ontology → ViewDefinition
│   └── ...
│
└── conf/
    └── conf.proto               # MODIFIED — thêm OntologyConfig message
```

---

## 6. Wire DI Dependency Graph

```
                     ┌──────────────┐
                     │ conf.proto   │
                     │ OntologyConf │
                     └──────┬───────┘
                            │
                            ▼
              ┌─────────────────────────┐
              │   OntologyValidator     │ ← biz/ontology_validator.go
              │                         │
              │  + OntologyRepo (data)  │──→ Redis cache + Postgres
              │  + GraphRepo   (data)   │──→ Neo4j (GetNode for label lookup)
              │  + Config      (conf)   │──→ Feature flags
              └────────────┬────────────┘
                           │ inject
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    GraphUsecase      batch.Usecase    (future: overlay)
    CreateNode()      Execute()
    CreateEdge()

              ┌──────────────────────────┐
              │   OntologyService        │ ← service/ontology.go (REWRITTEN)
              │                          │
              │  + gorm.DB              │──→ Postgres (kgs_entity_types, kgs_relation_types)
              │  + OntologyRepo (opt)   │──→ Cache invalidation on create/update
              └──────────────────────────┘
```

---

## 7. Observability

### 7.1 Metrics mới

| Metric | Type | Labels | Mô tả |
|--------|------|--------|-------|
| `kgs_ontology_validation_total` | Counter | result(pass/fail/skip), operation(entity/edge) | Số lần validate |
| `kgs_ontology_validation_duration_ms` | Histogram | operation | Thời gian validate (bao gồm cache lookup) |
| `kgs_ontology_violation_total` | Counter | mode(strict/soft), label/relation_type | Số vi phạm ontology |
| `kgs_ontology_cache_hit_total` | Counter | type(entity/relation) | Cache hit ratio |

### 7.2 Structured Logging

```
// Soft mode — warning log
[WARN] [ontology-soft] unknown entity type metadata=map[app_id:app1 label:InvalidType]

// Strict mode — error response
400 ERR_SCHEMA_INVALID "unknown entity type" metadata={label: "InvalidType", app_id: "app1"}

// Cache miss log
[INFO] ontology cache miss: entity type "Requirement" for app "app1" — fetched from Postgres
```

---

## 8. Migration Strategy

### Phase 0: Chuẩn bị (1-2 ngày)

| Task | File | Mô tả | Risk |
|------|------|-------|------|
| T0.1 | `conf/conf.proto` | Thêm `OntologyConfig` message | Zero — additive |
| T0.2 | `data/ontology.go` | Export `OntologyRepo` (rename `ontologyRepo` → `OntologyRepo`) | Zero — rename |
| T0.3 | `data/ontology.go` | Thêm `InvalidateEntityType/InvalidateRelationType` | Zero — additive |
| T0.4 | `biz/ontology_validator.go` | Tạo `OntologyValidator` (NEW file) | Zero — new file |
| T0.5 | `biz/errors.go` | Verify `ErrSchemaInvalid` hoạt động đúng | Đã có — no change |

### Phase 1: Wire Validation Gate (2-3 ngày)

| Task | File | Mô tả | Risk |
|------|------|-------|------|
| T1.1 | `biz/graph.go` | Thêm `validator` field + inject qua constructor | Low — thêm 1 field |
| T1.2 | `biz/graph.go:CreateNode` | Thêm `uc.validator.ValidateEntity()` sau OPA check | Low — 3 dòng code |
| T1.3 | `biz/graph.go:CreateEdge` | Thay `TODO: Validate relation whitelist` bằng `uc.validator.ValidateEdge()` | Low — thay 1 dòng TODO |
| T1.4 | `cmd/server/wire.go` | Wire `OntologyRepo` + `OntologyValidator` + inject vào `GraphUsecase` | Medium — DI change |
| T1.5 | Deploy | `validation_enabled=true, strict_mode=false` (soft mode) | Zero — log only |

### Phase 2: OntologyService Persistence (1-2 ngày)

| Task | File | Mô tả | Risk |
|------|------|-------|------|
| T2.1 | `service/ontology.go` | Rewrite sang Postgres persistence | Medium — rewrite |
| T2.2 | `data/data.go` | Verify AutoMigrate có `EntityType`, `RelationType` | Low — verify |
| T2.3 | `service/ontology.go` | Wire cache invalidation | Low |
| T2.4 | Deploy + Test | Create EntityType via API → verify validation rejects unknown types | — |

### Phase 3: Enable Strict Mode (1 ngày)

| Task | File | Mô tả | Risk |
|------|------|-------|------|
| T3.1 | `configs/config.yaml` | Set `strict_mode: true` | Medium — sẽ reject invalid writes |
| T3.2 | Monitor | Watch `kgs_ontology_violation_total` — xem có false positives | — |
| T3.3 | Seed data | Đảm bảo tất cả EntityTypes/RelationTypes đã được seed trước khi strict | — |

### Phase 4: Batch Validation + Projection Sync (2-3 ngày)

| Task | File | Mô tả | Risk |
|------|------|-------|------|
| T4.1 | `batch/batch.go` | Thêm `EntityValidator` interface + validate trong loop | Low |
| T4.2 | `projection/ontology_sync.go` | Tạo OntologyProjectionSync | Low — new file |
| T4.3 | Wire | Wire sync vào worker server hoặc cron | Low |

### Phase 5: JSON Schema Validation (2-3 ngày)

| Task | File | Mô tả | Risk |
|------|------|-------|------|
| T5.1 | `go.mod` | Thêm `github.com/santhosh-tekuri/jsonschema/v5` | Low |
| T5.2 | `biz/ontology_validator.go` | Implement `validateJSONSchema()` | Medium — new logic |
| T5.3 | Seed schemas | Create EntityTypes với JSON Schema definitions qua API | — |
| T5.4 | `configs/config.yaml` | Set `schema_validation: true` | Medium |

---

## 9. Seed Data — EntityTypes và RelationTypes cần có

Trước khi enable strict mode, cần seed tất cả entity/relation types mà hệ thống đang sử dụng:

### 9.1 EntityTypes (từ Ontology doc)

```json
[
    {"name": "Requirement", "schema": {"type": "object", "properties": {"priority": {"type": "string", "enum": ["HIGH","MEDIUM","LOW"]}, "status": {"type": "string"}, "source": {"type": "string"}}}},
    {"name": "UseCase", "schema": {"type": "object", "properties": {"actor": {"type": "string"}, "goal": {"type": "string"}}}},
    {"name": "Actor", "schema": {}},
    {"name": "APIEndpoint", "schema": {}},
    {"name": "DataModel", "schema": {}},
    {"name": "NFR", "schema": {}},
    {"name": "BusinessRule", "schema": {}},
    {"name": "Risk", "schema": {}},
    {"name": "Epic", "schema": {}},
    {"name": "UserStory", "schema": {}},
    {"name": "Feature", "schema": {}},
    {"name": "Stakeholder", "schema": {}},
    {"name": "UserFlow", "schema": {}},
    {"name": "Screen", "schema": {}},
    {"name": "Persona", "schema": {}},
    {"name": "Interaction", "schema": {}},
    {"name": "Integration", "schema": {}},
    {"name": "Sequence", "schema": {}}
]
```

### 9.2 RelationTypes (từ Ontology doc)

```json
[
    {"name": "DEPENDS_ON", "source_types": ["Requirement","UseCase"], "target_types": ["Requirement","UseCase","NFR"]},
    {"name": "IMPLEMENTS", "source_types": ["APIEndpoint"], "target_types": ["DataModel"]},
    {"name": "CONFLICTS_WITH", "source_types": ["Requirement"], "target_types": ["Requirement"]},
    {"name": "TRACED_TO", "source_types": ["UseCase"], "target_types": ["Requirement"]},
    {"name": "CALLS", "source_types": ["APIEndpoint"], "target_types": ["APIEndpoint"]},
    {"name": "EXTENDS", "source_types": ["DataModel"], "target_types": ["DataModel"]},
    {"name": "PART_OF", "source_types": ["UserStory","Feature"], "target_types": ["Epic"]},
    {"name": "BLOCKS", "source_types": ["UserStory"], "target_types": ["UserStory"]},
    {"name": "DELIVERS_VALUE_TO", "source_types": ["Feature"], "target_types": ["Stakeholder"]},
    {"name": "NAVIGATES_TO", "source_types": ["Screen"], "target_types": ["Screen"]},
    {"name": "TRIGGERED_BY", "source_types": ["Interaction"], "target_types": ["Screen","UserFlow"]}
]
```

### 9.3 Seed Script

```go
// cmd/seed/ontology_seed.go — chạy 1 lần trước khi enable strict mode

func SeedOntology(ctx context.Context, db *gorm.DB, appID, tenantID string) error {
    entityTypes := []biz.EntityType{
        {AppID: appID, TenantID: tenantID, Name: "Requirement", Schema: datatypes.JSON(`{"type":"object","properties":{"priority":{"type":"string","enum":["HIGH","MEDIUM","LOW"]}}}`)},
        {AppID: appID, TenantID: tenantID, Name: "UseCase", Schema: datatypes.JSON(`{"type":"object","properties":{"actor":{"type":"string"},"goal":{"type":"string"}}}`)},
        // ... remaining types ...
    }

    for _, et := range entityTypes {
        db.Clauses(clause.OnConflict{DoNothing: true}).Create(&et)
    }

    relationTypes := []biz.RelationType{
        {AppID: appID, TenantID: tenantID, Name: "DEPENDS_ON",
            SourceTypes: datatypes.JSON(`["Requirement","UseCase"]`),
            TargetTypes: datatypes.JSON(`["Requirement","UseCase","NFR"]`)},
        // ... remaining types ...
    }

    for _, rt := range relationTypes {
        db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rt)
    }

    return nil
}
```

---

## 10. Tổng hợp thay đổi

### Files mới

| File | Package | Mô tả |
|------|---------|-------|
| `internal/biz/ontology_validator.go` | biz | OntologyValidator — entity/edge validation logic |
| `internal/projection/ontology_sync.go` | projection | Sync Ontology → ViewDefinition |
| `cmd/seed/ontology_seed.go` | seed | Seed script cho EntityTypes/RelationTypes |

### Files sửa đổi

| File | Thay đổi | Risk |
|------|----------|------|
| `internal/biz/graph.go` | Thêm `validator` field, inject vào constructor, 3 dòng validate trong CreateNode, 3 dòng trong CreateEdge | Low |
| `internal/data/ontology.go` | Export `OntologyRepo` + thêm `InvalidateEntityType/InvalidateRelationType` | Low |
| `internal/service/ontology.go` | Rewrite: in-memory → Postgres persistence | Medium |
| `internal/batch/batch.go` | Thêm `EntityValidator` interface + validate loop | Low |
| `internal/conf/conf.proto` | Thêm `OntologyConfig` message | Zero |
| `cmd/server/wire.go` | Thêm `OntologyRepo` + `OntologyValidator` providers | Medium |
| `cmd/server/wire_gen.go` | Regenerate | — |

### Files KHÔNG ĐỔI

| File | Lý do |
|------|-------|
| `internal/biz/ontology.go` | GORM models đã đầy đủ — tái sử dụng 100% |
| `internal/biz/errors.go` | `ErrSchemaInvalid` đã sẵn sàng |
| `internal/data/graph_node.go` | Write path không đổi |
| `internal/data/graph_edge.go` | Write path không đổi |
| `internal/data/graph_query.go` | Read path không đổi |
| `internal/projection/projection.go` | Logic projection không đổi |
| `internal/projection/model.go` | ViewDefinition model không đổi |
| `internal/search/*` | Search không liên quan |
| `internal/overlay/*` | Overlay không liên quan |
| `internal/version/*` | Version không liên quan |
| `api/ontology/v1/ontology.proto` | Proto API contract giữ nguyên |

---

## 11. Đánh giá rủi ro

| Rủi ro | Mức | Giải pháp |
|--------|-----|----------|
| **Strict mode reject writes hợp lệ** | ⚠️ Medium | Soft mode trước (log only). Seed đầy đủ EntityTypes trước khi strict |
| **Ontology lookup thêm latency** | 🟢 Low | Redis cache 5 phút. Cache hit > 99% sau warm-up |
| **OntologyService rewrite** | 🟡 Medium | API contract giữ nguyên (proto không đổi). Chỉ persistence thay đổi |
| **Edge validation cần GetNode** | 🟡 Medium | Thêm 2 Neo4j reads per CreateEdge. Cache-able nếu cần |
| **JSON Schema library** | 🟢 Low | `santhosh-tekuri/jsonschema` — mature, well-tested library |
| **Wire DI change** | 🟡 Medium | `NewGraphUsecase` thêm 1 parameter. `wire_gen.go` regenerate |
| **Rollback** | 🟢 Easy | Set `validation_enabled: false` → instant rollback, zero code change |

---

## 12. Timeline tổng hợp

```
Phase 0 (1-2 ngày):  Chuẩn bị — config proto, OntologyValidator, export OntologyRepo
Phase 1 (2-3 ngày):  Wire Validation Gate — CreateNode + CreateEdge validation (soft mode)
Phase 2 (1-2 ngày):  OntologyService persistence — in-memory → Postgres
Phase 3 (1 ngày):    Enable strict mode — seed data + monitor
Phase 4 (2-3 ngày):  Batch validation + Projection sync
Phase 5 (2-3 ngày):  JSON Schema validation

Tổng: ~9-14 ngày (có thể song song Phase 2 với Phase 1)
```

```
Week 1:  Phase 0 + Phase 1 + Phase 2 (song song)
Week 2:  Phase 3 (enable strict) + Phase 4 (batch + projection)
Week 3:  Phase 5 (JSON Schema) + Testing + Documentation
```
