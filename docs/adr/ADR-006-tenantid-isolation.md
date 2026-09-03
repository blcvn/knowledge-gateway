# ADR-006 — TenantID Column-level Isolation

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-03 |
| **Deciders** | Platform + Security Team |
| **Feature** | F14 (Multi-tenancy), F22 (Governance) |

---

## Context

VNP Memory là multi-tenant SaaS. Yêu cầu: **zero cross-tenant data leaks** — tenant A không được đọc data của tenant B.

3 approaches phổ biến:
1. **Schema separation:** Mỗi tenant có PostgreSQL schema riêng
2. **Database separation:** Mỗi tenant có database riêng
3. **Column-level:** Shared tables, mọi row có `tenant_id` column

---

## Decision

**Column-level isolation với TenantID trong mọi table, enforced tại application layer.**

```go
// Middleware inject TenantID vào every request context
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := extractTenantFromJWT(r)  // từ JWT claims
        ctx := context.WithValue(r.Context(), TenantContextKey, tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Every database query MUST include tenant_id
// Pattern enforced by code review + linting rule
func (r *MemoryRepository) FindByUser(ctx context.Context, userID string) ([]*Memory, error) {
    tenantID := tenant.FromContext(ctx)  // panic if missing = developer error caught early
    return r.db.QueryContext(ctx,
        "SELECT * FROM memories WHERE tenant_id = $1 AND user_id = $2",
        tenantID, userID,
    )
}
```

**Integration test bắt buộc:**

```go
// Must pass before any deploy
func TestCrossTenantLeakPrevention(t *testing.T) {
    // Store data as tenant A
    storeAs(t, "tenant_a", "secret data")

    // Attempt read as tenant B → must return 0 results
    results := recallAs(t, "tenant_b", "secret data")
    assert.Empty(t, results, "Cross-tenant leak detected!")
}
```

---

## Consequences

**Positive:**
- **Simplest to implement** — không cần schema migration per tenant
- Efficient JOIN operations across tenants (admin queries)
- Single database instance cho nhiều tenants → giảm operational cost
- Easy to add new tenant (no DDL required)

**Negative:**
- Application code phải đúng mọi lúc — không có database-level enforcement
- "noisy neighbor" problem: 1 tenant heavy query ảnh hưởng others
- GDPR delete phức tạp hơn (cần DELETE WHERE tenant_id=X trên nhiều tables)

**Mitigations:**
- Mandatory `tenant.FromContext()` call trong mọi repository method
- Integration test suite cho cross-tenant isolation (CI gate)
- Query timeout per-tenant để giảm noisy neighbor
- GDPR cascading delete được automated (F22)

---

## Alternatives Considered

### A1 — PostgreSQL Schema Separation
- **Rejected:** Schema explosion với scale (1000 tenants = 1000 schemas); migration phải chạy trên mọi schema; cross-tenant admin query phức tạp

### A2 — Database Separation
- **Rejected:** 1 DB instance per tenant = rất expensive; không practical cho SaaS model

### A3 — Row-Level Security (PostgreSQL RLS)
- **Rejected:** Xem ADR-004; performance overhead + debug difficulty outweigh benefits
