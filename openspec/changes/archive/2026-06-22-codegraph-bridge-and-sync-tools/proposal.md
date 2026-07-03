# Proposal: CodeGraph Bridge And Sync Tools

## Problem

Ngay cả khi đã có CodeGraph local index và ontology `code-graph`, repo vẫn thiếu bridge/tooling để:

- extract symbols từ CodeGraph
- map chúng sang contract của `kg-service`
- sync/reconcile data vào domain `code-graph`
- expose MCP tools cho semantic/fulltext/template-backed queries

## Proposed Solution

Tạo `examples/codegraph/` như một bridge/tooling package riêng:

- extractor + mapping layer
- `KGServiceAdapter` bám `/v1/kg/*` hiện có
- reconcile/full-sync workflow
- MCP tools cho persistent lookup

## Scope

### In scope

- `examples/codegraph/`
- mapping sang `NodeCreateRequest` và `RelationshipCreateRequest`
- sync dry-run/full-run
- MCP tools `kg_semantic_search`, `kg_fulltext_search`, `kg_code_template_query`

### Out of scope

- ontology bootstrap
- platform extension endpoints mới
- CodeGraph local bootstrap

## Success Criteria

- `sync:dry` hiển thị đúng symbols/edges được extract
- full sync ghi được data vào `kg-service`
- search/template-backed query trả về kết quả từ domain `code-graph`
