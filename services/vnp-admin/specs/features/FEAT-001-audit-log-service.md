---
id: FEAT-001
title: Audit Log Storage and Query Service
service: vnp-admin
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: gateway/SOL-002 (T08)
linked_ux: "ux_spec.md §6.8 Governance Center — Audit Explorer"
---

## Mục Tiêu

Implement audit log storage và query API cho vnp-admin. Mọi hành động governance (tenant CRUD, policy changes, GDPR forget, API key operations) phải được ghi audit log và searchable.

## Scope

### In Scope
- gRPC `AuditService.RecordEvent(AuditEvent)` — ghi audit event
- gRPC `AuditService.QueryEvents(AuditQuery)` — search audit events
- PostgreSQL table `audit_logs` với indexing
- Domain events published to NATS `audit.events.*`

### Out of Scope
- UI rendering (Gateway/Console scope)
- Audit report generation (future)

## Thiết Kế Kỹ Thuật

### Data Model

```sql
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    actor_id    TEXT NOT NULL,
    actor_type  TEXT NOT NULL,          -- "user", "api_key", "system"
    action      TEXT NOT NULL,          -- "tenant.create", "policy.update", "gdpr.forget"
    entity_type TEXT NOT NULL,          -- "tenant", "policy", "user", "memory"
    entity_id   TEXT,
    engine      TEXT,                   -- "cognee", "graphiti", etc.
    details     JSONB,
    result      TEXT NOT NULL,          -- "success", "denied", "error"
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_tenant_time ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_actor ON audit_logs (actor_id);
CREATE INDEX idx_audit_action ON audit_logs (action);
CREATE INDEX idx_audit_engine ON audit_logs (engine);
```

### gRPC API

```protobuf
service AuditService {
    rpc RecordEvent(AuditEvent) returns (AuditEventID);
    rpc QueryEvents(AuditQuery) returns (AuditQueryResult);
}

message AuditEvent {
    string tenant_id = 1;
    string actor_id = 2;
    string actor_type = 3;
    string action = 4;
    string entity_type = 5;
    string entity_id = 6;
    string engine = 7;
    google.protobuf.Struct details = 8;
    string result = 9;
}

message AuditQuery {
    string tenant_id = 1;
    string actor_id = 2;
    string action = 3;
    string entity_type = 4;
    string engine = 5;
    google.protobuf.Timestamp from = 6;
    google.protobuf.Timestamp to = 7;
    string cursor = 8;
    int32 limit = 9;
}
```

## Acceptance Criteria
- [ ] AC-1: RecordEvent stores audit log in PostgreSQL within 50ms
- [ ] AC-2: QueryEvents supports filtering by actor, action, entity, engine, time range
- [ ] AC-3: QueryEvents returns cursor-based pagination
- [ ] AC-4: Audit events published to NATS `audit.events.{action}`
- [ ] AC-5: Retention policy: auto-delete logs older than configured TTL
- [ ] AC-6: Index queries return within 200ms for 1M+ records

## Test Requirements
- Unit tests: Query builder, filter validation
- Integration tests: PostgreSQL CRUD with test container
- Minimum coverage: 80%
