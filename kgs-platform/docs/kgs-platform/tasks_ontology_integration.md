# Tasks: Tích hợp Ontology vào ai-kg-service

> **Tham chiếu:** [Phương án tích hợp](plan_ontology_integration.md)
> **Ngày:** 10/03/2026

---

## Phase 0 — Chuẩn bị hạ tầng

### P0.1 Config Proto — Thêm OntologyConfig

- [x] Thêm message `OntologyConfig` vào `internal/conf/conf.proto` trong message `Data` (field number 9)
  - Fields: `validation_enabled`, `strict_mode`, `schema_validation`, `edge_constraint_check`, `sync_projection`
- [x] Chạy `make proto` để generate Go code từ proto
- [x] Thêm section `ontology` vào `configs/config.yaml` với tất cả flags mặc định `false`
- [x] Verify `confData.GetOntology()` trả về giá trị đúng từ config

Kết quả thực thi:
- Đã chạy `make config` tại `services/ai-kg-service/kgs-platform` để regenerate `internal/conf/conf.pb.go`.
- Đã thêm test `internal/conf/conf_test.go` để verify `GetOntology()` load đủ 5 flags với giá trị mặc định `false`.

### P0.2 Export OntologyRepo (data layer)

- [x] Rename struct `ontologyRepo` → `OntologyRepo` trong `internal/data/ontology.go`
- [x] Rename constructor `NewOntologyRepo` — đảm bảo return `*OntologyRepo`
- [x] Rename method receivers `(r *ontologyRepo)` → `(r *OntologyRepo)` cho `GetEntityType` và `GetRelationType`
- [x] Verify compile — không file nào khác reference `ontologyRepo` unexported

Kết quả thực thi:
- Đã verify compile bằng `go test ./internal/data -run '^$'` và `go test ./cmd/server`.

### P0.3 Cache Invalidation

- [x] Thêm method `InvalidateEntityType(ctx, appID, name)` vào `data/OntologyRepo` — xóa Redis key `ontology:entity:{appID}:{name}`
- [x] Thêm method `InvalidateRelationType(ctx, appID, name)` vào `data/OntologyRepo` — xóa Redis key `ontology:relation:{appID}:{name}`

### P0.4 Tạo OntologyValidator (biz layer)

- [x] Tạo file `internal/biz/ontology_validator.go`
- [x] Định nghĩa interface `OntologyRepo` trong biz package:
  ```go
  type OntologyRepo interface {
      GetEntityType(ctx, appID, name string) (*EntityType, error)
      GetRelationType(ctx, appID, name string) (*RelationType, error)
  }
  ```
- [x] Định nghĩa struct `OntologyValidatorConfig` với 4 fields: `Enabled`, `StrictMode`, `SchemaValidation`, `EdgeConstraintCheck`
- [x] Định nghĩa struct `OntologyValidator` với dependencies: `repo OntologyRepo`, `graph GraphRepo`, `config`, `log`
- [x] Implement `NewOntologyValidator(repo, graph, config, logger)` constructor
- [x] Implement `ValidateEntity(ctx, appID, label, properties)`:
  - Return nil nếu validator nil hoặc disabled
  - Return nil nếu repo nil (graceful degradation)
  - Gọi `repo.GetEntityType(ctx, appID, label)`
  - Nếu lookup fail → log warning, return nil (graceful degradation)
  - Nếu entity type không tồn tại → gọi `handleViolation("unknown entity type")`
  - Nếu `SchemaValidation` enabled và `EntityType.Schema` != nil → gọi `validateJSONSchema()` (placeholder, implement đầy đủ ở Phase 5)
- [x] Implement `ValidateEdge(ctx, appID, tenantID, relationType, sourceNodeID, targetNodeID)`:
  - Return nil nếu validator nil hoặc disabled
  - Gọi `repo.GetRelationType(ctx, appID, relationType)`
  - Nếu lookup fail → log warning, return nil
  - Nếu relation type không tồn tại → gọi `handleViolation("unknown relation type")`
  - Nếu `EdgeConstraintCheck` disabled → return nil
  - Parse `RelationType.SourceTypes` và `TargetTypes` từ JSON
  - Nếu cả hai rỗng → return nil (no constraint)
  - Gọi `graph.GetNode(ctx, appID, tenantID, sourceNodeID)` để lấy label
  - Check source label ∈ SourceTypes — nếu không → `handleViolation`
  - Gọi `graph.GetNode(ctx, appID, tenantID, targetNodeID)` để lấy label
  - Check target label ∈ TargetTypes — nếu không → `handleViolation`
