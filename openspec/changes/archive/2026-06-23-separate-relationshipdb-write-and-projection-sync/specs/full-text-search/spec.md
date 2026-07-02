## MODIFIED Requirements

### Requirement: Full-text search operation

The KG Service MUST execute full-text search against `fts db` or an equivalent FTS backend adapter, not
against ad hoc scans of `relationshipdb`.

#### Scenario: Full-text search uses fts db

- **GIVEN** a caller executes full-text search
- **WHEN** the service performs keyword retrieval
- **THEN** it SHALL query `fts db` through the configured FTS adapter
- **AND** it SHALL return results from the FTS projection plane

### Requirement: Hybrid search combining FTS and semantic scores

The KG Service MUST treat hybrid search as an orchestration of `vectordb` semantic retrieval plus `fts db`
keyword retrieval.

#### Scenario: Hybrid search merges vectordb and fts db results

- **GIVEN** a caller executes hybrid search
- **WHEN** the service builds the candidate set
- **THEN** it SHALL retrieve one ranked list from `vectordb`
- **AND** it SHALL retrieve another ranked list from `fts db`
- **AND** it SHALL merge or rerank those lists according to the documented hybrid policy
