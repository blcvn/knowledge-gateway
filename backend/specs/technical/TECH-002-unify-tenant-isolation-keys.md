---
id: TECH-002
title: Unify tenant isolation keys across engines
service: pkg/tenant
version: 1.0.0
status: Ready
priority: P2
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
risk_level: Medium
rollback_plan: Revert to per-engine tenant extraction middleware
---

## Mô Tả Thay Đổi

Thống nhất 6 tenant isolation keys (tenant_id, group_id, project_id, account_id, project_uuid, org_id) thành 1 canonical `tenant_id` với engine-specific aliases resolved tại Gateway.

## Lý Do

- 6 engines dùng 6 tên isolation key khác nhau → confusing, inconsistent
- Gateway đã resolve tenant từ JWT/APIKey → nên map một lần, propagate unified

## Các Bước Thực Hiện

1. Implement `pkg/tenant/resolver.go` với `TenantContext` struct
2. Update Gateway middleware: resolve `tenant_id` → set engine aliases
3. Update each engine's middleware: read alias from TenantContext instead of raw metadata
4. Update `vnp-platform` admin: manage mappings tenant_id ↔ engine keys
5. Integration test: verify isolation across all engines

## Verification Checklist

- [ ] Gateway sets `x-tenant-id` + engine aliases in gRPC metadata
- [ ] Each engine reads correct alias from TenantContext
- [ ] Multi-tenant queries isolated correctly
- [ ] No data leakage across tenants in any engine