- [x] Implement `handleViolation(ctx, message, metadata)`:
  - Nếu `StrictMode` → return `ErrSchemaInvalid(message, metadata)`
  - Nếu soft mode → log warning `[ontology-soft]`, return nil
- [x] Implement helper: `decodeJSONArray(raw json.RawMessage) []string`
- [x] Implement helper: `containsIgnoreCase(list []string, value string) bool`
- [x] Implement helper: `extractLabelFromNode(node map[string]any) string`

### P0.5 Verify ErrSchemaInvalid

- [x] Verify `ErrSchemaInvalid` trong `biz/errors.go` trả về 400 `ERR_SCHEMA_INVALID` đúng format
- [x] Verify kratos error serialization cho gRPC và HTTP responses

Kết quả thực thi:
- Đã thêm test `internal/biz/errors_test.go` verify code/reason/metadata.
- Đã verify serialization: gRPC `codes.InvalidArgument`, JSON payload HTTP chứa `code=400`, `reason=ERR_SCHEMA_INVALID`.

### P0.6 Unit Tests cho OntologyValidator

- [x] Tạo file `internal/biz/ontology_validator_test.go`
- [x] Test: `ValidateEntity` khi disabled → return nil
- [x] Test: `ValidateEntity` khi repo nil → return nil
- [x] Test: `ValidateEntity` entity type tồn tại → return nil (pass)
- [x] Test: `ValidateEntity` entity type không tồn tại + strict mode → return ErrSchemaInvalid
- [x] Test: `ValidateEntity` entity type không tồn tại + soft mode → return nil (log warning)
- [x] Test: `ValidateEntity` repo lookup fail → return nil (graceful degradation)
- [x] Test: `ValidateEdge` relation type tồn tại, source/target hợp lệ → return nil
- [x] Test: `ValidateEdge` relation type không tồn tại + strict → return error
- [x] Test: `ValidateEdge` source type không hợp lệ + strict → return error
- [x] Test: `ValidateEdge` target type không hợp lệ + strict → return error
- [x] Test: `ValidateEdge` SourceTypes/TargetTypes rỗng → return nil (no constraint)
- [x] Test: `ValidateEdge` EdgeConstraintCheck disabled → skip source/target check

Kết quả thực thi:
- Đã chạy `go test ./internal/conf ./internal/biz` và pass.

---

## Phase 1 — Wire Validation Gate vào Graph Write

### P1.1 Sửa GraphUsecase struct

- [x] Thêm field `validator *OntologyValidator` vào struct `GraphUsecase` trong `biz/graph.go`
- [x] Thêm parameter `validator *OntologyValidator` vào constructor `NewGraphUsecase()`
- [x] Gán `validator: validator` trong constructor body

### P1.2 Wire vào CreateNode

- [x] Thêm 3 dòng code vào `biz/graph.go:CreateNode()` — SAU OPA check (sau line 111), TRƯỚC data persistence (trước line 114):
  ```go
  if err := uc.validator.ValidateEntity(lockCtx, appID, label, properties); err != nil {
      observability.ObserveEntityWrite("create_node", err)
      return nil, err
  }
  ```

### P1.3 Wire vào CreateEdge

- [x] Thay dòng `// TODO: Validate relation whitelist` (line 174) trong `biz/graph.go:CreateEdge()` bằng:
  ```go
  if err := uc.validator.ValidateEdge(lockCtx, appID, tenantID, relationType, sourceNodeID, targetNodeID); err != nil {
      observability.ObserveEntityWrite("create_edge", err)
      return nil, err
  }
  ```

### P1.4 Cập nhật Wire DI

