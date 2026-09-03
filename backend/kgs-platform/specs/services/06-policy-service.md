# policy-service — Policy & Access Control Service

> **Role:** Quản lý và thực thi Access Control Policies sử dụng OPA (Open Policy Agent) với mô hình ABAC (Attribute-Based Access Control).

---

## 1. Trách Nhiệm (Single Responsibility)

`policy-service` chịu trách nhiệm **duy nhất** cho:
- **Policy CRUD**: Tạo, đọc, cập nhật, xóa OPA Rego policies per app
- **Policy Sync**: Đẩy policies lên OPA server khi có thay đổi
- **Policy Evaluation**: Evaluate access decision cho graph-service và query-intel-service
- **Policy Validation**: Validate Rego syntax trước khi lưu
- **Policy Bundling**: Nhóm policies theo app và bundle cho OPA

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         policy-service                                   │
│                                                                         │
│  gRPC Server (port 9006)                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  PolicyServiceServer                              │   │
│  │                                                                  │   │
│  │  CreatePolicy()    GetPolicy()    ListPolicies()                 │   │
│  │  UpdatePolicy()    DeletePolicy() ActivatePolicy()               │   │
│  │  Evaluate()                       [called by graph-service]      │   │
│  │  BulkEvaluate()                   [field-level access check]     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Policy Business Logic                           │   │
│  │                                                                  │   │
│  │  PolicyUsecase                                                   │   │
│  │  ├── Rego syntax validator (via OPA library)                     │   │
│  │  ├── OPAClient — REST calls to OPA server                        │   │
│  │  └── PolicySyncRunner — background sync on policy change         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Data Layer                                      │   │
│  │  PostgreSQL: policies table                                      │   │
│  │  OPA Server: bundle push, query evaluation                       │   │
│  │  Redis:      evaluation result cache (short TTL=30s)             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼ REST API
                         ┌───────────┐
                         │ OPA Server│
                         │ (port 8181│
                         └───────────┘
```

---

## 3. Data Models

### 3.1 Policy

```go
type Policy struct {
    ID          uint   `gorm:"primaryKey"`
    AppID       string `gorm:"type:varchar(50);not null;index:idx_app_policy"`
    Name        string `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_policy_name"`
    Description string `gorm:"type:text"`
    RegoContent string `gorm:"type:text;not null"`
    // Rego policy code:
    // package ba_agent.authz
    //
    // allow {
    //   input.action == "graph:read"
    //   input.role == "developer"
    // }
    IsActive    bool           `gorm:"default:true"`
    Version     int            `gorm:"default:1"`
    // Tăng mỗi khi update, dùng để detect stale
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

---

## 4. gRPC API

```protobuf
service PolicyService {
  // Policy CRUD
  rpc CreatePolicy(CreatePolicyRequest) returns (Policy);
  rpc GetPolicy(GetPolicyRequest) returns (Policy);
  rpc ListPolicies(ListPoliciesRequest) returns (ListPoliciesResponse);
  rpc UpdatePolicy(UpdatePolicyRequest) returns (Policy);
  rpc DeletePolicy(DeletePolicyRequest) returns (google.protobuf.Empty);
  rpc ActivatePolicy(ActivatePolicyRequest) returns (Policy);
  rpc DeactivatePolicy(DeactivatePolicyRequest) returns (Policy);

  // Policy Evaluation (called internally by graph-service, query-intel-service)
  rpc Evaluate(EvaluateRequest) returns (EvaluateResponse);
  rpc BulkEvaluate(BulkEvaluateRequest) returns (BulkEvaluateResponse);
}

message EvaluateRequest {
  string app_id = 1;
  EvaluateInput input = 2;
}

message EvaluateInput {
  string action = 1;        // e.g., "graph:write"
  string resource = 2;      // e.g., "Requirement"
  string role = 3;          // e.g., "developer"
  string tenant_id = 4;
  map<string, string> attributes = 5; // Extra ABAC attributes
}

message EvaluateResponse {
  bool allow = 1;
  string reason = 2;        // Lý do deny (nếu có)
  repeated string matched_rules = 3; // Tên rules đã match
}
```

---

## 5. ABAC Model

### 5.1 Input Context cho OPA

Mỗi lần evaluate, `policy-service` gửi đến OPA server:

```json
{
  "input": {
    "app_id":    "ba_agent",
    "action":    "graph:write",
    "resource":  "Requirement",
    "role":      "developer",
    "tenant_id": "tenant_001",
    "attributes": {
      "req_status": "DRAFT",
      "is_owner": "true"
    }
  }
}
```

### 5.2 Ví Dụ Rego Policy — BA Agent System

```rego
# package: ba_agent.authz
package ba_agent.authz

import future.keywords.if
import future.keywords.in

# Mặc định deny
default allow := false

# Developer có thể đọc mọi thứ
allow if {
    input.action == "graph:read"
    input.role in {"developer", "manager", "admin"}
}

# Developer chỉ write DRAFT Requirement
allow if {
    input.action == "graph:write"
    input.resource == "Requirement"
    input.role == "developer"
    input.attributes.req_status == "DRAFT"
}

# Manager có thể approve Requirement
allow if {
    input.action == "graph:write"
    input.resource == "Requirement"
    input.role == "manager"
    input.attributes.req_status in {"DRAFT", "APPROVED"}
}

# Admin có full access
allow if {
    input.role == "admin"
}
```

### 5.3 Field-Level Access Control (BulkEvaluate)

```json
// BulkEvaluate: kiểm tra nhiều field permissions cùng lúc
// Called by query-intel-service khi resolve view

Request:
{
  "app_id": "ba_agent",
  "evaluations": [
    { "action": "field:read", "resource": "Requirement.internal_notes", "role": "developer" },
    { "action": "field:read", "resource": "Requirement.title", "role": "developer" },
    { "action": "field:read", "resource": "Requirement.cost_estimate", "role": "developer" }
  ]
}

Response:
{
  "results": [
    { "allow": false, "reason": "internal_notes requires manager role" },
    { "allow": true },
    { "allow": false, "reason": "cost_estimate is restricted" }
  ]
}
```

---

## 6. Policy Sync Flow

```
PolicySyncRunner (background goroutine):
  1. Subscribe to PostgreSQL NOTIFY (hoặc poll every 30s)
  2. On policy change detected:
     a. Load all ACTIVE policies for affected app_id
     b. Build OPA bundle:
        {
          ".manifest": { "roots": ["ba_agent"] },
          "ba_agent/authz/policy.rego": <rego_content>
        }
     c. PUT /v1/policies/ba_agent/data → OPA server
     d. Verify: POST /v1/data/ba_agent/authz with test input
     e. Update sync_status in policy metadata
  3. If sync fails → mark policies as OUT_OF_SYNC, alert
```

### OPA Bundle Format

```
bundles/
  ba_agent/
    .manifest
    authz/
      policy.rego      ← main policy
      helpers.rego     ← helper functions
```

---

## 7. Evaluation Cache

Để tránh gọi OPA server cho mỗi request, kết quả evaluate được cache:

```
Cache Key: policy:eval:{app_id}:{hash(input)}
TTL: 30 seconds
Invalidation: Khi policy của app thay đổi → flush toàn bộ cache của app_id đó
```

> **Lưu ý:** TTL ngắn (30s) để đảm bảo policy changes có hiệu lực gần như tức thì.

---

## 8. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope | Mô tả |
|--------|------|-------|-------|
| POST | `/v1/policies` | `policies:write` | Tạo policy (Rego) |
| GET | `/v1/policies` | `policies:read` | List policies |
| GET | `/v1/policies/:id` | `policies:read` | Chi tiết policy |
| PUT | `/v1/policies/:id` | `policies:write` | Update policy |
| DELETE | `/v1/policies/:id` | `policies:write` | Xóa policy |
| POST | `/v1/policies/:id/activate` | `policies:write` | Activate policy |
| POST | `/v1/policies/:id/deactivate` | `policies:write` | Deactivate policy |
| POST | `/v1/policies/evaluate` | `policies:read` | Test evaluate (sandbox) |

---

## 9. Policy Validation

Trước khi lưu, Rego syntax được validate:

```go
// Sử dụng OPA Go SDK để compile Rego
func (u *PolicyUsecase) ValidateRego(regoContent string) error {
    parsed, err := ast.ParseModule("policy.rego", regoContent)
    if err != nil {
        return fmt.Errorf("rego syntax error: %w", err)
    }

    compiler := ast.NewCompiler()
    compiler.Compile(map[string]*ast.Module{"policy.rego": parsed})
    if compiler.Failed() {
        return fmt.Errorf("rego compile error: %v", compiler.Errors)
    }

    return nil
}
```

---

## 10. Configuration

```yaml
# configs/policy.yaml
policy_service:
  grpc_port: 9006

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_policy"

  opa:
    server_url: http://opa-server:8181
    bundle_push_path: /v1/policies
    query_path: /v1/data

  redis:
    addr: redis:6379
    eval_cache_ttl: 30s

  sync:
    poll_interval: 30s
    max_retry: 3

  observability:
    metrics_port: 9096
```

---

## 11. Observability

| Metric | Mô tả |
|--------|-------|
| `policy_evaluations_total{app_id, action, result}` | Số lần evaluate |
| `policy_evaluation_duration_seconds` | Latency của evaluate |
| `policy_cache_hits_total{app_id}` | Cache hit rate |
| `policy_sync_success_total{app_id}` | Successful OPA syncs |
| `policy_sync_failures_total{app_id}` | Failed OPA syncs |
| `policy_out_of_sync_apps_total` | Apps có policies chưa sync |
