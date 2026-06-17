## ADDED Requirements

### Requirement: Filter semantic search by ACL and deletion state

The KG Service MUST apply requester ACL and deletion-state constraints to every semantic search query.

#### Scenario: Search excludes deleted nodes

- **GIVEN** a node has been marked deleted in the source-of-truth store and projected to the vector store
- **WHEN** a semantic search is executed
- **THEN** that node is not returned

#### Scenario: Search excludes inaccessible nodes

- **GIVEN** a vector entry exists for a node outside the caller's ACL
- **WHEN** the caller performs semantic search
- **THEN** the node is not returned in results

#### Scenario: `POST /v1/kg/search/semantic` enforces ACL filters server-side

- **GIVEN** a caller submits a semantic search request
- **WHEN** the service executes `POST /v1/kg/search/semantic`
- **THEN** the vector filter always includes ACL and deletion constraints derived from the caller identity

### Requirement: Support domain-scoped semantic search

The KG Service MUST allow callers to restrict semantic search to one or more target domains.

#### Scenario: Search is limited to requested domains

- **GIVEN** the caller supplies a list of domain identifiers
- **WHEN** the search request executes
- **THEN** the vector filter restricts results to those domains

#### Scenario: Search without explicit domains uses all visible domains

- **GIVEN** the caller omits domain filters
- **WHEN** semantic search is executed
- **THEN** results can come from any domain visible to the caller

#### Scenario: `POST /v1/kg/search/rag` can retrieve from the same visible domain set

- **GIVEN** a caller invokes `POST /v1/kg/search/rag` without narrowing to a specific domain
- **WHEN** the service performs retrieval for downstream graph-RAG use
- **THEN** candidates are drawn only from the caller's visible domains

### Requirement: Apply lifecycle-aware search filtering only for configured domains

The KG Service MUST apply status-based search filtering only when the targeted domains declare lifecycle configuration.

#### Scenario: All targeted domains support status filtering

- **GIVEN** all requested domains declare lifecycle status configuration
- **WHEN** the search filter is built
- **THEN** the service includes status constraints using the union of valid values for those domains

#### Scenario: Mixed domains skip global status filter

- **GIVEN** the request spans domains where at least one has no lifecycle status configuration
- **WHEN** the search filter is built
- **THEN** the service does not enforce a global status filter that would incorrectly exclude unconfigured domains

### Requirement: Return projection metadata needed by downstream ranking and RAG

The KG Service MUST store and return the generic projection metadata needed for downstream ranking and graph-RAG expansion.

#### Scenario: Search result includes ownership and domain metadata

- **GIVEN** semantic search returns a vector match
- **WHEN** the result payload is returned to the caller or pipeline
- **THEN** it includes node identity, domain identity, and ownership metadata

#### Scenario: Authority score is available when configured

- **GIVEN** a domain defines authority ranking configuration
- **WHEN** its nodes are projected to the vector store and later retrieved
- **THEN** the result metadata includes authority score for downstream reranking

#### Scenario: `POST /v1/kg/search/semantic` returns metadata required by downstream expansion

- **GIVEN** semantic search finds matching projected nodes
- **WHEN** the response is returned from `POST /v1/kg/search/semantic`
- **THEN** each result includes enough metadata to support follow-up graph expansion or citation logic

### Requirement: Return consistent search API payloads and filter errors

The KG Service MUST return predictable success payloads and filter-validation errors for semantic and RAG search endpoints.

#### Scenario: `POST /v1/kg/search/semantic` returns ranked results

- **GIVEN** semantic retrieval succeeds
- **WHEN** the caller invokes `POST /v1/kg/search/semantic`
- **THEN** the service returns `200 OK`
- **AND** the response includes ranked results with score and metadata fields defined by the API contract

#### Scenario: Invalid domain filter returns validation error

- **GIVEN** the caller submits malformed or unsupported `domain_ids`
- **WHEN** the service validates `POST /v1/kg/search/semantic` or `POST /v1/kg/search/rag`
- **THEN** it returns `422 Unprocessable Entity`

#### Scenario: Search backend failure returns service error

- **GIVEN** the vector backend or embedding dependency is unavailable
- **WHEN** a search request is executed
- **THEN** the service returns a documented 5xx error without leaking internal backend details
