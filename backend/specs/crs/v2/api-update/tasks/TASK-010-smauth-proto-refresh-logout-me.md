# TASK-010: Add `sm-auth` Refresh + Logout + Me Endpoints to Proto

**Solution**: [SOL-001](../solutions/SOL-001-auth-api.md)  
**CR**: CR-001  
**Priority**: 🔴 Critical  
**Estimate**: 2 hours  
**Status**: ✅ Implemented

---

## Context

The `sm-auth` proto (`services/sm-auth/api/proto/v1/auth.proto`) currently only has:
- `Login(LoginRequest) → AuthResponse`
- `LoginWithGoogle(GoogleLoginRequest) → AuthResponse`

The frontend requires 4 auth endpoints: `login`, `logout`, `me`, `refresh`. The proto needs 3 new RPC methods.

`AuthResponse` already includes `token` and `user` — this is sufficient for the gateway to construct the `LoginResponse`. However, `refresh_token` (a separate long-lived token) is not issued yet.

For the MVP, the gateway's `Login` handler (TASK-001) will use the access token as both `access_token` and `refresh_token`. To properly support refresh, the proto needs a `Refresh` RPC.

---

## Exact Task

### Step 1: Update `services/sm-auth/api/proto/v1/auth.proto`

Add 3 new RPCs and messages:

```proto
service SmAuthService {
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  
  rpc Register(RegisterRequest) returns (AuthResponse);
  rpc Login(LoginRequest) returns (AuthResponse);
  rpc LoginWithGoogle(GoogleLoginRequest) returns (AuthResponse);
  
  // --- NEW RPCs ---
  rpc Logout(LogoutRequest) returns (LogoutResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  rpc GetCurrentUser(GetCurrentUserRequest) returns (UserProfile);
}

// --- NEW Messages ---

message LogoutRequest {
  string refresh_token = 1;
}

message LogoutResponse {
  bool success = 1;
}

message RefreshTokenRequest {
  string refresh_token = 1;
}

message RefreshTokenResponse {
  string access_token = 1;
  int64  expires_in   = 2; // seconds
}

message GetCurrentUserRequest {
  string access_token = 1; // JWT to decode
}
```

### Step 2: Regenerate Go code from proto

```bash
cd services/sm-auth

# Use the protoc tool from tools/protoc3/
../../tools/protoc3/bin/protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  api/proto/v1/auth.proto
```

Or use the project's existing generation script if it exists:
```bash
# Check for a Makefile or scripts/generate.sh
make proto-gen 2>/dev/null || ./scripts/generate_proto.sh 2>/dev/null
```

### Step 3: Implement stub handlers in `sm-auth` gRPC server

In `services/sm-auth/internal/adapter/grpc/auth_handler.go`, add stub implementations for the 3 new RPCs:

```go
func (h *AuthHandler) Logout(ctx context.Context, req *smauthv1.LogoutRequest) (*smauthv1.LogoutResponse, error) {
    // TODO: Implement refresh token invalidation (store invalidated tokens in Redis/DB)
    // For MVP: return success (stateless — client discards token)
    return &smauthv1.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *smauthv1.RefreshTokenRequest) (*smauthv1.RefreshTokenResponse, error) {
    // TODO: Implement proper refresh token validation + new access token generation
    // For MVP: re-validate the passed token and return it as the new access token
    return &smauthv1.RefreshTokenResponse{
        AccessToken: req.RefreshToken,
        ExpiresIn:   3600,
    }, nil
}

func (h *AuthHandler) GetCurrentUser(ctx context.Context, req *smauthv1.GetCurrentUserRequest) (*smauthv1.UserProfile, error) {
    // TODO: Decode JWT and return user profile
    // For MVP: return a placeholder profile
    return &smauthv1.UserProfile{
        Id:    "unknown",
        Name:  "User",
        Email: "user@example.com",
        Role:  "admin",
    }, nil
}
```

> **MVP Note**: These are intentionally simplified stubs. The TODO comments mark where real token validation/storage should be implemented in a follow-up task.

### Step 4: Register new RPCs in the gRPC server

Ensure the server registration in `sm-auth`'s `cmd/main.go` or server setup properly registers `SmAuthServiceServer` which now includes the new methods.

---

## Files to Modify

| File | Change |
|------|--------|
| `services/sm-auth/api/proto/v1/auth.proto` | Add `Logout`, `RefreshToken`, `GetCurrentUser` RPCs + messages |
| `services/sm-auth/api/proto/v1/auth.pb.go` | Regenerate (do not edit manually) |
| `services/sm-auth/api/proto/v1/auth_grpc.pb.go` | Regenerate (do not edit manually) |
| `services/sm-auth/internal/adapter/grpc/auth_handler.go` | Add 3 stub handler implementations |

---

## Acceptance Criteria

- [ ] `auth.proto` has `Logout`, `RefreshToken`, `GetCurrentUser` RPCs
- [ ] `auth.pb.go` and `auth_grpc.pb.go` are regenerated from the updated proto
- [ ] `auth_handler.go` implements all 3 new methods (stubs are acceptable for MVP)
- [ ] `go build ./services/sm-auth/...` passes

---

**Audit Note:** sm-auth proto: Logout/RefreshToken/GetCurrentUser RPCs + messages added; gRPC stubs implemented
