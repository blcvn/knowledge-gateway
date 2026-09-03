# Solution: SOL-SM-005 — External Connector Service

**CR ID:** CR-SM-005  
**Solution ID:** SOL-SM-005  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/connector-service/` với OAuth2 lifecycle đầy đủ, 6 provider adapters, AES-GCM token encryption, và cron job sync mỗi 4 giờ. Tích hợp trực tiếp với Document Service để ingest tài liệu.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `sm-connector` service | `apps/memory/internal/bootstrap/` | Đã có entry nhưng minimal |
| NATS events `sm.connector.synced` | Architecture docs | Defined nhưng chưa publish |
| MinIO/S3 | Infrastructure | Dùng cho object storage |

### Gap phân tích

- Không có OAuth2 flow thực sự
- Không có provider adapters (Google Drive, Notion, etc.)
- Không có cron job scheduler
- Không có AES-GCM token encryption
- Không có webhook support

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service Mới

```
services/connector-service/
├── internal/
│   ├── domain/
│   │   ├── connection.go      # Connection entity
│   │   ├── provider.go        # Provider interface + Registry
│   │   └── repository.go      # ConnectionRepository port
│   ├── usecase/
│   │   ├── create_connection.go   # OAuth URL generation
│   │   ├── complete_oauth.go      # Token exchange + store
│   │   ├── sync_connection.go     # Sync logic (cron + manual)
│   │   ├── delete_connection.go   # Cleanup
│   │   └── list_history.go        # Sync history
│   ├── adapter/
│   │   ├── grpc/                  # ConnectorService gRPC server
│   │   ├── webhook/               # Google Drive + Notion webhooks
│   │   └── cron/
│   │       └── sync_scheduler.go  # Cron mỗi 4 giờ
│   └── infra/
│       ├── postgres/
│       │   └── connection_repo.go
│       ├── crypto/
│       │   └── aes_gcm.go         # AES-GCM token encryption
│       └── provider/
│           ├── google_drive.go
│           ├── gmail.go
│           ├── notion.go
│           ├── onedrive.go
│           ├── github.go
│           └── web_crawler.go
```

### 3.2. Domain Model

```go
// services/connector-service/internal/domain/connection.go

package domain

import "time"

type Provider string

const (
    ProviderGoogleDrive Provider = "google_drive"
    ProviderGmail       Provider = "gmail"
    ProviderNotion      Provider = "notion"
    ProviderOneDrive    Provider = "onedrive"
    ProviderGitHub      Provider = "github"
    ProviderWeb         Provider = "web"
)

type ConnectionStatus string

const (
    StatusPending      ConnectionStatus = "pending"       // OAuth not yet completed
    StatusActive       ConnectionStatus = "active"        // Syncing normally
    StatusSyncing      ConnectionStatus = "syncing"       // Sync in progress
    StatusError        ConnectionStatus = "error"         // Sync failed
    StatusReauthNeeded ConnectionStatus = "reauth_needed" // Token expired + refresh failed
    StatusDisabled     ConnectionStatus = "disabled"      // User disabled
)

type Connection struct {
    ID             string
    OrgID          string
    UserID         string
    Provider       Provider
    Status         ConnectionStatus
    StateToken     string    // CSRF token (cleared after OAuth complete)

    // Encrypted tokens (AES-GCM)
    AccessToken    []byte    // AES-GCM encrypted
    RefreshToken   []byte    // AES-GCM encrypted
    TokenExpiresAt *time.Time

    // Config
    DocumentLimit  int       // Default 10,000
    ContainerTags  []string  // Applied to all imported docs
    CustomKey      *ConnectionConfig // Enterprise: custom OAuth client_id/secret
    WebhookID      *string   // Provider-side webhook registration ID

    // Stats
    LastSyncAt     *time.Time
    LastSyncError  *string
    DocumentCount  int

    Metadata       map[string]any
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type ConnectionConfig struct {
    ClientID     string
    ClientSecret string // AES-GCM encrypted
    Scopes       []string
}

type SyncHistory struct {
    ID              string
    ConnectionID    string
    StartedAt       time.Time
    CompletedAt     *time.Time
    DocumentsAdded  int
    DocumentsFailed int
    Error           *string
    Trigger         string // "cron" | "manual" | "webhook"
}
```

### 3.3. Provider Interface

```go
// services/connector-service/internal/domain/provider.go

type ProviderAdapter interface {
    // OAuth
    GetAuthURL(stateToken, redirectURL string, customKey *ConnectionConfig) string
    ExchangeCode(ctx context.Context, code string, redirectURL string, customKey *ConnectionConfig) (*TokenSet, error)
    RefreshToken(ctx context.Context, refreshToken string, customKey *ConnectionConfig) (*TokenSet, error)

    // Sync
    ListDocuments(ctx context.Context, accessToken string, since *time.Time) ([]RemoteDocument, error)
    GetDocumentContent(ctx context.Context, accessToken string, docID string) (content []byte, mimeType string, err error)

    // Webhook (optional)
    RegisterWebhook(ctx context.Context, accessToken, callbackURL string) (webhookID string, err error)
    UnregisterWebhook(ctx context.Context, accessToken, webhookID string) error

    Supports(provider Provider) bool
}

type TokenSet struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

type RemoteDocument struct {
    ID          string
    Title       string
    ContentURL  string
    MimeType    string
    ModifiedAt  time.Time
    Size        int64
}
```

### 3.4. AES-GCM Token Encryption

```go
// services/connector-service/internal/infra/crypto/aes_gcm.go

package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "io"
)

