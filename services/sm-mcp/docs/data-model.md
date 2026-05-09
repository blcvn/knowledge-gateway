---
id: DOC-S04
service: sm-mcp
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-mcp — Data Model

> **Database**: PostgreSQL (sessions) + In-Memory (tool registry)

## Tables

### mcp_sessions

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| session_id | UUID | PK | Session identifier |
| org_id | VARCHAR(36) | NOT NULL | Organization |
| user_id | VARCHAR(36) | NOT NULL | User |
| api_key_id | UUID | FK → api_keys | Auth key |
| created_at | TIMESTAMPTZ | NOT NULL | Session start |
| last_active_at | TIMESTAMPTZ | NOT NULL | Last activity |
| expires_at | TIMESTAMPTZ | | Session expiry (24h default) |

## Tool Registry (In-Memory)

| Tool Name | Input Schema | Target Service |
|-----------|-------------|---------------|
| add_memory | content, metadata?, containerTags? | sm-document |
| search_memory | q, limit?, containerTags? | sm-search |
| get_profile | — | sm-profile |
| list_documents | page?, limit? | sm-document |
| rag_query | q, limit? | sm-search |

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_session_org | org_id | List by org |
| idx_session_expiry | expires_at | Session cleanup |

## Note

sm-mcp is primarily a **stateless gateway** to other Supermemory services. Only session state is persisted.
