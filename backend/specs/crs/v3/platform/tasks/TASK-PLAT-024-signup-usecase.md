# TASK-PLAT-024 — Signup Usecase (Atomic User + Tenant Creation)

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-024 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-008](../solutions/SOL-PLAT-008-Tenant-Creation-Onboarding.md) §2 |
| **Component** | `services/vnp-platform/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Mục tiêu

Tạo `SignupUseCase` với atomic transaction tạo user + tenant cùng lúc.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/usecase/signup.go` [NEW]

```go
package usecase

type SignupRequest struct {
    Email    string `json:"email"    validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    OrgName  string `json:"org_name" validate:"required"`
    Tier     string `json:"tier"`
}

type SignupResponse struct {
    UserID    string `json:"user_id"`
    TenantID  string `json:"tenant_id"`
    EmailSent bool   `json:"email_sent"`
}

type SignupUseCase struct {
    userRepo   port.UserRepository
    tenantRepo port.TenantRepository
    tokenRepo  port.VerificationTokenRepository
    emailSvc   port.EmailService
    db         *pgxpool.Pool
}

func (u *SignupUseCase) Signup(ctx context.Context, req *SignupRequest) (*SignupResponse, error) {
    if req.Tier == "" { req.Tier = "free" }

    // Check duplicate email
    existing, _ := u.userRepo.GetByEmail(ctx, req.Email)
    if existing != nil { return nil, ErrEmailAlreadyExists }

    var userID, tenantID string
    err := pgxutil.WithTx(ctx, u.db, func(tx pgx.Tx) error {
        slug := slugify(req.OrgName)
        if ok, _ := u.tenantRepo.SlugExistsTx(ctx, tx, slug); ok {
            slug = slug + "-" + shortID()
        }
        t, err := u.tenantRepo.CreateTx(ctx, tx, &domain.Tenant{
            ID: uuid.NewString(), Name: req.OrgName, Slug: slug, Tier: req.Tier, Status: "active",
        })
        if err != nil { return err }
        tenantID = t.ID

        hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
        usr, err := u.userRepo.CreateTx(ctx, tx, &domain.User{
            ID: uuid.NewString(), Email: req.Email, PasswordHash: string(hash),
            TenantID: tenantID, Role: "admin",
        })
        if err != nil { return err }
        userID = usr.ID
        return nil
    })
    if err != nil { return nil, err }

    // Async: send verification email
    token := generateSecureToken()
    u.tokenRepo.Store(ctx, token, userID, 24*time.Hour)
    go u.emailSvc.SendVerification(req.Email, token)

    return &SignupResponse{UserID: userID, TenantID: tenantID, EmailSent: true}, nil
}

func slugify(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    return regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
}

func shortID() string {
    b := make([]byte, 3); rand.Read(b)
    return hex.EncodeToString(b)
}
```

### 2. DB Migration `deployment/dev/migrations/0045_signup.sql` [NEW]

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS verify_token    TEXT,
    ADD COLUMN IF NOT EXISTS verify_token_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. Tạo `services/vnp-platform/internal/usecase/signup_test.go` [NEW]

```go
func TestSignup_Success(t *testing.T) {
    uc := newTestSignupUC()
    resp, err := uc.Signup(ctx, &SignupRequest{
        Email: "test@example.com", Password: "password123", OrgName: "Test Org",
    })
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.UserID)
    assert.NotEmpty(t, resp.TenantID)
    assert.True(t, resp.EmailSent)
}

func TestSignup_DuplicateEmail(t *testing.T) {
    uc := newTestSignupUC()
    uc.Signup(ctx, &SignupRequest{Email: "test@example.com", Password: "pass1234", OrgName: "Org1"})
    _, err := uc.Signup(ctx, &SignupRequest{Email: "test@example.com", Password: "pass5678", OrgName: "Org2"})
    assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestSlugify(t *testing.T) {
    assert.Equal(t, "hello-world", slugify("Hello World!"))
    assert.Equal(t, "test-org", slugify("Test Org"))
}
```

---

## Acceptance Criteria

- [ ] Signup → user + tenant created atomically (transaction)
- [ ] Duplicate email → `ErrEmailAlreadyExists`
- [ ] Slug: lowercase alphanumeric + hyphens only
- [ ] Slug conflict → append short suffix
- [ ] Unit tests pass

## Files

```
services/vnp-platform/internal/usecase/signup.go       [NEW]
services/vnp-platform/internal/usecase/signup_test.go  [NEW]
deployment/dev/migrations/0045_signup.sql               [NEW]
```
