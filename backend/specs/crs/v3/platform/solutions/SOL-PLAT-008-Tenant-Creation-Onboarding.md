# Solution: SOL-PLAT-008 — Tenant Creation & Onboarding Flow

**CR:** CR-PLAT-008
**TDD refs:** `models/vnp-platform.md`, `architecture/01-gateway.md §4.1`
**Version:** v3/platform

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** Register usecase exists; email verification + onboarding checklist not implemented
---

## 1. Architecture Analysis

TDD `vnp-platform` model already defines:
```go
type Tenant struct {
    ID, Name, Slug, Tier, Status, Metadata, EngineAliases, CreatedAt, UpdatedAt
}
type SubscriptionTier = "free" | "pro" | "enterprise"
type TenantStatus     = "active" | "suspended" | "deleted"
```

Missing:
- Signup → tenant+user creation (atomic transaction)
- Email verification flow
- Onboarding checklist tracking

---

## 2. Signup Usecase

```go
// services/vnp-platform/internal/usecase/signup.go [NEW]
type SignupRequest struct {
    Email    string `json:"email"    validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    OrgName  string `json:"org_name" validate:"required"`
    Tier     string `json:"tier"`    // default: "free"
}

type SignupUseCase struct {
    userRepo   port.UserRepository
    tenantRepo port.TenantRepository
    emailSvc   port.EmailService
    db         *pgxpool.Pool
}

func (u *SignupUseCase) Signup(ctx context.Context, req *SignupRequest) (*SignupResponse, error) {
    if req.Tier == "" { req.Tier = "free" }

    // Check duplicate email
    if _, err := u.userRepo.GetByEmail(ctx, req.Email); err == nil {
        return nil, ErrEmailAlreadyExists
    }

    // Atomic: create user + tenant in transaction
    var (userID, tenantID string)
    err := pgxutil.WithTx(ctx, u.db, func(tx pgx.Tx) error {
        slug := slugify(req.OrgName)
        // Ensure unique slug
        if exists, _ := u.tenantRepo.SlugExists(ctx, slug); exists {
            slug = slug + "-" + shortID()
        }

        t, err := u.tenantRepo.CreateTx(ctx, tx, &Tenant{
            Name: req.OrgName, Slug: slug,
            Tier: req.Tier, Status: "active",
        })
        if err != nil { return err }
        tenantID = t.ID

        hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
        usr, err := u.userRepo.CreateTx(ctx, tx, &User{
            Email: req.Email, PasswordHash: string(hash),
            TenantID: tenantID, Role: "admin",
        })
        if err != nil { return err }
        userID = usr.ID
        return nil
    })
    if err != nil { return nil, err }

    // Send verification email (async)
    verifyToken := generateSecureToken()
    u.tokenRepo.Store(ctx, verifyToken, userID, 24*time.Hour)
    go u.emailSvc.SendVerification(req.Email, verifyToken)

    return &SignupResponse{UserID: userID, TenantID: tenantID, EmailSent: true}, nil
}

func slugify(s string) string {
    s = strings.ToLower(s)
    return regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
}
```

---

## 3. Email Verification

```go
// gateway/adapter/handler/auth_handler.go [MODIFY]

// GET /v1/auth/verify-email?token=xxx
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    userID, err := h.tokenRepo.Get(r.Context(), token)
    if err != nil || userID == "" {
        writeError(w, 400, "invalid_token", "verification link invalid or expired")
        return
    }
    if err := h.userRepo.MarkEmailVerified(r.Context(), userID); err != nil {
        writeError(w, 500, "verify_failed", err.Error())
        return
    }
    h.tokenRepo.Delete(r.Context(), token)
    writeJSON(w, 200, map[string]bool{"email_verified": true})
}
```

---

## 4. Onboarding Checklist

```go
// gateway/adapter/handler/console_handler.go [MODIFY]

// GET /v1/console/onboarding
func (h *ConsoleHandler) GetOnboarding(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    userID   := auth.UserIDFromContext(r.Context())

    var wg sync.WaitGroup; var mu sync.Mutex
    checklist := map[string]bool{}

    checks := map[string]func() bool{
        "email_verified":     func() bool { u, _ := h.userRepo.Get(r.Context(), userID); return u != nil && u.EmailVerified },
        "api_key_created":    func() bool { keys, _ := h.apiKeyRepo.CountForTenant(r.Context(), tenantID); return keys > 0 },
        "first_memory_stored": func() bool { count, _ := h.memRepo.CountForTenant(r.Context(), tenantID); return count > 0 },
        "mcp_connected":      func() bool { sessions, _ := h.sessionRepo.CountForTenant(r.Context(), tenantID); return sessions > 0 },
    }

    for name, check := range checks {
        wg.Add(1)
        go func(n string, c func() bool) {
            defer wg.Done()
            mu.Lock(); checklist[n] = c(); mu.Unlock()
        }(name, check)
    }
    wg.Wait()

    completed := 0
    for _, v := range checklist { if v { completed++ } }
    writeJSON(w, 200, map[string]any{
        "steps": checklist,
        "completed": completed,
        "total": len(checklist),
        "done": completed == len(checklist),
    })
}
```

---

## 5. DB Migration

```sql
-- deployment/dev/migrations/0045_signup.sql

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS verify_token    TEXT,
    ADD COLUMN IF NOT EXISTS verify_token_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 6. File Changes

| File | Action |
|---|---|
| `services/vnp-platform/internal/usecase/signup.go` | **[NEW]** Signup atomic tx |
| `services/vnp-platform/internal/usecase/signup_test.go` | **[NEW]** |
| `gateway/adapter/handler/auth_handler.go` | **[MODIFY]** add Signup + VerifyEmail |
| `gateway/adapter/handler/console_handler.go` | **[MODIFY]** add Onboarding |
| `gateway/adapter/handler/router.go` | **[MODIFY]** routes |
| `deployment/dev/migrations/0045_signup.sql` | **[NEW]** |
