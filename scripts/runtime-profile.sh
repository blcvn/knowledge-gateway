#!/usr/bin/env bash

kg_runtime_profile_defaults() {
  local profile="${1:-}"
  if [[ -z "${profile}" ]]; then
    echo "KG_RUNTIME_PROFILE is required" >&2
    return 1
  fi
  case "${profile}" in
    pgvector-memgraph)
      export KG_RUNTIME_PROFILE="${profile}"
      export GRAPH_ADAPTER="memgraph"
      export KG_GRAPH_ENDPOINT="${KG_GRAPH_ENDPOINT:-bolt://memgraph:7687}"
      export KG_GRAPH_DATABASE="${KG_GRAPH_DATABASE:-kg}"
      export VECTOR_ADAPTER="pgvector"
      export KG_VECTOR_ENDPOINT="${KG_VECTOR_ENDPOINT:-}"
      export KG_VECTOR_COLLECTION="${KG_VECTOR_COLLECTION:-kg_vectors}"
      export FTS_ADAPTER="postgres"
      ;;
    pgvector-neo4j)
      export KG_RUNTIME_PROFILE="${profile}"
      export GRAPH_ADAPTER="neo4j"
      export KG_GRAPH_ENDPOINT="${KG_GRAPH_ENDPOINT:-neo4j://neo4j:7687}"
      export KG_GRAPH_DATABASE="${KG_GRAPH_DATABASE:-neo4j}"
      export VECTOR_ADAPTER="pgvector"
      export KG_VECTOR_ENDPOINT="${KG_VECTOR_ENDPOINT:-}"
      export KG_VECTOR_COLLECTION="${KG_VECTOR_COLLECTION:-kg_vectors}"
      export FTS_ADAPTER="postgres"
      ;;
    qdrant-memgraph)
      export KG_RUNTIME_PROFILE="${profile}"
      export GRAPH_ADAPTER="memgraph"
      export KG_GRAPH_ENDPOINT="${KG_GRAPH_ENDPOINT:-bolt://memgraph:7687}"
      export KG_GRAPH_DATABASE="${KG_GRAPH_DATABASE:-kg}"
      export VECTOR_ADAPTER="qdrant"
      export KG_VECTOR_ENDPOINT="${KG_VECTOR_ENDPOINT:-http://qdrant:6333}"
      export KG_VECTOR_COLLECTION="${KG_VECTOR_COLLECTION:-kg_vectors}"
      export FTS_ADAPTER="postgres"
      ;;
    qdrant-neo4j)
      export KG_RUNTIME_PROFILE="${profile}"
      export GRAPH_ADAPTER="neo4j"
      export KG_GRAPH_ENDPOINT="${KG_GRAPH_ENDPOINT:-neo4j://neo4j:7687}"
      export KG_GRAPH_DATABASE="${KG_GRAPH_DATABASE:-neo4j}"
      export VECTOR_ADAPTER="qdrant"
      export KG_VECTOR_ENDPOINT="${KG_VECTOR_ENDPOINT:-http://qdrant:6333}"
      export KG_VECTOR_COLLECTION="${KG_VECTOR_COLLECTION:-kg_vectors}"
      export FTS_ADAPTER="postgres"
      ;;
    milvus-neo4j)
      export KG_RUNTIME_PROFILE="${profile}"
      export GRAPH_ADAPTER="neo4j"
      export KG_GRAPH_ENDPOINT="${KG_GRAPH_ENDPOINT:-neo4j://neo4j:7687}"
      export KG_GRAPH_DATABASE="${KG_GRAPH_DATABASE:-neo4j}"
      export VECTOR_ADAPTER="milvus"
      export KG_VECTOR_ENDPOINT="${KG_VECTOR_ENDPOINT:-http://milvus:19530}"
      export KG_VECTOR_COLLECTION="${KG_VECTOR_COLLECTION:-kg_vectors}"
      export FTS_ADAPTER="postgres"
      ;;
    qdrant-nebula)
      export KG_RUNTIME_PROFILE="${profile}"
      export GRAPH_ADAPTER="nebula"
      export KG_GRAPH_ENDPOINT="${KG_GRAPH_ENDPOINT:-nebula://nebula:9669}"
      export KG_GRAPH_DATABASE="${KG_GRAPH_DATABASE:-kg}"
      export VECTOR_ADAPTER="qdrant"
      export KG_VECTOR_ENDPOINT="${KG_VECTOR_ENDPOINT:-http://qdrant:6333}"
      export KG_VECTOR_COLLECTION="${KG_VECTOR_COLLECTION:-kg_vectors}"
      export FTS_ADAPTER="postgres"
      ;;
    *)
      echo "unsupported KG_RUNTIME_PROFILE: ${profile}" >&2
      return 1
      ;;
  esac

  export EMBEDDING_PROVIDER="${EMBEDDING_PROVIDER:-deterministic}"
}

kg_runtime_profile_requirements() {
  case "${KG_RUNTIME_PROFILE:-}" in
    pgvector-memgraph|qdrant-memgraph)
      echo "postgres redis memgraph ${VECTOR_ADAPTER} fts-postgres"
      ;;
    pgvector-neo4j|qdrant-neo4j|milvus-neo4j)
      echo "postgres redis neo4j ${VECTOR_ADAPTER} fts-postgres"
      ;;
    qdrant-nebula)
      echo "postgres redis nebula qdrant fts-postgres"
      ;;
    *)
      return 1
      ;;
  esac
}
