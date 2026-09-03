---
id: MERGE-P1-T1
title: "vnp-platform: Absorb sm-auth (JWT + Google SSO)"
phase: P1
service: vnp-platform
priority: P0
status: Done
estimated: 4h
created: 2026-06-11
linked_sol: SOL-003
depends_on: []
---

## Mục Tiêu

Tích hợp toàn bộ business logic từ `sm-auth` (service duy nhất có implementation thật) vào `vnp-platform`. Đây là task đầu tiên vì sm-auth đã có code real — chỉ cần di chuyển và re-wire, không cần viết mới.

## Context

`sm-auth` hiện là service **độc lập, fully implemented** với:
- JWT RS256 signing + verification
- Google OAuth2 SSO flow
- In-memory user repository (cần migrate sang PostgreSQL)
- Proto: `smauthv1.SmAuthServiceServer`

`vnp-platform` hiện có structure Clean Architecture nhưng **chưa có auth domain**.

## Scope

### Nguồn (sm-auth) — MOVE, không xóa ngay

```
services/sm-auth/
├── api/proto/v1/         → Copy proto defs vào vnp-platform/api/proto/v1/
├── internal/
│   ├── domain/           → Move vào vnp-platform/internal/domain/auth/
│   │   └── entity.go     (User, Credentials, Token entities)
│   ├── usecase/          → Move vào vnp-platform/internal/usecase/auth/
│   │   └── auth.go       (Register, Login, LoginWithGoogle)
│   ├── adapter/grpc/     → Move vào vnp-platform/internal/adapter/grpc/
│   │   └── auth_handler.go (SmAuthServiceServer impl)
│   └── adapter/repo/     → Move vào vnp-platform/internal/infra/persistence/
│       └── user_repo.go  (InMemoryUserRepository → PGUserRepository)
```

### Đích (vnp-platform) — EXTEND

```
services/vnp-platform/internal/
├── domain/auth/
│   └── entity.go         # User, Credentials, JWTClaims, AuthToken
├── usecase/auth/
│   └── service.go        # AuthUseCase: Register, Login, LoginWithGoogle
├── adapter/grpc/
│   └── auth_handler.go   # SmAuthServiceServer implementation
└── infra/persistence/
    └── pg_user_repo.go   # PostgreSQL UserRepository (replace in-memory)
```

## Thay Đổi Cần Thực Hiện

### 1. Domain Layer — `vnp-platform/internal/domain/auth/entity.go`

```go
package auth

import "time"

type User struct {
    ID           string
    Email        string
    Name         string
    PasswordHash string
    GoogleID     string
    CreatedAt    time.Time
}

type AuthToken struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    UserID       string
    Email        string
}

type Credentials struct {
    Email    string
    Password string
}
```

### 2. Usecase Layer — `vnp-platform/internal/usecase/auth/service.go`

Interface cần implement (từ sm-auth):
```go
type AuthUseCase interface {
    Register(ctx context.Context, name, email, password string) (*domain.User, *domain.AuthToken, error)
    Login(ctx context.Context, email, password string) (*domain.User, *domain.AuthToken, error)
    LoginWithGoogle(ctx context.Context, idToken string) (*domain.User, *domain.AuthToken, error)
    ValidateToken(ctx context.Context, token string) (*domain.AuthToken, error)
}
```

### 3. Infrastructure Layer — PostgreSQL UserRepository

Thay thế `InMemoryUserRepository` bằng PostgreSQL:
```go
// Table: vnp_users
// Columns: id, email, name, password_hash, google_id, created_at, updated_at

type PGUserRepository struct {
    pool *pgxpool.Pool
}

func (r *PGUserRepository) FindByEmail(ctx context.Context, email string) (*auth.User, error)
func (r *PGUserRepository) Create(ctx context.Context, user *auth.User) error
func (r *PGUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*auth.User, error)
```

### 4. Adapter Layer — gRPC Handler

Wire SmAuthServiceServer vào grpcServer trong `vnp-platform/cmd/server/main.go`:
```go
smauthv1.RegisterSmAuthServiceServer(grpcServer, authHandler)
```

### 5. ForwardService Routes — gRPC router

```go
router.Handle("POST", "/v1/auth/register",    authForward.Register)
router.Handle("POST", "/v1/auth/login",       authForward.Login)
router.Handle("POST", "/v1/auth/sso/google",  authForward.LoginWithGoogle)
```

### 6. Config — Environment Variables

```bash
AUTH_JWT_PRIVATE_KEY=<RSA PEM>     # required
GOOGLE_CLIENT_ID=<client_id>        # optional
DEFAULT_ADMIN_EMAIL=admin@vnp.io    # optional
DEFAULT_ADMIN_PASSWORD=secret       # optional
```

### 7. Database Migration

```sql
-- migrations/002_auth_users.sql
CREATE TABLE IF NOT EXISTS vnp_users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT UNIQUE NOT NULL,
    name         TEXT NOT NULL,
    password_hash TEXT,
    google_id    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_vnp_users_email ON vnp_users(email);
CREATE INDEX idx_vnp_users_google_id ON vnp_users(google_id) WHERE google_id IS NOT NULL;
```

## Acceptance Criteria

- [ ] `POST /v1/auth/register` returns JWT token
- [ ] `POST /v1/auth/login` với valid credentials returns JWT
- [ ] `POST /v1/auth/sso/google` với valid Google ID token returns JWT
- [ ] User data persisted in PostgreSQL (không còn in-memory)
- [ ] `vnp-platform` builds: `go build ./services/vnp-platform/...`
- [ ] sm-auth unit tests pass khi run từ vnp-platform package
- [ ] ENV `AUTH_JWT_PRIVATE_KEY` missing → startup fails với clear error

## Ghi Chú

- Giữ nguyên `services/sm-auth/` — chưa xóa cho đến P4 cleanup
- Proto package path phải tương thích với gateway handler `auth.go`
- JWT RS256 key format: PEM encoded RSA private key (same as sm-auth)
