# TASK-PLAT-026 — Onboarding Checklist API

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-026 |
| **Wave** | 3 |
| **Solution** | [SOL-PLAT-008](../solutions/SOL-PLAT-008-Tenant-Creation-Onboarding.md) §4 |
| **Component** | `gateway/adapter/handler/console_handler.go` |
| **Priority** | 🟠 Medium |
| **Depends On** | TASK-PLAT-025 |
| **Estimated** | 2h |

---

## Mục tiêu

Tạo `GET /v1/console/onboarding` trả về checklist progress.

---

## Công việc cụ thể

### 1. Sửa `gateway/adapter/handler/console_handler.go` [MODIFY]

```go
// GET /v1/console/onboarding
func (h *ConsoleHandler) GetOnboarding(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    userID   := auth.UserIDFromContext(r.Context())
    ctx := r.Context()

    type check struct{ name string; fn func() bool }
    checks := []check{
        {"email_verified",     func() bool { u, _ := h.userRepo.Get(ctx, userID); return u != nil && u.EmailVerified }},
        {"api_key_created",    func() bool { n, _ := h.apiKeyRepo.CountForTenant(ctx, tenantID); return n > 0 }},
        {"first_memory_stored", func() bool { n, _ := h.memoryRepo.CountForTenant(ctx, tenantID); return n > 0 }},
        {"mcp_connected",      func() bool { n, _ := h.sessionRepo.CountForTenant(ctx, tenantID); return n > 0 }},
    }

    var mu sync.Mutex
    var wg sync.WaitGroup
    steps := map[string]bool{}
    for _, c := range checks {
        wg.Add(1)
        go func(ch check) {
            defer wg.Done()
            result := ch.fn()
            mu.Lock(); steps[ch.name] = result; mu.Unlock()
        }(c)
    }
    wg.Wait()

    completed := 0
    for _, v := range steps { if v { completed++ } }
    writeJSON(w, 200, map[string]any{
        "steps": steps, "completed": completed,
        "total": len(steps), "done": completed == len(steps),
    })
}
```

### 2. Thêm route

```go
r.Get("/v1/console/onboarding", consoleHandler.GetOnboarding)
```

---

## Acceptance Criteria

- [ ] All 4 checks run in parallel (goroutines)
- [ ] `done: true` khi all 4 checks pass
- [ ] Response < 500ms
- [ ] Correct `completed` count

## Files

```
gateway/adapter/handler/console_handler.go  [MODIFY]
gateway/adapter/handler/router.go           [MODIFY]
```
