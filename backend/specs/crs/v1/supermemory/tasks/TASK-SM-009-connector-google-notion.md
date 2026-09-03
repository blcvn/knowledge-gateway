# TASK-SM-009 — services/connector-service: Google Drive & Notion Connectors

**Task ID:** TASK-SM-009  
**Wave:** 4 (Integrations)  
**Solution:** [SOL-SM-005](../solutions/SOL-SM-005-Connector-Service.md)  
**Depends on:** TASK-SM-001 (Auth OrgID), TASK-SM-005 (Document Service CreateDocument)  
**Ước tính:** 5h  
**Priority:** Medium

---

## Mục tiêu

Tạo `services/connector-service/` với:
1. `Connection` entity + OAuth2 token storage (AES-GCM encrypted)
2. Google Drive connector (OAuth2 → file sync)
3. Notion connector (API token → page sync)
4. `SyncJob` cron (incremental sync, every 15 min)
5. Webhook handler cho real-time updates
6. Rate limiting per provider

---

## Công việc cụ thể

### 1. Tạo Domain Model

**`services/connector-service/internal/domain/connection.go`**

```go
type Provider string
const (
    ProviderGoogleDrive Provider = "google_drive"
    ProviderNotion      Provider = "notion"
    ProviderGitHub      Provider = "github"
    ProviderSlack       Provider = "slack"
    ProviderOneDrive    Provider = "onedrive"
)

type Connection struct {
    ID           string
    OrgID        string
    UserID       string
    Provider     Provider
    DisplayName  string
    AccessToken  string    // AES-GCM encrypted (never store plaintext)
    RefreshToken string    // AES-GCM encrypted
    TokenExpiry  time.Time
    Config       ConnectorConfig  // JSONB: provider-specific settings
    Status       ConnectionStatus // "active" | "error" | "syncing" | "paused"
    LastSyncAt   *time.Time
    CreatedAt    time.Time
}

type SyncJob struct {
    ID           string
    ConnectionID string
    Status       SyncStatus // "pending" | "running" | "done" | "failed"
    FilesAdded   int
    FilesUpdated int
    FilesRemoved int
    ErrorMsg     *string
    StartedAt    *time.Time
    CompletedAt  *time.Time
}
```

### 2. Tạo PostgreSQL Schema

**`services/connector-service/migrations/001_create_connectors.sql`**

```sql
CREATE TABLE connections (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL,
    user_id       UUID NOT NULL,
    provider      TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    access_token  TEXT NOT NULL,      -- AES-GCM encrypted
    refresh_token TEXT,               -- AES-GCM encrypted
    token_expiry  TIMESTAMPTZ,
    config        JSONB DEFAULT '{}',
    status        TEXT NOT NULL DEFAULT 'active',
    last_sync_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE sync_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending',
    files_added   INT DEFAULT 0,
    files_updated INT DEFAULT 0,
    files_removed INT DEFAULT 0,
    error_msg     TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE synced_files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    document_id   UUID,                    -- Links to documents table
    provider_file_id TEXT NOT NULL,        -- Google Drive fileId / Notion pageId
    last_modified TIMESTAMPTZ NOT NULL,    -- For incremental sync
    UNIQUE (connection_id, provider_file_id)
);
```

### 3. Implement AES-GCM Token Encryption

**`services/connector-service/internal/infra/crypto/token_vault.go`**

```go
// AES-256-GCM encryption for OAuth tokens
// Key: 32-byte key from config (env: CONNECTOR_ENCRYPTION_KEY)
func EncryptToken(plaintext, keyBase64 string) (ciphertext string, err error) { ... }
func DecryptToken(ciphertext, keyBase64 string) (plaintext string, err error) { ... }
// Key rotation: store key version with ciphertext prefix "v1:{iv}:{ciphertext}"
```

### 4. Implement Google Drive Connector

**`services/connector-service/internal/infra/connector/google_drive.go`**

