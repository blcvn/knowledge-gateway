# TASK-OV-001 — `pkg/viking/` Shared Domain Types

**Wave:** 1 (Foundation)  
**Ưu tiên:** Critical  
**Phụ thuộc:** Không có  
**Ước tính:** 2 giờ  
**Solution tham chiếu:** [SOL-OV-007 §3](../solutions/SOL-OV-007-Shared-Infrastructure.md)

---

## Mục tiêu

Tạo package `pkg/viking/` chứa toàn bộ shared domain types cho hệ thống OpenViking: URI system, Identity, RBAC, Context types, và Error types. Package này **không có business logic** và **không import bất kỳ package nội bộ nào khác**.

---

## Ngữ cảnh

`pkg/viking/` là nền tảng của toàn bộ OpenViking. Mọi service khác (crypto, admin, fs, search, session, resource, gateway) đều import từ package này. Phải được build và test trước tiên.

---

## Các file cần tạo

### 1. `pkg/viking/uri.go`

Implement các hàm URI:
- `ValidateURI(uri string) error` — reject `..`, `\`, ký tự không hợp lệ
- `CanonicalizeURI(uri string) (string, error)` — chuẩn hóa double slashes
- `ResolveOwner(uri string) (accountID, userID, agentID string, err error)`
- `IsAccessible(uri string, ctx *RequestContext) bool`
- `ToAbstractURI(fileURI string) string` — thêm `.abstract.md`
- `ToOverviewURI(fileURI string) string` — thêm `.overview.md`
- `IsAbstractURI(uri string) bool`
- `IsOverviewURI(uri string) bool`

Constants:
```go
const (
    RootResources = "viking://resources/"
    RootUser      = "viking://user/"
    RootAgent     = "viking://agent/"
    RootSession   = "viking://session/"
    RootTemp      = "viking://temp/"
)
```

Logic `IsAccessible`:
- `RoleRoot` → luôn true
- `RoleAdmin` → đúng accountID → true
- `RoleUser` → đúng accountID + userID → true
- `RoleBot` → đúng agentID → true
- URI trong `viking://resources/` → readable cho mọi user trong cùng account

### 2. `pkg/viking/identity.go`

```go
type Role int
const (
    RoleUser  Role = 0
    RoleBot   Role = 1
    RoleAdmin Role = 2
    RoleRoot  Role = 3
)

type UserIdentifier struct {
    AccountID string
    UserID    string
    AgentID   string
}

type RequestContext struct {
    User            UserIdentifier
    Role            Role
    NamespacePolicy string
    APIKeyID        string
}

// Context injection helpers
func FromContext(ctx context.Context) (*RequestContext, bool)
func WithContext(ctx context.Context, rc *RequestContext) context.Context
```

### 3. `pkg/viking/context.go`

```go
type ContextType int
const (
    ContextTypeMemory   ContextType = 0
    ContextTypeResource ContextType = 1
    ContextTypeSkill    ContextType = 2
    ContextTypeSession  ContextType = 3
)

func (c ContextType) RootURIs() []string  // Returns namespace roots per type
func (c ContextType) String() string      // "MEMORY" | "RESOURCE" | "SKILL" | "SESSION"

type ContextLevel int
const (
    LevelAbstract ContextLevel = 0  // .abstract.md
    LevelOverview ContextLevel = 1  // .overview.md
    LevelDetail   ContextLevel = 2  // raw file
)
```

### 4. `pkg/viking/errors.go`

```go
type ErrorCode string
const (
    ErrInvalidArgument  ErrorCode = "INVALID_ARGUMENT"
    ErrUnauthenticated  ErrorCode = "UNAUTHENTICATED"
    ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"
    ErrNotFound         ErrorCode = "NOT_FOUND"
    ErrAlreadyExists    ErrorCode = "ALREADY_EXISTS"
    ErrResourceBusy     ErrorCode = "RESOURCE_BUSY"
    ErrNotInitialized   ErrorCode = "NOT_INITIALIZED"
    ErrInternal         ErrorCode = "INTERNAL"
)

type OpenVikingError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Details map[string]any
}

func (e *OpenVikingError) Error() string
func (e *OpenVikingError) Unwrap() error
func NewError(code ErrorCode, msg string) *OpenVikingError
func WrapError(code ErrorCode, msg string, cause error) *OpenVikingError
```

---

## Unit Tests (`pkg/viking/*_test.go`)

```
TestValidateURI_ValidURI              → "viking://user/acct1/alice/" → nil
TestValidateURI_PathTraversal         → "viking://../escape" → ErrInvalidArgument
TestValidateURI_Backslash             → "viking://user\\alice" → ErrInvalidArgument
TestValidateURI_NoPrefixRejected      → "http://foo" → ErrInvalidArgument
TestCanonicalizeURI_DoubleSlash       → "viking://user//alice/" → "viking://user/alice/"
TestCanonicalizeURI_TripleSlash       → "viking:///resources/" → "viking://resources/"
TestResolveOwner_UserURI              → "viking://user/acct1/alice/mem/" → ("acct1","alice","",nil)
TestResolveOwner_AgentURI             → "viking://agent/a/u/bot1/" → ("a","u","bot1",nil)
TestResolveOwner_ResourcesURI         → "viking://resources/" → ("","","",nil)
TestIsAccessible_Root_Always          → RoleRoot → true for any URI
TestIsAccessible_Admin_SameAccount    → RoleAdmin, same account → true
TestIsAccessible_Admin_OtherAccount   → RoleAdmin, different account → false
TestIsAccessible_User_OwnNamespace    → RoleUser, own user URI → true
TestIsAccessible_User_OtherUser       → RoleUser, other user URI → false
TestIsAccessible_Bot_OwnAgent         → RoleBot, own agent URI → true
TestIsAccessible_Resources_AnyUser    → RoleUser, resources URI same account → true
TestToAbstractURI                     → "foo.md" → "foo.md.abstract.md"
TestToOverviewURI                     → "foo.md" → "foo.md.overview.md"
TestContextType_RootURIs_Memory       → ContextTypeMemory → ["viking://user/","viking://agent/"]
TestContextType_RootURIs_Resource     → ContextTypeResource → ["viking://resources/"]
TestOpenVikingError_ErrorString       → formatted correctly with code + message
TestOpenVikingError_Unwrap            → returns Cause
TestFromContext_WithContext_RoundTrip → inject and retrieve RequestContext
```

---

## Cấu trúc thư mục kết quả

```
pkg/viking/
├── uri.go
├── uri_test.go
├── identity.go
├── identity_test.go
├── context.go
├── context_test.go
├── errors.go
└── errors_test.go
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./pkg/viking/...
go test ./pkg/viking/... -v -count=1
```

Tất cả tests phải **PASS** và không có compilation errors.

---

## Ghi chú triển khai

- Package name: `package viking`
- Module path: `vnp-memory/pkg/viking` (dựa trên `go.work`)
- Không import bất kỳ package nội bộ nào (`pkg/`, `services/`, v.v.)
- Chỉ import standard library + không có third-party deps
- Unexported context key `ctxKey{}` để tránh collision
