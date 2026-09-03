# TASK-ENT-001 — Tenant Middleware: Strict TenantID Validation

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-001 |
| **Wave** | 1 (Security Foundation) |
| **Solution** | [SOL-ENT-004](../solutions/SOL-ENT-004-MultiTenant-Isolation.md) §1.1 |
| **Component** | `shared/pkg/tenant/middleware.go` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Mục tiêu

Harden TenantMiddleware: strict validation, structured logging, context key safety.

---

## Công việc cụ thể

### `shared/pkg/tenant/middleware.go` [MODIFY]

```go
package tenant

type contextKey struct{}

// TenantMiddleware — strict tenant validation from JWT claims
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := jwtutil.ParseClaims(r)

        if claims == nil {
            writeError(w, 401, "missing_auth", "authorization required")
            return
        }
        if claims.TenantID == "" {
            writeError(w, 401, "missing_tenant", "tenant_id not found in token")
            return
        }
        // Validate format: UUID or slug
        if !isValidTenantID(claims.TenantID) {
            writeError(w, 400, "invalid_tenant", "tenant_id format invalid")
            return
        }

        ctx := context.WithValue(r.Context(), contextKey{}, claims.TenantID)
        slog.InfoContext(ctx, "request", "tenant_id", claims.TenantID,
            "method", r.Method, "path", r.URL.Path)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// FromContext — must always be called in usecase layer (panics in dev if missing)
func FromContext(ctx context.Context) string {
    v, ok := ctx.Value(contextKey{}).(string)
    if !ok || v == "" {
        if os.Getenv("APP_ENV") == "development" {
            panic("tenant.FromContext: no tenant_id in context — middleware missing?")
        }
        return ""
    }
    return v
}

func isValidTenantID(id string) bool {
    // Allow UUID or alphanumeric slug
    _, err := uuid.Parse(id)
    if err == nil { return true }
    return regexp.MustCompile(`^[a-z0-9_-]{3,64}$`).MatchString(id)
}
```

### Unit tests

```go
func TestTenantMiddleware_MissingJWT_Returns401(t *testing.T) {}
func TestTenantMiddleware_EmptyTenantID_Returns401(t *testing.T) {}
func TestTenantMiddleware_InvalidFormat_Returns400(t *testing.T) {}
func TestTenantMiddleware_Valid_SetsContext(t *testing.T) {}
func TestFromContext_Missing_PanicsInDev(t *testing.T) {}
```

---

## Acceptance Criteria

- [ ] Missing JWT → 401
- [ ] Empty TenantID → 401
- [ ] Invalid format → 400
- [ ] Valid UUID → sets context
- [ ] Valid slug (e.g., "acme-corp") → sets context
- [ ] FromContext panics in dev if no tenant in context
- [ ] `go test ./shared/pkg/tenant/...` passes

## Files

```
shared/pkg/tenant/middleware.go       [MODIFY — strict validation]
shared/pkg/tenant/middleware_test.go  [NEW]
```
