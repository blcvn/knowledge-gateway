# TASK-PLAT-006 — API Key HTTP Handlers & Routes

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-006 |
| **Wave** | 2 (Auth Flows) |
| **Solution** | [SOL-PLAT-001](../solutions/SOL-PLAT-001-Auth-API-Key-JWT.md) §2.4 API Endpoints |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-PLAT-005 |
| **Estimated** | 3h |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** console_sdk.go CreateKey/ListKeys/DeleteKey handlers present; backend usecase not fully wired
---

## Mục tiêu

Implement HTTP handlers cho API key lifecycle endpoints trong gateway. Register routes vào router.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/auth_apikey.go` [NEW]

```go
package handler

type APIKeyHandler struct {
    uc port.APIKeyUseCase
}

// GET /v1/console/sdk/keys
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    keys, err := h.uc.List(r.Context(), auth.TenantID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "key_error", err.Error())
        return
    }
    // Never expose secret_hash in response — only prefix + metadata
    type keyResponse struct {
        ID        string  `json:"id"`
        Name      string  `json:"name"`
        Prefix    string  `json:"prefix"`     // e.g. "vnp_abc12345"
        Status    string  `json:"status"`
        ExpiresAt *string `json:"expires_at"`
        CreatedAt string  `json:"created_at"`
    }
    resp := make([]keyResponse, 0, len(keys))
    for _, k := range keys {
        var exp *string
        if k.ExpiresAt != nil {
            s := k.ExpiresAt.Format(time.RFC3339)
            exp = &s
        }
        resp = append(resp, keyResponse{
            ID:        k.ID,
            Name:      k.Name,
            Prefix:    "vnp_" + k.Prefix,
            Status:    string(k.Status),
            ExpiresAt: exp,
            CreatedAt: k.CreatedAt.Format(time.RFC3339),
        })
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{"keys": resp})
}

// POST /v1/console/sdk/keys
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req struct {
        Name string  `json:"name"`
        TTL  *int    `json:"ttl_days"` // optional: expiry in days
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }
    if req.Name == "" {
        writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
        return
    }

    var ttl *time.Duration
    if req.TTL != nil {
        d := time.Duration(*req.TTL) * 24 * time.Hour
        ttl = &d
    }

    rawToken, key, err := h.uc.Create(r.Context(), usecase.CreateKeyRequest{
        TenantID: auth.TenantID,
        UserID:   auth.UserID,
        Name:     req.Name,
        TTL:      ttl,
        ActorID:  auth.UserID,
        IP:       r.RemoteAddr,
    })
    if err != nil {
        writeError(w, http.StatusInternalServerError, "key_create_error", err.Error())
        return
    }

    // raw_token: exposed ONCE at creation time
    writeJSON(w, http.StatusCreated, map[string]interface{}{
        "id":         key.ID,
        "name":       key.Name,
        "prefix":     "vnp_" + key.Prefix,
        "raw_token":  rawToken,   // ← only time this is shown
        "status":     key.Status,
        "created_at": key.CreatedAt.Format(time.RFC3339),
        "warning":    "Store this token securely. It will not be shown again.",
    })
}

// DELETE /v1/console/sdk/keys/{id}
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    keyID := chi.URLParam(r, "id")
    if err := h.uc.Revoke(r.Context(), keyID, auth.UserID, r.RemoteAddr); err != nil {
        writeError(w, http.StatusInternalServerError, "key_revoke_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// POST /v1/console/sdk/keys/{id}/rotate
func (h *APIKeyHandler) Rotate(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    oldKeyID := chi.URLParam(r, "id")
    rawToken, newKey, err := h.uc.Rotate(r.Context(), oldKeyID, auth.UserID, r.RemoteAddr)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "key_rotate_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "new_key_id":  newKey.ID,
        "prefix":      "vnp_" + newKey.Prefix,
        "raw_token":   rawToken,
        "old_key_id":  oldKeyID,
        "old_status":  "rotated",
        "warning":     "Store the new token securely. It will not be shown again.",
    })
}
```

### 2. Modify `gateway/adapter/handler/router.go` [MODIFY] — register API key routes

```go
// In console SDK routes section (requires admin role):
r.Route("/v1/console/sdk", func(r chi.Router) {
    r.Use(requireAdmin)

    // API Keys
    r.Get("/keys", apikeyH.List)
    r.Post("/keys", apikeyH.Create)
    r.Delete("/keys/{id}", apikeyH.Revoke)
    r.Post("/keys/{id}/rotate", apikeyH.Rotate)

    // JWKS (public — no auth required)
    // registered separately: r.Get("/.well-known/jwks.json", ...)
})
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/sdk/keys` returns list with prefix visible, no secret_hash
- [ ] `POST /v1/console/sdk/keys` returns `raw_token` + warning (only once)
- [ ] `DELETE /v1/console/sdk/keys/{id}` marks key revoked
- [ ] `POST /v1/console/sdk/keys/{id}/rotate` creates new key, old key status=rotated
- [ ] All endpoints require `admin` role (401/403 if not)
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/auth_apikey.go  [NEW]
gateway/adapter/handler/router.go       [MODIFY — register /v1/console/sdk/keys/* routes]
```
