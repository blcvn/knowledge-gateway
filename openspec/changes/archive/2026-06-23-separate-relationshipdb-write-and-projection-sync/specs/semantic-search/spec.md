## MODIFIED Requirements

### Requirement: Vector-backed semantic search

The KG Service MUST execute semantic search and vector-backed retrieval against `vectordb`, not directly
against `relationshipdb`.

#### Scenario: Semantic search uses vectordb

- **GIVEN** a caller executes semantic search
- **WHEN** the service performs vector retrieval
- **THEN** it SHALL query `vectordb`
- **AND** it SHALL use projected vector documents as the retrieval source

#### Scenario: RAG retrieval uses the same vector-backed search plane

- **GIVEN** a caller executes a RAG retrieval flow
- **WHEN** the service retrieves semantic candidates
- **THEN** it SHALL query `vectordb`
- **AND** any later expansion or reranking SHALL happen after that vector-backed retrieval step

#### Scenario: Projection lag affects search freshness but not backend routing

- **GIVEN** vectordb projection is behind the source state in `relationshipdb`
- **WHEN** semantic search or RAG retrieval executes
- **THEN** the service SHALL still treat `vectordb` as the retrieval backend for that search mode
- **AND** any freshness caveat SHALL be surfaced through projection lag semantics rather than by silently switching the search path to `relationshipdb`