- [x] Sửa `cmd/server/wire.go` — thêm `data.NewOntologyRepo` vào provider set
- [x] Sửa `cmd/server/wire.go` — thêm `biz.NewOntologyValidator` vào provider set
- [x] Sửa `cmd/server/wire.go` — cập nhật `biz.NewGraphUsecase` call với parameter `ontologyValidator`
- [x] Build `OntologyValidatorConfig` từ `confData.GetOntology()` trong wire
- [ ] Chạy `wire` để regenerate `wire_gen.go`
- [x] Verify compile thành công

Kết quả thực thi:
- `data.NewOntologyRepo` đã có sẵn trong `internal/data/data.go` ProviderSet, không cần bổ sung mới.
- Đã thêm `NewOntologyValidator` vào `internal/biz/biz.go` ProviderSet.
- Đã thêm provider config `newOntologyValidatorConfig(confData)` trong `cmd/server/wire_helpers.go`.
- Đã wire `ontologyValidator` vào `NewGraphUsecase` trong `cmd/server/wire_gen.go`.
- Đã thử chạy regenerate: `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go generate ./cmd/server` nhưng Wire fail do các lỗi DI nền sẵn có ngoài phạm vi ontology (thiếu providers cho registry/batch/search/overlay/projection interfaces).
- Compile đã pass với wiring mới: `go test ./internal/biz ./internal/service ./cmd/server`.

### P1.5 Integration Test — Soft Mode

- [ ] Deploy với config: `validation_enabled=true`, `strict_mode=false`
- [ ] Test CreateNode với label hợp lệ (đã có EntityType) → thành công, không log warning
- [ ] Test CreateNode với label không tồn tại trong ontology → thành công + log `[ontology-soft]` warning
- [ ] Test CreateEdge với relation type hợp lệ → thành công
- [ ] Test CreateEdge với relation type không tồn tại → thành công + log warning
- [ ] Verify không có regression trên existing APIs

Kết quả thực thi:
- Chưa thực hiện được trong môi trường hiện tại vì cần deploy runtime stack (Neo4j/Redis/Postgres/OPA + service running) để chạy integration flow end-to-end.

---

## Phase 2 — OntologyService Persistence

### P2.1 Rewrite OntologyService

- [x] Sửa `internal/service/ontology.go`:
  - Xóa toàn bộ in-memory fields: `mu sync.RWMutex`, `byApp map[string]*ontologyStore`, `nextEntityID`, `nextRelationID`
  - Xóa struct `ontologyStore`, `entityTypeItem`, `relationTypeItem`
  - Xóa method `ensureStoreLocked()`
  - Thêm field `db *gorm.DB`
  - Thêm field `ontologyRepo *data.OntologyRepo` (optional — cho cache invalidation)
- [x] Sửa constructor `NewOntologyService(db *gorm.DB) *OntologyService` — inject `*gorm.DB`
- [x] Rewrite `CreateEntityType`:
  - Build `biz.EntityType` từ request
  - Dùng `db.Clauses(clause.OnConflict{...}).Create()` để upsert
  - Gọi `ontologyRepo.InvalidateEntityType()` nếu available
  - Return reply với ID, Name, Status
- [x] Rewrite `CreateRelationType`:
  - Build `biz.RelationType` từ request — marshal `SourceTypes`/`TargetTypes` thành JSON
  - Upsert với OnConflict
  - Gọi `ontologyRepo.InvalidateRelationType()` nếu available
  - Return reply
- [x] Rewrite `ListEntityTypes`:
  - Query `db.Where("app_id = ?", appCtx.AppID).Find(&entities)`
  - Map sang `[]*pb.EntityTypeInfo`
- [x] Rewrite `ListRelationTypes`:
  - Query `db.Where("app_id = ?", appCtx.AppID).Find(&relations)`
  - Map sang `[]*pb.RelationTypeInfo`
  - Implement helper `decodeJSONStringSlice(raw datatypes.JSON) []string`

Kết quả thực thi:
- Đã rewrite toàn bộ `internal/service/ontology.go` sang Postgres/GORM persistence, dùng upsert cho entity/relation và có hook invalidation cache Redis khi `ontologyRepo` được inject.
- Constructor hiện tại: `NewOntologyService(db *gorm.DB, ontologyRepo *data.OntologyRepo)`.

