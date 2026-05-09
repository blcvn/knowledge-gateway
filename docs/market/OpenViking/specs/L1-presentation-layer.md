# L1 — Presentation Layer

> **Layer**: 1 (Presentation)  
> **Responsibility**: Expose OpenViking capabilities to external consumers via 6 parallel interfaces  
> **Dependencies**: L2 (Service Layer)

---

## 1. Tổng Quan

Layer 1 là tầng tiếp xúc trực tiếp với người dùng, AI Agents, và hệ thống bên ngoài. Nó cung cấp 6 interface song song, tất cả đều delegate xuống L2 (Service Layer).

| Interface | Path | Protocol | Entry Point |
|-----------|------|----------|-------------|
| **REST API** | `server/routers/` | HTTP/JSON (FastAPI) | `http://localhost:1933/api/v1/` |
| **MCP Endpoint** | `server/mcp_endpoint.py` | Streamable HTTP | `http://localhost:1933/mcp` |
| **WebDAV** | `server/routers/webdav.py` | WebDAV protocol | `http://localhost:1933/webdav` |
| **CLI** | `crates/ov_cli/` | Terminal (Rust binary) | `ov <command>` |
| **Python SDK** | `sync_client.py`, `async_client.py` | Python import | `from openviking import OpenViking` |
| **Bot Gateway** | `bot/vikingbot/` | Multi-channel | `http://localhost:18790` |

---

## 2. FastAPI Application (`server/app.py`)

### 2.1 Application Factory

```python
def create_app(config: ServerConfig) -> FastAPI:
    app = FastAPI(title="OpenViking", lifespan=_lifespan)
    # Register 17 routers
    # Configure middleware (CORS, timing, error mapping)
    # Register exception handlers
    # Mount MCP sub-app at /mcp
    # Mount WebDAV at /webdav
    return app
```

### 2.2 Middleware Stack (Processing Order)

| Order | Middleware | Chức năng |
|-------|-----------|-----------|
| 1 | CORS | Allow configurable origins |
| 2 | Request Timing | Log request duration + path |
| 3 | Auth Resolver | Resolve identity from headers/keys |
| 4 | Error Mapping | Map domain exceptions → HTTP status |

### 2.3 Exception Handlers

| Exception Type | HTTP Status | Error Code |
|---------------|-------------|------------|
| `InvalidArgumentError` | 400 | `INVALID_ARGUMENT` |
| `UnauthenticatedError` | 401 | `UNAUTHENTICATED` |
| `PermissionDeniedError` | 403 | `PERMISSION_DENIED` |
| `NotFoundError` | 404 | `NOT_FOUND` |
| `AlreadyExistsError` | 409 | `CONFLICT` |
| `FailedPreconditionError` | 412 | `FAILED_PRECONDITION` |
| `ResourceBusyError` | 423 | `RESOURCE_BUSY` |
| `NotInitializedError` | 503 | `UNAVAILABLE` |
| `RequestValidationError` | 422 | `INVALID_ARGUMENT` |

Tất cả trả về JSON chuẩn:
```json
{"error": {"code": "NOT_FOUND", "message": "...", "details": {}}}
```

### 2.4 Lifespan Management

```
Startup:
  1. Init MCP session manager
  2. Create OpenVikingService + initialize()
  3. Set service dependency for routers
  4. (Optional) Start VikingBot subprocess

Shutdown:
  1. Stop VikingBot
  2. Shutdown OpenVikingService
  3. Cleanup MCP sessions
```

---

## 3. REST API — 17 Routers

### 3.1 Router Directory Map