type TokenEncryptor struct {
    key []byte // 32 bytes = AES-256
}

func NewTokenEncryptor(keyHex string) (*TokenEncryptor, error) {
    key, err := hex.DecodeString(keyHex)
    if err != nil || len(key) != 32 {
        return nil, ErrInvalidKey
    }
    return &TokenEncryptor{key: key}, nil
}

func (e *TokenEncryptor) Encrypt(plaintext string) ([]byte, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil { return nil, err }

    gcm, err := cipher.NewGCM(block)
    if err != nil { return nil, err }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return ciphertext, nil
}

func (e *TokenEncryptor) Decrypt(ciphertext []byte) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil { return "", err }

    gcm, err := cipher.NewGCM(block)
    if err != nil { return "", err }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", ErrInvalidCiphertext
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil { return "", err }

    return string(plaintext), nil
}
```

Key management: Lấy từ env `VNP_MEMORY_CONNECTOR_ENCRYPTION_KEY` (32-byte hex). Rotate keys qua re-encryption job.

### 3.5. OAuth2 Flow Implementation

```go
// services/connector-service/internal/usecase/create_connection.go

func (uc *CreateConnectionUseCase) Execute(ctx context.Context, req CreateConnectionRequest) (*CreateConnectionResponse, error) {
    // 1. Generate CSRF state token
    stateToken := generateSecureToken(32)

    // 2. Create pending connection record
    conn := &Connection{
        OrgID:         req.OrgID,
        UserID:        req.UserID,
        Provider:      req.Provider,
        Status:        StatusPending,
        StateToken:    stateToken,
        DocumentLimit: defaultLimit(req.DocumentLimit, 10000),
        ContainerTags: req.ContainerTags,
        CustomKey:     req.CustomKey,
    }
    uc.repo.Create(ctx, conn)

    // 3. Get OAuth URL from provider adapter
    adapter := uc.registry.Get(req.Provider)
    authURL := adapter.GetAuthURL(stateToken, req.RedirectURL, req.CustomKey)

    return &CreateConnectionResponse{
        ConnectionID: conn.ID,
        AuthLink:     authURL,
    }, nil
}

// complete_oauth.go
func (uc *CompleteOAuthUseCase) Execute(ctx context.Context, req CompleteOAuthRequest) error {
    // 1. Load pending connection + validate state token (CSRF)
    conn, err := uc.repo.FindByStateToken(ctx, req.StateToken)
    if err != nil { return ErrInvalidState }

    // 2. Exchange code for tokens
    adapter := uc.registry.Get(conn.Provider)
    tokens, err := adapter.ExchangeCode(ctx, req.Code, req.RedirectURL, conn.CustomKey)
    if err != nil { return err }

    // 3. Encrypt tokens with AES-GCM
    encAccess, _ := uc.encryptor.Encrypt(tokens.AccessToken)
    encRefresh, _ := uc.encryptor.Encrypt(tokens.RefreshToken)

    // 4. Update connection
    conn.Status = StatusActive
    conn.AccessToken = encAccess
    conn.RefreshToken = encRefresh
    conn.TokenExpiresAt = &tokens.ExpiresAt
    conn.StateToken = "" // Clear CSRF token
    uc.repo.Update(ctx, conn)

    // 5. Trigger initial sync (async)
    go uc.syncUC.Execute(context.Background(), conn.ID, "initial")

    // 6. Register webhook (optional, provider-dependent)
    if adapter.SupportsWebhook() {
        webhookID, _ := adapter.RegisterWebhook(ctx, tokens.AccessToken, uc.webhookCallbackURL)
        conn.WebhookID = &webhookID
        uc.repo.Update(ctx, conn)
    }

    return nil
}
```

### 3.6. Sync Logic

```go
// services/connector-service/internal/usecase/sync_connection.go