### P2.2 Cập nhật Wire — OntologyService inject db

- [x] Sửa `cmd/server/wire.go` — `service.NewOntologyService(db)` thay vì `service.NewOntologyService()`
- [x] Optional: inject `data.OntologyRepo` vào `OntologyService` cho cache invalidation
- [ ] Chạy `wire` regenerate
- [x] Verify compile

Kết quả thực thi:
- Runtime wiring đã cập nhật trong `cmd/server/wire_gen.go`: `service.NewOntologyService(db, ontologyRepo)`.
- Đã thử regenerate bằng `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go generate ./cmd/server` nhưng vẫn fail do lỗi DI nền sẵn có ngoài scope ontology (registry/batch/search/overlay/projection).
- Compile pass: `go test ./internal/service ./internal/biz ./cmd/server`.

### P2.3 Verify AutoMigrate

- [x] Kiểm tra `data/data.go` — xác nhận `AutoMigrate` có `&biz.EntityType{}`, `&biz.RelationType{}`
- [x] Nếu chưa có → thêm vào danh sách AutoMigrate
- [ ] Test: khởi động server → verify bảng `kgs_entity_types` và `kgs_relation_types` tồn tại trong Postgres

Kết quả thực thi:
- Đã xác nhận `AutoMigrate` đã có đủ `&biz.EntityType{}` và `&biz.RelationType{}` từ trước, không cần chỉnh sửa.
- Chưa verify trực tiếp trên Postgres runtime do chưa khởi động full server stack trong môi trường hiện tại.

### P2.4 Tests — OntologyService Persistence

- [x] Test: `CreateEntityType` → verify record lưu trong Postgres
- [x] Test: `CreateEntityType` lần 2 cùng name → verify upsert (update description/schema)
- [x] Test: `ListEntityTypes` → verify trả về danh sách từ Postgres
- [x] Test: `CreateRelationType` với SourceTypes/TargetTypes → verify JSON array lưu đúng
- [x] Test: `ListRelationTypes` → verify SourceTypes/TargetTypes decode đúng
- [x] Test: Server restart → verify data vẫn còn (không mất như in-memory)

Kết quả thực thi:
- Đã thêm file test `internal/service/ontology_test.go` với 5 test persistence (sqlite) bao phủ đầy đủ các case P2.4, gồm cả kiểm tra data tồn tại sau restart DB file.
- Đã chạy: `go test ./internal/service -run 'TestOntologyService'` và pass.

### P2.5 Xóa OntologySyncManager stub

- [x] Xóa file `internal/biz/ontology_sync.go` hoặc deprecated (add `// Deprecated:` comment)
- [x] Remove reference `ontology *OntologySyncManager` trong `GraphUsecase` struct (nếu còn)
- [x] Clean up wire nếu cần

Kết quả thực thi:
- Đã thêm comment `// Deprecated:` cho constructor `NewOntologySyncManager`.
- Đã remove field `ontology *OntologySyncManager` khỏi `GraphUsecase`.
- Đã remove `NewOntologySyncManager` khỏi `biz.ProviderSet`.

---

## Phase 3 — Seed Data + Enable Strict Mode

### P3.1 Tạo Seed Script

- [ ] Tạo file `cmd/seed/ontology_seed.go` (hoặc `internal/seed/ontology.go`)
- [ ] Implement function `SeedOntology(ctx, db, appID, tenantID)`:
  - Seed EntityTypes: `Requirement`, `UseCase`, `Actor`, `APIEndpoint`, `DataModel`, `NFR`, `BusinessRule`, `Risk`, `Epic`, `UserStory`, `Feature`, `Stakeholder`, `UserFlow`, `Screen`, `Persona`, `Interaction`, `Integration`, `Sequence`
  - Seed RelationTypes: `DEPENDS_ON`, `IMPLEMENTS`, `CONFLICTS_WITH`, `TRACED_TO`, `CALLS`, `EXTENDS`, `PART_OF`, `BLOCKS`, `DELIVERS_VALUE_TO`, `NAVIGATES_TO`, `TRIGGERED_BY`
  - Mỗi RelationType phải có `SourceTypes` và `TargetTypes` đúng theo tài liệu ontology
  - Dùng `OnConflict{DoNothing: true}` để idempotent
