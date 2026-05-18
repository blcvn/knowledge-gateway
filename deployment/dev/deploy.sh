#!/usr/bin/env bash
###############################################################################
# VNP Memory — Full Deployment Script for Dev Server (172.20.2.39)
#
# All application services are Go projects built from apps/ directory.
# Build context = monorepo root so Dockerfiles can access shared pkg/.
#
# Usage:
#   ./deploy.sh                   # Full deploy (all services)
#   ./deploy.sh --monolith        # Deploy Memory Monolith only (API+UI)
#   ./deploy.sh --monolith-full   # Deploy Monolith + sync nginx on gateway
#   ./deploy.sh --config-only     # Sync config & restart (no image rebuild)
#   ./deploy.sh --images-only     # Build & upload images only (no restart)
#   ./deploy.sh --nginx           # Sync nginx config to gateway server
###############################################################################
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
DEV_SERVER="172.20.2.39"
DEV_USER="ubuntu"
DEPLOY_DIR="/opt/vnp-memory"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONOREPO_ROOT="${SCRIPT_DIR}/../.."

# Gateway server (nginx + certbot)
GATEWAY_SERVER="172.20.2.16"
GATEWAY_USER="ubuntu"

# App Dockerfiles (all Go-based, relative to monorepo root)
MEMORY_DOCKERFILE="${MONOREPO_ROOT}/apps/memory/Dockerfile"
COGNEE_DOCKERFILE="${MONOREPO_ROOT}/apps/cognee/Dockerfile"
GRAPHITI_DOCKERFILE="${MONOREPO_ROOT}/apps/graphiti/Dockerfile"
OPENVIKING_DOCKERFILE="${MONOREPO_ROOT}/apps/OpenViking/Dockerfile"
ZEP_DOCKERFILE="${MONOREPO_ROOT}/apps/zep/Dockerfile"
MEMOBASE_DOCKERFILE="${MONOREPO_ROOT}/apps/memobase/Dockerfile"
SUPERMEMORY_DOCKERFILE="${MONOREPO_ROOT}/apps/supermemory/Dockerfile"
COGNEE_FRONTEND_SRC="${MONOREPO_ROOT}/apps/cognee/cognee-frontend"

# ── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# ── Functions ────────────────────────────────────────────────────────────────

build_and_upload_image() {
    local name="$1"
    local dockerfile="$2"
    local context="$3"
    local image="vnp/${name}:latest"
    local tmp="/tmp/vnp-${name}.tar.gz"

    log "Building ${image} from ${dockerfile} (context: ${context}) ..."
    docker build -t "${image}" -f "${dockerfile}" "${context}"

    log "Exporting ${image} ..."
    docker save "${image}" | gzip > "${tmp}"

    log "Uploading to ${DEV_SERVER} ..."
    scp "${tmp}" "${DEV_USER}@${DEV_SERVER}:/tmp/"
    ssh "${DEV_USER}@${DEV_SERVER}" "docker load < /tmp/vnp-${name}.tar.gz && rm /tmp/vnp-${name}.tar.gz"
    rm -f "${tmp}"

    ok "${image} loaded on ${DEV_SERVER}"
}

build_and_upload_frontend() {
    local name="cognee-frontend"
    local image="vnp/${name}:latest"
    local tmp="/tmp/vnp-${name}.tar.gz"

    log "Building ${image} from ${COGNEE_FRONTEND_SRC} ..."
    docker build -t "${image}" "${COGNEE_FRONTEND_SRC}"

    log "Exporting ${image} ..."
    docker save "${image}" | gzip > "${tmp}"

    log "Uploading to ${DEV_SERVER} ..."
    scp "${tmp}" "${DEV_USER}@${DEV_SERVER}:/tmp/"
    ssh "${DEV_USER}@${DEV_SERVER}" "docker load < /tmp/vnp-${name}.tar.gz && rm /tmp/vnp-${name}.tar.gz"
    rm -f "${tmp}"

    ok "${image} loaded on ${DEV_SERVER}"
}

sync_config() {
    log "Syncing configuration to ${DEV_SERVER}:${DEPLOY_DIR} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" "mkdir -p ${DEPLOY_DIR}/config"

    scp "${SCRIPT_DIR}/docker-compose.yml" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
    scp "${SCRIPT_DIR}/init-db.sql"        "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
    scp "${SCRIPT_DIR}/config/zep.yaml"    "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/config/"
    scp "${SCRIPT_DIR}/config/kgs.yaml"    "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/config/"
    scp "${SCRIPT_DIR}/config/memory.yaml" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/config/"

    if [ -f "${SCRIPT_DIR}/.env" ]; then
        scp "${SCRIPT_DIR}/.env" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
        ok "Synced .env + configs"
    else
        scp "${SCRIPT_DIR}/.env.example" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/.env"
        warn "No .env found — synced .env.example as .env"
    fi
}

