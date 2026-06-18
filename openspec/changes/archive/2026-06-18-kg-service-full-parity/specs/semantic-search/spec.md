# semantic-search

## ADDED Requirements

### Requirement: Vector-backed semantic search
The system SHALL generate embeddings through a pluggable embedding provider and persist/query vectors through a vector adapter.

#### Scenario: Index a searchable node
- WHEN a node is projected for semantic search
- THEN the system SHALL build searchable text from the node content
- AND SHALL generate an embedding through the configured provider
- AND SHALL persist the embedding payload fields needed for downstream retrieval.

### Requirement: Distinct RAG retrieval
The system SHALL implement retrieval-augmented generation as a distinct pipeline from semantic search.

#### Scenario: Run a RAG query
- WHEN an actor calls the RAG search API or MCP tool
- THEN the system SHALL retrieve context using the vector-backed pipeline
- AND SHALL not treat the RAG request as a simple alias of semantic search
- AND SHALL preserve ACL, deletion, domain, lifecycle, and authority-score semantics in the retrieved context.