- [ ] Implement JSON Schema cho EntityTypes có schema rõ ràng:
  - `Requirement`: `{"type":"object","properties":{"priority":{"type":"string","enum":["HIGH","MEDIUM","LOW"]},"status":{"type":"string"},"source":{"type":"string"}}}`
  - `UseCase`: `{"type":"object","properties":{"actor":{"type":"string"},"goal":{"type":"string"}}}`
  - Các types còn lại: schema rỗng `{}` (không bắt buộc properties)

### P3.2 Chạy Seed

- [ ] Chạy seed script cho tất cả app/tenant đang hoạt động trên production
- [ ] Verify: `ListEntityTypes` trả về đầy đủ 18 entity types
- [ ] Verify: `ListRelationTypes` trả về đầy đủ 11 relation types
- [ ] Verify: Redis cache warm-up — gọi `GetEntityType` cho mỗi type → cache hit lần 2

### P3.3 Audit Soft Mode Logs

- [ ] Chạy với `strict_mode=false` trong 1-2 ngày
- [ ] Thu thập tất cả log `[ontology-soft]` → xác định entity/relation types đang dùng nhưng chưa seed
- [ ] Seed thêm bất kỳ types nào bị thiếu
- [ ] Verify: không còn `[ontology-soft]` warning sau khi seed đầy đủ

### P3.4 Enable Strict Mode

- [ ] Cập nhật `configs/config.yaml`: `strict_mode: true`
- [ ] Deploy
- [ ] Monitor metric `kgs_ontology_violation_total` — nếu > 0 → investigate
- [ ] Verify: CreateNode với label không tồn tại → trả về 400 `ERR_SCHEMA_INVALID`
- [ ] Verify: CreateEdge với relation type không tồn tại → trả về 400 `ERR_SCHEMA_INVALID`
- [ ] Verify: CreateEdge với source/target type sai → trả về 400 `ERR_SCHEMA_INVALID`

---

## Phase 4 — Batch Validation + Projection Sync

### P4.1 Batch Validation

- [ ] Định nghĩa interface `EntityValidator` trong `internal/batch/batch.go`:
  ```go
  type EntityValidator interface {
      ValidateEntity(ctx context.Context, appID, label string, properties map[string]any) error
  }
  ```
- [ ] Thêm field `validator EntityValidator` vào struct `batch.Usecase`
- [ ] Sửa constructor `NewUsecaseWithIndexer` → thêm parameter `validator EntityValidator`
- [ ] Thêm validation loop trong `Execute()` — sau dedup, trước bulk write:
  ```go
  if u.validator != nil {
      for i := range unique {
          if err := u.validator.ValidateEntity(ctx, req.AppID, unique[i].Label, unique[i].Properties); err != nil {
              return nil, fmt.Errorf("entity[%d] ontology validation: %w", i, err)
          }
      }
  }
  ```
- [x] Cập nhật Wire DI — inject `ontologyValidator` (ép kiểu `biz.OntologyValidator` → `batch.EntityValidator`) vào `batch.Usecase`
- [ ] Chạy `wire` regenerate

Kết quả thực thi:
- Đã thêm `EntityValidator` vào `batch.Usecase` và inject qua constructor `NewUsecaseWithIndexer(..., validator)`.
- Đã thêm helper DI `newBatchEntityValidator` trong `cmd/server/wire_helpers.go` và cập nhật wiring runtime trong `cmd/server/wire_gen.go`.
- Đã thử regenerate bằng `env GOWORK=off GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go generate ./cmd/server` nhưng vẫn fail do các DI gaps nền sẵn có ngoài scope Phase 4 (registry/search/overlay/analytics/projection interfaces).

### P4.2 Batch Validation Tests

- [x] Test: batch với tất cả labels hợp lệ → thành công
- [x] Test: batch có 1 entity label không hợp lệ + strict → reject toàn bộ batch
- [x] Test: batch với validator nil → pass through (backward compatible)