sync_nginx() {
    log "Syncing nginx config to ${GATEWAY_SERVER} ..."
    scp "${SCRIPT_DIR}/nginx/c6-openledger-vn.conf" "${GATEWAY_USER}@${GATEWAY_SERVER}:/tmp/c6-openledger-vn.conf"
    ssh "${GATEWAY_USER}@${GATEWAY_SERVER}" "\
        docker cp /tmp/c6-openledger-vn.conf nginx:/etc/nginx/conf.d/c6-openledger-vn.conf && \
        docker exec nginx nginx -t && \
        docker exec nginx nginx -s reload && \
        rm -f /tmp/c6-openledger-vn.conf"
    ok "Nginx config updated and reloaded on ${GATEWAY_SERVER}"
    echo "  https://c6.openledger.vn → ${DEV_SERVER}:8080"
}

remote_stop() {
    log "Stopping services on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down" \
        || true
}

remote_stop_monolith() {
    log "Stopping Memory Monolith on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile monolith down" \
        || true
}

remote_start() {
    log "Starting services on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile memory --profile ui up -d --pull never"
    ok "Services started!"
}

remote_start_monolith() {
    log "Starting Memory Monolith on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile monolith up -d --pull never"
    ok "Memory Monolith started!"
}

remote_pull() {
    log "Pulling latest public images on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose pull neo4j postgres redis qdrant" \
        || true
}

# ── Main ─────────────────────────────────────────────────────────────────────

MODE="${1:-full}"

case "${MODE}" in
    --monolith)
        log "🧊 Memory Monolith deployment"
        echo ""
        build_and_upload_image "memory" "${MEMORY_DOCKERFILE}" "${MONOREPO_ROOT}"
        sync_config
        remote_stop_monolith
        remote_start_monolith
        echo ""
        ok "🎉 Memory Monolith deployed!"
        echo ""
        echo "  REST API + UI: http://${DEV_SERVER}:8080"
        echo "  MCP:           http://${DEV_SERVER}:8082"
        echo "  Health:        http://${DEV_SERVER}:8083/healthz"
        echo ""
        echo "  💡 Run './deploy.sh --nginx' to update nginx on gateway server."
        echo ""
        ;;
    --monolith-full)
        log "🧊 Memory Monolith + Nginx full deployment"
        echo ""
        build_and_upload_image "memory" "${MEMORY_DOCKERFILE}" "${MONOREPO_ROOT}"
        sync_config
        remote_stop_monolith
        remote_start_monolith
        sync_nginx
        echo ""
        ok "🎉 Full Monolith deployment complete!"
        echo ""
        echo "  Public URL:    https://c6.openledger.vn"
        echo "  REST API:      https://c6.openledger.vn/v1/"
        echo "  UI Console:    https://c6.openledger.vn/"
        echo "  MCP:           https://c6.openledger.vn/mcp/"
        echo ""
        ;;
    --nginx)
        log "🌐 Nginx config sync only"
        sync_nginx
        ;;
    --config-only)
        log "🔧 Config-only deployment"
        sync_config
        remote_stop
        remote_start
        ;;
    --images-only)
        log "📦 Images-only deployment (all services)"
        build_and_upload_image "memory" "${MEMORY_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "cognee" "${COGNEE_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "graphiti" "${GRAPHITI_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "openviking" "${OPENVIKING_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "zep" "${ZEP_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "memobase" "${MEMOBASE_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "supermemory" "${SUPERMEMORY_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_frontend
        ;;
    --pull)
        log "⬇️  Pull latest public images"
        remote_pull
        ;;
    full|*)
        log "🚀 Full deployment to ${DEV_SERVER}"
        echo ""

        # Step 1: Build & upload all service images (including monolith)
        build_and_upload_image "memory" "${MEMORY_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "cognee" "${COGNEE_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "graphiti" "${GRAPHITI_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "openviking" "${OPENVIKING_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "zep" "${ZEP_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "memobase" "${MEMOBASE_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_image "supermemory" "${SUPERMEMORY_DOCKERFILE}" "${MONOREPO_ROOT}"
        build_and_upload_frontend

        # Step 2: Sync configuration
        sync_config

        # Step 3: Stop → Start
        remote_stop
        remote_start

        echo ""
        ok "🎉 Full deployment complete!"
        echo ""
        echo "  Memory Console: https://c6.openledger.vn"
        echo "  Memory API:     https://c6.openledger.vn/v1/"
        echo "  Cognee API:     http://${DEV_SERVER}:8000"
        echo "  Cognee UI:      http://${DEV_SERVER}:3000"
        echo "  Graphiti API:   http://${DEV_SERVER}:8001"
        echo "  Zep API:        http://${DEV_SERVER}:8002"
        echo "  OpenViking API: http://${DEV_SERVER}:1933"
        echo "  Neo4j Browser:  http://${DEV_SERVER}:7474"
        echo ""
        ;;
esac