func (uc *SyncConnectionUseCase) Execute(ctx context.Context, connID, trigger string) error {
    conn, _ := uc.repo.Get(ctx, connID)

    // Start sync history record
    history := &SyncHistory{ConnectionID: connID, StartedAt: time.Now(), Trigger: trigger}
    uc.historyRepo.Create(ctx, history)

    // Update status
    conn.Status = StatusSyncing
    uc.repo.Update(ctx, conn)

    // Decrypt access token
    accessToken, err := uc.encryptor.Decrypt(conn.AccessToken)
    if err != nil { return uc.handleSyncError(ctx, conn, history, err) }

    // Refresh token if expired
    if conn.TokenExpiresAt != nil && time.Now().After(*conn.TokenExpiresAt) {
        refreshToken, _ := uc.encryptor.Decrypt(conn.RefreshToken)
        adapter := uc.registry.Get(conn.Provider)
        newTokens, err := adapter.RefreshToken(ctx, refreshToken, conn.CustomKey)
        if err != nil {
            conn.Status = StatusReauthNeeded
            uc.repo.Update(ctx, conn)
            return ErrReauthNeeded
        }
        encAccess, _ := uc.encryptor.Encrypt(newTokens.AccessToken)
        conn.AccessToken = encAccess
        conn.TokenExpiresAt = &newTokens.ExpiresAt
        accessToken = newTokens.AccessToken
        uc.repo.Update(ctx, conn)
    }

    // Fetch documents from provider (since last sync)
    adapter := uc.registry.Get(conn.Provider)
    remoteDocs, err := adapter.ListDocuments(ctx, accessToken, conn.LastSyncAt)
    if err != nil { return uc.handleSyncError(ctx, conn, history, err) }

    added, failed := 0, 0
    for _, remoteDoc := range remoteDocs {
        // Enforce document limit
        if conn.DocumentCount+added >= conn.DocumentLimit {
            slog.Warn("connector: document limit reached",
                "connection_id", connID,
                "limit", conn.DocumentLimit,
            )
            break
        }

        // Fetch content
        content, mimeType, err := adapter.GetDocumentContent(ctx, accessToken, remoteDoc.ID)
        if err != nil { failed++; continue }

        // Ingest via Document Service
        _, err = uc.docClient.CreateDocument(ctx, CreateDocumentRequest{
            OrgID:         conn.OrgID,
            UserID:        conn.UserID,
            ConnectionID:  &conn.ID,
            Title:         &remoteDoc.Title,
            Content:       string(content),
            URL:           &remoteDoc.ContentURL,
            Type:          mimeTypeToDocType(mimeType),
            ContainerTags: conn.ContainerTags,
            CustomID:      &remoteDoc.ID, // Prevents re-ingestion on next sync
        })
        if err != nil {
            if errors.Is(err, ErrDuplicateCustomID) { continue } // Already ingested
            failed++
            continue
        }
        added++
    }

    // Complete sync
    now := time.Now()
    history.CompletedAt = &now
    history.DocumentsAdded = added
    history.DocumentsFailed = failed
    uc.historyRepo.Update(ctx, history)

    conn.Status = StatusActive
    conn.LastSyncAt = &now
    conn.DocumentCount += added
    uc.repo.Update(ctx, conn)

    uc.publisher.Publish(ctx, "sm.connector.synced", ConnectorSyncedEvent{
        ConnectionID: conn.ID,
        OrgID:        conn.OrgID,
        Added:        added,
    })

    return nil
}

func mimeTypeToDocType(mimeType string) DocumentType {
    switch {
    case strings.Contains(mimeType, "pdf"):   return DocTypePDF
    case strings.Contains(mimeType, "image"): return DocTypeImage
    default:                                   return DocTypeText
    }
}
```

### 3.7. Cron Scheduler (4-giờ)

```go
// services/connector-service/internal/adapter/cron/sync_scheduler.go