Kết quả thực thi:
- Đã thêm 3 test mới trong `internal/batch/batch_test.go`:
  - `TestUsecaseExecuteWithValidator_AllLabelsValid`
  - `TestUsecaseExecuteWithValidator_InvalidLabelRejectsBatch`
  - `TestUsecaseExecuteWithNilValidator_PassThrough`
- Đã verify `go test ./internal/batch` pass.

### P4.3 Projection Ontology Sync

- [x] Tạo file `internal/projection/ontology_sync.go`
- [x] Implement `OntologyProjectionSync` struct với `db *gorm.DB`
- [x] Implement `NewOntologyProjectionSync(db, logger)`
- [x] Implement `SyncRoleView(ctx, appID, tenantID, roleName)`:
  - Fetch tất cả `EntityType` cho app
  - Fetch `ViewDefinitionRecord` cho role
  - Nếu không có ViewDefinition → skip (log info)
  - Merge `AllowedEntityTypes` từ ontology vào ViewDefinition
  - Save updated ViewDefinitionRecord
- [x] Implement helper `mergeStringSlices(a, b []string) []string` — deduplicate merge

Kết quả thực thi:
- Đã thêm `OntologyProjectionSync` với các methods:
  - `SyncRoleView(ctx, appID, tenantID, roleName)`
  - `SyncAllRoleViews(ctx, appID, tenantID)` (bổ sung để hỗ trợ sync hàng loạt role)
- `mergeStringSlices` đã deduplicate theo giá trị trim + ignore-case.

### P4.4 Wire Projection Sync vào Worker

- [x] Thêm `OntologyProjectionSync` vào Wire provider set
- [ ] Tạo background job/cron chạy `SyncRoleView` định kỳ (mỗi 10 phút hoặc khi event trigger)
- [x] Hoặc: gọi `SyncRoleView` trong `OntologyService.CreateEntityType` sau khi create thành công

Kết quả thực thi:
- Đã thêm `projection.NewOntologyProjectionSync` vào `projection.ProviderSet`.
- Chọn phương án hook tại service thay vì cron worker: `OntologyService.CreateEntityType` gọi `projectionSync.SyncAllRoleViews(...)` sau khi upsert thành công.

### P4.5 Projection Sync Tests

- [x] Test: Tạo EntityType mới → ViewDefinition cho role BA tự động cập nhật AllowedEntityTypes
- [x] Test: Không có ViewDefinition cho role → skip, không lỗi
- [x] Test: Merge không tạo duplicate types

Kết quả thực thi:
- Đã thêm test mới:
  - `internal/projection/ontology_sync_test.go`:
    - `TestOntologyProjectionSyncSyncRoleView_MergesOntologyTypes`
    - `TestOntologyProjectionSyncSyncRoleView_NoViewDefinitionSkips`
    - `TestMergeStringSlices_DeduplicatesValues`
  - `internal/service/ontology_test.go`:
    - `TestOntologyServiceCreateEntityType_SyncsViewAllowedEntityTypes`
- Đã verify `go test ./internal/projection ./internal/service` pass.

### P4.6 Wire ViewResolver

- [x] Xóa `_ = viewResolver` trong `wire_gen.go:83`
- [x] Wire `ViewResolver` vào `GraphService` nếu cần thiết
- [x] Hoặc: inject trực tiếp `ProjectionEngine` (đã có trong `GraphService.projection`)
- [x] Verify `ViewResolver.Resolve()` hoạt động trong read path

Kết quả thực thi:
- Đã cập nhật `GraphService` constructor để nhận thêm `*biz.ViewResolver`.
- Đã chuyển read-path projection (`applyProjectionToSingleNode`, `applyProjectionToGraphReply`) sang dùng `viewResolver.Resolve(...)`.
- Đã cập nhật wiring runtime trong `cmd/server/wire_gen.go` để truyền `viewResolver` vào `NewGraphService`.
- Đã thêm test `TestGraphServiceGetNode_UsesViewResolver` và verify pass.

---

## Phase 5 — JSON Schema Validation

### P5.1 Thêm JSON Schema Library

