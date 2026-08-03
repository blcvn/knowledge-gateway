# Architecture Decision Records (ADR)

> Danh sách các quyết định thiết kế quan trọng trong KG Service  
> Format: [MADR](https://adr.github.io/madr/) (Markdown Architectural Decision Records)

---

| ADR | Tiêu đề | Status |
|:---:|:---|:---:|
| [ADR-001](./adr-001-cqrs-3mode.md) | CQRS 3-Mode: PostgreSQL + Graph DB + Vector DB | ✅ Accepted |
| [ADR-002](./adr-002-query-pattern-dsl.md) | Query Pattern DSL thay thế Cypher thô | ✅ Accepted |
| [ADR-003](./adr-003-acl-denormalization.md) | ACL denormalization qua `acl_visible_to` field | ✅ Accepted |
| [ADR-004](./adr-004-outbox-pattern.md) | Outbox Pattern cho cross-store sync | ✅ Accepted |
| [ADR-005](./adr-005-pluggable-backends.md) | Pluggable Backends qua Adapter Interface | ✅ Accepted |
| [ADR-006](./adr-006-domain-agnostic.md) | Domain-Agnostic Service với Ontology Plane | ✅ Accepted |
