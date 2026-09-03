#!/bin/bash
# scripts/archive-old-services.sh

ARCHIVE_DIR="services/archived"
mkdir -p "$ARCHIVE_DIR"

OLD_SERVICES=(
    # Cognee (merged into kg-service)
    "cognee-cognify"
    "cognee-ingestion"
    "cognee-pipeline"
    "cognee-search"
    
    # Graphiti (merged into kg-service)
    "graphiti-ingestion"
    "graphiti-knowledge"
    "graphiti-pipeline"
    "graphiti-search"
    "graphiti-store"
    
    # Memobase (merged into memory-service)
    "memobase-context"
    "memobase-engine"
    "memobase-ingestion"
    "memobase-pipeline"
    
    # OpenViking (merged into storage-service)
    "ov-admin"
    "ov-crypto"
    "ov-fs"
    "ov-resource"
    "ov-search"
    "ov-session"
    
    # Supermemory (split: platform + memory + search)
    "sm-analytics"
    "sm-connector"
    "sm-document"
    "sm-engine"
    "sm-mcp"
    "sm-memory"
    "sm-profile"
    "sm-project"
    "sm-search"
    "sm-auth"
    
    # VNP Core (merged into platform/obs/pipeline)
    "vnp-admin"
    "vnp-dashboard"
    "vnp-event"
    "vnp-infra"
    "vnp-observability"
    "vnp-pipelines"
    
    # Zep (merged into memory-service)
    "zep-admin"
    "zep-core"
    "zep-graph"
    "zep-memory"
    "zep-search"
    "zep-thread"
    "zep-user"
    
    # BA Knowledge (merged into pipeline-service)
    "ba-knowledge-service"
    "ba-knowledge-worker"
)

for svc in "${OLD_SERVICES[@]}"; do
    src="services/$svc"
    if [ -d "$src" ]; then
        echo "Archiving: $src"
        mv "$src" "$ARCHIVE_DIR/$svc"
    else
        echo "Already removed or not found: $src"
    fi
done

echo "Done. Archived $(ls $ARCHIVE_DIR | wc -l | tr -d ' ') services."