| Router | File | Prefix | Methods | Chức năng |
|--------|------|--------|---------|-----------|
| **filesystem** | `routers/filesystem.py` (7KB) | `/api/v1/filesystem` | GET/POST | `ls`, `tree`, `mkdir`, `rm` |
| **content** | `routers/content.py` (8KB) | `/api/v1/content` | GET/POST | `read`, `write`, `mv`, `cp`, `stat` |
| **search** | `routers/search.py` (8KB) | `/api/v1/search` | POST | `find`, `grep`, `glob` |
| **sessions** | `routers/sessions.py` (11KB) | `/api/v1/sessions` | POST/GET/DELETE | Session CRUD, `commit`, `used` |
| **resources** | `routers/resources.py` (10KB) | `/api/v1/resources` | POST/GET/DELETE | Resource ingestion, status |
| **relations** | `routers/relations.py` (2KB) | `/api/v1/relations` | GET/POST/DELETE | Context relations |
| **admin** | `routers/admin.py` (11KB) | `/api/v1/admin` | POST/GET/DELETE | Account/user/key CRUD |
| **observer** | `routers/observer.py` (3KB) | `/api/v1/observer` | GET | Retrieval stats, replay |
| **privacy_configs** | `routers/privacy_configs.py` (6KB) | `/api/v1/privacy-configs` | GET/POST | Privacy config CRUD |
| **tasks** | `routers/tasks.py` (2KB) | `/api/v1/tasks` | GET | Background task status |
| **system** | `routers/system.py` (6KB) | `/api/v1/system` | GET | Status, wait, debug info |
| **debug** | `routers/debug.py` (3KB) | `/api/v1/debug` | GET/POST | IO recording, diagnostics |
| **bot** | `routers/bot.py` (9KB) | `/api/v1/bot` | POST/GET | VikingBot lifecycle |
| **pack** | `routers/pack.py` (3KB) | `/api/v1/pack` | POST | Context packing/export |
| **maintenance** | `routers/maintenance.py` (4KB) | `/api/v1/maintenance` | POST | Storage maintenance |
| **stats** | `routers/stats.py` (3KB) | `/api/v1/stats` | GET | Usage statistics |
| **metrics** | `routers/metrics.py` (1KB) | `/metrics` | GET | Prometheus metrics |

### 3.2 Key Endpoints (Detail)

**Filesystem:**
```
GET  /api/v1/filesystem/ls?uri=viking://resources/
GET  /api/v1/filesystem/tree?uri=viking://resources/&level_limit=3
POST /api/v1/filesystem/mkdir    {"uri": "viking://resources/new_dir"}
POST /api/v1/filesystem/rm       {"uri": "viking://resources/old", "recursive": true}
```

**Content:**
```
GET  /api/v1/content/read?uri=viking://resources/file.md
POST /api/v1/content/write       {"uri": "...", "content": "..."}
POST /api/v1/content/mv          {"old_uri": "...", "new_uri": "..."}
GET  /api/v1/content/stat?uri=viking://resources/file.md
```

**Search:**
```
POST /api/v1/search/find         {"query": "auth flow", "limit": 10}
POST /api/v1/search/grep         {"uri": "viking://", "pattern": "TODO"}
POST /api/v1/search/glob         {"pattern": "**/*.py", "uri": "viking://resources/"}
```

**Sessions:**
```
POST   /api/v1/sessions                      Create session
GET    /api/v1/sessions/{id}                 Get session info
POST   /api/v1/sessions/{id}/messages        Add messages
POST   /api/v1/sessions/{id}/used            Record context usage
POST   /api/v1/sessions/{id}/commit          Trigger 2-phase commit
DELETE /api/v1/sessions/{id}                 Delete session
```

---

## 4. MCP Endpoint (`server/mcp_endpoint.py`)

**Protocol:** MCP Streamable HTTP (2025-03-26 spec)  
**Path:** `/mcp` (mounted as ASGI sub-app)  
**Framework:** FastMCP

### 4.1 Tool List (9 tools)

| Tool | Input | Output | Mô tả |
|------|-------|--------|--------|
| `search` | query, target_uri, limit, min_score | Ranked results + URI + abstract | Semantic search |
| `read` | uris (string\|list) | Full content | Batch-capable read |
| `list` | uri, recursive | Directory entries | Directory listing |
| `store` | messages [{role, content}] | Commit result | Auto session + commit |
| `add_resource` | path (URL), description | Task ID | Async resource ingestion |
| `grep` | uri, pattern(s), case_insensitive | Line matches | Regex content search |
| `glob` | pattern, uri, node_limit | URI matches | Filename glob |
| `forget` | uri | Deletion result | Permanent delete |
| `health` | — | Server status | Health check |