- [ ] Thêm dependency `github.com/santhosh-tekuri/jsonschema/v5` vào `go.mod`
- [ ] Chạy `go mod tidy` + `go mod vendor` (nếu dùng vendor)

### P5.2 Implement validateJSONSchema

- [ ] Implement method `validateJSONSchema(schema json.RawMessage, properties map[string]any) error` trong `biz/ontology_validator.go`:
  - Return nil nếu schema rỗng / nil / `{}` / `"null"`
  - Tạo `jsonschema.NewCompiler()`
  - `compiler.AddResource("entity_schema.json", reader)` từ schema
  - `compiler.Compile("entity_schema.json")`
  - Nếu compile fail → log warning, return nil (graceful degradation)
  - `compiled.Validate(properties)`
  - Nếu validate fail → return error với chi tiết validation errors

### P5.3 JSON Schema Validation Tests

- [ ] Test: properties match schema → pass
- [ ] Test: properties thiếu required field → fail
- [ ] Test: property sai type (string thay vì number) → fail
- [ ] Test: property enum value không hợp lệ → fail
- [ ] Test: schema rỗng `{}` → pass (không validate)
- [ ] Test: schema parse failure → graceful degradation (return nil)
- [ ] Test: nested object validation → pass/fail đúng

### P5.4 Enable Schema Validation

- [ ] Cập nhật `configs/config.yaml`: `schema_validation: true`
- [ ] Deploy
- [ ] Verify: CreateNode `Requirement` với `priority: "INVALID"` → reject
- [ ] Verify: CreateNode `Requirement` với `priority: "HIGH"` → pass

---

## Phase 6 — Observability + Cleanup

### P6.1 Metrics

- [ ] Register Prometheus counter `kgs_ontology_validation_total` với labels `result` (pass/fail/skip), `operation` (entity/edge)
- [ ] Register Prometheus histogram `kgs_ontology_validation_duration_ms` với label `operation`
- [ ] Register Prometheus counter `kgs_ontology_violation_total` với labels `mode` (strict/soft), `type_name`
- [ ] Instrument `ValidateEntity()` — record duration + result
- [ ] Instrument `ValidateEdge()` — record duration + result
- [ ] Instrument `handleViolation()` — increment violation counter

### P6.2 Cache Metrics

- [ ] Thêm counter `kgs_ontology_cache_hit_total` và `kgs_ontology_cache_miss_total` trong `data/OntologyRepo`
- [ ] Instrument `GetEntityType()` — record cache hit/miss
- [ ] Instrument `GetRelationType()` — record cache hit/miss

### P6.3 Dashboard

- [ ] Tạo Grafana dashboard cho ontology validation:
  - Panel: Validation pass/fail rate
  - Panel: Validation latency p50/p95/p99
  - Panel: Violation count by mode
  - Panel: Cache hit ratio
- [ ] Tạo alert khi `kgs_ontology_violation_total{mode="strict"}` > threshold

### P6.4 Cleanup

- [ ] Xóa file `internal/biz/ontology_sync.go` (nếu chưa xóa ở P2.5)
- [ ] Remove `ontology *OntologySyncManager` field khỏi `GraphUsecase` struct
- [ ] Clean up unused imports trong các files đã sửa
- [ ] Chạy `go vet ./...` + `golangci-lint run`
- [ ] Cập nhật tài liệu: [Báo cáo đối chiếu](repo_ontology_ai_kg_service.md) — đánh dấu các gaps đã fix

---

## Summary

| Phase | Số tasks | Ước lượng | Dependencies |
|-------|----------|-----------|-------------|
| Phase 0 | 18 | 1-2 ngày | Không |
| Phase 1 | 11 | 2-3 ngày | Phase 0 |
| Phase 2 | 12 | 1-2 ngày | Phase 0 (có thể song song Phase 1) |
| Phase 3 | 10 | 1-2 ngày | Phase 1 + Phase 2 |
| Phase 4 | 12 | 2-3 ngày | Phase 3 |
| Phase 5 | 8 | 2-3 ngày | Phase 1 (có thể song song Phase 3-4) |
| Phase 6 | 9 | 1-2 ngày | Phase 1 |
| **Tổng** | **80** | **~10-17 ngày** | |
