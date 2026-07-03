# Design: CodeGraph Bridge And Sync Tools

## Overview

Bridge này phụ thuộc vào ontology `code-graph` đã được freeze và chỉ dùng API surface hiện có của
`kg-service`.

## Core Adapter Contract

- `GET /healthz`
- `POST /v1/kg/write/nodes`
- `PUT /v1/kg/write/nodes/{id}` khi cần update
- `POST /v1/kg/write/relationships`
- `POST /v1/kg/search/semantic`
- `POST /v1/kg/search/fulltext`
- `POST /v1/kg/search/hybrid`
- `POST /v1/kg/read/template/{domain_id}/{template_name}`

## Mapping Rules

### Node mapping

- `external_ref = <project_id>:<symbol_id>`
- symbol -> `domain_id`, `node_type`, `properties`, `visibility`, `external_ref`

### Relationship mapping

- edge -> `rel_type`, `from_node_id`, `to_node_id`, `domain_id`, `properties`

## MCP Query Paths

- `kg_semantic_search` -> `/v1/kg/search/hybrid`
- `kg_fulltext_search` -> `/v1/kg/search/fulltext`
- `kg_code_template_query` -> `/v1/kg/read/template/code-graph/{template_name}`