### 4.2 Identity Propagation

```python
class _IdentityASGIMiddleware:
    """Reuse REST auth for MCP requests."""
    async def __call__(self, scope, receive, send):
        identity = await resolve_identity(request, auth_config)
        ctx = RequestContext(user=identity.user, role=identity.role)
        token = _mcp_ctx.set(ctx)  # contextvars
        await self.app(scope, receive, send)
        _mcp_ctx.reset(token)
```

Tất cả MCP tools gọi `_get_ctx()` để lấy identity từ `contextvars`.

---

## 5. Authentication (`server/auth.py`)

### 5.1 Three Auth Modes

| Mode | Config | Mechanism | Default Role |
|------|--------|-----------|--------------|
| `DEV` | `auth_mode: "dev"` | No auth, localhost only | ROOT |
| `API_KEY` | `auth_mode: "api_key"` | Root key + per-user keys | Per-key role |
| `TRUSTED` | `auth_mode: "trusted"` | Trust gateway headers | Header-based |

### 5.2 Identity Resolution Flow

```
Request arrives
  │
  ├── DEV mode: → ROOT, account/user from headers or "default"
  │
  ├── API_KEY mode:
  │   ├── Extract key from X-Api-Key or Authorization: Bearer
  │   ├── Root key match? → ROOT, override from headers
  │   ├── Admin key match? → ADMIN, locked to account
  │   └── User key match? → USER, locked to account+user
  │
  └── TRUSTED mode:
      ├── Optional root key validation
      ├── Read X-OpenViking-Account, X-OpenViking-User
      └── Lookup role via APIKeyManager (if available)
```

### 5.3 RBAC Enforcement

```python
# Dependency injection style
@router.post("/admin/accounts")
async def create_account(
    ctx: RequestContext = Depends(require_role(Role.ROOT))
):
    ...
```

---

## 6. CLI (`crates/ov_cli/`)

**Language:** Rust  
**Binary:** `ov`  
**Transport:** HTTP → OpenViking server

| Command | Chức năng |
|---------|-----------|
| `ov status` | Server health check |
| `ov add-resource <path/url>` | Resource ingestion |
| `ov ls <uri>` | Directory listing |
| `ov tree <uri> -L N` | Tree view with depth |
| `ov find <query>` | Semantic search |
| `ov grep <pattern> --uri <uri>` | Regex content search |
| `ov read <uri>` | Read file content |
| `ov chat` | Interactive VikingBot chat |

---

## 7. Python SDK

### 7.1 Sync Client (`sync_client.py`, 12KB)

```python
from openviking import OpenViking
client = OpenViking(url="http://localhost:1933")

client.add_resource("https://github.com/org/repo")
results = client.find("authentication flow", limit=5)
session = client.create_session()
session.add_message("user", "How does auth work?")
session.commit()
```

### 7.2 Async Client (`async_client.py`, 20KB)

```python
from openviking import AsyncOpenViking
client = AsyncOpenViking(url="http://localhost:1933")

results = await client.find("database schema", limit=10)
content = await client.read("viking://resources/README.md")
```

---

## 8. Key Files

| File | Size | Chức năng |
|------|------|-----------|
| `server/app.py` | 17KB | Application factory, middleware, exception handlers |
| `server/bootstrap.py` | 15KB | Server startup, config, VikingBot lifecycle |
| `server/config.py` | 10KB | ServerConfig schema, auth mode config |
| `server/auth.py` | 14KB | Authentication + RBAC middleware |
| `server/mcp_endpoint.py` | 13KB | MCP Streamable HTTP, 9 tools |
| `server/routers/__init__.py` | 2KB | Router registration |
| `sync_client.py` | 12KB | Synchronous Python SDK |
| `async_client.py` | 20KB | Asynchronous Python SDK |
