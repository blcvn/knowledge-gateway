# L3 — Core Domain Layer

> **Layer**: 3 (Domain Model)  
> **Responsibility**: Define domain types, namespace rules, and access control  
> **Dependencies**: L5 (Infrastructure — URI resolution helpers)

---

## 1. Tổng Quan

Layer 3 chứa domain model thuần túy — không phụ thuộc vào storage hay AI models. Định nghĩa các khái niệm cốt lõi: Context, Namespace, ContextType, và cơ chế access control.

**Path:** `openviking/core/`, `openviking/server/identity.py`

| Module | File | Chức năng |
|--------|------|-----------|
| **Context** | `core/context.py` (261 lines) | Primary record type |
| **Namespace** | `core/namespace.py` (337 lines) | URI resolution + ownership |
| **Directories** | `core/directories.py` | Bootstrap root dirs |
| **URI Validation** | `core/uri_validation.py` | Input sanitization |
| **Identity** | `server/identity.py` | RequestContext, Role, AuthMode |
| **Exceptions** | `openviking_cli/exceptions.py` | Domain error hierarchy |

---

## 2. Context — Primary Record Type

**File:** `core/context.py` (261 lines)

### 2.1 Data Model

```python
@dataclass
class Context:
    uri: str                    # viking:// URI (primary key)
    parent_uri: str             # Parent directory URI
    context_type: ContextType   # MEMORY | RESOURCE | SKILL | SESSION
    level: int                  # 0=Abstract, 1=Overview, 2=Detail
    owner_account_id: str       # Tenant account
    owner_user_id: str          # User within account
    owner_agent_id: str         # Agent within user scope
    abstract: str               # L0 summary (~100 tokens)
    category: str               # Sub-classification
    active_count: int           # Usage counter (hotness)
    created_at: datetime
    updated_at: datetime
    meta: Dict[str, Any]        # Extensible metadata
```

### 2.2 ContextType Enum

| Type | URI Root | Mô tả |
|------|----------|--------|
| `MEMORY` | `user/*/memories/`, `agent/*/memories/` | Long-term extracted memories |
| `RESOURCE` | `resources/` | Project documents, code repos, web pages |
| `SKILL` | `agent/*/skills/` | Agent tools and capabilities |
| `SESSION` | `session/` | Active conversation data |

### 2.3 ContextLevel

| Level | File | Token Budget | Mô tả |
|-------|------|-------------|--------|
| 0 | `.abstract.md` | ~100 | "What is this?" — quick relevance check |
| 1 | `.overview.md` | ~2,000 | Key points + usage guidance |
| 2 | (original file) | Full | Complete content |

### 2.4 Context Lifecycle

```
Resource Ingestion:
  parse_file → create Context(level=2) → generate L1 (.overview.md)
                                        → generate L0 (.abstract.md)
  → embed dense+sparse vectors → upsert to VikingDB

Memory Extraction:
  session_commit → VLM extract → create Context(type=MEMORY, level=2)
                                → generate L0 summary
  → embed → upsert

Usage Tracking:
  search_result_used → increment active_count → recalculate hotness
```

---

## 3. Namespace — URI Resolution

**File:** `core/namespace.py` (337 lines)

### 3.1 URI Structure

```
viking://{space}/{account_id}/{user_id}/{agent_id}/{...path}
```

### 3.2 Canonical Roots

| Root URI | Space | Owner |
|----------|-------|-------|
| `viking://resources/` | Shared | Account-level |
| `viking://user/{acct}/{user}/` | User | Specific user |
| `viking://agent/{acct}/{user}/{agent}/` | Agent | Specific agent |
| `viking://session/{session_id}/` | Session | Session creator |
| `viking://temp/` | Temp | ROOT only (write) |

### 3.3 URI Canonicalization

```python
def canonicalize_uri(uri: str) -> str:
    """
    Normalize URI to canonical form:
    - Ensure viking:// prefix
    - Remove trailing slashes (for files)
    - Collapse double slashes
    - Validate namespace structure
    """
```

### 3.4 Ownership Resolution

```python
def resolve_owner(uri: str) -> tuple[str, str, str]:
    """
    Extract (account_id, user_id, agent_id) from URI.

    viking://user/{acct}/{user}/... → (acct, user, "")
    viking://agent/{acct}/{user}/{agent}/... → (acct, user, agent)
    viking://resources/... → ("", "", "")
    viking://session/... → from session metadata
    """
```

### 3.5 Accessibility Check

```python
def is_accessible(uri: str, ctx: RequestContext) -> bool:
    """
    Check if user can access URI based on role + namespace rules.

    ROOT → always True
    ADMIN → own account + managed users
    USER → own user/agent space + shared resources
    """
```

---

## 4. Identity & Access Control

**File:** `server/identity.py`

### 4.1 RequestContext

```python
@dataclass
class RequestContext:
    user: UserIdentifier     # (account_id, user_id, agent_id)
    role: Role               # ROOT | ADMIN | USER
    namespace_policy: str    # Namespace access policy
    api_key_id: str          # Key used for auth (if any)
```

### 4.2 Role Hierarchy

```
ROOT
 └── ADMIN (account-scoped)
      └── USER (user-scoped)
```

| Role | Admin APIs | Own Account Data | Other Account Data | Shared Resources |
|------|-----------|------------------|-------------------|-----------------|
| ROOT | ✅ | ✅ | ✅ | ✅ |
| ADMIN | ❌ | ✅ | ❌ | ✅ (read) |
| USER | ❌ | ✅ (own user only) | ❌ | ✅ (read) |

### 4.3 AuthMode Enum

| Mode | Config Value | Mô tả |
|------|-------------|--------|
| `DEV` | `"dev"` | No auth, ROOT, localhost only |
| `API_KEY` | `"api_key"` | Root key + per-user keys |
| `TRUSTED` | `"trusted"` | Trust gateway headers |

### 4.4 UserIdentifier

```python
@dataclass
class UserIdentifier:
    account_id: str     # Tenant
    user_id: str        # User within tenant
    agent_id: str       # Agent within user (optional)

    @staticmethod
    def the_default_user() -> "UserIdentifier":
        return UserIdentifier("default", "default", "")
```

---

## 5. DirectoryInitializer

**File:** `core/directories.py`

Bootstraps the filesystem structure on first startup:

```
initialize():
  1. Create root directories:
     viking://resources/
     viking://user/
     viking://agent/
     viking://session/
     viking://temp/

  2. Load built-in skills:
     Copy from package data → viking://agent/.../skills/

  3. Ensure per-user directories for default user:
     viking://user/{default_acct}/{default_user}/memories/
     viking://agent/{default_acct}/{default_user}/{default_agent}/
```

---

## 6. Exception Hierarchy

```
OpenVikingError (base)
├── InvalidArgumentError      # 400 — bad input
├── UnauthenticatedError      # 401 — no/invalid credentials
├── PermissionDeniedError     # 403 — insufficient role
├── NotFoundError             # 404 — URI not found
├── AlreadyExistsError        # 409 — duplicate resource
├── FailedPreconditionError   # 412 — state mismatch
├── ResourceBusyError         # 423 — locked by another op
├── NotInitializedError       # 503 — service not ready
└── InternalError             # 500 — unexpected failure
```

Mỗi exception mang `code`, `message`, và optional `details` dict.
