# TASK-ENT-003 — Cross-Tenant Isolation Integration Test

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-003 |
| **Wave** | 1 |
| **Solution** | [SOL-ENT-004](../solutions/SOL-ENT-004-MultiTenant-Isolation.md) §1.3 |
| **Component** | `tests/integration/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-ENT-001 |
| **Estimated** | 3h |

---

## Mục tiêu

CI gate: integration tests proving zero cross-tenant data access.

---

## Công việc cụ thể

### `tests/integration/tenant_isolation_test.go` [NEW]

```go
package integration

func TestCrossTenantIsolation_MemoryStore(t *testing.T) {
    // Setup: 2 tenants
    tenantA := setupTestTenant(t, "tenant-a-test")
    tenantB := setupTestTenant(t, "tenant-b-test")

    // Tenant A stores memory
    storeResp := postJSON(t, "/v1/memory/store", tenantA.Token, map[string]any{
        "content": "secret data for tenant A only",
        "type": "semantic",
    })
    assert.Equal(t, 202, storeResp.StatusCode)
    time.Sleep(100 * time.Millisecond) // wait for async storage

    // Tenant B recalls — must not find Tenant A's memory
    recallResp := postJSON(t, "/v1/memory/recall", tenantB.Token, map[string]any{
        "query": "secret data for tenant A",
        "limit": 10,
    })
    var results RecallResponse
    json.NewDecoder(recallResp.Body).Decode(&results)
    assert.Empty(t, results.Results, "tenant B must NOT see tenant A's memory")
}

func TestCrossTenantIsolation_ForgetUser(t *testing.T) {
    tenantA := setupTestTenant(t, "tenant-a-forget")
    tenantB := setupTestTenant(t, "tenant-b-forget")

    // Tenant B tries to forget Tenant A's user — must fail
    resp := postJSON(t, "/v1/admin/forget", tenantB.AdminToken, map[string]any{
        "user_id": "user-from-tenant-a",
        "tenant_id": tenantA.ID, // cross-tenant attempt
        "reason": "test",
    })
    assert.Equal(t, 403, resp.StatusCode, "cross-tenant forget must be forbidden")
}

func TestCrossTenantIsolation_Sessions(t *testing.T) {
    // Tenant B cannot read Tenant A's session replay
    tenantA := setupTestTenant(t, "tenant-a-session")
    tenantB := setupTestTenant(t, "tenant-b-session")

    sessionID := createTestSession(t, tenantA)
    resp := getWithToken(t, "/v1/observe/sessions/"+sessionID, tenantB.Token)
    assert.Equal(t, 404, resp.StatusCode) // not 403 to avoid enumeration
}
```

---

## Acceptance Criteria

- [ ] Memory store: tenant B cannot see tenant A's memories
- [ ] Forget: cross-tenant forget returns 403
- [ ] Session: cross-tenant session access returns 404
- [ ] Tests run in CI pipeline as gate
- [ ] Tests clean up after themselves (testcontainers or seeded DB)

## Files

```
tests/integration/tenant_isolation_test.go  [NEW]
tests/integration/helpers_test.go           [NEW — test helpers]
```
