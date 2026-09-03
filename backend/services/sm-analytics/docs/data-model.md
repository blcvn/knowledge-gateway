---
id: DOC-S04
service: sm-analytics
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-analytics — Data Model

> **Database**: PostgreSQL

## Tables

### api_requests

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Request ID |
| type | VARCHAR(20) | NOT NULL | add/search/fast_search/update/delete/chat/search_v4 |
| org_id | VARCHAR(36) | NOT NULL, INDEX | Organization |
| user_id | VARCHAR(36) | | User |
| key_id | VARCHAR(36) | | API key used |
| status_code | INT | NOT NULL | HTTP status code |
| duration_ms | INT | | Request duration |
| input | JSONB | | Request payload |
| output | JSONB | | Response payload |
| original_tokens | INT | | Input token count |
| final_tokens | INT | | Output token count |
| tokens_saved | INT | | Computed savings |
| cost_saved_usd | DECIMAL(10,4) | | Estimated cost saved |
| model | VARCHAR(100) | | LLM model (chat) |
| provider | VARCHAR(50) | | LLM provider |
| conversation_id | VARCHAR(36) | | Chat session |
| context_modified | BOOL | DEFAULT false | Context was modified |
| metadata | JSONB | | Extra metadata |
| origin | VARCHAR(10) | DEFAULT 'api' | api/mcp/web |
| created_at | TIMESTAMPTZ | NOT NULL, INDEX | |

### daily_aggregates (Materialized View)

| Column | Type | Description |
|--------|------|-------------|
| org_id | VARCHAR(36) | Organization |
| date | DATE | Aggregation date |
| request_type | VARCHAR(20) | Request type |
| count | BIGINT | Total requests |
| total_duration_ms | BIGINT | Sum of durations |
| total_tokens | BIGINT | Sum of tokens |
| total_saved_tokens | BIGINT | Sum of saved tokens |
| total_cost_saved | DECIMAL | Sum of cost saved |

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_req_org_created | (org_id, created_at DESC) | Time-range queries |
| idx_req_org_type | (org_id, type, created_at) | Per-type aggregation |
| idx_req_org_key | (org_id, key_id) | Per-key analytics |
| idx_req_conversation | (conversation_id) | Chat session grouping |
