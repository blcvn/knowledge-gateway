#!/usr/bin/env bash
###############################################################################
# VNP Memory — Full Deployment Script for Dev Server (172.20.2.39)
#
# Usage:
#   ./deploy.sh                  # Full deploy (build + upload + start)
#   ./deploy.sh --config-only    # Sync config & restart (no image rebuild)
#   ./deploy.sh --images-only    # Build & upload images only (no restart)
###############################################################################
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
DEV_SERVER="172.20.2.39"
DEV_USER="ubuntu"
DEPLOY_DIR="/opt/vnp-memory"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COGNEE_SRC="${SCRIPT_DIR}/../../services/cognee"
GRAPHITI_SRC="${SCRIPT_DIR}/../../services/graphiti"
COGNEE_FRONTEND_SRC="${SCRIPT_DIR}/../../services/cognee/cognee-frontend"

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
    local src="$2"
    local image="vnp/${name}:latest"
    local tmp="/tmp/vnp-${name}.tar.gz"

    log "Building ${image} from ${src} ..."
    docker build -t "${image}" "${src}"

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

    if [ -f "${SCRIPT_DIR}/.env" ]; then
        scp "${SCRIPT_DIR}/.env" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
        ok "Synced .env + configs"
    else
        scp "${SCRIPT_DIR}/.env.example" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/.env"
        warn "No .env found — synced .env.example as .env"
    fi
}

remote_stop() {
    log "Stopping services on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down" \
        || true
}

remote_start() {
    log "Starting services on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile memory --profile ui up -d"
    ok "Services started!"
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
    --config-only)
        log "🔧 Config-only deployment"
        sync_config
        remote_stop
        remote_start
        ;;
    --images-only)
        log "📦 Images-only deployment"
        build_and_upload_image "cognee" "${COGNEE_SRC}"
        build_and_upload_image "graphiti" "${GRAPHITI_SRC}"
        build_and_upload_image "cognee-frontend" "${COGNEE_FRONTEND_SRC}"
        ;;
    --pull)
        log "⬇️  Pull latest public images"
        remote_pull
        ;;
    full|*)
        log "🚀 Full deployment to ${DEV_SERVER}"
        echo ""

        # Step 1: Build & upload custom images
        build_and_upload_image "cognee" "${COGNEE_SRC}"
        build_and_upload_image "graphiti" "${GRAPHITI_SRC}"
        build_and_upload_image "cognee-frontend" "${COGNEE_FRONTEND_SRC}"

        # Step 2: Sync configuration
        sync_config

        # Step 3: Stop → Start
        remote_stop
        remote_start

        echo ""
        ok "🎉 Full deployment complete!"
        echo ""
        echo "  Cognee API:     http://${DEV_SERVER}:8000"
        echo "  Cognee UI:      http://${DEV_SERVER}:3000"
        echo "  Graphiti MCP:   http://${DEV_SERVER}:8001"
        echo "  Zep MCP:        http://${DEV_SERVER}:8002"
        echo "  OpenViking MCP: http://${DEV_SERVER}:1933"
        echo "  Neo4j Browser:  http://${DEV_SERVER}:7474"
        echo ""
        ;;
esac