type SyncScheduler struct {
    repo   ConnectionRepository
    syncUC *SyncConnectionUseCase
}

func (s *SyncScheduler) Start(ctx context.Context) {
    // Chạy mỗi 4 giờ
    ticker := time.NewTicker(4 * time.Hour)
    for {
        select {
        case <-ticker.C:
            s.runSync(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (s *SyncScheduler) runSync(ctx context.Context) {
    // Lấy tất cả active connections
    connections, _ := s.repo.ListActive(ctx)

    // Sync concurrently (max 10 parallel)
    sem := make(chan struct{}, 10)
    var wg sync.WaitGroup

    for _, conn := range connections {
        wg.Add(1)
        sem <- struct{}{}
        go func(connID string) {
            defer wg.Done()
            defer func() { <-sem }()
            s.syncUC.Execute(ctx, connID, "cron")
        }(conn.ID)
    }
    wg.Wait()
}
```

---

## 4. Database Schema

```sql
CREATE TABLE connections (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL,
    user_id          UUID NOT NULL,
    provider         TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    state_token      TEXT,                              -- CSRF, cleared after OAuth
    access_token     BYTEA,                             -- AES-GCM encrypted
    refresh_token    BYTEA,                             -- AES-GCM encrypted
    token_expires_at TIMESTAMPTZ,
    document_limit   INT NOT NULL DEFAULT 10000,
    container_tags   TEXT[] DEFAULT '{}',
    custom_key       JSONB,                             -- AES-GCM encrypted fields
    webhook_id       TEXT,
    last_sync_at     TIMESTAMPTZ,
    last_sync_error  TEXT,
    document_count   INT NOT NULL DEFAULT 0,
    metadata         JSONB DEFAULT '{}',
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE sync_history (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id     UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    started_at        TIMESTAMPTZ NOT NULL,
    completed_at      TIMESTAMPTZ,
    documents_added   INT DEFAULT 0,
    documents_failed  INT DEFAULT 0,
    error             TEXT,
    trigger           TEXT NOT NULL  -- cron | manual | webhook | initial
);

CREATE INDEX idx_conn_org ON connections(org_id);
CREATE INDEX idx_conn_status ON connections(status) WHERE status = 'active';
CREATE INDEX idx_sync_hist_conn ON sync_history(connection_id, started_at DESC);
```

---

## 5. API Endpoints (Gateway)

```
POST   /api/v1/connections/:provider   → CreateConnection (trả về {authLink, connectionId})
GET    /api/v1/oauth/callback          → CompleteOAuth (redirect từ provider)
GET    /api/v1/connections             → ListConnections
DELETE /api/v1/connections/:id         → DeleteConnection (ngắt sync, giữ docs)
POST   /api/v1/connections/:id/sync    → ManualSync
GET    /api/v1/connections/:id/history → SyncHistory (paginated)

POST   /api/v1/webhooks/google-drive   → Google Drive push notification handler
POST   /api/v1/webhooks/notion         → Notion webhook handler
```

---

## 6. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + AES-GCM encryption | 1 ngày |
| **P2** | OAuth2 flow (create + complete) | 2 ngày |
| **P3** | Google Drive + Notion adapters | 3 ngày |
| **P4** | GitHub + OneDrive + Web Crawler adapters | 3 ngày |
| **P5** | Sync logic + Document Service integration | 2 ngày |
| **P6** | Cron scheduler (4-giờ) | 1 ngày |
| **P7** | Webhook handlers (Google Drive + Notion) | 2 ngày |
| **P8** | Gateway integration + REST handlers | 1 ngày |
| **P9** | Tests + Acceptance Criteria | 2 ngày |

**Tổng:** ~17 ngày (Wave 4)

---

## 7. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| Google Drive → OAuth → tài liệu ingest | Full OAuth flow + ListDocuments + CreateDocument |
| 4 giờ sau → cron sync | Cron scheduler ticker 4h |
| Xóa connection → docs giữ nguyên | DeleteConnection chỉ xóa connection record, không cascade docs |
| Token encrypted trong DB | AES-GCM encryption, plaintext chỉ in memory |
| 10,000 docs limit | `conn.DocumentCount >= conn.DocumentLimit` check |
| Custom OAuth keys cho Enterprise | `CustomKey *ConnectionConfig` field + adapter.GetAuthURL(customKey) |
