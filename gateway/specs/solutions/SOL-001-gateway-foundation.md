---
id: SOL-001
title: Gateway Foundation — Core Infrastructure Setup
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_cr: TDD-vnp-gateway
approved_by: TBD
---

## Yêu Cầu Gốc

Xây dựng vnp-gateway — Unified API Gateway — làm single entry point cho toàn bộ 35 domain services của VNP Memory. Gateway phải hỗ trợ REST, gRPC, MCP, WebDAV; có auth, rate limiting, circuit breaking.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng
| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| vnp-gateway | New service creation | Cao |
| All 35 services | gRPC client connections | Trung bình |
| vnp-admin | API key/tenant store dependency | Trung bình |

### Breaking Changes
- [ ] API response format thay đổi? — No (new service)
- [ ] Database schema migration cần thiết? — Yes (tenants, api_keys, route_configs)
- [ ] Consumer downstream cần cập nhật? — No

### Ràng Buộc Kiến Trúc
- Clean Architecture 4-layer, dependency rule strictly enforced
- All infrastructure via pkg/ shared adapters
- Google Wire for DI, no service locator
- chi/v5 for HTTP (stdlib-compatible)

## Giải Pháp Đề Xuất

### Approach
Phased implementation: foundation → auth → routing → protocols → observability

### Alternatives Đã Xem Xét
| Alternative | Lý do loại bỏ |
|---|---|
| Envoy/Kong API Gateway | Not Go-native, can't embed MCP server |
| net/http stdlib router | Lacks middleware composition of chi |
| gRPC-Gateway code-gen | Too rigid for MCP + WebDAV multi-protocol |

### Trade-offs
- **Ưu điểm:** Full control, MCP-native, single binary, Go performance
- **Nhược điểm:** More code to maintain vs off-the-shelf gateway

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)
```
T01: Domain layer (entities, errors, events)        ← No dependencies
T02: Usecase ports (interfaces)                      ← After T01
T03: Config + Wire setup                             ← After T01
T04: Auth middleware (JWT + API Key)                  ← After T02, T03
T05: HTTP router skeleton (chi/v5)                   ← After T02
T06: REST handlers (8 namespaces)                    ← After T04, T05
T07: gRPC client registry (ServiceRegistry)          ← After T02
T08: Rate limiting middleware                         ← After T03
T09: Circuit breaker middleware                       ← After T07
T10: MCP server                                      ← After T07
T11: WebDAV proxy                                    ← After T07
T12: Observability (metrics, traces, health)          ← After T06
T13: Integration tests                                ← After all
```

### Danh Sách Tác Vụ
| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Domain layer: entities, errors, events | ARCH | vnp-gateway | - | 2h |
| T02 | Usecase port interfaces | ARCH | vnp-gateway | T01 | 2h |
| T03 | Config + Wire DI setup | TECH | vnp-gateway | T01 | 3h |
| T04 | Auth middleware (JWT RS256 + API Key) | FEAT | vnp-gateway | T02, T03 | 4h |
| T05 | HTTP router skeleton (chi/v5) | FEAT | vnp-gateway | T02 | 2h |
| T06 | REST handlers (8 namespaces, 50+ routes) | FEAT | vnp-gateway | T04, T05 | 8h |
| T07 | gRPC client registry + connection pool | FEAT | vnp-gateway | T02 | 4h |
| T08 | Rate limiting (Redis sliding window) | FEAT | vnp-gateway | T03 | 3h |
| T09 | Circuit breaker (sony/gobreaker) | FEAT | vnp-gateway | T07 | 2h |
| T10 | MCP server (16 tools, SSE transport) | FEAT | vnp-gateway | T07 | 6h |
| T11 | WebDAV proxy to ov-fs | FEAT | vnp-gateway | T07 | 3h |
| T12 | Observability (OTel, Prometheus, health) | FEAT | vnp-gateway | T06 | 4h |
| T13 | Integration test suite | QA | vnp-gateway | T06-T12 | 6h |

### Rollback Plan
Gateway is a new service — rollback = stop deployment, no data migration needed.

## Acceptance Criteria (Solution Level)
- [ ] SOL-AC-1: All 13 tasks completed and verified
- [ ] SOL-AC-2: Gateway routes to all 35 services successfully
- [ ] SOL-AC-3: Auth (JWT + API Key) works for all routes
- [ ] SOL-AC-4: Rate limiting enforced per tenant/tier
- [ ] SOL-AC-5: Circuit breaker isolates failures per service
- [ ] SOL-AC-6: MCP server exposes 16 tools
- [ ] SOL-AC-7: p99 gateway overhead < 50ms
- [ ] SOL-AC-8: Health check aggregates all 35 services
- [ ] SOL-AC-9: Docs updated (api.md, changelog.md)