```go
type GoogleDriveConnector struct {
    docClient DocumentServiceClient  // gRPC → document-service
    tokenVault *TokenVault
}

// OAuth2 flow:
// 1. GET /api/v1/connections/google_drive/auth → redirect to Google OAuth2
// 2. GET /api/v1/connections/google_drive/callback → exchange code → store tokens
func (c *GoogleDriveConnector) GetAuthURL(state string) string

// Full sync (initial) + Incremental sync (delta via lastSyncAt)
// For each file: GET content → CreateDocument via document-service
// Rate limit: 10 req/s (Google API quota)
func (c *GoogleDriveConnector) Sync(ctx context.Context, conn *Connection) (*SyncResult, error)

// Webhook: POST /api/v1/connections/google_drive/webhook
// Validate X-Goog-Channel-Token header
// Trigger incremental sync for changed files
func (c *GoogleDriveConnector) HandleWebhook(w http.ResponseWriter, r *http.Request)

// Token refresh: check TokenExpiry, refresh if < 5 min remaining
func (c *GoogleDriveConnector) refreshTokenIfNeeded(ctx context.Context, conn *Connection) error
```

### 5. Implement Notion Connector

**`services/connector-service/internal/infra/connector/notion.go`**

```go
type NotionConnector struct {
    docClient DocumentServiceClient
    tokenVault *TokenVault
}

// Uses Notion API token (not OAuth2 for simplicity)
// GET /api/v1/connections/notion/auth → prompt for API token (integration token)
// Sync: list all pages → for each page: GET content → CreateDocument
// Rate limit: 3 req/s (Notion API)
func (c *NotionConnector) Sync(ctx context.Context, conn *Connection) (*SyncResult, error)
```

### 6. Implement Sync Cron Job

**`services/connector-service/internal/adapter/cron/sync_job.go`**

```go
// Run every 15 minutes: find active connections with lastSyncAt > 15 min ago
// For each connection: create SyncJob → trigger connector.Sync
// Concurrency: max 3 concurrent sync jobs (via goroutine pool)
func (j *SyncCronJob) Run(ctx context.Context)
```

### 7. REST Endpoints

```
POST   /api/v1/connections/{provider}/auth        → InitiateAuth (requires connection:create)
GET    /api/v1/connections/{provider}/callback    → OAuthCallback
GET    /api/v1/connections                        → ListConnections
DELETE /api/v1/connections/{id}                   → DeleteConnection
POST   /api/v1/connections/{id}/sync              → TriggerManualSync
GET    /api/v1/connections/{id}/jobs              → ListSyncJobs
POST   /api/v1/connections/{provider}/webhook     → WebhookHandler (no auth)
```

### 8. Tests

- `TestEncryptDecryptToken_Roundtrip`: encrypt → decrypt → same plaintext
- `TestEncryptToken_DifferentCiphertext`: same input → different ciphertext (IV random)
- `TestGoogleDriveConnector_TokenRefresh`: near-expired token → refresh called
- `TestSyncJob_IncrementalSync`: lastModified unchanged → file skipped
- `TestSyncJob_NewFile_CreatesDocument`: new file → DocumentService.CreateDocument called
- `TestSyncCron_MaxConcurrency`: 10 connections → max 3 concurrent goroutines

---

## Acceptance Criteria

- [ ] `go build ./services/connector-service/...` không lỗi
- [ ] EncryptToken(plaintext) → decrypt(ciphertext) = plaintext
- [ ] Same plaintext → different ciphertext each time (AES-GCM random IV)
- [ ] Google Drive OAuth2 redirect URL generated correctly
- [ ] New file in Drive → CreateDocument called with correct OrgID
- [ ] Rate limit: Google connector ≤ 10 req/s
- [ ] Sync job incremental: unmodified file → NOT re-ingested
- [ ] `go test ./services/connector-service/...` pass

---

## Files tạo ra

```
services/connector-service/
├── internal/
│   ├── domain/
│   │   └── connection.go
│   ├── usecase/
│   │   ├── create_connection.go
│   │   ├── delete_connection.go
│   │   ├── list_connections.go
│   │   └── trigger_sync.go
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── connector_server.go
│   │   ├── http/
│   │   │   └── connector_handler.go
│   │   └── cron/
│   │       └── sync_job.go
│   └── infra/
│       ├── postgres/
│       │   ├── connection_repo.go
│       │   └── sync_job_repo.go
│       ├── crypto/
│       │   ├── token_vault.go
│       │   └── token_vault_test.go
│       └── connector/
│           ├── google_drive.go
│           ├── notion.go
│           └── connector_test.go
└── migrations/
    └── 001_create_connectors.sql

gateway/adapter/handler/
└── connector_handler.go
```

## Sau khi hoàn thành

Chạy: `go build ./services/connector-service/... && go test ./services/connector-service/...`
