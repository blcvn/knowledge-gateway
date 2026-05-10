---
id: ARCH-008
title: Absorb sm-mcp into vnp-gateway MCP Server
service: vnp-gateway
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

`sm-mcp` là một standalone MCP server dành riêng cho Supermemory engine. Tuy nhiên `vnp-gateway` đã có MCP server endpoint tại `:8082` hỗ trợ tools cho tất cả engines. sm-mcp chỉ là thin proxy → sm-memory/sm-search/sm-profile, hoàn toàn redundant.

## Kiến Trúc Mới

Absorb sm-mcp tools vào vnp-gateway MCP server:
- `add_memories` → proxy tới sm-engine.SmDocumentService.CreateDocument
- `search_memories` → proxy tới sm-search.SmSearchService.Search
- `get_profile` → proxy tới sm-engine.SmProfileService.GetProfile

## Phạm Vi Refactor

### Files cần sửa
- `services/vnp-gateway/internal/adapter/mcp/tools.go`: Register sm-mcp tools
- `services/vnp-gateway/internal/infra/config/config.go`: Add sm-engine gRPC target

### Files cần xóa (sau migration)
- `services/sm-mcp/` → toàn bộ

## Acceptance Criteria

- [ ] AC-1: MCP tools `add_memories`, `search_memories`, `get_profile` available via gateway :8082
- [ ] AC-2: MCP tools route correctly to sm-engine and sm-search gRPC endpoints
- [ ] AC-3: sm-mcp container no longer in docker-compose
