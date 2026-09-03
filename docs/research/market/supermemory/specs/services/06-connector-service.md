# 06 — Connector Service

> **gRPC**: 9005 | **Health**: 9085

---

## 1. Purpose

External data synchronization: OAuth-based connectors cho Google Drive, Gmail, Notion, OneDrive, GitHub, và Web Crawler. Manages connection lifecycle, periodic sync (cron 4h), webhook handling, và document ingestion delegation.

---

## 2. Clean Architecture

```
services/connector-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Connection, SyncJob, SyncLog
│   │   ├── value_object.go     # Provider, ConnectionStatus, OAuthTokens
│   │   ├── event.go            # ConnectionSynced, ConnectionFailed
│   │   └── errors.go           # ErrProviderNotSupported, ErrTokenExpired
│   ├── usecase/
│   │   ├── create_connection.go  # OAuth flow initiation
│   │   ├── complete_oauth.go     # Code exchange → store tokens
│   │   ├── sync_connection.go    # Fetch documents from provider
│   │   ├── list_connections.go   # List org connections
│   │   ├── delete_connection.go  # Revoke + cleanup
│   │   ├── port/
│   │   │   ├── input.go          # CreateConnectionUC, SyncConnectionUC
│   │   │   └── output.go        # ConnectionRepo, OAuthClient, DocumentCreator,
│   │   │                         # EventPublisher, TokenEncryptor
│   │   └── dto/
│   │       └── connection.go     # CreateConnectionInput, SyncResult
│   ├── adapter/
│   │   ├── grpc/handler.go       # ConnectorServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── connection.go      # Connection CRUD
│   │   │       └── sync_log.go        # Sync history tracking
│   │   ├── provider/                  # Provider-specific adapters
│   │   │   ├── google_drive.go        # Google Drive API v3
│   │   │   ├── gmail.go              # Gmail API
│   │   │   ├── notion.go             # Notion API
│   │   │   ├── onedrive.go           # Microsoft Graph API
│   │   │   ├── github.go             # GitHub API
│   │   │   ├── web_crawler.go        # URL crawling
│   │   │   └── registry.go           # Provider → Adapter mapping
│   │   ├── oauth/
│   │   │   └── client.go             # Generic OAuth2 flow handler
│   │   ├── crypto/
│   │   │   └── token_encryptor.go    # AES-GCM encryption for OAuth tokens at rest
│   │   ├── grpc_client/
│   │   │   └── document.go           # Call Document Service to ingest
│   │   ├── event/
│   │   │   └── publisher.go          # NATS: connection.synced
│   │   └── scheduler/
│   │       └── sync_cron.go          # Cron: sync all connections every 4h
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   └── 001_create_connections.up.sql
└── Dockerfile
```

---

## 3. Connection Lifecycle

```
CreateConnection(provider, redirectURL, containerTags, documentLimit)
       │
       ▼
┌── 1. Generate OAuth URL ──────────────────────┐
│  stateToken = crypto.RandomToken()             │
│  Store ConnectionState (provider, stateToken)  │
│  Return authLink + connectionID                │
└────────────────────┬──────────────────────────┘
                     ▼ User authorizes at provider
┌── 2. OAuth Callback ──────────────────────────┐
│  Validate stateToken (CSRF protection)         │
│  Exchange auth code → access + refresh tokens  │
│  Encrypt tokens (AES-GCM) → store in DB       │
│  Trigger initial sync                          │
└────────────────────┬──────────────────────────┘
                     ▼
┌── 3. Sync (Cron 4h / Webhook / Manual) ───────┐
│  Refresh token if expired                      │
│  Fetch document list from provider             │
│  For each new/updated document:                │
│    → Call Document Service.CreateDocument()     │
│    → Apply containerTags from connection config │
│  Enforce documentLimit (max 10,000)            │
│  Log sync result (success/failure/count)       │
│  Publish connection.synced event               │
└────────────────────────────────────────────────┘
```

---

## 4. Custom OAuth Keys (Enterprise)

```go
type ConnectionConfig struct {
    // Default: platform OAuth keys
    // Enterprise: organization-specific keys
    CustomKeyEnabled  bool
    ClientID          string  // From OrganizationSettings
    ClientSecret      string  // Encrypted at rest
}
```

---

## 5. gRPC Interface

```protobuf
service ConnectorService {
  rpc CreateConnection(CreateConnectionRequest) returns (CreateConnectionResponse);
  rpc CompleteOAuth(CompleteOAuthRequest) returns (CompleteOAuthResponse);
  rpc ListConnections(ListConnectionsRequest) returns (ListConnectionsResponse);
  rpc DeleteConnection(DeleteConnectionRequest) returns (google.protobuf.Empty);
  rpc SyncConnection(SyncConnectionRequest) returns (SyncConnectionResponse);
  rpc GetSyncHistory(GetSyncHistoryRequest) returns (SyncHistoryResponse);
}
```
