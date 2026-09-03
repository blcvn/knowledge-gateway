# Bug Report — F17: Graph Studio

> Feature: Subgraph query, entity detail, temporal graph, ontology management, Cypher/NL query
> Luồng: `apps/memory → gateway/console.go (GraphHandler) → kg-service`

---

## BUG-F17-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:149-194`

---

## BUG-F17-002: `kg-service` Thiếu Forward Service Implementation

**Severity:** HIGH  
**File:** `services/kg-service/`

**Mô tả:**  
`kg-service` được dùng cho cả Graphiti lẫn Graph Studio. Cần verify service này có implement ForwardService protocol và xử lý được tất cả Graph Studio paths.

---

## BUG-F17-003: `UpdateOntology` Không Có Audit Log

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/console.go:181-186`

**Mô tả:**  
`PUT /v1/console/graph/ontology` là sensitive operation thay đổi schema của Knowledge Graph. Không có audit log được ghi khi operation này được thực hiện. `AuditUseCase` đã được implement nhưng không được dùng trong GraphHandler.

**Impact:**  
- Không có audit trail cho ontology changes — vi phạm governance requirements.
